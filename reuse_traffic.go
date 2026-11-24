package eyebw

func (b *tileBuilder) reusedTile(eye EyeID, prefix string, target Tile, source Tile, sourceKeys tileKeys, motion float64, independent bool) tileKeys {
	keys := makeTileKeys(prefix, eye, target.IndexX, target.IndexY)
	colorBPP := uint64(b.cfg.ColorBytesPerSample)
	depthBPP := uint64(b.cfg.DepthBytesPerSample)
	historyBPP := uint64(b.cfg.HistoryBytesPerTexel)

	sourceColorBytes := source.Texels * colorBPP
	sourceDepthBytes := source.Texels * depthBPP
	sourceHistoryBytes := source.Texels * historyBPP

	// These reads use the source eye's keys. A resident source tile is a
	// cache hit and therefore contributes logical bytes but zero transfer.
	b.meter.ReadReason(StageReuse, eye, ClassColor, sourceKeys.ColorResolved, "source_eye_color", sourceColorBytes)
	b.meter.ReadReason(StageReuse, eye, ClassDepth, sourceKeys.DepthResolved, "source_eye_depth", sourceDepthBytes)
	b.meter.ReadReason(StageReuse, eye, ClassHistory, sourceKeys.History, "source_eye_history", sourceHistoryBytes)

	frontBytes := target.FullPixels * colorBPP
	b.meter.WriteReason(StageReproject, eye, ClassColor, keys.Front, "cross_eye_warp", frontBytes)
	b.meter.ReadUncached(StageScanout, eye, ClassColor, keys.Front, "display_scanout", frontBytes)

	b.emitDisocclusion(eye, keys, target, motion)

	if target.QualityCenter {
		*b.observations = append(*b.observations, CenterObservation{
			Eye:                   eye,
			TileX:                 target.IndexX,
			TileY:                 target.IndexY,
			Divisor:               target.Divisor,
			MSAASamples:           b.cfg.msaa(),
			IndependentlyRendered: independent,
			HistoryValid:          false,
			FullRepaint:           false,
		})
	}
	return keys
}

func (s *PipelineSim) buildReport(frame int, meter *Meter, foveation []EyeFoveation, repaint RepaintReport, observations []CenterObservation, probes []QualityProbe, notes []string) *PipelineReport {
	accesses := meter.Snapshot()
	agg := AggregateAccesses(accesses)
	hotspots := BuildHotspots(agg.Stages)
	budget := CheckBudget(s.cfg.Budget, agg.TransferTotal(), agg.Stages)
	quality := CheckQuality(s.cfg, s.id, observations, probes)
	report := &PipelineReport{
		ID:                 s.id,
		Label:              s.id.LabelZH(),
		FrameIndex:         frame,
		Aggregate:          agg,
		Hotspots:           hotspots,
		Budget:             budget,
		Quality:            quality,
		FoveationByEye:     foveation,
		Repaint:            repaint,
		CacheResidentBytes: s.cache.Size(),
		Notes:              notes,
	}
	if len(hotspots) > 0 {
		report.TopHotspot = &hotspots[0]
	}
	if s.cfg.RetainAccesses {
		report.Accesses = accesses
	}
	return report
}
