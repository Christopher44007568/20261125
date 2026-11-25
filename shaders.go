package main

import "math"

// Hair appearance parameters shared by all three models.
var (
	HairBase   = v(0.11, 0.06, 0.03) // dark brown absorption color
	HairSpec   = v(0.95, 0.90, 0.82)
	HairTint   = v(0.58, 0.36, 0.17)
	ScalpColor = v(0.24, 0.15, 0.11)
)

const (
	KKExponent = 26.0
	AmbientMix = 0.06

	// Absorption and scale for the global multiple-scattering transport.
	HairAbsorb     = 0.9
	TransportScale = 0.012

	// Target integrated specular radiance of each model's highlight term so
	// the energy comparison is not dominated by arbitrary lobe weights.
	TargetSpecEnergy = 0.55
)

// ShadeContext holds per-pixel geometry projected into fiber-local space.
type ShadeContext struct {
	View, Light    Vec3
	Tangent        Vec3
	Normal         Vec3
	Shell          float64
	H              Vec3
	ThetaI, ThetaH float64
	Phi            float64
	NDotL          float64
	Pos            Vec3
}

func makeShadeContext(sc *Scene, p *Pixel) ShadeContext {
	view := sc.Cam.Eye.Sub(p.Pos).Normalize()
	t := p.Tangent.Normalize()
	h := view.Add(sc.LightDir).Normalize()
	thetaI := math.Asin(clampf(sc.LightDir.Dot(t), -1, 1))
	thetaH := math.Asin(clampf(h.Dot(t), -1, 1))

	// Fiber normal-plane basis: project the view direction onto the plane
	// perpendicular to the fiber tangent (the true fiber cross-section frame).
	u := view.Sub(t.Scale(view.Dot(t))).Normalize()
	w := t.Cross(u).Normalize()
	lp := sc.LightDir.Sub(t.Scale(sc.LightDir.Dot(t)))
	vp := view.Sub(t.Scale(view.Dot(t)))
	phi := math.Atan2(lp.Dot(w), lp.Dot(u)) -
		math.Atan2(vp.Dot(w), vp.Dot(u))

	return ShadeContext{
		View: view, Light: sc.LightDir, Tangent: t, Normal: p.Normal, Pos: p.Pos,
		Shell: p.ShellDepth, H: h, ThetaI: thetaI, ThetaH: thetaH,
		Phi: wrapPi(phi), NDotL: p.Normal.Dot(sc.LightDir),
	}
}

func scalpShade(_ *Scene, c *ShadeContext) Vec3 {
	diff := clamp01(c.NDotL)
	return ScalpColor.Scale(AmbientMix + 0.9*diff)
}

// hairDiffuse is shared by all three models: a tangent-cone diffuse term plus
// an ambient floor. Only the specular/highlight term differs between models.
func hairDiffuse(c *ShadeContext) Vec3 {
	sinTL := math.Sqrt(math.Max(0, 1-c.Light.Dot(c.Tangent)*c.Light.Dot(c.Tangent)))
	sinTV := math.Sqrt(math.Max(0, 1-c.View.Dot(c.Tangent)*c.View.Dot(c.Tangent)))
	return HairBase.Scale(AmbientMix + 0.8*sinTL*sinTV)
}

// ---- Kajiya-Kay (1989): single tangent-cone specular lobe -------------------

func kajiyaSpec(c *ShadeContext) float64 {
	tDotH := c.H.Dot(c.Tangent)
	sinTH := math.Sqrt(math.Max(0, 1-tDotH*tDotH))
	return math.Pow(sinTH, KKExponent)
}

// ---- Compact single-scattering Marschner-style model -----------------------

func marschnerSpec(c *ShadeContext, betaScale float64) Vec3 {
	betaR := 0.10 * betaScale
	betaTT := 0.085 * betaScale
	betaTRT := 0.12 * betaScale
	longR := gauss(c.ThetaH-1.5*c.ThetaI, betaR)
	longTT := gauss(c.ThetaH-c.ThetaI, betaTT)
	longTRT := gauss(c.ThetaH+2.0*c.ThetaI, betaTRT)

	azR := periodicGauss(c.Phi, 0, 0.63)
	azTT := periodicGauss(c.Phi, 0, 0.32)
	azTRT := periodicGauss(c.Phi, math.Pi, 0.28)

	return HairSpec.Scale(0.78 * longR * azR).
		Add(HairTint.Scale(0.92 * longTT * azTT)).
		Add(HairBase.Scale(2.2 * 0.32 * longTRT * azTRT))
}

// ---- Zinke-style double-scattering approximation ---------------------------

func doubleGlobal(c *ShadeContext) Vec3 {
	avgTT := 0.32 / math.Sqrt(2*math.Pi)
	avgTRT := 0.28 / math.Sqrt(2*math.Pi)
	globalLong := gauss(c.ThetaH-c.ThetaI, 0.085*1.9) +
		0.6*gauss(c.ThetaH+2.0*c.ThetaI, 0.12*1.9)
	transport := math.Exp(-HairAbsorb * c.Shell / FiberRadius * TransportScale)
	return HairTint.Scale(0.92 * avgTT * globalLong).
		Add(HairBase.Scale(2.2 * 0.32 * avgTRT * globalLong)).
		Scale(1.35 * transport)
}

