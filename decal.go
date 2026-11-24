package decallab

import "math"

// Decal stores all physical dimensions in meters. UVs are derived from world
// position; screen-space derivatives or a monocular tangent buffer are not
// authoritative for scale.
type Decal struct {
	ID        string  `json:"id"`
	SurfaceID string  `json:"surface_id"`
	Center    Vec3    `json:"center"`
	Normal    Vec3    `json:"normal"`
	AxisU     Vec3    `json:"axis_u"`
	AxisV     Vec3    `json:"axis_v"`
	WidthM    float64 `json:"width_m"`
	HeightM   float64 `json:"height_m"`
	AngleDeg  float64 `json:"orientation_deg"`
	Opacity   float64 `json:"opacity"`
	Pattern   string  `json:"pattern"`
}

func NewDecal(id string, surface Surface, anchor Vec3, width, height, angleDeg, opacity float64, pattern string) Decal {
	frame := surface.FrameAt(anchor)
	axisU, axisV := RotateFrameAxes(frame, angleDeg)
	return Decal{
		ID:        id,
		SurfaceID: surface.ID(),
		Center:    anchor,
		Normal:    frame.Normal,
		AxisU:     axisU,
		AxisV:     axisV,
		WidthM:    width,
		HeightM:   height,
		AngleDeg:  angleDeg,
		Opacity:   opacity,
		Pattern:   pattern,
	}
}

type DecalSample struct {
	DecalID             string  `json:"decal_id"`
	U                   float64 `json:"u"`
	V                   float64 `json:"v"`
	InBounds            bool    `json:"in_bounds"`
	PatternValue        float64 `json:"pattern_value"`
	EdgeDistanceM       float64 `json:"edge_distance_m"`
	ProjectedWidthM     float64 `json:"projected_width_m"`
	ProjectedHeightM    float64 `json:"projected_height_m"`
	SurfaceArcWidthM    float64 `json:"surface_arc_width_m"`
	SurfaceArcHeightM   float64 `json:"surface_arc_height_m"`
	CurvatureStretchPct float64 `json:"curvature_stretch_pct"`
}

func (d Decal) UV(point Vec3) (u, v float64) {
	delta := point.Sub(d.Center)
	return 0.5 + delta.Dot(d.AxisU)/d.WidthM,
		0.5 + delta.Dot(d.AxisV)/d.HeightM
}

func (d Decal) Sample(point Vec3, surfaceFrame Frame) DecalSample {
	u, v := d.UV(point)
	offsetU := (u - 0.5) * d.WidthM
	offsetV := (v - 0.5) * d.HeightM
	edgeU := d.WidthM/2 - math.Abs(offsetU)
	edgeV := d.HeightM/2 - math.Abs(offsetV)
	edgeDistance := math.Min(edgeU, edgeV)

	gradU := d.AxisU.Scale(1 / d.WidthM)
	gradV := d.AxisV.Scale(1 / d.HeightM)
	duDT := gradU.Dot(surfaceFrame.Tangent)
	duDB := gradU.Dot(surfaceFrame.Bitangent)
	dvDT := gradV.Dot(surfaceFrame.Tangent)
	dvDB := gradV.Dot(surfaceFrame.Bitangent)
	det := duDT*dvDB - duDB*dvDT

	arcWidth := math.Inf(1)
	arcHeight := math.Inf(1)
	if math.Abs(det) > 1e-10 {
		// Inverse Jacobian column for delta UV=(1,0), expressed in the local
		// Tangent/Bitangent basis. Its length is the surface arc distance for
		// one complete U span.
		iuT := dvDB / det
		iuB := -dvDT / det
		widthVector := surfaceFrame.Tangent.Scale(iuT).Add(surfaceFrame.Bitangent.Scale(iuB))
		arcWidth = widthVector.Norm()

		ivT := -duDB / det
		ivB := duDT / det
		heightVector := surfaceFrame.Tangent.Scale(ivT).Add(surfaceFrame.Bitangent.Scale(ivB))
		arcHeight = heightVector.Norm()
	}

	stretch := 0.0
	if !math.IsInf(arcWidth, 1) && d.WidthM != 0 {
		stretch = math.Max(
			math.Abs(percentError(arcWidth, d.WidthM)),
			math.Abs(percentError(arcHeight, d.HeightM)),
		)
	} else {
		stretch = math.Inf(1)
	}

	return DecalSample{
		DecalID:             d.ID,
		U:                   u,
		V:                   v,
		InBounds:            u >= 0 && u <= 1 && v >= 0 && v <= 1,
		PatternValue:        d.patternValue(u, v),
		EdgeDistanceM:       edgeDistance,
		ProjectedWidthM:     d.WidthM,
		ProjectedHeightM:    d.HeightM,
		SurfaceArcWidthM:    arcWidth,
		SurfaceArcHeightM:   arcHeight,
		CurvatureStretchPct: stretch,
	}
}

// patternValue is a deterministic directional arrow/stripe mask. It gives the
// report a high-frequency signal without images or random dependencies.
func (d Decal) patternValue(u, v float64) float64 {
	if d.Pattern == "stripe" {
		phase := math.Abs(math.Mod(u*6, 2) - 1)
		return 1 - smoothStep(0.18, 0.28, phase)
	}
	body := 1 - smoothStep(0.34, 0.39, math.Abs(v-0.5))
	shaftEnd := smoothStep(0.10, 0.18, u) * (1 - smoothStep(0.58, 0.66, u))
	head := 1 - smoothStep(0.16, 0.22, math.Abs(v-0.5)/(0.18+0.34*(u-0.62)))
	headZone := smoothStep(0.58, 0.66, u) * (1 - smoothStep(0.84, 0.91, u))
	return clamp01(body*shaftEnd + head*headZone)
}

// CopiedTangentDecalError measures how far a copied UV is from the UV that the
// destination eye should evaluate at its own world hit.
type CopiedDecalError struct {
	SourceU       float64 `json:"source_u"`
	SourceV       float64 `json:"source_v"`
	CorrectU      float64 `json:"correct_u"`
	CorrectV      float64 `json:"correct_v"`
	DeltaU        float64 `json:"delta_u"`
	DeltaV        float64 `json:"delta_v"`
	ReprojectionM float64 `json:"reprojection_error_m"`
	ScaleErrorPct float64 `json:"scale_error_pct"`
}

func (d Decal) MeasureCopiedTangent(sourcePoint, _ Vec3, copied DecalSample, correct DecalSample) CopiedDecalError {
	sourceU, sourceV := d.UV(sourcePoint)
	deltaU := copied.U - correct.U
	deltaV := copied.V - correct.V
	// Convert UV mismatch back to projector-plane meters. This is the physical
	// texture-coordinate displacement caused by direct monocular reuse.
	worldError := Vec3{}.
		Add(d.AxisU.Scale(deltaU * d.WidthM)).
		Add(d.AxisV.Scale(deltaV * d.HeightM)).
		Norm()
	scaleError := percentError(copied.SurfaceArcWidthM, correct.SurfaceArcWidthM)
	if math.IsInf(copied.SurfaceArcWidthM, 1) || math.IsNaN(copied.SurfaceArcWidthM) {
		scaleError = math.Inf(1)
	}
	return CopiedDecalError{
		SourceU:       sourceU,
		SourceV:       sourceV,
		CorrectU:      correct.U,
		CorrectV:      correct.V,
		DeltaU:        deltaU,
		DeltaV:        deltaV,
		ReprojectionM: worldError,
		ScaleErrorPct: math.Abs(scaleError),
	}
}
