package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	webfiles "github.com/Bernardo-Txa/printlab/web"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("printlab web listening on %s", addr)

	if err := http.ListenAndServe(addr, newHandler()); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", homeHandler)
	mux.HandleFunc("GET /health", healthHandler)
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
