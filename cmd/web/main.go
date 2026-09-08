package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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
	mux.HandleFunc("GET /health", healthHandler)

	return mux
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "ok")
}
