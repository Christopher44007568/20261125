package main

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"time"

	"highlightlab/sim"
)

//go:embed web/*
var webFiles embed.FS

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/timeline", handleTimeline)
	mux.HandleFunc("/api/ledger.csv", handleLedger)

	sub, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	server := &http.Server{
		Addr:              *addr,
		Handler:           requestLogger(mux),
		ReadHeaderTimeout: 4 * time.Second,
	}
	log.Printf("highlight lab listening on http://%s", *addr)
	log.Fatal(server.ListenAndServe())
}

func handleTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	seconds := 6.0
	if raw := r.URL.Query().Get("seconds"); raw != "" {
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil || parsed < 2 || parsed > 20 {
			http.Error(w, "seconds must be between 2 and 20", http.StatusBadRequest)
			return
		}
		seconds = parsed
	}
	timeline := sim.BuildTimeline(seconds)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(timeline); err != nil {
		log.Printf("encode timeline: %v", err)
	}
}

func handleLedger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	timeline := sim.BuildTimeline(6)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="highlight-ledger.csv"`)
	writer := csv.NewWriter(w)
	header := []string{
		"frame", "time_s", "eye", "object", "center_x", "center_y",
		"stretch_px", "peak", "energy", "history_x", "history_y",
		"history_energy", "composite_energy", "history_eye", "capped",
	}
	if err := writer.Write(header); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, eye := range timeline.Eyes {
		for _, frame := range eye.Frames {
			for _, track := range frame.Tracks {
				row := []string{
					strconv.Itoa(frame.Index),
					fmt.Sprintf("%.4f", frame.Time),
					eye.Label,
					track.ObjectID,
					fmt.Sprintf("%.3f", track.Center.X),
					fmt.Sprintf("%.3f", track.Center.Y),
					fmt.Sprintf("%.3f", track.Stretch),
					fmt.Sprintf("%.6f", track.Peak),
					fmt.Sprintf("%.6f", track.Energy),
					fmt.Sprintf("%.3f", track.History.Center.X),
					fmt.Sprintf("%.3f", track.History.Center.Y),
					fmt.Sprintf("%.6f", track.History.SampledEnergy),
					fmt.Sprintf("%.6f", track.History.CompositeEnergy),
					eye.Label,
					strconv.FormatBool(track.History.EnergyCapped),
				}
				if err := writer.Write(row); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Printf("write ledger: %v", err)
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
