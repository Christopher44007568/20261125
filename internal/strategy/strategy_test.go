package strategy

import (
	"testing"

	"handlate/internal/sim"
	"handlate/internal/vec3"
)

func TestLossWindowStrategiesDiverge(t *testing.T) {
	scen := sim.LossScenario{}
	tl := sim.Build(scen)

	var lateSum, freezeSum, syncSum float64
	var lateN, freezeN, syncN int
	for _, df := range tl.Displays {
		if df.NowMs < 1900 || df.NowMs > 2300 {
			continue
		}
		collect := func(k Kind, sum *float64, n *int) {
			est := Build(k, df)
			if est.Points == nil {
				return
			}
			var frameErr float64
			for i, p := range est.Points {
				frameErr += vec3.Dist(p, df.Ground.Points[i])
			}
			*sum += frameErr / float64(len(est.Points))
			*n++
		}
		collect(LateCompose, &lateSum, &lateN)
		collect(Freeze, &freezeSum, &freezeN)
		collect(WorldSync, &syncSum, &syncN)
	}
	if lateN == 0 || freezeN == 0 || syncN == 0 {
		t.Fatalf("missing loss-window frames: late=%d freeze=%d sync=%d", lateN, freezeN, syncN)
	}
	lateErr, freezeErr, syncErr := lateSum/float64(lateN), freezeSum/float64(freezeN), syncSum/float64(syncN)
	if !(lateErr < freezeErr && freezeErr < syncErr) {
		t.Fatalf("expected late < freeze < world-sync error, got late=%.4f freeze=%.4f sync=%.4f",
			lateErr, freezeErr, syncErr)
	}
}

func TestWorldSyncNeverUsesNewHand(t *testing.T) {
	scen := sim.WaveScenario{}
	tl := sim.Build(scen)
	for _, df := range tl.Displays[5:] {
		est := Build(WorldSync, df)
		if est.Points == nil {
			continue
		}
		pairedSampleTime := df.NowMs - est.HandAgeMs
		if pairedSampleTime > df.World.SampleTime {
			t.Fatalf("world-sync used hand@%dms newer than world@%dms", pairedSampleTime, df.World.SampleTime)
		}
	}
}
