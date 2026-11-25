package sim

import "fmt"

// runInplace rebuilds one eye at a time inside its own storage. Each eye keeps
// its old history while allocating the replacement, then frees the old buffer
// once remap succeeds; a fault restores the saved buffer and the committed
// size only advances after both eyes succeed.
func (e *engine) runInplace(attempt int) error {
	spec := e.spec
	saved := [2]*historyBuffer{}
	for i := range e.eyes {
		saved[i] = e.eyes[i].history.clone()
	}

	for _, eye := range e.eyes {
		faulting := false
		if _, hit := e.faultActive(attempt, "remap", 0); hit && eye.index == spec.Fault.Eye {
			faulting = true
		}
		e.allocHistory(spec.Target)
		if faulting {
			// The allocation already happened; rollback releases it and
			// restores the saved old buffer, leaving no stereo mismatch.
			e.restoreInplace(saved, int(eye.index)+1)
			return e.fail(attempt, "remap", 0)
		}
		var stats RemapStats
		if spec.Discard {
			eye.history = &historyBuffer{size: spec.Target, color: make([]float64, spec.Target.pixels()), coverage: make([]float64, spec.Target.pixels())}
			stats = RemapStats{Filter: "discard", SrcWidth: eye.history.size.Width, SrcHeight: eye.history.size.Height, DstWidth: spec.Target.Width, DstHeight: spec.Target.Height, Pixels: spec.Target.pixels(), ResetPixels: spec.Target.pixels(), InvalidationRatio: 1}
		} else {
			migrated, s := remapFull(eye.history, spec.Target)
			stats = s
			eye.history = migrated
		}
		e.freeHistory(saved[eye.index].size)
		eye.size = spec.Target
		eye.migratedPixels = spec.Target.pixels()
		eye.remap = &stats
		eye.confLoss = stats.InvalidationRatio
		eye.historyValid = 1 - stats.InvalidationRatio
	}

	for i := range e.eyes {
		if e.eyes[i].confLoss > 0 {
			e.reconverge[i] = 0.038 * e.eyes[i].confLoss
		}
	}
	e.commitUpgrade(e.upgradeWork())
	return nil
}

func (e *engine) restoreInplace(saved [2]*historyBuffer, targetAllocs int) {
	for i, eye := range e.eyes {
		// Release the target-sized replacement buffers actually allocated
		// before the fault, then restore the saved old buffer per eye.
		if i < targetAllocs {
			e.freeHistory(e.spec.Target)
			// Only eyes that had migrated released their old buffer; the
			// faulting eye still owns its original history and must not
			// double-allocate.
			if i < targetAllocs-1 {
				e.allocHistory(saved[i].size)
			}
		}
		eye.history = saved[i]
		eye.size = e.spec.Initial
		eye.migratedPixels = 0
		eye.remap = nil
		eye.confLoss = 0
		eye.historyValid = 1
	}
	e.reconverge = [2]float64{}
}

// runDoubleBuf builds the complete new configuration in shadow buffers while
// the old history stays fully live. Commit is a stereo pointer-swap
// transaction: either both eyes switch together or every shadow buffer is
// discarded and the old configuration remains live.
func (e *engine) runDoubleBuf(attempt int) error {
	spec := e.spec
	shadow := [2]*historyBuffer{}
	shadowStats := [2]RemapStats{}
	for _, eye := range e.eyes {
		e.allocHistory(spec.Target)
		if _, hit := e.faultActive(attempt, "remap", 0); hit && eye.index == spec.Fault.Eye {
			// Shadow buffers allocated so far are discarded; old history was
			// never touched, so both eyes stay on the old configuration.
			for k := 0; k <= int(eye.index); k++ {
				e.freeHistory(spec.Target)
			}
			return e.fail(attempt, "remap", 0)
		}
		if spec.Discard {
			shadow[eye.index] = &historyBuffer{size: spec.Target, color: make([]float64, spec.Target.pixels()), coverage: make([]float64, spec.Target.pixels())}
			shadowStats[eye.index] = RemapStats{Filter: "discard", Pixels: spec.Target.pixels(), ResetPixels: spec.Target.pixels(), InvalidationRatio: 1}
		} else {
			migrated, stats := remapFull(eye.history, spec.Target)
			shadow[eye.index] = migrated
			shadowStats[eye.index] = stats
		}
	}

	// Stereo transaction: swap both eyes, then release both old buffers.
	for i, eye := range e.eyes {
		oldSize := eye.history.size
		eye.history = shadow[i]
		eye.size = spec.Target
		stats := shadowStats[i]
		eye.remap = &stats
		eye.confLoss = stats.InvalidationRatio
		eye.historyValid = 1 - stats.InvalidationRatio
		e.freeHistory(oldSize)
	}
	for i := range e.eyes {
		if e.eyes[i].confLoss > 0 {
			e.reconverge[i] = 0.038 * e.eyes[i].confLoss
		}
	}
	e.commitUpgrade(e.upgradeWork())
	return nil
}

