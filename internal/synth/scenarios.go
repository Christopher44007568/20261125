package synth

import (
	"fmt"
	"math/rand"

	"eyediag/internal/model"
)

// Scenario names supported by the generator.
const (
	ScenClean            = "clean"
	ScenSorting          = "sorting"
	ScenParallax         = "parallax"
	ScenAlpha            = "alpha"
	ScenAA               = "aa"
	ScenMulti            = "multi"
	ScenOneEye           = "one_eye"
	ScenBudget           = "budget"
	ScenEqualDiffDiffCau = "equal_diff_different_cause"
)

// Scenarios lists every built-in scenario.
func Scenarios() []string {
	return []string{
		ScenClean, ScenSorting, ScenParallax, ScenAlpha, ScenAA,
		ScenMulti, ScenOneEye, ScenBudget, ScenEqualDiffDiffCau,
	}
}

// Generate builds the named scenario. The seed only perturbs marker screen
// positions; the fault signals themselves are fully deterministic.
func Generate(name string, seed int64) (model.Case, error) {
	rng := rand.New(rand.NewSource(seed))
	c := model.Case{Name: name}

	switch name {
	case ScenClean:
		c.Markers = cleanMarkers()
	case ScenSorting:
		c.Markers = simpleMarkers("sort", healthyBudget(), sortingFault)
	case ScenParallax:
		c.Markers = simpleMarkers("parallax", healthyBudget(), parallaxFault)
	case ScenAlpha:
		c.Markers = simpleMarkers("alpha", healthyBudget(), func(o model.Observation) model.Observation {
			return alphaFault(o, 0.42)
		})
	case ScenAA:
		c.Markers = simpleMarkers("aa", healthyBudget(), aaFault)
	case ScenMulti:
		c.Markers = multiMarkers()
	case ScenOneEye:
		c.Markers = oneEyeMarkers()
	case ScenBudget:
		c.Markers = budgetMarkers()
	case ScenEqualDiffDiffCau:
		c.Markers = equalDiffMarkers()
	default:
		return model.Case{}, fmt.Errorf("unknown scenario %q", name)
	}

	jitterPositions(rng, c.Markers)
	return c, nil
}

func cleanMarkers() []model.Marker {
	markers := make([]model.Marker, 0, 3)
	for i := 0; i < 3; i++ {
		m := baseMarker(fmt.Sprintf("clean-%d", i), 10+i*20, 10, healthyBudget())
		m.Eyes = bothEyes(func(o model.Observation) model.Observation { return o })
		markers = append(markers, m)
	}
	return markers
}

func simpleMarkers(prefix string, budget model.BudgetStatus,
	mutate func(model.Observation) model.Observation) []model.Marker {
	markers := make([]model.Marker, 0, 3)
	for i := 0; i < 3; i++ {
		m := baseMarker(fmt.Sprintf("%s-%d", prefix, i), 10+i*20, 30, budget)
		m.Eyes = bothEyes(mutate)
		markers = append(markers, m)
	}
	return markers
}

// multiMarkers stacks sorting + alpha on one marker and parallax + AA on
// another, proving multi-label classification on the same eye.
func multiMarkers() []model.Marker {
	m1 := baseMarker("multi-sort-alpha", 10, 50, healthyBudget())
	m1.Eyes = bothEyes(func(o model.Observation) model.Observation {
		return alphaFault(sortingFault(o), 0.45)
	})
	m2 := baseMarker("multi-parallax-aa", 40, 50, healthyBudget())
	m2.Eyes = bothEyes(func(o model.Observation) model.Observation {
		return aaFault(parallaxFault(o))
	})
	return []model.Marker{m1, m2}
}

// oneEyeMarkers keeps the left eye correct and injects faults only into the
// right eye, per marker, covering all four causes one-sidedly.
func oneEyeMarkers() []model.Marker {
	faults := []struct {
		id     string
		mutate func(model.Observation) model.Observation
	}{
		{"one-sort", sortingFault},
		{"one-parallax", parallaxFault},
		{"one-alpha", func(o model.Observation) model.Observation { return alphaFault(o, 0.4) }},
		{"one-aa", aaFault},
	}
	markers := make([]model.Marker, 0, len(faults))
	for i, f := range faults {
		m := baseMarker(f.id, 10+i*20, 70, healthyBudget())
		m.Eyes = map[model.Eye]model.Observation{
			model.Left:  goodObs(model.Left),
			model.Right: f.mutate(goodObs(model.Right)),
		}
		markers = append(markers, m)
	}
	return markers
}

// budgetMarkers injects faults while the frame is under a cut transparent/AA
// budget, so degradation factors must be attached instead of inventing causes.
func budgetMarkers() []model.Marker {
	m1 := baseMarker("budget-sort", 10, 90, degradedBudget())
	m1.Eyes = bothEyes(sortingFault)
	m2 := baseMarker("budget-aa", 40, 90, degradedBudget())
	m2.Eyes = bothEyes(aaFault)
	m3 := baseMarker("budget-alpha", 70, 90, degradedBudget())
	m3.Eyes = bothEyes(func(o model.Observation) model.Observation { return alphaFault(o, 0.43) })
	return []model.Marker{m1, m2, m3}
}

// equalDiffMarkers gives two markers the same total pixel difference while
// their true causes differ, proving pixel difference cannot replace causal
// classification.
func equalDiffMarkers() []model.Marker {
	shared := model.RGBA{R: 0.62, G: 0.46, B: 0.26, A: expCover}

	mSort := baseMarker("eqdiff-sorting", 10, 110, healthyBudget())
	sortObs := sortingFault(goodObs(model.Left))
	sortObs.Color = shared
	mSort.Eyes = map[model.Eye]model.Observation{
		model.Left:  sortObs,
		model.Right: withColor(sortingFault(goodObs(model.Right)), shared),
	}

	mPar := baseMarker("eqdiff-parallax", 50, 110, healthyBudget())
	parObs := parallaxFault(goodObs(model.Left))
	parObs.Color = shared
	mPar.Eyes = map[model.Eye]model.Observation{
		model.Left:  parObs,
		model.Right: withColor(parallaxFault(goodObs(model.Right)), shared),
	}
	return []model.Marker{mSort, mPar}
}

func withColor(o model.Observation, c model.RGBA) model.Observation {
	o.Color = c
	return o
}
