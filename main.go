package main

import (
	"embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

//go:embed web
var webAssets embed.FS

func main() {
	outDir := flag.String("out", "artifacts", "directory for PNGs and the comparison page")
	serveAddr := flag.String("serve", "", "serve the generated page, for example 127.0.0.1:8080")
	flag.Parse()

	start := time.Now()
	report, err := exportLab(*outDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ditherlab:", err)
		os.Exit(1)
	}
	for _, scene := range report.Scenes {
		none := scene.Variants["none"]
		ordered := scene.Variants["ordered"]
		diffusion := scene.Variants["diffusion"]
		fmt.Printf("%-7s flatErr=%.0f  RMSE none=%.2f ordered=%.2f diffusion=%.2f\n",
			scene.ID, none.FlatError, none.RMSE, ordered.RMSE, diffusion.RMSE)
	}
	fmt.Printf("wrote 9 PNGs and comparison page to %s in %s\n", *outDir, time.Since(start).Round(time.Millisecond))

	if *serveAddr == "" {
		return
	}
	fileServer := http.FileServer(http.Dir(*outDir))
	fmt.Printf("serving http://%s/\n", *serveAddr)
	if err := http.ListenAndServe(*serveAddr, cacheControl(fileServer)); err != nil {
		fmt.Fprintln(os.Stderr, "serve:", err)
		os.Exit(1)
	}
}

func cacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}
