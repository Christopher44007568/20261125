package main

import (
	"encoding/csv"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"handlate/internal/metrics"
	"handlate/internal/render"
	"handlate/internal/sim"
	"handlate/internal/strategy"
)

const (
	outDir  = "artifacts"
	headerH = 26
	margin  = 8
)

func main() {
	scenarios := []sim.Scenario{sim.WaveScenario{}, sim.GrabScenario{}, sim.LossScenario{}}
	kinds := []strategy.Kind{strategy.WorldSync, strategy.LateCompose, strategy.Freeze}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}
	csvFile, err := os.Create(filepath.Join(outDir, "metrics.csv"))
	if err != nil {
		fatal(err)
	}
	defer csvFile.Close()
	w := csv.NewWriter(csvFile)
	defer w.Flush()
	_ = w.Write([]string{"scenario", "strategy", "traj_rmse_m", "traj_p95_m", "pinch_rmse_m",
		"contact_rmse_m", "through_violations", "over_hide_pct", "avg_hand_age_ms",
		"avg_world_age_ms", "loss_window_rmse_m"})

	for _, scen := range scenarios {
		runScenario(scen, kinds, w)
	}
	fmt.Println("artifacts written to", outDir)
}

func runScenario(scen sim.Scenario, kinds []strategy.Kind, w *csv.Writer) {
	tl := sim.Build(scen)
	fmt.Printf("\n=== %s (%s) duration=%dms ===\n", scen.Title(), scen.ID(), scen.Duration())
	fmt.Printf("%-13s %9s %9s %9s %9s %8s %11s %8s %9s %10s\n",
		"strategy", "trajRMSE", "trajP95", "pinchRMSE", "contact", "through", "overHide", "handAge", "worldAge", "lossWin")

	results := make([]metrics.Result, 0, len(kinds))
	for _, k := range kinds {
		r := metrics.Evaluate(scen, tl, k)
		results = append(results, r)
		row := r.Row()
		fmt.Printf("%-13s %9s %9s %9s %9s %8s %11s %8s %9s %10s\n",
			row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8], row[9])
		_ = w.Write(append([]string{scen.ID()}, r.Row()...))
	}

	if err := writeFrames(scen, tl, kinds); err != nil {
		fatal(err)
	}
	if err := writePlot(scen, tl); err != nil {
		fatal(err)
	}
}

// writeFrames renders three representative instants per scenario.
func writeFrames(scen sim.Scenario, tl *sim.Timeline, kinds []strategy.Kind) error {
	picks := pickTimes(scen, tl)
	for _, t := range picks {
		df := frameAt(tl, t)
		width := margin + len(kinds)*(render.PanelW+margin)
		height := headerH + render.PanelH + margin
		c := render.NewCanvas(width, height)
		c.Fill(render.ColDark)
		title := fmt.Sprintf("%s T=%dMS", scen.ID(), t)
		c.Text(margin, 12, title, render.ColWhite)

		for i, k := range kinds {
			est := strategy.Build(k, df)
			ox := margin + i*(render.PanelW+margin)
			render.DrawPanel(c, ox, headerH, scen, df, est)
		}
		path := filepath.Join(outDir, fmt.Sprintf("%s_frame_%04dms.png", scen.ID(), t))
		if err := savePNG(path, c); err != nil {
			return err
		}
	}
	return nil
}

func writePlot(scen sim.Scenario, tl *sim.Timeline) error {
	c := render.NewCanvas(render.PlotW+2*margin, render.PlotH+2*margin)
	c.Fill(render.ColDark)
	render.DrawErrorChart(c, margin, margin, scen, tl)
	path := filepath.Join(outDir, scen.ID()+"_trajectory.png")
	return savePNG(path, c)
}

func pickTimes(scen sim.Scenario, tl *sim.Timeline) []int {
	d := scen.Duration()
	candidates := []int{d / 4, d / 2, (3 * d) / 4}
	if scen.ID() == "loss" {
		candidates = []int{1200, 2100, 2700}
	}
	out := make([]int, 0, len(candidates))
	for _, want := range candidates {
		best := tl.Displays[0].NowMs
		for _, df := range tl.Displays {
			if abs(df.NowMs-want) < abs(best-want) {
				best = df.NowMs
			}
		}
		out = append(out, best)
	}
	return out
}

func frameAt(tl *sim.Timeline, t int) sim.DisplayFrame {
	for _, df := range tl.Displays {
		if df.NowMs == t {
			return df
		}
	}
	return tl.Displays[0]
}

func savePNG(path string, c *render.Canvas) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, c.Img)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
