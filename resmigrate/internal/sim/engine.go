package sim

import (
	"fmt"
	"math"
)

// eyeState is the mutable state of one eye: committed size, live history and
// the currently rendered frame image.
type eyeState struct {
	index          EyeIndex
	size           Size
	history        *historyBuffer
	image          *frameImage
	spp            int
	migratedPixels int
	remap          *RemapStats
	confLoss       float64
	// historyValid is the fraction of live history usable for temporal
	// reprojection (1 after warmup, reduced by remap invalidation).
	historyValid float64
}

type engine struct {
	spec   ScenarioSpec
	rng    *rng
	meter  meter
	eyes   [2]*eyeState
	frames []FrameRecord
	traj   []TrajectoryPoint

	committed Size
	tick      float64
	// prevImageBytes is the previous frame's linear buffers, which stay
	// allocated until the new frame is fully presented (double-buffered swap).
	prevImageBytes int

	outcome          Outcome
	retriesUsed      int
	injectedErr      string
	upgradeFrame     int
	windowStartFrame int
	upgradeMS        float64
	totalWork        int
	deadlineMiss     int
	maxFrameMS       float64

	// per-eye frame jump scores (frame-to-frame luma MAD on the fixed grid)
	eyeScores [2][]float64
	prevGrid  [2]float64
	hasPrev   bool
	// reconverge is a per-eye decaying shock applied after history loss.
	reconverge [2]float64
	// reconvergeDecay controls how fast the shock fades; staged migration
	// uses a slower decay than one-shot discard/remap.
	reconvergeDecay float64
	// resolutionShock is a one-time resampling jump applied on the upgrade
	// frame, scaled by the pixel-count ratio of the new size.
	resolutionShock float64
	// kindOverride replaces the kind label of the next presented frame.
	kindOverride string
}

func newEngine(spec ScenarioSpec) *engine {
	e := &engine{
		spec:            spec,
		rng:             newRNG(0x1234abcd + uint64(hashSpec(spec.ID))),
		committed:       spec.Initial,
		outcome:         OutcomeUpgraded,
		reconvergeDecay: 0.48,
	}
	for i, idx := range []EyeIndex{Left, Right} {
		hist := newHistory(spec.Initial, newRNG(0x77+uint64(i)*131+uint64(hashSpec(spec.ID))))
		e.eyes[i] = &eyeState{index: idx, size: spec.Initial, history: hist, spp: spec.SPP, historyValid: 1}
		e.meter.allocBytes(spec.Initial.pixels()*historyBytesPerPx, true)
	}
	e.meter.markBaseline()
	return e
}

