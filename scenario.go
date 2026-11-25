package main

// Scenario describes one D65 -> D50 -> D65 round trip.
type Scenario struct {
	Name    string
	FwdCAT  Mat3    // CAT used D65 -> D50
	FwdT    float64 // forward adaptation strength
	BwdCAT  Mat3    // CAT used D50 -> D65
	BwdT    float64 // backward adaptation strength
	BwdFrom Vec3   // white the backward step *assumes* the data is in
	BwdTo   Vec3
}

// RoundTrip pushes an sRGB color through the scenario and returns the
// resulting sRGB color.
func (s Scenario) RoundTrip(srgb Vec3) Vec3 {
	xyz := SRGBToXYZ(srgb)
	fwd := AdaptMatrix(WhiteD65, WhiteD50, s.FwdCAT, s.FwdT)
	xyz50 := matVec(fwd, xyz)
	bwd := AdaptMatrix(s.BwdFrom, s.BwdTo, s.BwdCAT, s.BwdT)
	back := matVec(bwd, xyz50)
	return XYZToSRGB(back)
}

func scenarios() []Scenario {
	return []Scenario{
		{
			Name:    "paired Bradford t=1 (correct)",
			FwdCAT:  CATBradford, FwdT: 1,
			BwdCAT: CATBradford, BwdT: 1,
			BwdFrom: WhiteD50, BwdTo: WhiteD65,
		},
		{
			Name:    "mismatched white (no back adaptation)",
			FwdCAT:  CATBradford, FwdT: 1,
			BwdCAT: CATBradford, BwdT: 1,
			BwdFrom: WhiteD65, BwdTo: WhiteD65, // assumes data is already D65
		},
		{
			Name:    "mismatched matrix (fwd Bradford, back XYZ-scale)",
			FwdCAT:  CATBradford, FwdT: 1,
			BwdCAT: CATXYZScale, BwdT: 1,
			BwdFrom: WhiteD50, BwdTo: WhiteD65,
		},
		{
			Name:    "mismatched matrix (fwd Bradford, back CAT02)",
			FwdCAT:  CATBradford, FwdT: 1,
			BwdCAT: CATCAT02, BwdT: 1,
			BwdFrom: WhiteD50, BwdTo: WhiteD65,
		},
		{
			Name:    "partial adaptation t=0.5 both ways",
			FwdCAT:  CATBradford, FwdT: 0.5,
			BwdCAT: CATBradford, BwdT: 0.5,
			BwdFrom: WhiteD50, BwdTo: WhiteD65,
		},
	}
}

// Patch is a named display color (gamma-encoded sRGB).
type Patch struct {
	Name string
	RGB  Vec3
}

func patches() []Patch {
	// D50 paper white converted for a D65 display.
	d50Adapted := XYZToSRGB(matVec(AdaptMatrix(WhiteD50, WhiteD65, CATBradford, 1), WhiteD50))
	// D50 paper white dumped straight into sRGB with no adaptation.
	d50Raw := XYZToSRGB(WhiteD50)
	ps := []Patch{
		{"D65 white", Vec3{1, 1, 1}},
		{"D50 white (adapted)", d50Adapted},
		{"D50 white (unadapted)", d50Raw},
	}
	for _, g := range []float64{0.15, 0.3, 0.5, 0.7, 0.85} {
		ps = append(ps, Patch{"gray", Vec3{g, g, g}})
	}
	ps = append(ps,
		Patch{"red", Vec3{1, 0, 0}},
		Patch{"green", Vec3{0, 1, 0}},
		Patch{"blue", Vec3{0, 0, 1}},
		Patch{"cyan", Vec3{0, 1, 1}},
		Patch{"magenta", Vec3{1, 0, 1}},
		Patch{"yellow", Vec3{1, 1, 0}},
		Patch{"skin", Vec3{0.76, 0.57, 0.45}},
		Patch{"foliage", Vec3{0.35, 0.5, 0.25}},
	)
	return ps
}
