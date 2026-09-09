package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/internal/database"
	"github.com/Bernardo-Txa/printlab/internal/products"
	webfiles "github.com/Bernardo-Txa/printlab/web"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

const readyTimeout = 3 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("database configuration error")
	}

	db, err := database.New(context.Background(), database.Config{
		DatabaseURL: cfg.DatabaseURL,
		MaxConns:    cfg.DBMaxConns,
	})
	if err != nil {
		log.Fatal("database configuration error")
	}
	defer db.Close()

	if db.Configured() {
		log.Print("database configured")
	}

	addr := ":" + cfg.Port
	log.Printf("printlab web listening on %s", addr)

	if err := http.ListenAndServe(addr, newHandler(db)); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func newHandler(db *database.Database) http.Handler {
	var catalog catalogService
	if db != nil && db.Configured() {
		catalog = products.NewService(products.NewPostgresRepository(db.Pool()))
	}

	return newHandlerWithCatalog(db, catalog)
}

func newHandlerWithCatalog(db *database.Database, catalog catalogService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", homeHandler)
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /ready", readyHandler(db))
	mux.HandleFunc("GET /produtos", catalogHandler(catalog))
	mux.HandleFunc("GET /produtos/{slug}", productHandler(catalog))
	mux.Handle("GET /static/", staticFileHandler(webfiles.StaticFS()))

	return mux
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Home().Render(r.Context(), w); err != nil {
		log.Printf("render home: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "ok")
}

func readyHandler(db *database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		if db == nil || !db.Configured() {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			log.Print("database unavailable")
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	}
}

func staticFileHandler(staticFS fs.FS) http.Handler {
	fileServer := http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, "/static/")
		clean := path.Clean("/" + rel)
		name := strings.TrimPrefix(clean, "/")

		if rel == "" || strings.HasSuffix(rel, "/") || strings.Contains(clean, "/.") {
			http.NotFound(w, r)
			return
		}

		info, err := fs.Stat(staticFS, name)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}

		request := r.Clone(r.Context())
		request.URL.Path = "/static/" + name
		fileServer.ServeHTTP(w, request)
	})
}
