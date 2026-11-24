package decallab

import (
	"fmt"
	"math"
)

type SamplingMode string

const (
	ModeIndependent   SamplingMode = "independent_per_eye"
	ModeSharedWorld   SamplingMode = "shared_world_cache"
	ModeCopiedTangent SamplingMode = "copied_monocular_tangent"
)

type DensityConfig struct {
	Label            string  `json:"label"`
	PixelStride      float64 `json:"pixel_stride"`
	MinSpacingM      float64 `json:"min_spacing_m"`
	FallbackSpacingM float64 `json:"fallback_spacing_m"`
	SharedCellM      float64 `json:"shared_cell_m"`
}

type MaterialSample struct {
	Point    Vec3         `json:"point"`
	Frame    Frame        `json:"frame"`
	SpacingM float64      `json:"spacing_m"`
	LOD      int          `json:"lod"`
	Decal    DecalSample  `json:"decal"`
	Normal   NormalSample `json:"normal"`
	Shadow   ShadowSample `json:"shadow"`
}

func EvaluateMaterial(point Vec3, surface Surface, decal Decal, detail DetailNormal, shadow ContactShadow, spacingM float64, forcedLOD int) MaterialSample {
	frame := surface.FrameAt(point)
	lod := forcedLOD
	if lod < 0 {
		lod = detail.LODForSpacing(spacingM, 2.2)
	}
	return MaterialSample{
		Point:    point,
		Frame:    frame,
		SpacingM: spacingM,
		LOD:      lod,
		Decal:    decal.Sample(point, frame),
		Normal:   detail.Evaluate(point, frame, lod, spacingM),
		Shadow:   shadow.Sample(point),
	}
}

type SharedCacheEntry struct {
	Key      string         `json:"key"`
	CellX    int            `json:"cell_x"`
	CellY    int            `json:"cell_y"`
	FirstEye Eye            `json:"first_eye"`
	Sample   MaterialSample `json:"sample"`
}

// WorldSampleCache models shared sampling on a meter-sized surface lattice in
// the decal's world anchor frame. Both eyes reuse the cell-center material
// sample. Independent per-eye evaluation instead uses each exact hit point.
type WorldSampleCache struct {
	CellSizeM float64                     `json:"cell_size_m"`
	Anchor    Vec3                        `json:"anchor"`
	AxisU     Vec3                        `json:"axis_u"`
	AxisV     Vec3                        `json:"axis_v"`
	Normal    Vec3                        `json:"normal"`
	Entries   map[string]SharedCacheEntry `json:"entries"`
	Hits      int                         `json:"hits"`
	Misses    int                         `json:"misses"`
}

func NewWorldSampleCache(cellSizeM float64, decal Decal) *WorldSampleCache {
	if cellSizeM <= 0 {
		cellSizeM = 0.025
	}
	return &WorldSampleCache{
		CellSizeM: cellSizeM,
		Anchor:    decal.Center,
		AxisU:     decal.AxisU,
		AxisV:     decal.AxisV,
		Normal:    decal.Normal,
		Entries:   make(map[string]SharedCacheEntry),
	}
}

func (c *WorldSampleCache) CellCoordinates(point Vec3) (ix, iy int) {
	delta := point.Sub(c.Anchor)
	u := delta.Dot(c.AxisU) / c.CellSizeM
	v := delta.Dot(c.AxisV) / c.CellSizeM
	return int(math.Floor(u)), int(math.Floor(v))
}

func (c *WorldSampleCache) Key(point Vec3, surfaceID string) string {
	ix, iy := c.CellCoordinates(point)
	return fmt.Sprintf("%s/%d/%d", surfaceID, ix, iy)
}

// SnapPoint returns the shared lattice center, mapped back onto curved surfaces.
func (c *WorldSampleCache) SnapPoint(point Vec3, surface Surface) Vec3 {
	ix, iy := c.CellCoordinates(point)
	offsetU := (float64(ix) + 0.5) * c.CellSizeM
	offsetV := (float64(iy) + 0.5) * c.CellSizeM
	offset := c.AxisU.Scale(offsetU).Add(c.AxisV.Scale(offsetV))
	distance := offset.Norm()
	if distance < 1e-10 {
		return c.Anchor
	}
	return PointOnSurfaceAtDistance(surface, c.Anchor, offset.Normalize(), distance)
}

// GetOrEvaluatePoint returns a cached sample when the destination eye lands in
// the same world cell. Otherwise it evaluates and stores the destination point.
func (c *WorldSampleCache) GetOrEvaluatePoint(eye Eye, point Vec3, surface Surface, decal Decal, detail DetailNormal, shadow ContactShadow, spacingM float64, forcedLOD int) (sample MaterialSample, key string, shared bool) {
	ix, iy := c.CellCoordinates(point)
	key = fmt.Sprintf("%s/%d/%d", surface.ID(), ix, iy)
	if entry, ok := c.Entries[key]; ok {
		c.Hits++
		return entry.Sample, key, true
	}
	c.Misses++
	samplePoint := c.SnapPoint(point, surface)
	sample = EvaluateMaterial(samplePoint, surface, decal, detail, shadow, spacingM, forcedLOD)
	c.Entries[key] = SharedCacheEntry{
		Key:      key,
		CellX:    ix,
		CellY:    iy,
		FirstEye: eye,
		Sample:   sample,
	}
	return sample, key, false
}

func MaterialDifference(a, b MaterialSample) (pointErrorM, normalAngleDeg, arcWidthPct, arcHeightPct, shadowAlphaDelta float64) {
	pointErrorM = a.Point.Distance(b.Point)
	normalAngleDeg = angleBetweenDeg(a.Normal.WorldNormal, b.Normal.WorldNormal)
	arcWidthPct = math.Abs(percentError(a.Decal.SurfaceArcWidthM, b.Decal.SurfaceArcWidthM))
	arcHeightPct = math.Abs(percentError(a.Decal.SurfaceArcHeightM, b.Decal.SurfaceArcHeightM))
	shadowAlphaDelta = math.Abs(a.Shadow.Alpha - b.Shadow.Alpha)
	return
}
