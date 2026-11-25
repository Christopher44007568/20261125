package main

import (
	"math"
	"math/rand"
)

const (
	PatchA = 82.5
	PatchB = 52.5
)

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func (a Vec3) Add(b Vec3) Vec3    { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3    { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Len() float64       { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Unit() Vec3         { return a.Mul(1 / a.Len()) }
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Reference is the analytic preview shell used to synthesize the scan and to
// run the post-fit dense audit. It is never passed to the fitting routine.
type Reference struct{}

func (Reference) Z(x, y float64) float64 {
	nx := x / PatchA
	ny := y / PatchB
	r2 := nx*nx + ny*ny
	base := 10.5
	dome := 26.5 * math.Exp(-0.72*r2)
	ridge := 8.5 * nx * math.Exp(-0.56*r2)
	scoop := -5.4 * ny * ny * math.Exp(-0.50*r2)
	crown := 4.8 * math.Exp(-math.Pow((nx-0.22)/0.33, 2)-math.Pow((ny+0.06)/0.43, 2))
	shoulder := 1.7 * nx * ny * math.Exp(-0.40*r2)
	return base + dome + ridge + scoop + crown + shoulder
}

func (Reference) Normal(x, y float64) Vec3 {
	h := 0.20
	fx := (Reference{}.Z(x+h, y) - Reference{}.Z(x-h, y)) / (2 * h)
	fy := (Reference{}.Z(x, y+h) - Reference{}.Z(x, y-h)) / (2 * h)
	return Vec3{-fx, -fy, 1}.Unit()
}

func (Reference) Curvature(x, y float64) (mean, gaussian float64) {
	h := 0.42
	z0 := Reference{}.Z(x, y)
	zx := (Reference{}.Z(x+h, y) - Reference{}.Z(x-h, y)) / (2 * h)
	zy := (Reference{}.Z(x, y+h) - Reference{}.Z(x, y-h)) / (2 * h)
	zxx := (Reference{}.Z(x+h, y) - 2*z0 + Reference{}.Z(x-h, y)) / (h * h)
	zyy := (Reference{}.Z(x, y+h) - 2*z0 + Reference{}.Z(x, y-h)) / (h * h)
	zxy := (Reference{}.Z(x+h, y+h) - Reference{}.Z(x+h, y-h) -
		Reference{}.Z(x-h, y+h) + Reference{}.Z(x-h, y-h)) / (4 * h * h)
	g := 1 + zx*zx + zy*zy
	mean = ((1+zy*zy)*zxx - 2*zx*zy*zxy + (1+zx*zx)*zyy) / (2 * math.Pow(g, 1.5))
	gaussian = (zxx*zyy - zxy*zxy) / (g * g)
	return
}

func trimRadius(x, y float64) float64 {
	nx, ny := x/PatchA, y/PatchB
	return nx*nx + ny*ny
}

func inTrim(x, y, inset float64) bool {
	return trimRadius(x, y) <= 1-inset
}

const (
	PointClean = iota
	PointNoise
)

type ScanPoint struct {
	X, Y, Z      float64
	Kind         int
	Residual     float64
	Weight       float64
	Rejected     bool
	HoleAdjacent bool
}

type holeSpec struct {
	CX, CY, RX, RY, Angle float64
}

var scanHoles = []holeSpec{
	{CX: -0.34, CY: 0.31, RX: 0.145, RY: 0.205, Angle: -0.28},
	{CX: 0.37, CY: -0.28, RX: 0.125, RY: 0.155, Angle: 0.66},
	{CX: 0.10, CY: 0.60, RX: 0.105, RY: 0.075, Angle: 0.12},
}

func inScanHole(nx, ny float64) bool {
	for _, h := range scanHoles {
		dx, dy := nx-h.CX, ny-h.CY
		ca, sa := math.Cos(h.Angle), math.Sin(h.Angle)
		u := (ca*dx + sa*dy) / h.RX
		v := (-sa*dx + ca*dy) / h.RY
		if u*u+v*v <= 1 {
			return true
		}
	}
	return false
}

func halton(index, base int) float64 {
	f, r := 1.0, 0.0
	for index > 0 {
		f /= float64(base)
		r += f * float64(index%base)
		index /= base
	}
	return r
}

func GenerateScan(ref Reference) []ScanPoint {
	rng := rand.New(rand.NewSource(20261125))
	points := make([]ScanPoint, 0, 520)

	for i := 1; len(points) < 365 && i < 12000; i++ {
		nx := 2*halton(i, 2) - 1
		ny := 2*halton(i, 3) - 1
		if nx*nx+ny*ny >= 0.982 || inScanHole(nx, ny) {
			continue
		}
		x := nx*PatchA + rng.NormFloat64()*0.12
		y := ny*PatchB + rng.NormFloat64()*0.12
		z := ref.Z(x, y) + rng.NormFloat64()*0.20
		points = append(points, ScanPoint{X: x, Y: y, Z: z, Kind: PointClean, Weight: 1})
	}

	for i := 0; i < 128; i++ {
		theta := 2*math.Pi*(float64(i)+0.37)/128 + rng.NormFloat64()*0.007
		nx := 0.992 * math.Cos(theta)
		ny := 0.992 * math.Sin(theta)
		if inScanHole(nx, ny) {
			continue
		}
		x, y := nx*PatchA, ny*PatchB
		z := ref.Z(x, y) + rng.NormFloat64()*0.12
		points = append(points, ScanPoint{X: x, Y: y, Z: z, Kind: PointClean, Weight: 1})
	}

	for i := 0; i < 24; i++ {
		var x, y float64
		for attempt := 0; attempt < 200; attempt++ {
			radius := 0.18 + 0.68*rng.Float64()
			angle := 2 * math.Pi * rng.Float64()
			nx := radius * math.Cos(angle)
			ny := radius * math.Sin(angle)
			if !inScanHole(nx, ny) {
				x, y = nx*PatchA, ny*PatchB
				break
			}
		}
		if x == 0 && y == 0 {
			x, y = 0.58*PatchA, 0.58*PatchB
		}
		offset := 0.45 + math.Abs(rng.NormFloat64())*1.05
		if i >= 17 {
			offset = 3.2 + rng.Float64()*6.8
		}
		if rng.Intn(2) == 0 {
			offset = -offset
		}
		x += rng.NormFloat64() * 0.65
		y += rng.NormFloat64() * 0.65
		points = append(points, ScanPoint{
			X: x, Y: y, Z: ref.Z(clamp(x, -PatchA, PatchA), clamp(y, -PatchB, PatchB)) + offset,
			Kind: PointNoise, Weight: 1,
		})
	}
	return points
}
