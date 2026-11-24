package main

import (
	"fmt"
	"os"
)

func main() {
	outputDir := "artifacts/output"
	if len(os.Args) > 1 {
		outputDir = os.Args[1]
	}

	report, err := run(outputDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("rendered %d×%d stereo scene to %s\n", report.Width, report.Height, outputDir)
	for _, stats := range report.ErrorStats {
		fmt.Printf(
			"eye %d %s: mean linear RMSE %.6f, max %.6f, mean |gamma delta| %.6f, valid pixels %d\n",
			stats.Eye,
			stats.Mode,
			stats.MeanLinearRMSE,
			stats.MaxLinearRMSE,
			stats.MeanGammaDelta,
			stats.ValidGammaPixels,
		)
	}
}
