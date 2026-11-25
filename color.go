package main

import "math"

// Vec3 is a 3-component color vector (XYZ, Lab, RGB, cone response).
type Vec3 [3]float64

// Mat3 is a row-major 3x3 matrix.
type Mat3 [3][3]float64

// CIE standard illuminants, Y = 1.
var (
	WhiteD65 = Vec3{0.95047, 1.0, 1.08883}
	WhiteD50 = Vec3{0.96422, 1.0, 0.82521}
)

// Chromatic adaptation transforms (cone-response matrices).
var (
	CATBradford = Mat3{
		{0.8951, 0.2664, -0.1614},
		{-0.7502, 1.7135, 0.0367},
		{0.0389, -0.0685, 1.0296},
	}
	CATCAT02 = Mat3{
		{0.7328, 0.4296, -0.1624},
		{-0.7036, 1.6975, 0.0061},
		{0.0030, 0.0136, 0.9834},
	}
	// CATXYZScale is the "wrong von Kries": diagonal scaling in XYZ.
	CATXYZScale = Mat3{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
)

// sRGB (D65) primary matrix, linear sRGB -> XYZ.
var sRGBToXYZMat = Mat3{
	{0.4124564, 0.3575761, 0.1804375},
	{0.2126729, 0.7151522, 0.0721750},
	{0.0193339, 0.1191920, 0.9503041},
}

func matVec(m Mat3, v Vec3) Vec3 {
	var r Vec3
	for i := 0; i < 3; i++ {
		r[i] = m[i][0]*v[0] + m[i][1]*v[1] + m[i][2]*v[2]
	}
	return r
}

func matMul(a, b Mat3) Mat3 {
	var r Mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			r[i][j] = a[i][0]*b[0][j] + a[i][1]*b[1][j] + a[i][2]*b[2][j]
		}
	}
	return r
}

func matInv(m Mat3) Mat3 {
	a, b, c := m[0][0], m[0][1], m[0][2]
	d, e, f := m[1][0], m[1][1], m[1][2]
	g, h, i := m[2][0], m[2][1], m[2][2]
	A := e*i - f*h
	B := -(d*i - f*g)
	C := d*h - e*g
	det := a*A + b*B + c*C
	inv := Mat3{
		{A, -(b*i - c*h), b*f - c*e},
		{B, a*i - c*g, -(a*f - c*d)},
		{C, -(a*h - b*g), a*e - b*d},
	}
	for r := 0; r < 3; r++ {
		for col := 0; col < 3; col++ {
			inv[r][col] /= det
		}
	}
	return inv
}

// AdaptMatrix builds M^-1 * D * M where D scales cone responses from
// srcWhite toward dstWhite by fraction t (0 = none, 1 = full).
func AdaptMatrix(srcWhite, dstWhite Vec3, cat Mat3, t float64) Mat3 {
	srcCone := matVec(cat, srcWhite)
	dstCone := matVec(cat, dstWhite)
	var d Mat3
	for ch := 0; ch < 3; ch++ {
		target := srcCone[ch] + t*(dstCone[ch]-srcCone[ch])
		d[ch][ch] = target / srcCone[ch]
	}
	return matMul(matInv(cat), matMul(d, cat))
}

func srgbToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func linearToSRGB(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// SRGBToXYZ converts gamma-encoded sRGB (D65) to CIE XYZ.
func SRGBToXYZ(srgb Vec3) Vec3 {
	lin := Vec3{srgbToLinear(srgb[0]), srgbToLinear(srgb[1]), srgbToLinear(srgb[2])}
	return matVec(sRGBToXYZMat, lin)
}

// XYZToSRGB converts CIE XYZ to gamma-encoded sRGB (D65), clamped to gamut.
func XYZToSRGB(xyz Vec3) Vec3 {
	lin := matVec(matInv(sRGBToXYZMat), xyz)
	return Vec3{
		clamp01(linearToSRGB(lin[0])),
		clamp01(linearToSRGB(lin[1])),
		clamp01(linearToSRGB(lin[2])),
	}
}

// XYZToLab converts XYZ to CIE L*a*b* relative to the given white point.
func XYZToLab(xyz, white Vec3) Vec3 {
	const eps = 216.0 / 24389.0 // (6/29)^3
	const kappa = 24389.0 / 27.0
	f := func(t float64) float64 {
		if t > eps {
			return math.Cbrt(t)
		}
		return (kappa*t + 16) / 116
	}
	fx := f(xyz[0] / white[0])
	fy := f(xyz[1] / white[1])
	fz := f(xyz[2] / white[2])
	return Vec3{116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)}
}
