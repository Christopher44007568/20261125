package main

import "math"

const (
	sceneW = 640
	sceneH = 360
	ssaa   = 2

	grayNits       = 18.0
	grayLinear     = grayNits / sdrWhiteNits
	grayProofCode  = 118.0
	sdrWhiteNits   = 100.0
	pqPeakNits     = 1000.0
	hlgPeakNits    = 1000.0
	hlgSystemGamma = 1.2
)

type Rect struct {
	X0, Y0, X1, Y1 float64
}

func (r Rect) contains(x, y float64) bool {
	return x >= r.X0 && x < r.X1 && y >= r.Y0 && y < r.Y1
}

func (r Rect) center() (float64, float64) {
	return (r.X0 + r.X1) / 2, (r.Y0 + r.Y1) / 2
}

type Scene struct {
	ID       string
	Title    string
	Subtitle string
	Gray     Rect
	Pixels   []RGB
	Width    int
	Height   int
}

func (s *Scene) index(x, y int) int {
	return y*s.Width + x
}

func (s *Scene) at(x, y int) RGB {
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return RGB{}
	}
	return s.Pixels[s.index(x, y)]
}

func hash2(x, y int64) float64 {
	n := uint64(x)*0x9e3779b97f4a7c15 ^ uint64(y)*0xbf58476d1ce4e5b9
	n ^= n >> 30
	n *= 0xbf58476d1ce4e5b9
	n ^= n >> 27
	n *= 0x94d049bb133111eb
	n ^= n >> 31
	return float64(n>>11) / float64(1<<53)
}

func valueNoise(x, y float64) float64 {
	x0 := math.Floor(x)
	y0 := math.Floor(y)
	tx := smoothstep(0, 1, x-x0)
	ty := smoothstep(0, 1, y-y0)
	ix0 := int64(x0)
	iy0 := int64(y0)
	a := hash2(ix0, iy0)
	b := hash2(ix0+1, iy0)
	c := hash2(ix0, iy0+1)
	d := hash2(ix0+1, iy0+1)
	return mix(mix(a, b, tx), mix(c, d, tx), ty)
}

func fbm(x, y float64) float64 {
	sum := 0.0
	amp := 0.5
	for i := 0; i < 4; i++ {
		sum += amp * valueNoise(x, y)
		x = x*2.03 + 17.1
		y = y*2.01 - 9.2
		amp *= 0.5
	}
	return sum
}
