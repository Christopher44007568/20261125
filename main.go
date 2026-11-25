package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"meshscope/internal/scene"
)

//go:embed web
var webAssets embed.FS

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	registry := scene.BuildRegistry()
	web, err := fs.Sub(webAssets, "web")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","solver":"meshscope-go"}`))
	})
	mux.HandleFunc("/api/scenes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, scene.List(registry))
	})
	mux.HandleFunc("/api/scene/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/scene/")
		s, ok := registry.ByID[id]
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, s)
	})
	mux.Handle("/", http.FileServer(http.FS(web)))

	server := &http.Server{
		Addr:              *addr,
		Handler:           logging(mux),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	shown := *addr
	if strings.HasPrefix(shown, ":") {
		shown = "localhost" + shown
	}
	log.Printf("MESHSCOPE ready · http://%s", shown)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		log.Printf("json encode: %v", err)
	}
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if !strings.HasPrefix(r.URL.Path, "/api/scene/") && !strings.Contains(r.URL.Path, "health") {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}
