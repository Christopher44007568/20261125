package main

import (
	"math"
	"testing"
)

func TestDeepOpacityAccumulatesLayers(t *testing.T) {
	dm := &DeepOpacityMap{
		Size:    2,
		Samples: make([][]DeepSample, 4),
	}
	attenuation := layerAttenuation(0.55, Vec(0.45, 0.20, 0.08))
	dm.Samples[0] = []DeepSample{
		{Depth: 1.0, StrandID: 1, Alpha: 0.55, Attenuation: attenuation},
		{Depth: 2.0, StrandID: 2, Alpha: 0.55, Attenuation: attenuation},
		{Depth: 3.0, StrandID: 3, Alpha: 0.55, Attenuation: attenuation},
	}

	one := dm.tileTransmittance(0, 0, 1.5, -1)
	two := dm.tileTransmittance(0, 0, 2.5, -1)
	three := dm.tileTransmittance(0, 0, 3.5, -1)

	if luminance(one) >= 0.99 {
		t.Fatalf("first deep layer did not attenuate light: %+v", one)
	}
	if !(luminance(two) < luminance(one)) {
		t.Fatalf("second layer did not darken transmittance: one=%v two=%v", one, two)
	}
	if !(luminance(three) < luminance(two)) {
		t.Fatalf("third layer did not add occlusion: two=%v three=%v", two, three)
	}

	allThree := three
	excluded := dm.tileTransmittance(0, 0, 3.5, 2)
	if luminance(excluded) <= luminance(allThree) {
		t.Fatalf("excluding guide layer did not brighten complete transmittance: excluded=%v all=%v", excluded, allThree)
	}
	if math.Abs(luminance(excluded)-luminance(two)) > 1e-9 {
		t.Fatalf("excluding middle layer should leave two equal attenuations: excluded=%v two=%v", excluded, two)
	}
}

func TestCompressMergesOnlySameGuideJoint(t *testing.T) {
	dm := &DeepOpacityMap{
		Size:    1,
		Samples: make([][]DeepSample, 1),
	}
	layer := layerAttenuation(0.5, Vec(0.5, 0.25, 0.1))
	dm.Samples[0] = []DeepSample{
		{Depth: 1.00, StrandID: 7, Alpha: 0.5, Attenuation: layer},
		{Depth: 1.01, StrandID: 7, Alpha: 0.8, Attenuation: layer},
		{Depth: 1.20, StrandID: 8, Alpha: 0.6, Attenuation: layer},
	}

	dm.Compress(0.025)
	got := dm.Samples[0]
	if len(got) != 2 {
		t.Fatalf("expected one guide-7 joint layer and guide-8 layer, got %d: %+v", len(got), got)
	}
	if got[0].StrandID != 7 || got[0].Alpha != 0.8 || got[1].StrandID != 8 {
		t.Fatalf("same-guide joint was not reduced to strongest coverage: %+v", got[0])
	}
}

func TestProceduralSceneContainsMultilayerTiles(t *testing.T) {
	scene := NewProceduralScene(160, 200, 42)
	dm := BuildDeepOpacityMap(scene, 192)

	maxLayers := 0
	for y := 0; y < dm.Size; y++ {
		for x := 0; x < dm.Size; x++ {
			if n := dm.LayerCount(x, y); n > maxLayers {
				maxLayers = n
			}
		}
	}
	if maxLayers < 3 {
		t.Fatalf("expected overlapping guide layers, maximum compressed depth layers = %d", maxLayers)
	}
}
