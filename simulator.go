package eyebw

import "math"

type FrameInput struct {
	Index  int            `json:"index"`
	Gazes  [2]Gaze        `json:"gazes"`
	Motion float64        `json:"motion,omitempty"`
	Probes []QualityProbe `json:"probes,omitempty"`
}

type Simulator struct {
	cfg        FrameConfig
	pipelines  map[PipelineID]*PipelineSim
	lastGazes  *[2]Gaze
	frameCount int
}

type PipelineSim struct {
	id      PipelineID
	cfg     FrameConfig
	cache   *Cache
	history map[string]int
}

func NewSimulator(cfg FrameConfig) (*Simulator, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	s := &Simulator{cfg: cfg, pipelines: make(map[PipelineID]*PipelineSim, 3)}
	for _, id := range AllPipelines() {
		s.pipelines[id] = &PipelineSim{
			id:      id,
			cfg:     cfg,
			cache:   NewCache(cfg.CacheCapacityBytes),
			history: make(map[string]int),
		}
	}
	return s, nil
}

func (s *Simulator) Config() FrameConfig { return s.cfg }

func (s *Simulator) Step(input FrameInput) (*FrameReport, error) {
	if _, err := NewSimulator(s.cfg); err != nil {
		return nil, err
	}
	if input.Index == 0 && s.frameCount > 0 {
		input.Index = s.frameCount
	}
	motion := input.Motion
	if motion <= 0 && s.lastGazes != nil {
		leftDelta := math.Hypot(input.Gazes[0].X-s.lastGazes[0].X, input.Gazes[0].Y-s.lastGazes[0].Y)
		rightDelta := math.Hypot(input.Gazes[1].X-s.lastGazes[1].X, input.Gazes[1].Y-s.lastGazes[1].Y)
		motion = math.Min(1, math.Max(leftDelta, rightDelta)/0.35)
	}
	s.lastGazes = &[2]Gaze{input.Gazes[0], input.Gazes[1]}
	s.frameCount++

	frame := &FrameReport{Index: input.Index, Gazes: input.Gazes, Motion: motion}
	for _, id := range AllPipelines() {
		frame.Pipelines = append(frame.Pipelines, s.pipelines[id].step(input.Index, input.Gazes, motion, input.Probes))
	}
	return frame, nil
}

func foveationByEye(cfg FrameConfig, gazes [2]Gaze) ([]Tile, []Tile, []EyeFoveation) {
	left := BuildTiles(cfg, gazes[0])
	right := BuildTiles(cfg, gazes[1])
	summary := []EyeFoveation{
		{Eye: EyeLeft, FoveationSummary: SummarizeTiles(left)},
		{Eye: EyeRight, FoveationSummary: SummarizeTiles(right)},
	}
	return left, right, summary
}

func (p *PipelineSim) step(frameIndex int, gazes [2]Gaze, motion float64, probes []QualityProbe) *PipelineReport {
	leftTiles, rightTiles, foveation := foveationByEye(p.cfg, gazes)
	meter := NewMeter(frameIndex, p.id, p.cache)
	observations := make([]CenterObservation, 0)
	repaint := RepaintReport{}
	builder := tileBuilder{
		cfg:          p.cfg,
		meter:        meter,
		history:      p.history,
		repaint:      &repaint,
		observations: &observations,
	}
	notes := []string{}

	switch p.id {
	case PipelineSeparateEyes:
		for i := range leftTiles {
			builder.renderedTile(EyeLeft, "separate", leftTiles[i])
		}
		for i := range rightTiles {
			builder.renderedTile(EyeRight, "separate", rightTiles[i])
		}
		notes = append(notes, "Two unique eye surfaces; no eye-pixel deduplication.")

	case PipelineAtlas:
		for i := range leftTiles {
			builder.renderedTile(EyeLeft, "atlas", leftTiles[i])
		}
		for i := range rightTiles {
			builder.renderedTile(EyeRight, "atlas", rightTiles[i])
		}
		if p.cfg.AtlasGapPixels > 0 {
			gapBytes := uint64(p.cfg.AtlasGapPixels) * uint64(p.cfg.Height) * uint64(p.cfg.ColorBytesPerSample)
			meter.Write(StageOpaquePass, EyeAtlas, ClassColor, "atlas:gap", gapBytes)
			meter.ReadUncached(StageScanout, EyeAtlas, ClassColor, "atlas:gap", "atlas_gap_scanout", gapBytes)
		}
		notes = append(notes, "One atlas allocation; packing improves locality but does not deduplicate stereo pixels.")

	case PipelineSingleReuse:
		for i := range leftTiles {
			sourceKeys := builder.renderedTile(EyeLeft, "single-source", leftTiles[i])
			target := rightTiles[i]
			if target.QualityCenter && !p.cfg.SingleEye.AggressiveCenterReuse {
				builder.renderedTile(EyeRight, "single-center-repaint", target)
			} else {
				builder.reusedTile(EyeRight, "single-reuse", target, leftTiles[i], sourceKeys, motion, false)
			}
		}
		if p.cfg.SingleEye.AggressiveCenterReuse {
			notes = append(notes, "UNSAFE: center tiles are reused from the other eye; quality gate must reject silent bandwidth savings.")
		} else {
			notes = append(notes, "Safe reuse: peripheral color/depth/history are warped from one eye; both center regions remain full-rate and independently rendered.")
		}
	}

	return p.buildReport(frameIndex, meter, foveation, repaint, observations, probes, notes)
}
