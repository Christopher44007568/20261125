package decallab

import "math"

type ContactShadow struct {
	ID        string  `json:"id"`
	SurfaceID string  `json:"surface_id"`
	Center    Vec3    `json:"center"`
	AxisU     Vec3    `json:"axis_u"`
	AxisV     Vec3    `json:"axis_v"`
	RadiusU   float64 `json:"radius_u_m"`
	RadiusV   float64 `json:"radius_v_m"`
	SoftnessM float64 `json:"softness_m"`
	Opacity   float64 `json:"opacity"`
}

func NewContactShadow(id string, surface Surface, center Vec3, radiusU, radiusV, softness, opacity, angleDeg float64) ContactShadow {
	frame := surface.FrameAt(center)
	u, v := RotateFrameAxes(frame, angleDeg)
	return ContactShadow{
		ID:        id,
		SurfaceID: surface.ID(),
		Center:    center,
		AxisU:     u,
		AxisV:     v,
		RadiusU:   radiusU,
		RadiusV:   radiusV,
		SoftnessM: softness,
		Opacity:   opacity,
	}
}

type ShadowSample struct {
	ShadowID   string  `json:"shadow_id"`
	Alpha      float64 `json:"alpha"`
	HalfAlpha  float64 `json:"half_alpha"`
	InsideEdge bool    `json:"inside_half_edge"`
}

func (s ContactShadow) Coordinates(point Vec3) (u, v, rho float64) {
	delta := point.Sub(s.Center)
	u = delta.Dot(s.AxisU) / s.RadiusU
	v = delta.Dot(s.AxisV) / s.RadiusV
	return u, v, math.Hypot(u, v)
}

func (s ContactShadow) AlphaAt(point Vec3) float64 {
	_, _, rho := s.Coordinates(point)
	meanRadius := math.Sqrt(s.RadiusU * s.RadiusV)
	halfWidth := s.SoftnessM / meanRadius
	if halfWidth < 1e-5 {
		halfWidth = 1e-5
	}
	coverage := 1 - smoothStep(1-halfWidth, 1+halfWidth, rho)
	return s.Opacity * coverage
}

func (s ContactShadow) Sample(point Vec3) ShadowSample {
	alpha := s.AlphaAt(point)
	half := s.Opacity * 0.5
	return ShadowSample{
		ShadowID:   s.ID,
		Alpha:      alpha,
		HalfAlpha:  half,
		InsideEdge: alpha >= half,
	}
}

func (s ContactShadow) Direction(angleDeg float64) Vec3 {
	rad := angleDeg * math.Pi / 180
	return s.AxisU.Scale(math.Cos(rad)).Add(s.AxisV.Scale(math.Sin(rad))).Normalize()
}

func (s ContactShadow) ExpectedEdgeRadiusM(direction Vec3) float64 {
	du := math.Abs(direction.Dot(s.AxisU))
	dv := math.Abs(direction.Dot(s.AxisV))
	denom := math.Hypot(du/s.RadiusU, dv/s.RadiusV)
	if denom < 1e-12 {
		return math.Inf(1)
	}
	return 1 / denom
}

type arcMover interface {
	PointAtArcDistance(origin, direction Vec3, distanceM float64) Vec3
}

func PointOnSurfaceAtDistance(surface Surface, origin, direction Vec3, distanceM float64) Vec3 {
	if mover, ok := surface.(arcMover); ok {
		return mover.PointAtArcDistance(origin, direction.Normalize(), distanceM)
	}
	return origin.Add(direction.Normalize().Scale(distanceM))
}

func (p *Plane) PointAtArcDistance(origin, direction Vec3, distanceM float64) Vec3 {
	return origin.Add(direction.Normalize().Scale(distanceM))
}

func (s *Sphere) PointAtArcDistance(origin, direction Vec3, distanceM float64) Vec3 {
	normal := origin.Sub(s.Center).Normalize()
	d := direction.Sub(normal.Scale(direction.Dot(normal))).Normalize()
	theta := distanceM / s.Radius
	return s.Center.Add(normal.Scale(s.Radius * math.Cos(theta))).
		Add(d.Scale(s.Radius * math.Sin(theta)))
}

