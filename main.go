package main

import (
	"fmt"
	"math"
	"os"
)

const heatMaxDE = 10.0

func run() error {
	ps := patches()
	ss := scenarios()

	orig := make([]Vec3, len(ps))
	for i, p := range ps {
		orig[i] = p.RGB
	}

	mosaic := [][]Vec3{orig}
	heat := make([][]float64, len(ss))

	fmt.Printf("%-46s", "scenario \\ patch")
	for _, p := range ps {
		fmt.Printf(" %7.7s", p.Name)
	}
	fmt.Println("      max     mean")

	for si, s := range ss {
		row := make([]Vec3, len(ps))
		heat[si] = make([]float64, len(ps))
		maxDE, sumDE := 0.0, 0.0
		for i, p := range ps {
			out := s.RoundTrip(p.RGB)
			row[i] = out
			dE := DeltaEFromSRGB(p.RGB, out)
			heat[si][i] = dE
			sumDE += dE
			if dE > maxDE {
				maxDE = dE
			}
		}
		mosaic = append(mosaic, row)
		fmt.Printf("%-46s", s.Name)
		for _, dE := range heat[si] {
			fmt.Printf(" %7.3f", dE)
		}
		fmt.Printf("  %7.3f %7.3f\n", maxDE, sumDE/float64(len(ps)))
	}

	fmt.Println("\npartial adaptation sweep (D65 white, Bradford both ways):")
	for _, t := range []float64{0, 0.25, 0.5, 0.75, 1} {
		s := Scenario{
			FwdCAT: CATBradford, FwdT: t,
			BwdCAT: CATBradford, BwdT: t,
			BwdFrom: WhiteD50, BwdTo: WhiteD65,
		}
		dE := DeltaEFromSRGB(Vec3{1, 1, 1}, s.RoundTrip(Vec3{1, 1, 1}))
		fmt.Printf("  t=%4.2f  dE2000=%.4f\n", t, dE)
	}

	if err := os.MkdirAll("artifacts", 0o755); err != nil {
		return err
	}
	if err := RenderMosaic("artifacts/patches.png", mosaic); err != nil {
		return err
	}
	if err := RenderHeatmap("artifacts/heatmap.png", heat, heatMaxDE); err != nil {
		return err
	}

	worst := 0.0
	for _, row := range heat {
		for _, dE := range row {
			worst = math.Max(worst, dE)
		}
	}
	fmt.Printf("\nwrote artifacts/patches.png and artifacts/heatmap.png (worst dE2000 = %.3f)\n", worst)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
