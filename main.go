package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	format := flag.String("format", "table", "output format: table, json, or csv")
	outPath := flag.String("out", "", "optional output file; defaults to stdout")
	flag.Parse()

	rows := RunSuite(DefaultSuite())

	var writer *os.File
	if *outPath != "" {
		file, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create output: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		writer = file
	} else {
		writer = os.Stdout
	}

	var err error
	switch *format {
	case "table":
		err = WriteTable(writer, rows)
	case "json":
		err = WriteJSON(writer, rows)
	case "csv":
		err = WriteCSV(writer, rows)
	default:
		err = fmt.Errorf("unsupported format %q", *format)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "write report: %v\n", err)
		os.Exit(1)
	}
}
