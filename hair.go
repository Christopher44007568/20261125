package main

import (
	"math"
	"math/rand"
)

type RGB struct {
	R float64
	G float64
	B float64
}

func (c RGB) mul(s float64) RGB { return RGB{c.R * s, c.G * s, c.B * s} }

type Strip struct {
	ID     int
	Points []Vec3
	Widths []float64
	Color  RGB
	Parent int
}

type HairModel struct {
	Center Vec3
	Radii  Vec3
	LOD2   []Strip
	LOD1   []Strip
	LOD0   []Strip
	Tris2  int
	Tris1  int
	Tris0  int
}

func NewHairModel() *HairModel {
	const seed = 0x484149524c4f44 // "HAIRLOD"
	rng := rand.New(rand.NewSource(seed))
	m := &HairModel{
		Center: Vec3{X: 0, Y: 1.43, Z: 0},
		Radii:  Vec3{X: 0.47, Y: 0.57, Z: 0.46},
	}

	clumpCount := 12
	locksPerClump := 4
	strandsPerLock := 6

	for ci := 0; ci < clumpCount; ci++ {
		_, _, rootDir := m.scalpRoot(ci, clumpCount, 0.12)
		clumpColor := RGB{
			R: 0.205 + rng.Float64()*0.075,
			G: 0.105 + rng.Float64()*0.040,
			B: 0.060 + rng.Float64()*0.030,
		}
		clump := makeStrip(ci, m.Center, m.Radii, rootDir, 4, 0.158+rng.Float64()*0.022,
			0.72+rng.Float64()*0.18, rng, clumpColor, -1)
		m.LOD2 = append(m.LOD2, clump)

		for li := 0; li < locksPerClump; li++ {
			lockID := len(m.LOD1)
			lockColor := varyColor(clumpColor, rng, 0.025)
			lockAngle := float64(li)/float64(locksPerClump)*2*math.Pi + float64(ci)*0.31
			lock := makeChildStrip(lockID, clump, m.Center, 7,
				0.074+rng.Float64()*0.010, rng, lockColor, ci, 0.030, lockAngle)
			m.LOD1 = append(m.LOD1, lock)

			for si := 0; si < strandsPerLock; si++ {
				strandID := len(m.LOD0)
				strandColor := varyColor(lockColor, rng, 0.035)
				strandAngle := float64(si)/float64(strandsPerLock)*2*math.Pi + rng.Float64()*0.35
				strand := makeChildStrip(strandID, lock, m.Center, 12,
					0.0115+rng.Float64()*0.0035, rng, strandColor, lockID, 0.021, strandAngle)
				m.LOD0 = append(m.LOD0, strand)
			}
		}
	}

	m.Tris2 = countStripTriangles(m.LOD2)
	m.Tris1 = countStripTriangles(m.LOD1)
	m.Tris0 = countStripTriangles(m.LOD0)
	return m
}

func (m *HairModel) scalpRoot(index, count int, shell float64) (Vec3, Vec3, Vec3) {
	k := float64(index) + 0.5
	azimuth := float64(index)*math.Pi*(3-math.Sqrt(5)) + 0.35
	front := math.Max(0, math.Sin(azimuth))
	thetaMax := lerpFloat(1.88, 0.98, math.Pow(front, 1.7))
	cosTheta := 1 - (k/float64(count))*(1-math.Cos(thetaMax))
	theta := math.Acos(clampFloat(cosTheta, -1, 1))
	dir := Vec3{
		X: math.Sin(theta) * math.Cos(azimuth),
		Y: math.Cos(theta),
		Z: math.Sin(theta) * math.Sin(azimuth),
	}
	root := Vec3{
		X: m.Center.X + m.Radii.X*(1+shell*0.18)*dir.X,
		Y: m.Center.Y + m.Radii.Y*(1+shell*0.10)*dir.Y,
		Z: m.Center.Z + m.Radii.Z*(1+shell*0.18)*dir.Z,
	}
	normal := ellipsoidNormal(root, m.Center, m.Radii)
	root = vadd(root, vmul(normal, shell*0.035))
	return root, normal, dir
}

func ellipsoidNormal(p, center, radii Vec3) Vec3 {
	return vnorm(Vec3{
		X: (p.X - center.X) / (radii.X * radii.X),
		Y: (p.Y - center.Y) / (radii.Y * radii.Y),
		Z: (p.Z - center.Z) / (radii.Z * radii.Z),
	})
}

func tangentBasis(n Vec3) (Vec3, Vec3) {
	ref := Vec3{Y: 1}
	if math.Abs(vdot(n, ref)) > 0.92 {
		ref = Vec3{Z: 1}
	}
	a := vnorm(vcross(n, ref))
	b := vnorm(vcross(n, a))
	return a, b
}

