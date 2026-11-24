package main

import (
	"fmt"
	"strings"
	"time"
)

func main() {
	scenario := buildScenario()
	reports := runAll(scenario)
	printSummary(reports)
	for _, report := range reports {
		printReport(report)
	}
}

func printSummary(reports []SimulationReport) {
	fmt.Println("=== strategy comparison ===")
	fmt.Printf("%-22s %-8s %-14s %-14s\n", "strategy", "outputs", "blocked-ticks", "changed-frames")
	for _, report := range reports {
		changed := 0
		for _, delta := range report.VisualDeltas {
			if len(delta.Changed) > 0 {
				changed++
			}
		}
		fmt.Printf("%-22s %-8d %-14d %-14d\n",
			report.Name, len(report.Frames), len(report.MissedTicks), changed)
	}
}

func printReport(report SimulationReport) {
	fmt.Printf("\n=== %s ===\n", report.Name)
	fmt.Println("output frames:")
	fmt.Printf("%-4s %-8s %-18s %-18s %-18s %-16s\n",
		"tick", "time", "world(seq,age)", "ui(seq,age)", "hand(seq,age)", "visual change")

	for index, frame := range report.Frames {
		change := "first-frame"
		if index > 0 {
			change = formatChanged(report.VisualDeltas[index-1].Changed)
		}
		fmt.Printf("%-4d %-8s %-18s %-18s %-18s %-16s %s\n",
			frame.DisplayTick,
			formatDuration(frame.At),
			formatSource(frame.Layers[World]),
			formatSource(frame.Layers[UI]),
			formatSource(frame.Layers[Hand]),
			change,
			frame.Digest,
		)
	}

	fmt.Println("\nlayer provenance:")
	for _, frame := range report.Frames {
		parts := make([]string, 0, len(layers))
		for _, layer := range layers {
			source := frame.Layers[layer]
			parts = append(parts, fmt.Sprintf("%s=f%d@t%d age=%s %s",
				layer,
				source.Sequence,
				source.AvailableAt/displayInterval,
				formatDuration(source.Age),
				source.Status,
			))
		}
		fmt.Printf("tick%02d: %s\n", frame.DisplayTick, strings.Join(parts, " | "))
	}

	fmt.Printf("\nmissed/blocked display ticks: %d\n", len(report.MissedTicks))
	for _, missed := range report.MissedTicks {
		fmt.Printf("tick%02d %s: waiting=%s\n",
			missed.DisplayTick,
			formatDuration(missed.At),
			strings.Join(names(missed.WaitingOn), ","),
		)
	}

	changes := 0
	for _, delta := range report.VisualDeltas {
		if len(delta.Changed) > 0 {
			changes++
		}
	}
	fmt.Printf("outputs=%d changed-frames=%d blocking-ticks=%d\n",
		len(report.Frames), changes, len(report.MissedTicks))
}

func formatSource(source LayerSource) string {
	return fmt.Sprintf("f%d,%s,%s",
		source.Sequence,
		formatDuration(source.Age),
		source.Status,
	)
}

func formatChanged(changed []LayerName) string {
	if len(changed) == 0 {
		return "none"
	}
	return strings.Join(names(changed), "+")
}

func names(values []LayerName) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func formatDuration(value time.Duration) string {
	ticks := float64(value) / float64(displayInterval)
	return fmt.Sprintf("%.2ft", ticks)
}
