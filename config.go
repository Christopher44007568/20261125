package eyebw

import (
	"errors"
	"fmt"
)

type Resolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Gaze struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type FoveationConfig struct {
	Enabled       bool    `json:"enabled"`
	TileWidth     int     `json:"tile_width"`
	TileHeight    int     `json:"tile_height"`
	CenterRadius  float64 `json:"center_radius"`
	MidRadius     float64 `json:"mid_radius"`
	MediumDivisor int     `json:"medium_divisor"`
	EdgeDivisor   int     `json:"edge_divisor"`
}

type QualityContract struct {
	CenterRadius           float64 `json:"center_radius"`
	RequiredDivisor        int     `json:"required_divisor"`
	RequiredMSAASamples    int     `json:"required_msaa_samples"`
	RequireIndependentEyes bool    `json:"require_independent_eyes"`
	MinimumCenterPSNRDB    float64 `json:"minimum_center_psnr_db,omitempty"`
	MinimumCenterSSIM      float64 `json:"minimum_center_ssim,omitempty"`
}

type SingleEyePolicy struct {
	// AggressiveCenterReuse deliberately allows reusing the source-eye center.
	// The quality gate must reject it when independent center rendering is
	// required; it exists to prove savings are not accepted silently.
	AggressiveCenterReuse bool    `json:"aggressive_center_reuse"`
	DisocclusionFraction  float64 `json:"disocclusion_fraction"`
	MotionFraction        float64 `json:"motion_fraction"`
}

type Budget struct {
	FrameBytes uint64             `json:"frame_bytes"`
	StageBytes map[StageID]uint64 `json:"stage_bytes,omitempty"`
}

type FrameConfig struct {
	Width                int             `json:"width"`
	Height               int             `json:"height"`
	MSAASamples          int             `json:"msaa_samples"`
	ColorBytesPerSample  int             `json:"color_bytes_per_sample"`
	DepthBytesPerSample  int             `json:"depth_bytes_per_sample"`
	HistoryBytesPerTexel int             `json:"history_bytes_per_texel"`
	RepaintBytesPerTexel int             `json:"repaint_bytes_per_texel"`
	AtlasGapPixels       int             `json:"atlas_gap_pixels"`
	CacheCapacityBytes   uint64          `json:"cache_capacity_bytes"`
	Budget               Budget          `json:"budget"`
	Foveation            FoveationConfig `json:"foveation"`
	Quality              QualityContract `json:"quality"`
	SingleEye            SingleEyePolicy `json:"single_eye"`
	RetainAccesses       bool            `json:"-"`
}

func DefaultConfig() FrameConfig {
	return FrameConfig{
		Width:                2400,
		Height:               2400,
		MSAASamples:          4,
		ColorBytesPerSample:  4,
		DepthBytesPerSample:  4,
		HistoryBytesPerTexel: 16,
		RepaintBytesPerTexel: 8,
		AtlasGapPixels:       0,
		CacheCapacityBytes:   32 * MebiByte,
		Budget: Budget{
			FrameBytes: 512 * MebiByte,
		},
		Foveation: FoveationConfig{
			Enabled:       true,
			TileWidth:     120,
			TileHeight:    120,
			CenterRadius:  0.30,
			MidRadius:     0.66,
			MediumDivisor: 2,
			EdgeDivisor:   4,
		},
		Quality: QualityContract{
			CenterRadius:           0.24,
			RequiredDivisor:        1,
			RequireIndependentEyes: true,
			MinimumCenterPSNRDB:    38,
			MinimumCenterSSIM:      0.98,
		},
		SingleEye: SingleEyePolicy{
			AggressiveCenterReuse: false,
			DisocclusionFraction:  0.08,
			MotionFraction:        0.10,
		},
	}
}

func (c FrameConfig) withDefaults() FrameConfig {
	d := DefaultConfig()
	if c.Width == 0 {
		c.Width = d.Width
	}
	if c.Height == 0 {
		c.Height = d.Height
	}
	if c.MSAASamples == 0 {
		c.MSAASamples = 1
	}
	if c.ColorBytesPerSample == 0 {
		c.ColorBytesPerSample = d.ColorBytesPerSample
	}
	if c.DepthBytesPerSample == 0 {
		c.DepthBytesPerSample = d.DepthBytesPerSample
	}
	if c.HistoryBytesPerTexel == 0 {
		c.HistoryBytesPerTexel = d.HistoryBytesPerTexel
	}
	if c.RepaintBytesPerTexel == 0 {
		c.RepaintBytesPerTexel = d.RepaintBytesPerTexel
	}
	if c.CacheCapacityBytes == 0 {
		c.CacheCapacityBytes = d.CacheCapacityBytes
	}
	if c.Foveation.TileWidth == 0 {
		c.Foveation.TileWidth = d.Foveation.TileWidth
	}
	if c.Foveation.TileHeight == 0 {
		c.Foveation.TileHeight = d.Foveation.TileHeight
	}
	if c.Foveation.CenterRadius == 0 {
		c.Foveation.CenterRadius = d.Foveation.CenterRadius
	}
	if c.Foveation.MidRadius == 0 {
		c.Foveation.MidRadius = d.Foveation.MidRadius
	}
	if c.Foveation.MediumDivisor == 0 {
		c.Foveation.MediumDivisor = d.Foveation.MediumDivisor
	}
	if c.Foveation.EdgeDivisor == 0 {
		c.Foveation.EdgeDivisor = d.Foveation.EdgeDivisor
	}
	if c.Quality.CenterRadius == 0 {
		c.Quality.CenterRadius = d.Quality.CenterRadius
	}
	if c.Quality.RequiredDivisor == 0 {
		c.Quality.RequiredDivisor = 1
	}
	if c.SingleEye.DisocclusionFraction == 0 {
		c.SingleEye.DisocclusionFraction = d.SingleEye.DisocclusionFraction
	}
	if c.SingleEye.MotionFraction == 0 {
		c.SingleEye.MotionFraction = d.SingleEye.MotionFraction
	}
	return c
}

func (c FrameConfig) Validate() error {
	if c.Width <= 0 || c.Height <= 0 {
		return errors.New("width and height must be positive")
	}
	if c.MSAASamples < 1 || c.MSAASamples > 64 {
		return fmt.Errorf("msaa samples must be in [1,64], got %d", c.MSAASamples)
	}
	if c.ColorBytesPerSample <= 0 || c.DepthBytesPerSample <= 0 || c.HistoryBytesPerTexel <= 0 {
		return errors.New("pixel byte coefficients must be positive")
	}
	if c.Foveation.MediumDivisor < 1 || c.Foveation.EdgeDivisor < c.Foveation.MediumDivisor {
		return errors.New("foveation divisors must satisfy 1 <= medium <= edge")
	}
	if c.Quality.RequiredDivisor != 1 {
		return errors.New("center quality contract currently requires divisor 1")
	}
	if c.Quality.CenterRadius <= 0 || c.Quality.CenterRadius > 2 {
		return errors.New("quality center radius must be in (0,2]")
	}
	if c.SingleEye.DisocclusionFraction < 0 || c.SingleEye.DisocclusionFraction > 1 {
		return errors.New("disocclusion fraction must be in [0,1]")
	}
	return nil
}

func (c FrameConfig) msaa() int {
	if c.MSAASamples < 1 {
		return 1
	}
	return c.MSAASamples
}
