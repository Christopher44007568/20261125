package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"notificationframe"
)

func main() {
	outDir := flag.String("out", filepath.Join("artifacts", "notifyframe"), "directory for report.json and report.md; use - for compact stdout JSON")
	flag.Parse()

	report := notify.DefaultReport()

	if *outDir == "-" {
		if err := report.WriteJSON(os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if !report.Pass {
			os.Exit(1)
		}
		return
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(2)
	}

	jsonPath := filepath.Join(*outDir, "report.json")
	markdownPath := filepath.Join(*outDir, "report.md")
	if err := writeJSON(jsonPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "write JSON: %v\n", err)
		os.Exit(2)
	}
	if err := writeMarkdown(markdownPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "write Markdown: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("wrote %s\n", jsonPath)
	fmt.Printf("wrote %s\n", markdownPath)
	if !report.Pass {
		os.Exit(1)
	}
}

func writeJSON(path string, report *notify.Report) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return report.WriteJSON(file)
}

func writeMarkdown(path string, report *notify.Report) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return report.WriteMarkdown(file)
}
