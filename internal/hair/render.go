package hair

import "math"

type Camera struct {
	Width    int
	Height   int
	Distance float64
	Target   Vec3
	FOV      float64
	Focal    float64
}

func NewCamera(cfg CameraConfig, width, height int, distance float64) Camera {
	if width < 8 {
		width = 8
	}
	if height < 8 {
		height = 8
	}
	fov := cfg.FOVDegrees
	if fov <= 1 || fov >= 170 {
		fov = 48
	}
	return Camera{
		Width:    width,
		Height:   height,
		Distance: distance,
		Target:   Vec3{X: 0, Y: -0.55, Z: 0},
		FOV:      fov,
		Focal:    0.5 * float64(height) / math.Tan(fov*math.Pi/360),
	}
}

func (c Camera) Project(p Vec3) (Vec2, bool) {
	z := c.Distance - (p.Z - c.Target.Z)
	if z < 0.08 {
		return Vec2{}, false
	}
	scale := c.Focal / z
	return Vec2{
		X: float64(c.Width)*0.5 + (p.X-c.Target.X)*scale,
		Y: float64(c.Height)*0.5 - (p.Y-c.Target.Y)*scale,
	}, true
}

type DensityField struct {
	Width  int
	Height int
	Data   []float64
}

func NewDensityField(width, height int) *DensityField {
	return &DensityField{Width: width, Height: height, Data: make([]float64, width*height)}
}

func (f *DensityField) AddAt(x, y int, value float64) {
	if x < 0 || x >= f.Width || y < 0 || y >= f.Height {
		return
	}
	f.Data[y*f.Width+x] += value
}

func (a *Asset) RenderLevel(camera Camera, level int) *DensityField {
	field := NewDensityField(camera.Width, camera.Height)
	if level < 0 || level >= len(a.Levels) {
		return field
	}
	lod := a.Levels[level]
	meanMass := 1.0
	if len(lod.Entries) > 0 {
		meanMass = lod.TotalMass / float64(len(lod.Entries))
	}
	for _, entry := range lod.Entries {
		if len(entry.Points) < 2 || entry.Weight <= 0 {
			continue
		}
		massRatio := entry.Weight / meanMass
		segments := len(entry.Points) - 1
		segmentMass := entry.Weight / float64(segments)
		for i := 0; i < segments; i++ {
			p0, ok0 := camera.Project(entry.Points[i])
			p1, ok1 := camera.Project(entry.Points[i+1])
			if !ok0 || !ok1 {
				continue
			}
			midZ := camera.Distance - (entry.Points[i].Z+entry.Points[i+1].Z)*0.5 + camera.Target.Z
			baseRadius := clamp(camera.Focal*0.0045/math.Max(midZ, 0.1), 0.62, 1.15)
			radius := clamp(baseRadius*(0.72+0.36*math.Sqrt(massRatio)), 0.62, 2.25)
			splatSegment(field, p0, p1, segmentMass, radius)
		}
	}
	return field
}

func splatSegment(field *DensityField, a, b Vec2, mass, radius float64) {
	dx := b.X - a.X
	dy := b.Y - a.Y
	steps := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy))))
	if steps < 1 {
		steps = 1
	}
	sampleMass := mass / float64(steps+1)
	sigma := math.Max(radius, 0.55)
	if sigma < 0.9 {
		for step := 0; step <= steps; step++ {
			t := float64(step) / float64(steps)
			cx := a.X + dx*t - 0.5
			cy := a.Y + dy*t - 0.5
			x0 := int(math.Floor(cx))
			y0 := int(math.Floor(cy))
			fx := cx - float64(x0)
			fy := cy - float64(y0)
			field.AddAt(x0, y0, sampleMass*(1-fx)*(1-fy))
			field.AddAt(x0+1, y0, sampleMass*fx*(1-fy))
			field.AddAt(x0, y0+1, sampleMass*(1-fx)*fy)
			field.AddAt(x0+1, y0+1, sampleMass*fx*fy)
		}
		return
	}
	limit := int(math.Ceil(sigma * 2.2))
	for step := 0; step <= steps; step++ {
		t := float64(step) / float64(steps)
		cx := a.X + dx*t
		cy := a.Y + dy*t
		px := int(math.Floor(cx))
		py := int(math.Floor(cy))
		for oy := -limit; oy <= limit; oy++ {
			y := py + oy
			if y < 0 || y >= field.Height {
				continue
			}
			for ox := -limit; ox <= limit; ox++ {
				x := px + ox
				if x < 0 || x >= field.Width {
					continue
				}
				ddx := float64(x) + 0.5 - cx
				ddy := float64(y) + 0.5 - cy
				d2 := ddx*ddx + ddy*ddy
				if d2 <= 0.2 {
					field.Data[y*field.Width+x] += sampleMass
					continue
				}
				if d2 > (sigma*2.2)*(sigma*2.2) {
					continue
				}
				field.Data[y*field.Width+x] += sampleMass * math.Exp(-d2/(2*sigma*sigma))
			}
		}
	}
}

func (a *Asset) RenderLOD(camera Camera, distance float64) (*DensityField, Selection) {
	sel := a.Select(distance)
	field := a.RenderLevel(camera, sel.LevelA)
	if sel.LevelB == sel.LevelA || sel.Blend <= 1e-9 {
		return field, sel
	}
	coarse := a.RenderLevel(camera, sel.LevelB)
	if sel.Blend >= 1-1e-9 {
		return coarse, sel
	}
	blend := sel.Blend
	for i := range field.Data {
		field.Data[i] = field.Data[i]*(1-blend) + coarse.Data[i]*blend
	}
	return field, sel
}

func AlphaAt(field *DensityField, x, y int) float64 {
	if x < 0 || x >= field.Width || y < 0 || y >= field.Height {
		return 0
	}
	return 1 - math.Exp(-6*field.Data[y*field.Width+x])
}

type Coverage struct {
	Pixels int     `json:"pixels"`
	Sum    float64 `json:"sumAlpha"`
	MinX   int     `json:"minX"`
	MinY   int     `json:"minY"`
	MaxX   int     `json:"maxX"`
	MaxY   int     `json:"maxY"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
}

func (f *DensityField) Coverage(threshold float64) Coverage {
	c := Coverage{MinX: f.Width, MinY: f.Height, MaxX: -1, MaxY: -1}
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			a := AlphaAt(f, x, y)
			if a < threshold {
				continue
			}
			c.Pixels++
			c.Sum += a
			if x < c.MinX {
				c.MinX = x
			}
			if y < c.MinY {
				c.MinY = y
			}
			if x > c.MaxX {
				c.MaxX = x
			}
			if y > c.MaxY {
				c.MaxY = y
			}
		}
	}
	if c.MaxX >= 0 {
		c.Width = c.MaxX - c.MinX + 1
		c.Height = c.MaxY - c.MinY + 1
	}
	return c
}

func (f *DensityField) Occupancy(threshold float64) []uint8 {
	mask := make([]uint8, len(f.Data))
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			if AlphaAt(f, x, y) >= threshold {
				mask[y*f.Width+x] = 1
			}
		}
	}
	return mask
}

func maskIoU(a, b []uint8) float64 {
	var intersection, union int
	for i := range a {
		if a[i] != 0 || b[i] != 0 {
			union++
		}
		if a[i] != 0 && b[i] != 0 {
			intersection++
		}
	}
	if union == 0 {
		return 1
	}
	return float64(intersection) / float64(union)
}
