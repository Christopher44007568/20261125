package gamut

import "math"

// Matrix rows are stored explicitly; all matrices below operate on column
// vectors and include the D65 white point.
var rec2020ToXYZ = [3][3]float64{
	{0.6369580483012914, 0.14461690358620832, 0.1688809751641721},
	{0.2627002120112671, 0.6779980715188708, 0.05930171646986196},
	{0, 0.028072693049087428, 1.060985057710791},
}

var xyzToLinearSRGB = [3][3]float64{
	{3.2409699419045226, -1.537383177570094, -0.4986107602930034},
	{-0.9692436362808796, 1.8759675015077202, 0.04155505740717559},
	{0.05563007969699366, -0.20397695888897652, 1.0569715142428786},
}

var xyzToRec2020 = [3][3]float64{
	{1.716651187971268, -0.355670783776392, -0.253366281373660},
	{-0.666684351832489, 1.616481236634939, 0.015768545813911},
	{0.017639857445311, -0.042770613257809, 0.942103121235474},
}

var srgbToXYZ = [3][3]float64{
	{0.41239079926595934, 0.357584339383878, 0.1804807884018343},
	{0.21263900587151027, 0.715168678767756, 0.07219231536073371},
	{0.01933081871559182, 0.11919477979462598, 0.9505321522496607},
}

var xyzToLMS = [3][3]float64{
	{0.8189330101, 0.3618667424, -0.1288597137},
	{0.0329845436, 0.9293118715, 0.0361456387},
	{0.0482003018, 0.2643662691, 0.6338517070},
}

var lmsToOklab = [3][3]float64{
	{0.2104542553, 0.7936177850, -0.0040720468},
	{1.9779984951, -2.4285922050, 0.4505937099},
	{0.0259040371, 0.7827717662, -0.8086757660},
}

var oklabToLMS = [3][3]float64{
	{1, 0.3963377774, 0.2158037573},
	{1, -0.1055613458, -0.0638541728},
	{1, -0.0894841775, -1.2914855480},
}

var lmsToXYZ = [3][3]float64{
	{1.2270138511, -0.5577999807, 0.2812561490},
	{-0.0405801784, 1.1122568696, -0.0716766787},
	{-0.0763812845, -0.4214819784, 1.5861632204},
}

type xyz struct {
	x, y, z float64
}

type lab struct {
	l, a, b float64
}

func linearSRGBToXYZ(in RGB) xyz {
	r, g, b := in.R, in.G, in.B
	return xyz{
		x: srgbToXYZ[0][0]*r + srgbToXYZ[0][1]*g + srgbToXYZ[0][2]*b,
		y: srgbToXYZ[1][0]*r + srgbToXYZ[1][1]*g + srgbToXYZ[1][2]*b,
		z: srgbToXYZ[2][0]*r + srgbToXYZ[2][1]*g + srgbToXYZ[2][2]*b,
	}
}

func xyzToLinearSRGBColor(in xyz) RGB {
	x, y, z := in.x, in.y, in.z
	return RGB{
		R: xyzToLinearSRGB[0][0]*x + xyzToLinearSRGB[0][1]*y + xyzToLinearSRGB[0][2]*z,
		G: xyzToLinearSRGB[1][0]*x + xyzToLinearSRGB[1][1]*y + xyzToLinearSRGB[1][2]*z,
		B: xyzToLinearSRGB[2][0]*x + xyzToLinearSRGB[2][1]*y + xyzToLinearSRGB[2][2]*z,
	}
}

func xyzToRec2020Linear(in xyz) RGB {
	x, y, z := in.x, in.y, in.z
	return RGB{
		R: xyzToRec2020[0][0]*x + xyzToRec2020[0][1]*y + xyzToRec2020[0][2]*z,
		G: xyzToRec2020[1][0]*x + xyzToRec2020[1][1]*y + xyzToRec2020[1][2]*z,
		B: xyzToRec2020[2][0]*x + xyzToRec2020[2][1]*y + xyzToRec2020[2][2]*z,
	}
}

func rec2020LinearToXYZ(in RGB) xyz {
	r, g, b := in.R, in.G, in.B
	return xyz{
		x: rec2020ToXYZ[0][0]*r + rec2020ToXYZ[0][1]*g + rec2020ToXYZ[0][2]*b,
		y: rec2020ToXYZ[1][0]*r + rec2020ToXYZ[1][1]*g + rec2020ToXYZ[1][2]*b,
		z: rec2020ToXYZ[2][0]*r + rec2020ToXYZ[2][1]*g + rec2020ToXYZ[2][2]*b,
	}
}

func mul(m [3][3]float64, v [3]float64) [3]float64 {
	return [3]float64{
		m[0][0]*v[0] + m[0][1]*v[1] + m[0][2]*v[2],
		m[1][0]*v[0] + m[1][1]*v[1] + m[1][2]*v[2],
		m[2][0]*v[0] + m[2][1]*v[1] + m[2][2]*v[2],
	}
}

func xyzToOklab(in xyz) lab {
	lms := mul(xyzToLMS, [3]float64{in.x, in.y, in.z})
	lms = [3]float64{
		cbroot(lms[0]),
		cbroot(lms[1]),
		cbroot(lms[2]),
	}
	out := mul(lmsToOklab, lms)
	return lab{l: out[0], a: out[1], b: out[2]}
}

func oklabToXYZ(in lab) xyz {
	lms := mul(oklabToLMS, [3]float64{in.l, in.a, in.b})
	lms = [3]float64{
		lms[0] * lms[0] * lms[0],
		lms[1] * lms[1] * lms[1],
		lms[2] * lms[2] * lms[2],
	}
	out := mul(lmsToXYZ, lms)
	return xyz{x: out[0], y: out[1], z: out[2]}
}

func cbroot(v float64) float64 {
	// The LMS values of display colors are non-negative in practice. The
	// signed branch protects extrapolated values from returning NaN.
	if v < 0 {
		return -math.Pow(-v, 1.0/3.0)
	}
	return math.Pow(v, 1.0/3.0)
}

func oklabToOklch(in lab) (l, c, h float64) {
	l = in.l
	c = math.Hypot(in.a, in.b)
	h = math.Atan2(in.b, in.a)
	return l, c, h
}

func oklchToOklab(l, c, h float64) lab {
	return lab{
		l: l,
		a: c * math.Cos(h),
		b: c * math.Sin(h),
	}
}
