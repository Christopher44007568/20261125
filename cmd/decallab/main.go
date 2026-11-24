package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"stereodecal"
)

func main() {
	outDir := flag.String("out", filepath.Join("artifacts", "decallab"), "directory for JSON/Markdown/CSV records; use - for JSON on stdout")
	flag.Parse()

	report := decallab.Run()

	if *outDir == "-" {
		if err := report.WriteJSON(os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		exitBasedOnPass(report)
		return
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(2)
	}

	files := map[string]func(*os.File) error{
		"report.json": func(file *os.File) error { return report.WriteJSON(file) },
		"report.md":   func(file *os.File) error { return report.WriteMarkdown(file) },
	}
	csvFiles := map[string]func(*os.File) error{
		"normal_probes.csv":  func(file *os.File) error { return decallab.WriteNormalProbesCSV(report, file) },
		"decal_scales.csv":   func(file *os.File) error { return decallab.WriteDecalScalesCSV(report, file) },
		"shadow_edges.csv":   func(file *os.File) error { return decallab.WriteShadowEdgesCSV(report, file) },
		"binocular_diff.csv": func(file *os.File) error { return decallab.WriteBinocularCSV(report, file) },
	}

	for name, writer := range files {
		if err := writeFile(filepath.Join(*outDir, name), writer); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", name, err)
			os.Exit(2)
		}
		fmt.Printf("wrote %s\n", filepath.Join(*outDir, name))
	}
	for name, writer := range csvFiles {
		if err := writeFile(filepath.Join(*outDir, name), writer); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", name, err)
			os.Exit(2)
		}
		fmt.Printf("wrote %s\n", filepath.Join(*outDir, name))
	}

	exitBasedOnPass(report)
}

func writeFile(path string, write func(*os.File) error) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return write(file)
}

func exitBasedOnPass(report *decallab.Report) {
	if !report.Pass {
		os.Exit(1)
	}
}
