package main

import (
	"flag"
	"fmt"
	"os"

	"normalcone"
)

func main() {
	samples := flag.Int("samples", 1024, "number of deterministic Fibonacci-sphere camera samples per scenario")
	flag.Parse()

	scenarios := normalcone.BuildScenarios(*samples)
	reports := normalcone.Evaluate(scenarios)
	report := normalcone.FormatTextReport(reports)

	if _, err := fmt.Println(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
