package render_test

import (
	"bytes"
	"testing"

	"ruledintersect/intersect"
	"ruledintersect/render"
	"ruledintersect/scenes"
)

func TestThreeViewsUsesAllStatusColors(t *testing.T) {
	scene, ok := scenes.ByKind(scenes.Tangent)
	if !ok {
		t.Fatal("tangent scene missing")
	}
	result := intersect.Surfaces(scene.A, scene.B, intersect.DefaultOptions())
	im := render.ThreeViews("TANGENT TEST", scene.A, scene.B, result)
	if im.W != 1438 || im.H != 620 {
		t.Fatalf("unexpected PPM size %dx%d", im.W, im.H)
	}

	var valid, duplicate, blue int
	for panel := 0; panel < 3; panel++ {
		x0, y0, w, h := render.PanelRect(panel)
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				c := im.Pix[y*im.W+x]
				switch c {
				case render.ValidColor:
					valid++
				case render.DuplicateColor:
					duplicate++
				}
				if int(c.B) > int(c.R)+14 && int(c.B) > int(c.G)+6 {
					blue++
				}
			}
		}
	}
	if valid == 0 {
		t.Fatal("valid segments were not drawn")
	}
	if duplicate == 0 {
		t.Fatal("duplicate segments were not drawn")
	}
	if blue < 1000 {
		t.Fatal("untrimmed surface regions were not drawn")
	}

	var ppm bytes.Buffer
	if err := render.WritePPM(&ppm, im); err != nil {
		t.Fatal(err)
	}
	data := ppm.Bytes()
	if !bytes.HasPrefix(data, []byte("P6\n#")) {
		t.Fatal("missing binary PPM header")
	}
	marker := []byte("\n255\n")
	end := bytes.Index(data, marker)
	if end < 0 {
		t.Fatal("missing PPM max value")
	}
	pixels := data[end+len(marker):]
	if len(pixels) != im.W*im.H*3 {
		t.Fatalf("PPM payload has %d bytes, want %d", len(pixels), im.W*im.H*3)
	}
}

func TestCropKeepsPanelPixels(t *testing.T) {
	scene, _ := scenes.ByKind(scenes.Crossing)
	result := intersect.Surfaces(scene.A, scene.B, intersect.DefaultOptions())
	im := render.ThreeViews("CROP TEST", scene.A, scene.B, result)
	x, y, w, h := render.PanelRect(1)
	crop := render.Crop(im, x, y, w, h)
	if crop.W != w || crop.H != h || crop.Pix[0] != im.Pix[y*im.W+x] {
		t.Fatal("crop does not preserve panel geometry")
	}
}
