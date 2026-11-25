package render

import (
	"bytes"
	"image/png"
	"testing"

	"brepsection/brep"
	"brepsection/geom"
	"brepsection/section"
)

func reports() []*section.Report {
	solid := brep.SampleSolid()
	c := solid.AABB().Center()
	planes := []geom.Plane{
		geom.NewPlane(geom.V3(0, 0, 1.25), geom.V3(0, 0, 1), "z-extrusion-cut"),
		geom.NewPlane(geom.V3(6, 0, 0), geom.V3(1, 0, 0), "x-tangent-via"),
		geom.NewPlane(c, geom.V3(0.3, 0.7, 0.65), "oblique-center-cut"),
	}
	var reps []*section.Report
	for _, pl := range planes {
		reps = append(reps, section.Extract(solid, pl, section.StandardModes[0]))
	}
	return reps
}

func TestMontageEncodes(t *testing.T) {
	c := Montage(reports(), 1200, 380)
	var buf bytes.Buffer
	if err := c.SavePNG(&buf); err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(&buf); err != nil {
		t.Fatalf("montage produced invalid PNG: %v", err)
	}
}

func TestHeatmapEncodes(t *testing.T) {
	c := Heatmap(reports(), 1200, 320)
	var buf bytes.Buffer
	if err := c.SavePNG(&buf); err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(&buf); err != nil {
		t.Fatalf("heatmap produced invalid PNG: %v", err)
	}
}

func TestCutawayEncodes(t *testing.T) {
	solid := brep.SampleSolid()
	pl := geom.NewPlane(geom.V3(0, 0, 1.25), geom.V3(0, 0, 1), "z-extrusion-cut")
	rep := section.Extract(solid, pl, section.StandardModes[0])
	c := Cutaway(solid, rep, pl, 800, 360)
	var buf bytes.Buffer
	if err := c.SavePNG(&buf); err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(&buf); err != nil {
		t.Fatalf("cutaway produced invalid PNG: %v", err)
	}
}