type ShadowEdgeRecord struct {
	ShadowID        string  `json:"shadow_id"`
	DensityLabel    string  `json:"density_label"`
	DirectionDeg    float64 `json:"direction_deg"`
	SampleSpacingM  float64 `json:"sample_spacing_m"`
	ExpectedRadiusM float64 `json:"expected_radius_m"`
	SampledRadiusM  float64 `json:"sampled_radius_m"`
	EdgeBiasM       float64 `json:"edge_bias_m"`
	EdgeAbsBiasM    float64 `json:"edge_abs_bias_m"`
	EdgeErrorPct    float64 `json:"edge_error_pct"`
	Transition1090M float64 `json:"transition_10_90_m"`
	CrossingSamples int     `json:"crossing_samples"`
}

func MeasureShadowEdge(surface Surface, shadow ContactShadow, directionDeg, spacingM float64) ShadowEdgeRecord {
	direction := shadow.Direction(directionDeg)
	expected := shadow.ExpectedEdgeRadiusM(direction)
	if spacingM <= 0 {
		spacingM = 0.005
	}

	limit := expected + 4*spacingM + shadow.SoftnessM*4
	previousDistance := 0.0
	previousAlpha := shadow.AlphaAt(shadow.Center)
	sampledRadius := math.NaN()
	crossings := 0

	for distance := spacingM; distance <= limit+spacingM*0.5; distance += spacingM {
		point := PointOnSurfaceAtDistance(surface, shadow.Center, direction, distance)
		alpha := shadow.AlphaAt(point)
		threshold := shadow.Opacity * 0.5
		if previousAlpha >= threshold && alpha < threshold {
			fraction := (threshold - previousAlpha) / (alpha - previousAlpha)
			sampledRadius = previousDistance + clamp01(fraction)*(distance-previousDistance)
			crossings++
			break
		}
		previousDistance = distance
		previousAlpha = alpha
	}
	if math.IsNaN(sampledRadius) {
		sampledRadius = expected
	}

	highResolution := math.Max(expected/160, 0.00025)
	r90 := findAlphaCrossing(surface, shadow, direction, expected*0.25, expected*1.75, highResolution, shadow.Opacity*0.9)
	r10 := findAlphaCrossing(surface, shadow, direction, expected*0.25, expected*1.75, highResolution, shadow.Opacity*0.1)
	transition := math.Abs(r10 - r90)

	bias := sampledRadius - expected
	return ShadowEdgeRecord{
		ShadowID:        shadow.ID,
		DensityLabel:    "",
		DirectionDeg:    directionDeg,
		SampleSpacingM:  spacingM,
		ExpectedRadiusM: expected,
		SampledRadiusM:  sampledRadius,
		EdgeBiasM:       bias,
		EdgeAbsBiasM:    math.Abs(bias),
		EdgeErrorPct:    math.Abs(percentError(sampledRadius, expected)),
		Transition1090M: transition,
		CrossingSamples: crossings,
	}
}

func findAlphaCrossing(surface Surface, shadow ContactShadow, direction Vec3, start, end, step, threshold float64) float64 {
	previousD := start
	previousA := shadow.AlphaAt(PointOnSurfaceAtDistance(surface, shadow.Center, direction, start))
	for d := start + step; d <= end; d += step {
		a := shadow.AlphaAt(PointOnSurfaceAtDistance(surface, shadow.Center, direction, d))
		if (previousA >= threshold && a < threshold) || (previousA <= threshold && a > threshold) {
			fraction := (threshold - previousA) / (a - previousA)
			if math.IsNaN(fraction) {
				return previousD
			}
			return previousD + clamp01(fraction)*(d-previousD)
		}
		previousD = d
		previousA = a
	}
	return math.NaN()
}