func makeStrip(id int, center, radii, rootDir Vec3, segments int, width, length float64,
	rng *rand.Rand, color RGB, parent int) Strip {
	theta0 := math.Acos(clampFloat(rootDir.Y, -1, 1))
	phi0 := math.Atan2(rootDir.Z, rootDir.X)
	front := math.Max(0, math.Sin(phi0))
	thetaTravel := lerpFloat(1.48, 0.78, math.Pow(front, 1.45))
	thetaTravel += (rng.Float64() - 0.5) * 0.18
	if maxEnd := 2.70 - theta0; thetaTravel > maxEnd {
		thetaTravel = math.Max(0.35, maxEnd)
	}
	phiDrift := (rng.Float64() - 0.5) * 0.28
	if front > 0.45 {
		if id%2 == 0 {
			phiDrift += 0.42 + rng.Float64()*0.16
		} else {
			phiDrift -= 0.42 + rng.Float64()*0.16
		}
	}
	waveAmp := 0.045 + rng.Float64()*0.060
	waveCount := 1.6 + rng.Float64()*1.4
	phase := rng.Float64() * math.Pi * 2
	extraDrop := length * 0.18

	points := make([]Vec3, segments+1)
	widths := make([]float64, segments+1)
	for i := 0; i <= segments; i++ {
		u := float64(i) / float64(segments)
		theta := theta0 + thetaTravel*math.Pow(u, 0.92)
		phi := phi0 + phiDrift*u
		phi += waveAmp * math.Sin(u*math.Pi*2*waveCount+phase) * u
		sinTheta := math.Sin(theta)
		dir := Vec3{
			X: sinTheta * math.Cos(phi),
			Y: math.Cos(theta),
			Z: sinTheta * math.Sin(phi),
		}
		surface := Vec3{
			X: center.X + radii.X*dir.X,
			Y: center.Y + radii.Y*dir.Y,
			Z: center.Z + radii.Z*dir.Z,
		}
		normal := ellipsoidNormal(surface, center, radii)
		lift := 0.030 + 0.032*math.Sin(math.Pi*u)
		lift += 0.060 * smoothstep(0.55, 1, u)
		p := vadd(surface, vmul(normal, lift))
		p.Y -= extraDrop * smoothstep(0.62, 1, u)
		points[i] = p
		widths[i] = width * widthProfile(u)
	}
	return Strip{ID: id, Points: points, Widths: widths, Color: color, Parent: parent}
}

func makeChildStrip(id int, parent Strip, center Vec3, segments int, width float64,
	rng *rand.Rand, color RGB, parentIndex int, spread, angle float64) Strip {
	rootRadial := vnorm(vsub(parent.Points[0], center))
	axisA, axisB := tangentBasis(rootRadial)
	offsetDirection := vnorm(vadd(vmul(axisA, math.Cos(angle)), vmul(axisB, math.Sin(angle))))
	waveDirection := vnorm(vcross(rootRadial, offsetDirection))
	if vlen(waveDirection) < 1e-4 {
		waveDirection = axisB
	}
	waveAmplitude := 0.006 + rng.Float64()*0.012
	waveCount := 1.5 + rng.Float64()*1.8
	phase := rng.Float64() * math.Pi * 2
	tipShift := (rng.Float64() - 0.42) * 0.11

	points := make([]Vec3, segments+1)
	widths := make([]float64, segments+1)
	for i := 0; i <= segments; i++ {
		u := float64(i) / float64(segments)
		base, _ := sampleStrip(parent, u)
		tangent := sampleStripTangent(parent, u)
		side := vnorm(vsub(offsetDirection, vmul(tangent, vdot(offsetDirection, tangent))))
		if vlen(side) < 1e-4 {
			side = offsetDirection
		}
		envelope := 0.58 + 0.72*u
		p := vadd(base, vmul(side, spread*envelope))
		wave := math.Sin(u*math.Pi*2*waveCount+phase) * waveAmplitude * u
		p = vadd(p, vmul(waveDirection, wave))
		p.Y -= tipShift * u * u
		points[i] = p
		widths[i] = width * widthProfile(u)
	}
	return Strip{
		ID: id, Points: points, Widths: widths, Color: color, Parent: parentIndex,
	}
}

func sampleStripTangent(s Strip, u float64) Vec3 {
	const eps = 0.015
	a, _ := sampleStrip(s, math.Max(0, u-eps))
	b, _ := sampleStrip(s, math.Min(1, u+eps))
	return vnorm(vsub(b, a))
}

func widthProfile(u float64) float64 {
	if u < 0.18 {
		return 0.55 + 0.45*(u/0.18)
	}
	return 1 - 0.94*smoothstep(0.18, 1, u)
}

func varyColor(c RGB, rng *rand.Rand, amount float64) RGB {
	return RGB{
		R: clampFloat(c.R+(rng.Float64()-0.5)*amount, 0, 1),
		G: clampFloat(c.G+(rng.Float64()-0.5)*amount*0.7, 0, 1),
		B: clampFloat(c.B+(rng.Float64()-0.5)*amount*0.5, 0, 1),
	}
}

func countStripTriangles(strips []Strip) int {
	total := 0
	for _, s := range strips {
		total += 2 * (len(s.Points) - 1)
	}
	return total
}
