// Package synth produces deterministic synthetic stereo frames with known
// injected faults, so the classifier can be exercised and validated without
// any external assets or components.
package synth

import (
	"math/rand"

	"eyediag/internal/model"
)

const (
	expDepth   = 0.5
	expCover   = 0.6
	expSamples = 8
	layers     = 4
)

var expectedColor = model.RGBA{R: 0.7, G: 0.4, B: 0.2, A: expCover}

func expectedOrder() []int { return []int{0, 1, 2, 3} }

func healthyBudget() model.BudgetStatus {
	return model.BudgetStatus{SampleBudget: expSamples}
}

func degradedBudget() model.BudgetStatus {
	return DegradedBudgetForTest()
}

// DegradedBudgetForTest exposes the degraded budget profile to tests in
// other packages.
func DegradedBudgetForTest() model.BudgetStatus {
	return model.BudgetStatus{
		Degraded:        true,
		DepthPrepassOff: true,
		AlphaToCoverage: true,
		MSAAHalved:      true,
		SampleBudget:    4,
		OverdrawRatio:   2.2,
	}
}

func baseMarker(id string, x, y int, budget model.BudgetStatus) model.Marker {
	return model.Marker{
		ID:              id,
		X:               x,
		Y:               y,
		LayerCount:      layers,
		ExpectedOrder:   expectedOrder(),
		ExpectedDepth:   expDepth,
		ExpectedColor:   expectedColor,
		ExpectedCover:   expCover,
		ExpectedSamples: expSamples,
		Budget:          budget,
		Eyes:            map[model.Eye]model.Observation{},
	}
}

// goodObs returns a fault-free observation. Left/right differ only by the
// mirrored parallax offset, exactly as a correct stereo pair should.
func goodObs(eye model.Eye) model.Observation {
	offset := 0.06
	if eye == model.Left {
		offset = -0.06
	}
	return model.Observation{
		Depth:          expDepth,
		Color:          expectedColor,
		Coverage:       expCover,
		EdgeCoverage:   expCover * 0.5,
		SampleCount:    expSamples,
		DrawOrder:      expectedOrder(),
		VertexDepth:    expDepth,
		ParallaxOffset: offset,
	}
}

func sortingFault(obs model.Observation) model.Observation {
	obs.DrawOrder = []int{3, 2, 1, 0}
	obs.Depth = expDepth // equal-depth layers: classic sort ambiguity
	obs.Color = model.RGBA{R: 0.58, G: 0.52, B: 0.32, A: obs.Color.A}
	return obs
}

func parallaxFault(obs model.Observation) model.Observation {
	obs.VertexDepth = expDepth + 0.08
	obs.Depth = expDepth + 0.08
	obs.ParallaxOffset = -obs.ParallaxOffset // wrong-eye offset direction
	obs.Color = model.RGBA{R: 0.66, G: 0.44, B: 0.24, A: obs.Color.A}
	return obs
}

func alphaFault(obs model.Observation, coverage float64) model.Observation {
	obs.Coverage = coverage
	obs.EdgeCoverage = expCover * 0.5 // interior leak, edge intact
	obs.Color = blendProxy(expectedColor, coverage)
	return obs
}

func aaFault(obs model.Observation) model.Observation {
	obs.SampleCount = expSamples / 4
	obs.EdgeCoverage = expCover*0.5 + 0.15 // jagged edge
	obs.Color = model.RGBA{R: 0.66, G: 0.43, B: 0.23, A: obs.Color.A}
	return obs
}

func blendProxy(src model.RGBA, coverage float64) model.RGBA {
	bg := 0.18
	a := coverage
	return model.RGBA{
		R: src.R*a + bg*(1-a),
		G: src.G*a + bg*(1-a),
		B: src.B*a + bg*(1-a),
		A: a,
	}
}

func bothEyes(mutate func(model.Observation) model.Observation) map[model.Eye]model.Observation {
	return map[model.Eye]model.Observation{
		model.Left:  mutate(goodObs(model.Left)),
		model.Right: mutate(goodObs(model.Right)),
	}
}

func jitterPositions(rng *rand.Rand, markers []model.Marker) {
	for i := range markers {
		markers[i].X += rng.Intn(6)
		markers[i].Y += rng.Intn(6)
	}
}
