package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"ruledintersect/intersect"
	"ruledintersect/render"
	"ruledintersect/scenes"
)

type sceneReport struct {
	Name              string   `json:"name"`
	Kind              string   `json:"kind"`
	RawSegments       int      `json:"raw_segments"`
	ValidSegments     int      `json:"valid_segments"`
	DuplicateSegments int      `json:"duplicate_segments"`
	StitchedCurves    int      `json:"stitched_curves"`
	RejectedSegments  int      `json:"rejected_segments"`
	MaxSurfaceGap     float64  `json:"max_surface_gap"`
	CompositePPM      string   `json:"composite_ppm"`
	IndividualPPMs    []string `json:"individual_ppms"`
}

func main() {
	out := flag.String("out", "artifacts", "output directory")
	which := flag.String("case", "all", "all, tangent, near_parallel, or crossing")
	flag.Parse()

	selected, err := selectScenes(*which)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	opts := intersect.DefaultOptions()
	var reports []sceneReport
	for _, scene := range selected {
		result := intersect.Surfaces(scene.A, scene.B, opts)
		title := sceneTitle(scene.Kind) + " / SURFACE-SURFACE INTERSECTION"
		im := render.ThreeViews(title, scene.A, scene.B, result)
		composite := filepath.Join(*out, scene.Name+"_3views.ppm")
		if err := writePPM(composite, im); err != nil {
			fatal(err)
		}
		var paths []string
		for i, name := range []string{"front", "top", "side"} {
			x, y, w, h := render.PanelRect(i)
			crop := render.Crop(im, x, y, w, h)
			path := filepath.Join(*out, scene.Name+"_"+name+".ppm")
			if err := writePPM(path, crop); err != nil {
				fatal(err)
			}
			paths = append(paths, path)
		}
		reports = append(reports, sceneReport{
			Name: scene.Name, Kind: string(scene.Kind),
			RawSegments: result.RawCount, ValidSegments: len(result.Valid),
			DuplicateSegments: len(result.Duplicates),
			StitchedCurves:    len(result.Curves),
			RejectedSegments:  result.RejectedCount,
			MaxSurfaceGap:     result.MaxSurfaceGap,
			CompositePPM:      composite, IndividualPPMs: paths,
		})
		fmt.Printf("%-14s raw=%3d valid=%3d duplicate=%3d curves=%d maxgap=%.3g\n",
			scene.Name, result.RawCount, len(result.Valid), len(result.Duplicates),
			len(result.Curves), result.MaxSurfaceGap)
	}
	reportPath := filepath.Join(*out, "intersection_report.json")
	data, _ := json.MarshalIndent(reports, "", "  ")
	if err := os.WriteFile(reportPath, append(data, '\n'), 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %d scene(s) to %s\n", len(reports), *out)
}

func selectScenes(which string) ([]scenes.Scene, error) {
	switch which {
	case "", "all":
		return scenes.All(), nil
	case string(scenes.Tangent):
		s, _ := scenes.ByKind(scenes.Tangent)
		return []scenes.Scene{s}, nil
	case string(scenes.NearParallel), "near-parallel":
		s, _ := scenes.ByKind(scenes.NearParallel)
		return []scenes.Scene{s}, nil
	case string(scenes.Crossing):
		s, _ := scenes.ByKind(scenes.Crossing)
		return []scenes.Scene{s}, nil
	default:
		return nil, fmt.Errorf("unknown case %q", which)
	}
}

func sceneTitle(kind scenes.Kind) string {
	switch kind {
	case scenes.NearParallel:
		return "NEAR-PARALLEL"
	case scenes.Tangent:
		return "TANGENT"
	case scenes.Crossing:
		return "CROSSING"
	default:
		return string(kind)
	}
}

func writePPM(path string, im *render.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := render.WritePPM(f, im); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
