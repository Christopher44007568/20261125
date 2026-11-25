package hair

import (
	"math"
	"math/rand"
)

const (
	scalpRadiusX = 1.02
	scalpRadiusY = 1.14
	scalpRadiusZ = 0.90
	scalpCenterY = 0.18
)

func Generate(count int, seed int64, pointsPerStrand int) *Asset {
	if count < 1 {
		count = 1
	}
	if pointsPerStrand < 4 {
		pointsPerStrand = 4
	}

	rng := rand.New(rand.NewSource(seed))
	strands := make([]Strand, 0, count)
	for len(strands) < count {
		root, outward, ok := sampleRoot(rng)
		if !ok {
			continue
		}
		points := makeStrand(rng, root, outward, pointsPerStrand)
		strands = append(strands, Strand{ID: len(strands), Points: points})
	}

	entries := make([]Entry, len(strands))
	for i, s := range strands {
		entries[i] = Entry{
			ID:                  i,
			Points:              s.Points,
			Weight:              1,
			RepresentativeID:    s.ID,
			RepresentativeLevel: 0,
			SourceIDs:           []int{s.ID},
		}
	}

	return &Asset{
		Seed:    seed,
		Strands: strands,
		Bounds:  boundsOf(strands),
		Levels: []Level{{
			Index:     0,
			Label:     "full",
			Entries:   entries,
			TotalMass: float64(len(entries)),
		}},
		Camera: DefaultCamera(),
	}
}

func sampleRoot(rng *rand.Rand) (Vec3, Vec3, bool) {
	// A spherical cap with a carved-out face region produces a readable
	// hairline and enough side/back mass for silhouette tests.
	theta := math.Acos(1 - 0.97*rng.Float64())
	phi := 2 * math.Pi * rng.Float64()
	sx := math.Sin(theta)

	root := Vec3{
		X: scalpRadiusX * sx * math.Cos(phi),
		Y: scalpCenterY + scalpRadiusY*math.Cos(theta),
		Z: scalpRadiusZ * sx * math.Sin(phi),
	}
	if root.Z > 0.30 && root.Y < 0.5 && math.Abs(root.X) < 0.72 {
		return Vec3{}, Vec3{}, false
	}
	outward := Vec3{
		X: root.X / (scalpRadiusX * scalpRadiusX),
		Y: (root.Y - scalpCenterY) / (scalpRadiusY * scalpRadiusY),
		Z: root.Z / (scalpRadiusZ * scalpRadiusZ),
	}.Normalize()
	return root, outward, true
}

func makeStrand(rng *rand.Rand, root, outward Vec3, pointCount int) []Vec3 {
	length := 2.15 + 1.50*rng.Float64()
	if root.Y > 0.85 {
		length *= 1.12 + 0.12*rng.Float64()
	}
	phase := 2 * math.Pi * rng.Float64()
	frequency := 4.2 + 1.8*rng.Float64()
	side := Vec3{X: outward.Z, Y: 0, Z: -outward.X}.Normalize()
	if side.Length() < 0.5 {
		side = Vec3{X: 1}
	}
	waveAmount := 0.055 + 0.075*rng.Float64()
	drape := 0.10 + 0.13*rng.Float64()
	points := make([]Vec3, pointCount)

	for i := range points {
		t := float64(i) / float64(pointCount-1)
		fall := length * (0.18*t + 0.82*t*t)
		swell := drape * (1 - math.Exp(-3.2*t))
		p := root.Add(Vec3{Y: -fall}).Add(outward.Scale(swell))

		wave := math.Sin(frequency*t+phase) * waveAmount * (0.2 + 0.8*t)
		lateralWave := math.Cos(frequency*0.73*t+phase*1.31) * waveAmount * 0.55 * t
		p = p.Add(side.Scale(wave)).Add(Vec3{X: lateralWave, Z: lateralWave * 0.6})

		curl := math.Sin(phase+math.Pi*1.7*t) * 0.035 * t * t
		p = p.Add(Vec3{
			X: side.X * curl,
			Y: curl * 0.35,
			Z: side.Z * curl,
		})
		points[i] = p
	}
	return points
}
