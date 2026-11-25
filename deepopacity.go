package main

import (
	"math"
	"sort"
)

// OrthoProjector is the stable orthographic coordinate system shared by the
// deep-opacity construction and world-space shadow queries.
type OrthoProjector struct {
	Center  Vec3
	Forward Vec3
	Right   Vec3
	Up      Vec3
	Span    float64
	Size    int
}

func NewLightProjector(lightToward Vec3, center Vec3, size int, span float64) OrthoProjector {
	forward := lightToward.Mul(-1).Normalized()
	right := forward.Cross(Vec(0, 1, 0)).Normalized()
	if right.LenSqr() < 1e-10 {
		right = Vec(1, 0, 0)
	}
	up := right.Cross(forward).Normalized()
	return OrthoProjector{
		Center:  center,
		Forward: forward,
		Right:   right,
		Up:      up,
		Span:    span,
		Size:    size,
	}
}

// ProjectPixel returns continuous image coordinates with y growing downward.
func (p OrthoProjector) ProjectPixel(point Vec3) (float64, float64, float64) {
	rel := point.Sub(p.Center)
	x := rel.Dot(p.Right)
	y := rel.Dot(p.Up)
	depth := rel.Dot(p.Forward)
	u := 0.5 + x/p.Span
	v := 0.5 + y/p.Span
	return u * float64(p.Size), (1 - v) * float64(p.Size), depth
}

func (p OrthoProjector) WorldToTile(point Vec3) (int, int, float64, bool) {
	px, py, depth := p.ProjectPixel(point)
	x := int(math.Floor(px))
	y := int(math.Floor(py))
	if x < 0 || y < 0 || x >= p.Size || y >= p.Size {
		return 0, 0, depth, false
	}
	return x, y, depth, true
}

// DeepSample is one compressed opacity layer in a light-space pixel.
// Attenuation is the colored multiplier contributed by this layer.
type DeepSample struct {
	Depth       float64
	StrandID    int
	Alpha       float64
	Attenuation Vec3
}

type DeepOpacityMap struct {
	Projector OrthoProjector
	Size      int
	Samples   [][]DeepSample
}

func NewDeepOpacityMap(projector OrthoProjector) *DeepOpacityMap {
	return &DeepOpacityMap{
		Projector: projector,
		Size:      projector.Size,
		Samples:   make([][]DeepSample, projector.Size*projector.Size),
	}
}

func BuildDeepOpacityMap(scene *Scene, size int) *DeepOpacityMap {
	projector := NewLightProjector(
		scene.LightDir,
		Vec(0, 0.60, 0.46),
		size,
		3.15,
	)
	dm := NewDeepOpacityMap(projector)

	for _, segment := range scene.Segments {
		dm.rasterizeSegment(segment)
	}
	dm.Compress(0.022)
	return dm
}

func (dm *DeepOpacityMap) rasterizeSegment(segment Segment) {
	x0, y0, d0 := dm.Projector.ProjectPixel(segment.A)
	x1, y1, d1 := dm.Projector.ProjectPixel(segment.B)
	scale := float64(dm.Size) / dm.Projector.Span
	r0 := segment.RA * scale
	r1 := segment.RB * scale
	radius := math.Max(r0, r1) + 0.9

	minX := maxInt(0, int(math.Floor(math.Min(x0, x1)-radius-1)))
	maxX := minInt(dm.Size-1, int(math.Ceil(math.Max(x0, x1)+radius+1)))
	minY := maxInt(0, int(math.Floor(math.Min(y0, y1)-radius-1)))
	maxY := minInt(dm.Size-1, int(math.Ceil(math.Max(y0, y1)+radius+1)))

	dx := x1 - x0
	dy := y1 - y0
	lenSqr := dx*dx + dy*dy
	guideAlpha := guideLayerAlpha(segment.StrandID)
	coveredTransmission := segment.Tint

	for y := minY; y <= maxY; y++ {
		py := float64(y) + 0.5
		for x := minX; x <= maxX; x++ {
			px := float64(x) + 0.5
			t := 0.0
			if lenSqr > 1e-9 {
				t = clamp01(((px-x0)*dx + (py-y0)*dy) / lenSqr)
			}
			cx := x0 + dx*t
			cy := y0 + dy*t
			dist := math.Hypot(px-cx, py-cy)
			pixelRadius := lerp(r0, r1, t)
			coverage := 1 - smoothstep(pixelRadius*0.62, pixelRadius+0.85, dist)
			alpha := coverage * guideAlpha
			if alpha < 0.025 {
				continue
			}

			attenuation := Vec3{
				X: (1 - alpha) + alpha*coveredTransmission.X,
				Y: (1 - alpha) + alpha*coveredTransmission.Y,
				Z: (1 - alpha) + alpha*coveredTransmission.Z,
			}
			depth := lerp(d0, d1, t)
			idx := y*dm.Size + x
			dm.Samples[idx] = append(dm.Samples[idx], DeepSample{
				Depth:       depth,
				StrandID:    segment.StrandID,
				Alpha:       alpha,
				Attenuation: attenuation,
			})
		}
	}
}

