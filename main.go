// Command brepsection cuts one BRep solid along three different directions,
// extracts and validates the section curves, and renders a solid cutaway, a
// 2D section montage and an endpoint error heat map. It uses only the Go
// standard library.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"brepsection/brep"
	"brepsection/geom"
	"brepsection/render"
	"brepsection/section"
)

func main() {
	outDir := "artifacts"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}

	solid := brep.SampleSolid()
	planes := cuttingPlanes(solid)

	// Run the full pipeline in both precision modes for every direction.
	results := map[string][]*section.Report{}
	var allOK = true
	for _, mode := range section.StandardModes {
		var reps []*section.Report
		for _, pl := range planes {
			rep := section.Extract(solid, pl, mode)
			reps = append(reps, rep)
			if !rep.Closed || !rep.OrderConsistent || rep.SelfIntersect || !rep.WindingConsistent {
				allOK = false
			}
		}
		results[mode.Name] = reps
	}

	f64 := results["float64"]
	f32 := results["float32"]

	// 1. Solid cutaway: one view per cutting direction, stacked vertically.
	cutH := 360
	cutaway := render.NewCanvas(1200, cutH*len(planes)+40, render.BgColor())
	for i, pl := range planes {
		panel := render.Cutaway(solid, f64[i], pl, 1200, cutH)
		cutaway.Blit(panel, 0, 20+i*cutH)
	}
	writePNG(filepath.Join(outDir, "solid_cutaway.png"), cutaway)

	// 2. 2D section montage for both precision modes.
	montage := render.Montage(append(append([]*section.Report{}, f64...), f32...), 1200, 380)
	writePNG(filepath.Join(outDir, "section_montage.png"), montage)

	// 3. Endpoint error heat map for both precision modes.
	heat := render.Heatmap(append(append([]*section.Report{}, f64...), f32...), 1200, 320)
	writePNG(filepath.Join(outDir, "endpoint_error_heatmap.png"), heat)

	printReport(solid, planes, f64, f32)
	if !allOK {
		os.Exit(1)
	}
}

// cuttingPlanes returns the three required directions: an axis-aligned plane
// normal to the extrusion, an axis-aligned plane tangent to the circular via,
// and an oblique plane through the solid center.
func cuttingPlanes(solid *brep.BRep) []geom.Plane {
	center := solid.AABB().Center()
	return []geom.Plane{
		geom.NewPlane(geom.V3(0, 0, 1.25), geom.V3(0, 0, 1), "z-extrusion-cut"),
		geom.NewPlane(geom.V3(6, 0, 0), geom.V3(1, 0, 0), "x-tangent-via"),
		geom.NewPlane(center, geom.V3(0.3, 0.7, 0.65), "oblique-center-cut"),
	}
}

func writePNG(path string, c *render.Canvas) {
	f, err := os.Create(path)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	if err := c.SavePNG(f); err != nil {
		fatal(err)
	}
	fmt.Println("wrote", path)
}

func printReport(solid *brep.BRep, planes []geom.Plane, f64, f32 []*section.Report) {
	fmt.Printf("\nBRep: %s  vertices=%d faces=%d  bbox diag=%.4f\n\n",
		solid.Name, len(solid.Points), len(solid.Faces), solid.AABB().Diagonal())
	for i, pl := range planes {
		a, b := f64[i], f32[i]
		fmt.Printf("== %s (normal=%v) ==\n", pl.Name, pl.Normal)
		fmt.Printf("  float64: rawSeg=%d junctions=%d loops=%d closed=%v orderOK=%v windingOK=%v selfIntersect=%v maxGap=%s\n",
			a.RawSegments, a.Junctions, len(a.Loops), a.Closed, a.OrderConsistent, a.WindingConsistent,
			a.SelfIntersect, sci(a.MaxGap))
		for _, l := range a.Loops {
			fmt.Printf("    loop %d level=%d ccw=%v nodes=%d gap=%s\n",
				l.Index, l.Level, l.CCW, len(l.Nodes), sci(l.ClosureGap))
		}
		fmt.Printf("  float32: closed=%v orderOK=%v windingOK=%v selfIntersect=%v maxGap=%s\n\n",
			b.Closed, b.OrderConsistent, b.WindingConsistent, b.SelfIntersect, sci(b.MaxGap))
	}
}

func sci(v float64) string {
	return strconv.FormatFloat(v, 'G', 3, 64)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(2)
}