func hashSpec(s string) int {
	h := 0
	for _, c := range s {
		h = h*131 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// sppFor computes samples-per-pixel for a size under the budget mode.
func (e *engine) sppFor(size Size) int {
	if e.spec.Budget == BudgetFixedSPP {
		return e.spec.SPP
	}
	initialTotal := e.spec.SPP * e.spec.Initial.pixels()
	spp := initialTotal / size.pixels()
	if spp < 1 {
		spp = 1
	}
	return spp
}

func (e *engine) budgetFor(size Size) (spp, samples int) {
	spp = e.sppFor(size)
	return spp, spp * size.pixels()
}

func (e *engine) allocHistory(size Size) {
	e.meter.allocBytes(size.pixels()*historyBytesPerPx, true)
}

func (e *engine) freeHistory(size Size) {
	e.meter.freeBytes(size.pixels()*historyBytesPerPx, true)
}

// present renders one visible frame. Both eyes always present the same
// committed size; a half-migrated eye is never shown. The previous frame's
// linear buffers remain live while the new frame renders, modeling a real
// double-buffered swap.
func (e *engine) present(kind string, workUnits int, event *string) FrameRecord {
	e.tick += sceneTimeStep
	idx := len(e.frames) + 1
	frameKind := kind
	if e.kindOverride != "" {
		frameKind = e.kindOverride
		e.kindOverride = ""
	}
	rec := FrameRecord{Index: idx, Kind: frameKind, Event: event, WorkUnits: workUnits}

	newImageBytes := 0
	for _, eye := range e.eyes {
		newImageBytes += eye.size.pixels() * imageBytesPerPx
	}
	// Both old and new linear frame buffers are live during present.
	e.meter.allocBytes(newImageBytes, false)
	firstFrame := e.prevImageBytes == 0
	if firstFrame {
		// The first frame models a double-buffered swap: an initial blank
		// linear buffer exists alongside the first rendered frame, then the
		// blank buffer is retired once rendering completes.
		e.meter.allocBytes(newImageBytes, false)
	}

	worst := 0.0
	for i := range e.eyes {
		eye := e.eyes[i]
		eye.image = renderFrame(eye.size, e.tick)
		spp, budget := e.budgetFor(eye.size)
		eye.spp = spp
		fb := FrameBudget{Size: eye.size, SPP: spp, BudgetSamples: budget, SpentSamples: budget}
		if i == 0 {
			rec.Left = fb
		} else {
			rec.Right = fb
		}
		grid := eye.image.gridMean()
		jump := grid
		if e.hasPrev {
			jump = math.Abs(grid - e.prevGrid[i])
		}
		// History loss adds a decaying re-convergence shock to the visible
		// frame-to-frame jump; fully valid history adds nothing.
		if e.hasPrev && e.reconverge[i] > 0 {
			jump += e.reconverge[i]
			e.reconverge[i] *= e.reconvergeDecay
			if e.reconverge[i] < 0.0004 {
				e.reconverge[i] = 0
			}
		}
		if e.resolutionShock > 0 {
			jump += e.resolutionShock
		}
		e.eyeScores[i] = append(e.eyeScores[i], jump)
		e.prevGrid[i] = grid
		if jump > worst {
			worst = jump
		}
		if i == 0 {
			rec.Jump.LeftMeanLuma = jump
		} else {
			rec.Jump.RightMeanLuma = jump
		}
	}
	e.hasPrev = true
	rec.Jump.StereoWorst = worst
	e.resolutionShock = 0

	frameMS := float64(workUnits) / unitsPerMS
	rec.ModeledFrameMS = frameMS
	if frameMS > frameBudgetMS {
		rec.DeadlineMiss = true
		e.deadlineMiss++
	}
	if frameMS > e.maxFrameMS {
		e.maxFrameMS = frameMS
	}
	e.totalWork += workUnits
	rec.PeakMemoryBytes = e.meter.snapshot()
	e.frames = append(e.frames, rec)
	e.appendTrajectory(idx, "")

	// Swap completes after the frame is presented: previous buffers are
	// released only once the new frame is the active one.
	if firstFrame {
		e.meter.freeBytes(newImageBytes, false)
	} else if e.prevImageBytes > 0 {
		e.meter.freeBytes(e.prevImageBytes, false)
	}
	e.prevImageBytes = newImageBytes
	return rec
}

func (e *engine) appendTrajectory(frame int, event string) {
	left := e.eyes[0]
	right := e.eyes[1]
	e.traj = append(e.traj, TrajectoryPoint{
		Frame:                 frame,
		Event:                 event,
		Left:                  left.size,
		Right:                 right.size,
		Committed:             e.committed,
		LeftMigratedFraction:  e.migratedFraction(Left),
		RightMigratedFraction: e.migratedFraction(Right),
		Consistent:            left.size.equals(right.size),
	})
}

func (e *engine) migratedFraction(idx EyeIndex) float64 {
	eye := e.eyes[idx]
	if eye.size.equals(e.committed) {
		return 0
	}
	if eye.migratedPixels <= 0 {
		return 0
	}
	return float64(eye.migratedPixels) / float64(eye.size.pixels())
}

func (e *engine) faultActive(attempt int, phase string, stage int) (*FaultSpec, bool) {
	f := e.spec.Fault
	if f == nil || f.OnAttempt != attempt || f.Phase != phase {
		return nil, false
	}
	if phase == "stage" && f.Stage != stage {
		return nil, false
	}
	return f, true
}

func (e *engine) fail(attempt int, phase string, stage int) error {
	f := e.spec.Fault
	if phase == "stage" {
		return fmt.Errorf("injected upgrade failure: eye=%s phase=%s attempt=%d stage=%d", f.Eye, phase, attempt, stage)
	}
	return fmt.Errorf("injected upgrade failure: eye=%s phase=%s attempt=%d", f.Eye, phase, attempt)
}
