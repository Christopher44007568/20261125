package eyebw

import "fmt"

type CenterObservation struct {
	Eye                   EyeID `json:"eye"`
	TileX                 int   `json:"tile_x"`
	TileY                 int   `json:"tile_y"`
	Divisor               int   `json:"divisor"`
	MSAASamples           int   `json:"msaa_samples"`
	IndependentlyRendered bool  `json:"independently_rendered"`
	HistoryValid          bool  `json:"history_valid"`
	FullRepaint           bool  `json:"full_repaint"`
}

type QualityProbe struct {
	Eye    EyeID   `json:"eye"`
	PSNRDB float64 `json:"psnr_db"`
	SSIM   float64 `json:"ssim"`
	Source string  `json:"source,omitempty"`
}

type QualityReport struct {
	Passed                 bool                `json:"passed"`
	Pipeline               PipelineID          `json:"pipeline"`
	CenterTileObservations int                 `json:"center_tile_observations"`
	ExpectedCenterEyes     int                 `json:"expected_center_eyes"`
	MinDivisor             int                 `json:"min_divisor"`
	MinMSAASamples         int                 `json:"min_msaa_samples"`
	IndependentCenterTiles int                 `json:"independent_center_tiles"`
	Violations             []string            `json:"violations"`
	Probes                 []QualityProbe      `json:"probes,omitempty"`
	ProbeViolations        []string            `json:"probe_violations,omitempty"`
	Observations           []CenterObservation `json:"-"`
}

func CheckQuality(cfg FrameConfig, pipeline PipelineID, observations []CenterObservation, probes []QualityProbe) QualityReport {
	cfg = cfg.withDefaults()
	requiredMSAA := cfg.Quality.RequiredMSAASamples
	if requiredMSAA == 0 {
		requiredMSAA = cfg.MSAASamples
	}
	out := QualityReport{
		Passed:             true,
		Pipeline:           pipeline,
		ExpectedCenterEyes: 2,
		MinDivisor:         1 << 30,
		MinMSAASamples:     1 << 30,
		Observations:       observations,
		Probes:             probes,
	}

	seen := make(map[EyeID]int)
	for _, obs := range observations {
		out.CenterTileObservations++
		seen[obs.Eye]++
		if obs.Divisor < out.MinDivisor {
			out.MinDivisor = obs.Divisor
		}
		if obs.MSAASamples < out.MinMSAASamples {
			out.MinMSAASamples = obs.MSAASamples
		}
		if obs.IndependentlyRendered {
			out.IndependentCenterTiles++
		}
		if obs.Divisor > cfg.Quality.RequiredDivisor {
			out.Violations = append(out.Violations, fmt.Sprintf(
				"%s center tile (%d,%d) divisor=%d > required %d",
				obs.Eye, obs.TileX, obs.TileY, obs.Divisor, cfg.Quality.RequiredDivisor))
		}
		if obs.MSAASamples < requiredMSAA {
			out.Violations = append(out.Violations, fmt.Sprintf(
				"%s center tile (%d,%d) MSAA=%d < required %d",
				obs.Eye, obs.TileX, obs.TileY, obs.MSAASamples, requiredMSAA))
		}
		if cfg.Quality.RequireIndependentEyes && !obs.IndependentlyRendered {
			out.Violations = append(out.Violations, fmt.Sprintf(
				"%s center tile (%d,%d) reused another eye without independent full-rate render/repaint",
				obs.Eye, obs.TileX, obs.TileY))
		}
	}
	for _, eye := range []EyeID{EyeLeft, EyeRight} {
		if seen[eye] == 0 {
			out.Violations = append(out.Violations, fmt.Sprintf("missing center observations for %s eye", eye))
		}
	}
	if out.CenterTileObservations == 0 {
		out.MinDivisor = 0
		out.MinMSAASamples = 0
		out.Violations = append(out.Violations, "no center quality observations")
	}
	if out.MinDivisor == 1<<30 {
		out.MinDivisor = 0
	}
	if out.MinMSAASamples == 1<<30 {
		out.MinMSAASamples = 0
	}

	for _, probe := range probes {
		if cfg.Quality.MinimumCenterPSNRDB > 0 && probe.PSNRDB > 0 && probe.PSNRDB < cfg.Quality.MinimumCenterPSNRDB {
			out.ProbeViolations = append(out.ProbeViolations, fmt.Sprintf(
				"%s center PSNR %.2f dB < %.2f dB", probe.Eye, probe.PSNRDB, cfg.Quality.MinimumCenterPSNRDB))
		}
		if cfg.Quality.MinimumCenterSSIM > 0 && probe.SSIM > 0 && probe.SSIM < cfg.Quality.MinimumCenterSSIM {
			out.ProbeViolations = append(out.ProbeViolations, fmt.Sprintf(
				"%s center SSIM %.4f < %.4f", probe.Eye, probe.SSIM, cfg.Quality.MinimumCenterSSIM))
		}
	}

	out.Violations = dedupeStrings(out.Violations)
	out.ProbeViolations = dedupeStrings(out.ProbeViolations)
	if len(out.Violations)+len(out.ProbeViolations) > 0 {
		out.Passed = false
	}
	return out
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// RejectSilentBandwidthSaving returns an error if a candidate uses less
// transfer bandwidth than the baseline but fails the center quality gate.
func RejectSilentBandwidthSaving(baseline, candidate PipelineQualityBandwidth) error {
	if candidate.TransferBytes >= baseline.TransferBytes {
		return nil
	}
	if candidate.QualityPassed {
		return nil
	}
	return fmt.Errorf("pipeline %s saves %s bandwidth but center quality failed: %v",
		candidate.Pipeline, FormatBytes(baseline.TransferBytes-candidate.TransferBytes), candidate.Violations)
}

type PipelineQualityBandwidth struct {
	Pipeline      PipelineID
	TransferBytes uint64
	QualityPassed bool
	Violations    []string
}
