package main

import "math"

// DeltaE2000 computes CIEDE2000 between two Lab colors (kL=kC=kH=1).
func DeltaE2000(lab1, lab2 Vec3) float64 {
	L1, a1, b1 := lab1[0], lab1[1], lab1[2]
	L2, a2, b2 := lab2[0], lab2[1], lab2[2]

	C1 := math.Hypot(a1, b1)
	C2 := math.Hypot(a2, b2)
	cBar := (C1 + C2) / 2
	cBar7 := math.Pow(cBar, 7)
	g := 0.5 * (1 - math.Sqrt(cBar7/(cBar7+math.Pow(25, 7))))

	a1p := a1 * (1 + g)
	a2p := a2 * (1 + g)
	C1p := math.Hypot(a1p, b1)
	C2p := math.Hypot(a2p, b2)

	hue := func(ap, b float64) float64 {
		if ap == 0 && b == 0 {
			return 0
		}
		h := math.Atan2(b, ap)
		if h < 0 {
			h += 2 * math.Pi
		}
		return h
	}
	h1p := hue(a1p, b1)
	h2p := hue(a2p, b2)

	dLp := L2 - L1
	dCp := C2p - C1p
	var dhp float64
	if C1p*C2p != 0 {
		dhp = h2p - h1p
		if dhp > math.Pi {
			dhp -= 2 * math.Pi
		} else if dhp < -math.Pi {
			dhp += 2 * math.Pi
		}
	}
	dHp := 2 * math.Sqrt(C1p*C2p) * math.Sin(dhp/2)

	lBarP := (L1 + L2) / 2
	cBarP := (C1p + C2p) / 2
	var hBarP float64
	if C1p*C2p == 0 {
		hBarP = h1p + h2p
	} else if math.Abs(h1p-h2p) <= math.Pi {
		hBarP = (h1p + h2p) / 2
	} else if h1p+h2p < 2*math.Pi {
		hBarP = (h1p + h2p + 2*math.Pi) / 2
	} else {
		hBarP = (h1p + h2p - 2*math.Pi) / 2
	}

	deg := math.Pi / 180
	T := 1 - 0.17*math.Cos(hBarP-30*deg) + 0.24*math.Cos(2*hBarP) +
		0.32*math.Cos(3*hBarP+6*deg) - 0.20*math.Cos(4*hBarP-63*deg)
	dTheta := 30 * deg * math.Exp(-math.Pow((hBarP/deg-275)/25, 2))
	cBarP7 := math.Pow(cBarP, 7)
	rc := 2 * math.Sqrt(cBarP7/(cBarP7+math.Pow(25, 7)))
	sl := 1 + 0.015*math.Pow(lBarP-50, 2)/math.Sqrt(20+math.Pow(lBarP-50, 2))
	sc := 1 + 0.045*cBarP
	sh := 1 + 0.015*cBarP*T
	rt := -math.Sin(2*dTheta) * rc

	dL := dLp / sl
	dC := dCp / sc
	dH := dHp / sh
	return math.Sqrt(dL*dL + dC*dC + dH*dH + rt*dC*dH)
}

// DeltaEFromSRGB converts both sRGB colors to Lab (D65) and returns dE2000.
func DeltaEFromSRGB(c1, c2 Vec3) float64 {
	return DeltaE2000(XYZToLab(SRGBToXYZ(c1), WhiteD65), XYZToLab(SRGBToXYZ(c2), WhiteD65))
}
