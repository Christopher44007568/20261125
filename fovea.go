package eyebw

import "math"

type Tile struct {
	IndexX        int     `json:"index_x"`
	IndexY        int     `json:"index_y"`
	X             int     `json:"x"`
	Y             int     `json:"y"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	Divisor       int     `json:"divisor"`
	Texels        uint64  `json:"texels"`
	FullPixels    uint64  `json:"full_pixels"`
	Samples       uint64  `json:"samples"`
	Distance      float64 `json:"distance"`
	MinDistance   float64 `json:"min_distance"`
	FoveaCenter   bool    `json:"fovea_center"`
	QualityCenter bool    `json:"quality_center"`
}

func clampUnit(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	if v < -1 {
		return -1
	}
	if v > 1 {
		return 1
	}
	return v
}

func tileDivisor(cfg FrameConfig, centerDistance, nearestDistance float64) (int, bool) {
	f := cfg.Foveation
	if !f.Enabled || nearestDistance <= f.CenterRadius {
		return 1, true
	}
	if centerDistance <= f.MidRadius {
		return f.MediumDivisor, false
	}
	return f.EdgeDivisor, false
}

func nearestTileDistance(cfg FrameConfig, tile Tile, gaze Gaze) float64 {
	left := float64(2*tile.X)/float64(cfg.Width) - 1
	right := float64(2*(tile.X+tile.Width))/float64(cfg.Width) - 1
	top := float64(2*tile.Y)/float64(cfg.Height) - 1
	bottom := float64(2*(tile.Y+tile.Height))/float64(cfg.Height) - 1
	closestX := clampFloat(gaze.X, left, right)
	closestY := clampFloat(gaze.Y, top, bottom)
	aspectX := float64(cfg.Width) / float64(cfg.Height)
	return math.Hypot((closestX-gaze.X)*aspectX, closestY-gaze.Y)
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func BuildTiles(cfg FrameConfig, gaze Gaze) []Tile {
	cfg = cfg.withDefaults()
	gx := clampUnit(gaze.X)
	gy := clampUnit(gaze.Y)
	aspectX := float64(cfg.Width) / float64(cfg.Height)
	msaa := cfg.msaa()

	tiles := make([]Tile, 0, (cfg.Width/cfg.Foveation.TileWidth+1)*(cfg.Height/cfg.Foveation.TileHeight+1))
	for y := 0; y < cfg.Height; y += cfg.Foveation.TileHeight {
		th := cfg.Foveation.TileHeight
		if y+th > cfg.Height {
			th = cfg.Height - y
		}
		iy := y / cfg.Foveation.TileHeight
		for x := 0; x < cfg.Width; x += cfg.Foveation.TileWidth {
			tw := cfg.Foveation.TileWidth
			if x+tw > cfg.Width {
				tw = cfg.Width - x
			}
			ix := x / cfg.Foveation.TileWidth
			nx := (float64(x)+float64(tw)/2)/float64(cfg.Width)*2 - 1
			ny := (float64(y)+float64(th)/2)/float64(cfg.Height)*2 - 1
			dx := (nx - gx) * aspectX
			dy := ny - gy
			distance := math.Hypot(dx, dy)
			preTile := Tile{X: x, Y: y, Width: tw, Height: th}
			minDistance := nearestTileDistance(cfg, preTile, Gaze{X: gx, Y: gy})
			divisor, foveaCenter := tileDivisor(cfg, distance, minDistance)
			rw := (tw + divisor - 1) / divisor
			rh := (th + divisor - 1) / divisor
			full := mulSize(tw, th)
			texels := mulSize(rw, rh)
			tiles = append(tiles, Tile{
				IndexX:        ix,
				IndexY:        iy,
				X:             x,
				Y:             y,
				Width:         tw,
				Height:        th,
				Divisor:       divisor,
				Texels:        texels,
				FullPixels:    full,
				Samples:       texels * uint64(msaa),
				Distance:      distance,
				MinDistance:   minDistance,
				FoveaCenter:   foveaCenter,
				QualityCenter: minDistance <= cfg.Quality.CenterRadius,
			})
		}
	}
	return tiles
}

type FoveationSummary struct {
	TileCount        int     `json:"tile_count"`
	FullPixels       uint64  `json:"full_pixels"`
	EffectiveTexels  uint64  `json:"effective_texels"`
	EffectiveRatio   float64 `json:"effective_ratio"`
	CenterTiles      int     `json:"center_tiles"`
	CenterFullPixels uint64  `json:"center_full_pixels"`
	CenterEffective  uint64  `json:"center_effective_texels"`
	MinDivisor       int     `json:"min_divisor"`
	MaxDivisor       int     `json:"max_divisor"`
}

func SummarizeTiles(tiles []Tile) FoveationSummary {
	out := FoveationSummary{TileCount: len(tiles), MinDivisor: 1 << 30}
	for _, tile := range tiles {
		out.FullPixels += tile.FullPixels
		out.EffectiveTexels += tile.Texels
		if tile.QualityCenter {
			out.CenterTiles++
			out.CenterFullPixels += tile.FullPixels
			out.CenterEffective += tile.Texels
		}
		if tile.Divisor < out.MinDivisor {
			out.MinDivisor = tile.Divisor
		}
		if tile.Divisor > out.MaxDivisor {
			out.MaxDivisor = tile.Divisor
		}
	}
	if out.FullPixels > 0 {
		out.EffectiveRatio = float64(out.EffectiveTexels) / float64(out.FullPixels)
	}
	if out.MinDivisor == 1<<30 {
		out.MinDivisor = 0
	}
	return out
}

// SweepGaze is a deterministic dynamic-gaze script. It moves between both
// horizontal edges so that foveated tiles are promoted and demoted.
func SweepGaze(frame, frames int) [2]Gaze {
	if frames <= 1 {
		return [2]Gaze{{X: 0, Y: 0}, {X: 0, Y: 0}}
	}
	phase := float64(frame) / float64(frames-1)
	x := math.Sin(phase*math.Pi*2-math.Pi/2) * 0.72
	y := math.Sin(phase*math.Pi*4) * 0.22
	ipdOffset := 0.035
	return [2]Gaze{
		{X: clampUnit(x - ipdOffset), Y: y},
		{X: clampUnit(x + ipdOffset), Y: y + 0.01},
	}
}