// Compress sorts linked fragments and merges only nearly coincident fragments.
// Distinct guide layers remain separate, which is what removes the binary
// block shadow produced by a nearest-depth-only map.
func (dm *DeepOpacityMap) Compress(mergeDepth float64) {
	for tile, samples := range dm.Samples {
		if len(samples) == 0 {
			continue
		}
		samples = mergeSameGuideSamples(samples)
		sort.Slice(samples, func(i, j int) bool {
			return samples[i].Depth < samples[j].Depth
		})

		compressed := samples[:0]
		for _, sample := range samples {
			if len(compressed) == 0 {
				compressed = append(compressed, sample)
				continue
			}
			last := &compressed[len(compressed)-1]
			if sample.Depth-last.Depth <= mergeDepth {
				totalAlpha := 1 - (1-last.Alpha)*(1-sample.Alpha)
				weightA := last.Alpha / math.Max(totalAlpha, 1e-6)
				weightB := sample.Alpha / math.Max(totalAlpha, 1e-6)
				last.Depth = last.Depth*weightA + sample.Depth*weightB
				last.Alpha = totalAlpha
				last.Attenuation = last.Attenuation.MulVec(sample.Attenuation)
			} else {
				compressed = append(compressed, sample)
			}
		}
		dm.Samples[tile] = compressed
	}
}

func mergeSameGuideSamples(samples []DeepSample) []DeepSample {
	sort.Slice(samples, func(i, j int) bool {
		if samples[i].StrandID != samples[j].StrandID {
			return samples[i].StrandID < samples[j].StrandID
		}
		return samples[i].Depth < samples[j].Depth
	})

	merged := samples[:0]
	for _, sample := range samples {
		if len(merged) == 0 || merged[len(merged)-1].StrandID != sample.StrandID {
			merged = append(merged, sample)
			continue
		}
		// Adjacent polyline segments overlap at their joint. Treat that as one
		// guide layer and retain its strongest coverage instead of squaring it.
		if sample.Alpha > merged[len(merged)-1].Alpha {
			merged[len(merged)-1] = sample
		}
	}
	return merged
}

// TransmittanceAt returns colored light transmission through every deep layer
// strictly before point. A small depth bias prevents a strand from shadowing
// its own immediately adjacent fragment.
func (dm *DeepOpacityMap) TransmittanceAt(point Vec3) Vec3 {
	return dm.TransmittanceAtExcluding(point, -1)
}

// TransmittanceAtExcluding skips one guide ID. Hair shading uses this to avoid
// treating the guide currently being shaded as an earlier opacity layer.
func (dm *DeepOpacityMap) TransmittanceAtExcluding(point Vec3, excludeStrand int) Vec3 {
	px, py, depth := dm.Projector.ProjectPixel(point)
	if px < -2 || py < -2 || px > float64(dm.Size)+2 || py > float64(dm.Size)+2 {
		return Vec(1, 1, 1)
	}

	texelX := px - 0.5
	texelY := py - 0.5
	x0f := math.Floor(texelX)
	y0f := math.Floor(texelY)
	x0 := int(x0f)
	y0 := int(y0f)
	tx := texelX - x0f
	ty := texelY - y0f

	x0c := clampInt(x0, 0, dm.Size-1)
	x1c := clampInt(x0+1, 0, dm.Size-1)
	y0c := clampInt(y0, 0, dm.Size-1)
	y1c := clampInt(y0+1, 0, dm.Size-1)

	c00 := dm.tileTransmittance(x0c, y0c, depth, excludeStrand)
	c10 := dm.tileTransmittance(x1c, y0c, depth, excludeStrand)
	c01 := dm.tileTransmittance(x0c, y1c, depth, excludeStrand)
	c11 := dm.tileTransmittance(x1c, y1c, depth, excludeStrand)
	low := mixVec(c00, c10, tx)
	high := mixVec(c01, c11, tx)
	return mixVec(low, high, ty)
}

func (dm *DeepOpacityMap) tileTransmittance(x, y int, depth float64, excludeStrand int) Vec3 {
	const depthBias = 0.014
	result := Vec(1, 1, 1)
	for _, sample := range dm.Samples[y*dm.Size+x] {
		if sample.StrandID == excludeStrand {
			continue
		}
		if sample.Depth+depthBias >= depth {
			break
		}
		result = result.MulVec(sample.Attenuation)
	}
	return result
}

// TotalTransmittance returns the complete light-space deep-opacity slice.
func (dm *DeepOpacityMap) TotalTransmittance(x, y int) Vec3 {
	result := Vec(1, 1, 1)
	for _, sample := range dm.Samples[y*dm.Size+x] {
		result = result.MulVec(sample.Attenuation)
	}
	return result
}

// LayerCount is a diagnostic/test helper.
func (dm *DeepOpacityMap) LayerCount(x, y int) int {
	if x < 0 || y < 0 || x >= dm.Size || y >= dm.Size {
		return 0
	}
	return len(dm.Samples[y*dm.Size+x])
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
