package main

import (
	"flag"
	"fmt"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"

	"hairroots/internal/poisson"
	"hairroots/internal/render"
	"hairroots/internal/scalp"
)

func main() {
	out := flag.String("out", "artifacts", "output directory")
	rMin := flag.Float64("rmin", 0.034, "minimum Poisson-disc radius (crown spacing)")
	seed := flag.Int64("seed", 20261125, "RNG seed")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}

	s := scalp.New()
	rng := rand.New(rand.NewSource(*seed))
	sampler := poisson.New(s, rng, *rMin)
	roots := sampler.Run(24)

	fmt.Printf("generated roots: %d (rMin=%.4f)\n", len(roots), *rMin)
	if err := writeRootsCSV(filepath.Join(*out, "roots.csv"), roots); err != nil {
		fatal(err)
	}

	const size = 1024

	// Perspective view from the front-top-right.
	persp := render.NewCanvas(size, size)
	persp.Fill(color.RGBA{R: 16, G: 18, B: 24, A: 255})
	camPersp := render.NewCamera(
		scalp.Vec3{X: 2.0, Y: 2.3, Z: 3.0},
		scalp.Vec3{X: 0, Y: 1, Z: 0},
		0.62, false,
	)
	camPersp.Zoom = 4.2
	render.PaintScalp(persp, camPersp, s, [3]float64{0.42, 0.34, 0.29})
	render.DrawRoots(persp, camPersp, s, roots)
	if err := writePNG(filepath.Join(*out, "roots_perspective.png"), persp); err != nil {
		fatal(err)
	}

	// Orthographic side view from the ear direction (+X).
	side := render.NewCanvas(size, size)
	side.Fill(color.RGBA{R: 16, G: 18, B: 24, A: 255})
	camSide := render.NewCamera(
		scalp.Vec3{X: 4.0, Y: 0.35, Z: 0},
		scalp.Vec3{X: 0, Y: 1, Z: 0},
		1.0, true,
	)
	render.PaintScalp(side, camSide, s, [3]float64{0.42, 0.34, 0.29})
	render.DrawRoots(side, camSide, s, roots)
	if err := writePNG(filepath.Join(*out, "roots_side.png"), side); err != nil {
		fatal(err)
	}

	// Parameter-space density heat map, normalized by physical bin area.
	grid := render.BuildDensityGrid(s, roots, 48, 96)
	heat := render.NewCanvas(1024, 512)
	grid.DrawHeatMap(heat, s)
	if err := writePNG(filepath.Join(*out, "density_heatmap.png"), heat); err != nil {
		fatal(err)
	}

	// Top-down orthographic view carrying the hair direction vectors.
	top := render.NewCanvas(size, size)
	top.Fill(color.RGBA{R: 16, G: 18, B: 24, A: 255})
	camTop := render.NewCamera(
		scalp.Vec3{X: 0, Y: 4.2, Z: 1e-3},
		scalp.Vec3{X: 0, Y: 0, Z: 1},
		1.0, true,
	)
	render.PaintScalp(top, camTop, s, [3]float64{0.42, 0.34, 0.29})
	render.DrawFlowGlyphs(top, camTop, s, roots, max(1, len(roots)/260))
	if err := writePNG(filepath.Join(*out, "flow_vectors_top.png"), top); err != nil {
		fatal(err)
	}

	report := Analyze(s, roots, grid)
	if err := os.WriteFile(filepath.Join(*out, "stats.txt"), []byte(report), 0o644); err != nil {
		fatal(err)
	}
	fmt.Print(report)
}

func writePNG(path string, c *render.Canvas) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, c.Image())
}

func writeRootsCSV(path string, roots []poisson.Root) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintln(f, "id,theta,phi,x,y,z,weight,radius,flow_u,flow_v,flow_x,flow_y,flow_z"); err != nil {
		return err
	}
	for _, r := range roots {
		_, err := fmt.Fprintf(f,
			"%d,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
			r.ID, r.Theta, r.Phi,
			r.Position.X, r.Position.Y, r.Position.Z,
			r.Weight, r.Radius, r.FlowU, r.FlowV,
			r.FlowWorld.X, r.FlowWorld.Y, r.FlowWorld.Z)
		if err != nil {
			return err
		}
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
