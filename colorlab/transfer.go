package colorlab

import "math"

const (
	// Exact intersection of 12.92*x and 1.055*x^(1/2.4)-0.055. The commonly
	// published 0.04045/0.0031308 pair is rounded and leaves a tiny backward
	// step at the segment boundary.
	srgbLinearThreshold  = 0.0031306684425005367
	srgbEncodedThreshold = 0.04044823627710693

	rec2020Alpha = 1.09929682680944
	rec2020Beta  = 0.018053968510807

	pqM1 = 2610.0 / 16384.0
	pqM2 = 2523.0 / 4096.0 * 128.0
	pqC1 = 3424.0 / 4096.0
	pqC2 = 2413.0 / 4096.0 * 32.0
	pqC3 = 2392.0 / 4096.0 * 32.0

	hlgA = 0.17883277
	// BT.2100 publishes b=0.28466892 and c=0.55991073. These values retain
	// the same curve to 1e-7 while making both the 1/12 breakpoint and the
	// [0,1] endpoints continuous.
	hlgB = 0.28466894101887735
	hlgC = 0.5599107347842814
)

func sRGBToLinear(v float64) float64 {
	if v <= srgbEncodedThreshold {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func linearToSRGB(v float64) float64 {
	if v <= srgbLinearThreshold {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1.0/2.4) - 0.055
}

func rec2020ToLinear(v float64) float64 {
	if v < 4.5*rec2020Beta {
		return v / 4.5
	}
	return math.Pow((v+rec2020Alpha-1.0)/rec2020Alpha, 1.0/0.45)
}

func linearToRec2020(v float64) float64 {
	if v < rec2020Beta {
		return 4.5 * v
	}
	return rec2020Alpha*math.Pow(v, 0.45) - (rec2020Alpha - 1.0)
}

func pqToLinear(v float64) float64 {
	if v <= 0 {
		return 0
	}
	p := math.Pow(v, 1.0/pqM2)
	numerator := math.Max(p-pqC1, 0)
	denominator := pqC2 - pqC3*p
	if denominator <= 0 {
		return 0
	}
	return math.Pow(numerator/denominator, 1.0/pqM1)
}

func linearToPQ(v float64) float64 {
	if v <= 0 {
		return 0
	}
	p := math.Pow(v, pqM1)
	return math.Pow((pqC1+pqC2*p)/(1.0+pqC3*p), pqM2)
}

func hlgToLinear(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v <= 0.5 {
		return v * v / 3.0
	}
	return (math.Exp((v-hlgC)/hlgA) + hlgB) / 12.0
}

func linearToHLG(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v <= 1.0/12.0 {
		return math.Sqrt(3.0 * v)
	}
	return hlgA*math.Log(12.0*v-hlgB) + hlgC
}