// ---- Calibrated shaders -----------------------------------------------------

var (
	kajiyaGain float64
	singleGain Vec3
	doubleGlob Vec3
	globalGain Vec3
)

func calibrateModels() {
	kajiyaGain = TargetSpecEnergy / integrateScalar(kajiyaSpec)
	singleEnergy := integrateColor(marschnerSpecUnit)
	singleGain = v(
		TargetSpecEnergy/singleEnergy.X,
		TargetSpecEnergy/singleEnergy.Y,
		TargetSpecEnergy/singleEnergy.Z,
	)
	doubleGlob = integrateColor(func(c *ShadeContext) Vec3 { return doubleGlobal(c) })
	globalGain = v(
		TargetSpecEnergy*0.45/doubleGlob.X,
		TargetSpecEnergy*0.45/doubleGlob.Y,
		TargetSpecEnergy*0.45/doubleGlob.Z,
	)
}

func marschnerSpecUnit(c *ShadeContext) Vec3 {
	return marschnerSpec(c, 1.0)
}

// integrateScalar averages a scalar highlight term over the full sphere of
// outgoing directions for a fixed incoming direction and fixed fiber frame.
func integrateScalar(fn func(*ShadeContext) float64) float64 {
	const n = 120
	tangent := v(1, 0, 0)
	normal := v(0, 1, 0)
	light := v(0.4, 0.7, 0.59).Normalize()
	dTheta := math.Pi / n
	dPhi := math.Pi / n
	var sum, weight float64
	for j := 0; j < n; j++ {
		theta := (float64(j) + 0.5) * dTheta
		st, ct := math.Sin(theta), math.Cos(theta)
		for k := 0; k < 2*n; k++ {
			phi := float64(k) * dPhi
			view := v(st*math.Cos(phi), ct, st*math.Sin(phi)).Normalize()
			c := buildCalibContext(view, light, tangent, normal)
			w := st * dTheta * dPhi
			sum += fn(&c) * w
			weight += w
		}
	}
	return sum / weight
}

func integrateColor(fn func(*ShadeContext) Vec3) Vec3 {
	const n = 120
	tangent := v(1, 0, 0)
	normal := v(0, 1, 0)
	light := v(0.4, 0.7, 0.59).Normalize()
	dTheta := math.Pi / n
	dPhi := math.Pi / n
	var sum Vec3
	var weight float64
	for j := 0; j < n; j++ {
		theta := (float64(j) + 0.5) * dTheta
		st, ct := math.Sin(theta), math.Cos(theta)
		for k := 0; k < 2*n; k++ {
			phi := float64(k) * dPhi
			view := v(st*math.Cos(phi), ct, st*math.Sin(phi)).Normalize()
			c := buildCalibContext(view, light, tangent, normal)
			w := st * dTheta * dPhi
			sum = sum.Add(fn(&c).Scale(w))
			weight += w
		}
	}
	return sum.Scale(1 / weight)
}

func buildCalibContext(view, light, tangent, normal Vec3) ShadeContext {
	h := view.Add(light).Normalize()
	thetaI := math.Asin(clampf(light.Dot(tangent), -1, 1))
	thetaH := math.Asin(clampf(h.Dot(tangent), -1, 1))
	u := view.Sub(tangent.Scale(view.Dot(tangent))).Normalize()
	wv := tangent.Cross(u).Normalize()
	lp := light.Sub(tangent.Scale(light.Dot(tangent)))
	vp := view.Sub(tangent.Scale(view.Dot(tangent)))
	phi := wrapPi(math.Atan2(lp.Dot(wv), lp.Dot(u)) -
		math.Atan2(vp.Dot(wv), vp.Dot(u)))
	return ShadeContext{
		View: view, Light: light, Tangent: tangent, Normal: normal,
		H: h, ThetaI: thetaI, ThetaH: thetaH, Phi: phi,
		NDotL: normal.Dot(light),
	}
}

func shadeKajiyaKay(_ *Scene, c *ShadeContext) Vec3 {
	return hairDiffuse(c).Add(HairSpec.Scale(kajiyaGain * kajiyaSpec(c)))
}

func shadeSingleScatter(_ *Scene, c *ShadeContext) Vec3 {
	return hairDiffuse(c).Add(componentMul(marschnerSpec(c, 1.0), singleGain))
}

func shadeDoubleScatter(_ *Scene, c *ShadeContext) Vec3 {
	// Local single scattering (same calibration) plus the azimuthally averaged
	// global transport term; its gain is calibrated independently.
	local := componentMul(marschnerSpec(c, 1.0), singleGain)
	global := componentMul(doubleGlobal(c), globalGain)
	return hairDiffuse(c).Add(local).Add(global)
}

func componentMul(a, b Vec3) Vec3 { return v(a.X*b.X, a.Y*b.Y, a.Z*b.Z) }
