package metrics

import (
	"fmt"
	"math"

	"handlate/internal/camera"
	"handlate/internal/sim"
	"handlate/internal/strategy"
	"handlate/internal/vec3"
)

// Result aggregates one strategy over one scenario.
type Result struct {
	Strategy       string
	Frames         int
	TrajRMSE       float64
	TrajP95        float64
	PinchRMSE      float64
	ContactRMSE    float64
	ContactN       int
	ThroughCount   int
	OverHideCount  int
	TotalPoints    int
	HiddenPoints   int
	LostFrames     int
	AvgHandAgeMs   float64
	AvgWorldAgeMs  float64
	LossWindowRMSE float64
}

func (r Result) Row() []string {
	return []string{
		r.Strategy,
		fmt.Sprintf("%.4f", r.TrajRMSE),
		fmt.Sprintf("%.4f", r.TrajP95),
		fmt.Sprintf("%.4f", r.PinchRMSE),
		fmt.Sprintf("%.4f", r.ContactRMSE),
		fmt.Sprintf("%d", r.ThroughCount),
		fmt.Sprintf("%.1f%%", 100*float64(r.OverHideCount)/float64(max1(r.TotalPoints))),
		fmt.Sprintf("%.1f", r.AvgHandAgeMs),
		fmt.Sprintf("%.1f", r.AvgWorldAgeMs),
		fmt.Sprintf("%.4f", r.LossWindowRMSE),
	}
}

// Evaluate runs one strategy over the whole timeline.
func Evaluate(scen sim.Scenario, tl *sim.Timeline, k strategy.Kind) Result {
	r := Result{Strategy: k.Name()}
	var sq, pinchSq, contactSq, handAge, worldAge, lossSq []float64

	for _, df := range tl.Displays {
		est := strategy.Build(k, df)
		r.Frames++
		if est.Points == nil {
			r.LostFrames++
			continue
		}
		handAge = append(handAge, float64(est.HandAgeMs))
		worldAge = append(worldAge, float64(est.WorldAgeMs))

		var frameErr float64
		for i, p := range est.Points {
			frameErr += vec3.Dist(p, df.Ground.Points[i])
		}
		frameErr /= float64(len(est.Points))
		sq = append(sq, frameErr*frameErr)
		if scen.ID() == "loss" && df.NowMs >= 1800 && df.NowMs <= 2400 {
			lossSq = append(lossSq, frameErr*frameErr)
		}

		estPinch := sim.PinchPoint(est.Points)
		pd := vec3.Dist(estPinch, df.Ground.Pinch)
		pinchSq = append(pinchSq, pd*pd)

		ref := scen.GroundAt(est.SampleMs)
		if scen.MeasureContact(ref) {
			target := scen.ContactTarget(ref)
			cd := vec3.Dist(estPinch, target)
			contactSq = append(contactSq, cd*cd)
			r.ContactN++
		}

		countLayer(scen, df, est, &r)
	}

	r.TrajRMSE = rmse(sq)
	r.TrajP95 = p95(sqrtAll(sq))
	r.PinchRMSE = rmse(pinchSq)
	r.ContactRMSE = rmse(contactSq)
	r.AvgHandAgeMs = mean(handAge)
	r.AvgWorldAgeMs = mean(worldAge)
	r.LossWindowRMSE = rmse(lossSq)
	return r
}

func countLayer(scen sim.Scenario, df sim.DisplayFrame, est strategy.Estimate, r *Result) {
	boxes := scen.Occluders()
	for i := range est.Points {
		r.TotalPoints++
		truthHidden := camera.Occluded(df.Ground.Head, df.Ground.Points[i], boxes)
		rendererHidden := est.Hidden[i]
		if rendererHidden {
			r.HiddenPoints++
		}
		// Through: renderer shows a point that truth geometry places behind a panel.
		if truthHidden && !rendererHidden {
			r.ThroughCount++
		}
		// Over-hide: renderer hides a point that truth geometry places in front.
		if !truthHidden && rendererHidden {
			r.OverHideCount++
		}
	}
}

func rmse(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	return math.Sqrt(mean(v))
}

func sqrtAll(v []float64) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = math.Sqrt(x)
	}
	return out
}

func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	var sum float64
	for _, x := range v {
		sum += x
	}
	return sum / float64(len(v))
}

func p95(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sorted := append([]float64(nil), v...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1] > sorted[j]; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}
	idx := int(math.Ceil(0.95*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	return sorted[idx]
}

func max1(n int) int {
	if n == 0 {
		return 1
	}
	return n
}