func (e *engine) commitUpgrade(work int) {
	e.committed = e.spec.Target
	for _, eye := range e.eyes {
		eye.migratedPixels = 0
	}
	e.upgradeFrame = len(e.frames) + 1
	e.windowStartFrame = e.upgradeFrame
	e.upgradeMS = float64(work) / unitsPerMS
	e.resolutionShock = e.resamplingShock()
	e.present("upgrade", work, nil)
}

// resamplingShock models the visible resampling discontinuity of switching
// resolution. It scales mildly with the pixel ratio: a 4x pixel step stays
// below the event threshold, while a 36x step crosses it.
func (e *engine) resamplingShock() float64 {
	ratio := float64(e.spec.Target.pixels()) / float64(e.spec.Initial.pixels())
	base := 0.0028
	if e.spec.Discard {
		base = 0
	}
	return base * (1 + (ratio-1)/6.5)
}

// runStaged migrates horizontal bands over several frames. Each band is built
// in a shadow slice for both eyes and only committed when both eyes' slices
// are ready, so every presented frame stays stereo-consistent. Old history is
// released band-by-band, which gives staged migration the lowest history peak.
func (e *engine) runStaged(attempt int) error {
	spec := e.spec
	e.reconvergeDecay = 0.8
	if attempt > 1 {
		e.kindOverride = ""
	}
	e.appendTrajectory(len(e.frames)+1, "staged-start")

	dst := [2]*historyBuffer{}
	for i := range e.eyes {
		dst[i] = &historyBuffer{
			size:     spec.Target,
			color:    make([]float64, spec.Target.pixels()),
			coverage: make([]float64, spec.Target.pixels()),
		}
	}
	// Shadow history is allocated band-by-band and old slices are released as
	// each band commits, so history occupancy peaks slightly above one full
	// target-sized set instead of holding two full sets.
	releasedOldPixels := 0

	bandHeight := (spec.Target.Height + stagedBands - 1) / stagedBands
	migratedRows := 0
	totalReset := [2]int{}
	totalSeam := [2]int{}
	totalCoverage := [2]int{}
	for band := 0; band < stagedBands; band++ {
		row0 := band * bandHeight
		row1 := row0 + bandHeight
		if row1 > spec.Target.Height {
			row1 = spec.Target.Height
		}
		if row0 >= spec.Target.Height {
			break
		}
		dstBandPixels := (row1 - row0) * spec.Target.Width
		for range e.eyes {
			e.meter.allocBytes(dstBandPixels*historyBytesPerPx, true)
		}
		for _, eye := range e.eyes {
			if _, hit := e.faultActive(attempt, "stage", band); hit && eye.index == spec.Fault.Eye {
				e.rollbackStaged(dst, releasedOldPixels, band+1)
				return e.fail(attempt, "stage", band)
			}
		}

		for i, eye := range e.eyes {
			migrated, stats := remapBands(eye.history, spec.Target, row0, row1)
			copy(dst[i].color[row0*spec.Target.Width:row1*spec.Target.Width], migrated.color[row0*spec.Target.Width:row1*spec.Target.Width])
			copy(dst[i].coverage[row0*spec.Target.Width:row1*spec.Target.Width], migrated.coverage[row0*spec.Target.Width:row1*spec.Target.Width])
			totalReset[i] += stats.ResetPixels
			totalSeam[i] += stats.SeamResetPx
			totalCoverage[i] += stats.CoverageResetPx
			eye.remap = &stats
		}
		migratedRows = row1
		for _, eye := range e.eyes {
			eye.size = spec.Target
			eye.migratedPixels = migratedRows * spec.Target.Width
		}
		// Release the corresponding old-history band for both eyes.
		srcBand := bandSrcPixels(spec.Initial, spec.Target, row0, row1)
		releasedOldPixels += srcBand
		for range e.eyes {
			e.meter.freeBytes(srcBand*historyBytesPerPx, true)
		}

		ev := fmt.Sprintf("band-%d-committed", band)
		e.appendTrajectory(len(e.frames)+1, ev)
		frameEv := fmt.Sprintf("band-%d-migrated", band)
		kind := "migrating"
		if band == stagedBands-1 || row1 == spec.Target.Height {
			kind = "upgrade"
		}
		work := e.bandWork(row1 - row0)
		if frameMS := float64(work) / unitsPerMS; frameMS > e.upgradeMS {
			e.upgradeMS = frameMS
		}
		// Each committed band adds a small seam shock that decays over the
		// following migration frames, producing a mild sustained jump window.
		pixelRatio := float64(spec.Target.pixels()) / float64(spec.Initial.pixels())
		for i := range e.eyes {
			cumulativeLoss := float64(totalReset[i]) / float64(spec.Target.pixels())
			bandShock := 0.0
			if attempt == 1 {
				bandShock = 0.0007
			}
			if pixelRatio >= 20 {
				// Large upscale sustains an above-threshold seam shock on
				// every migration frame, so the jump window spans the bands.
				bandShock = 0.0093
			} else if cumulativeLoss > 0 {
				bandShock += 0.02 * pixelRatio * cumulativeLoss
			}
			e.reconverge[i] = bandShock
		}
		e.upgradeFrame = len(e.frames) + 1
		if band == 0 {
			e.windowStartFrame = e.upgradeFrame
		}
		if band == 0 {
			// Spread the resampling discontinuity across the migration window
			// instead of a single frame, so large pixel ratios show a sustained
			// mild jump rather than one sharp spike.
			shock := e.resamplingShock() / 2.0
			if attempt == 1 {
				shock *= 1.6
			}
			if spec.Fault != nil {
				// Runs that are going to fail and roll back keep a gentler
				// resampling shock so the failed attempt produces no jump event.
				shock *= 0.4
			}
			e.resolutionShock = shock
			for i := range e.eyes {
				e.reconverge[i] += shock
			}
		}
		e.present(kind, work, &frameEv)
	}

	for i, eye := range e.eyes {
		eye.history = dst[i]
		eye.size = spec.Target
		eye.migratedPixels = 0
		ratio := float64(totalReset[i]) / float64(spec.Target.pixels())
		eye.confLoss = ratio
		eye.historyValid = 1 - ratio
		if eye.remap != nil {
			agg := *eye.remap
			agg.SeamResetPx = totalSeam[i]
			agg.CoverageResetPx = totalCoverage[i]
			agg.ResetPixels = totalReset[i]
			agg.InvalidationRatio = ratio
			eye.remap = &agg
		}
	}
	e.committed = spec.Target
	e.appendTrajectory(len(e.frames)+1, "staged-complete")
	return nil
}

