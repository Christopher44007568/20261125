package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	serveAddr := flag.String("serve", "", "start interactive status UI at address, e.g. :8080")
	outDir := flag.String("out", "artifacts", "directory for generated reports, trace and PNGs")
	flag.Parse()

	if *serveAddr != "" {
		log.Printf("degradation console listening on http://localhost%s", *serveAddr)
		if err := runServer(*serveAddr); err != nil {
			log.Fatal(err)
		}
		return
	}

	report, err := GenerateAll(*outDir, DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote report for %d scenarios (%d episodes, %d chains) to %s\n",
		report.Summary.ScenarioCount, report.Summary.EpisodeCount,
		report.Summary.ChainCount, *outDir)
}
