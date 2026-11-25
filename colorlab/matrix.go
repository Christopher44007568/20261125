package colorlab

import "math"

// Matrix3 is a row-major 3x3 color matrix.
type Matrix3 [3][3]float64

// Apply multiplies a column vector by the matrix.
func (m Matrix3) Apply(c RGB) RGB {
	return RGB{
		R: m[0][0]*c.R + m[0][1]*c.G + m[0][2]*c.B,
		G: m[1][0]*c.R + m[1][1]*c.G + m[1][2]*c.B,
		B: m[2][0]*c.R + m[2][1]*c.G + m[2][2]*c.B,
	}
}

// Mul returns m*n.
func (m Matrix3) Mul(n Matrix3) Matrix3 {
	var out Matrix3
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			for k := 0; k < 3; k++ {
				out[row][col] += m[row][k] * n[k][col]
			}
		}
	}
	return out
}

// Inverse returns the inverse matrix. The supported RGB matrices are
// well-conditioned; a singular input produces a zero matrix.
func (m Matrix3) Inverse() Matrix3 {
	a, b, c := m[0][0], m[0][1], m[0][2]
	d, e, f := m[1][0], m[1][1], m[1][2]
	g, h, i := m[2][0], m[2][1], m[2][2]

	cofA := e*i - f*h
	cofB := f*g - d*i
	cofC := d*h - e*g
	det := a*cofA + b*cofB + c*cofC
	if math.Abs(det) < 1e-15 {
		return Matrix3{}
	}

	invDet := 1.0 / det
	return Matrix3{
		{cofA * invDet, (c*h - b*i) * invDet, (b*f - c*e) * invDet},
		{cofB * invDet, (a*i - c*g) * invDet, (c*d - a*f) * invDet},
		{cofC * invDet, (b*g - a*h) * invDet, (a*e - b*d) * invDet},
	}
}

var (
	// IEC 61966-2-1 sRGB primaries with a D65 white point.
	sRGBToXYZ = Matrix3{
		{0.41239079926595934, 0.3575843393838780, 0.18048078840183430},
		{0.21263900587151027, 0.7151686787677560, 0.07219231536073371},
		{0.01933081871559182, 0.11919477979462598, 0.95053215224966070},
	}

	// ITU-R BT.2020 primaries with a D65 white point.
	rec2020ToXYZ = Matrix3{
		{0.6369580483012914, 0.14461690358620832, 0.1688809751641721},
		{0.2627002120112671, 0.6779980715188708, 0.05930171646986196},
		{0.0, 0.028072693049087428, 1.060985057710791},
	}

	xyzToSRGB     = sRGBToXYZ.Inverse()
	xyzToRec2020  = rec2020ToXYZ.Inverse()
	srgbToRec2020 = xyzToRec2020.Mul(sRGBToXYZ)
	rec2020ToSRGB = srgbToRec2020.Inverse()
)

// SRGBToRec2020Matrix exposes the linear-light primary conversion used by the
// sRGB and BT.2020 transforms.
func SRGBToRec2020Matrix() Matrix3 {
	return srgbToRec2020
}

// Rec2020ToSRGBMatrix exposes the inverse linear-light conversion.
func Rec2020ToSRGBMatrix() Matrix3 {
	return rec2020ToSRGB
}