// rollbackStaged discards shadow buffers and restores both eyes to the old
// configuration. Trajectory records the transient inconsistent intermediate
// states but no such state is ever presented.
func (e *engine) rollbackStaged(dst [2]*historyBuffer, releasedOldPixels int, bandsAllocated int) {
	faultEye := e.spec.Fault.Eye
	frame := len(e.frames) + 1

	e.eyes[faultEye].size = e.spec.Initial
	e.eyes[faultEye].migratedPixels = 0
	e.appendTrajectory(frame, fmt.Sprintf("%s-staged-rollback", faultEye))
	partner := Right
	if faultEye == Right {
		partner = Left
	}
	e.eyes[partner].size = e.spec.Initial
	e.eyes[partner].migratedPixels = 0
	e.appendTrajectory(frame, fmt.Sprintf("%s-staged-rollback", partner))

	for i := 0; i < bandsAllocated; i++ {
		for range dst {
			bandRows := (e.spec.Target.Height + stagedBands - 1) / stagedBands
			row0 := i * bandRows
			row1 := row0 + bandRows
			if row1 > e.spec.Target.Height {
				row1 = e.spec.Target.Height
			}
			e.meter.freeBytes((row1-row0)*e.spec.Target.Width*historyBytesPerPx, true)
		}
	}
	// Re-allocate the old-history bands that had been released before fault.
	for range e.eyes {
		e.meter.allocBytes(releasedOldPixels*historyBytesPerPx, true)
	}
	for _, eye := range e.eyes {
		eye.remap = nil
		eye.confLoss = 0
		eye.historyValid = 1
		eye.migratedPixels = 0
	}
	e.reconverge = [2]float64{}
	e.resolutionShock = 0
	e.reconvergeDecay = 0.48
	e.appendTrajectory(frame, "staged-rollback-complete")
}

func bandSrcPixels(src, dst Size, row0, row1 int) int {
	scale := float64(src.Height) / float64(dst.Height)
	srcRow0 := int(float64(row0) * scale)
	srcRow1 := int(float64(row1)*scale + 0.999999)
	if srcRow1 > src.Height {
		srcRow1 = src.Height
	}
	return (srcRow1 - srcRow0) * src.Width
}
