package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outDir := "out"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fail(err)
	}

	fmt.Println("1/6 generating procedural strand groom ...")
	g := generateGroom()

	fmt.Println("2/6 baking strands into hair cards ...")
	res := bakeGroom(g)
	packAtlas(res.Cards)

	fmt.Println("3/6 rasterizing card atlases ...")
	atlas := bakeAtlas(g, res.Cards)

	fmt.Println("4/6 rendering reference and card scenes ...")
	front := viewDef{"front", 0, 12, 30, 240}
	refImg := sceneSize(front, res.Cards, atlas, g, res, "ref", false, renderSize)
	cardImg := sceneSize(front, res.Cards, atlas, g, res, "card", false, renderSize)
	res.SilhouetteIOU = maskIOU(
		silhouetteMask(refImg, bgColor),
		silhouetteMask(cardImg, bgColor),
	)
	refEnergy := highlightEnergy(refImg, bgColor)
	cardEnergy := highlightEnergy(cardImg, bgColor)
	if refEnergy > 0 {
		res.SpecRetention = cardEnergy / refEnergy
	}

	fmt.Println("5/6 compositing audit images ...")
	outputs := []struct {
		name string
		im   *RGBA
	}{
		{"atlas_color.png", atlasToRGBA(atlas, false)},
		{"atlas_flow_spec.png", atlasToRGBA(atlas, true)},
		{"atlas_annotated.png", annotateAtlas(atlas, res.Cards)},
		{"side_by_side.png", sideBySide(g, res, atlas)},
		{"closeup.png", closeup(g, res, atlas)},
		{"montage_views.png", montage(g, res, atlas)},
		{"render_reference_front.png", refImg},
		{"render_cards_front.png", cardImg},
	}
	for _, o := range outputs {
		path := filepath.Join(outDir, o.name)
		if err := o.im.writePNG(path); err != nil {
			fail(err)
		}
		fmt.Printf("    wrote %s (%dx%d)\n", path, o.im.W, o.im.H)
	}

	fmt.Println("6/6 writing report and machine-readable summary ...")
	if err := writeReport(filepath.Join(outDir, "REPORT.md"), g, res); err != nil {
		fail(err)
	}
	if err := writeSummaryJSON(filepath.Join(outDir, "summary.json"), g, res); err != nil {
		fail(err)
	}
	fmt.Println("done.")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "hairbake:", err)
	os.Exit(1)
}
