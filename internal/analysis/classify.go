package analysis

import (
	"math"

	"meshscope/internal/mesh"
)

type Region struct {
	ID         int
	Kind       mesh.Feature
	Faces      []int
	Confidence float64
	Area       float64
	Plane      planeFit
	Cylinder   cylinderFit
	Axis       mesh.Vec3
	Radius     float64
	Span       float64
	Signed     float64
	RadialRMS  float64
}

type FaceResult struct {
	Kind       mesh.Feature
	RegionID   int
	Confidence float64
	Deviation  float64
}

type Boundary struct {
	A, B       mesh.Vec3
	N          mesh.Vec3
	KindA      mesh.Feature
	KindB      mesh.Feature
	RegionA    int
	RegionB    int
	Confidence float64
	Length     float64
}

type Callout struct {
	Position   mesh.Vec3
	N          mesh.Vec3
	KindA      mesh.Feature
	KindB      mesh.Feature
	Confidence float64
	Label      string
}

type ErrorProbe struct {
	Position mesh.Vec3
	N        mesh.Vec3
	Value    float64
	Kind     mesh.Feature
}

type Result struct {
	Faces       []FaceResult
	Regions     []Region
	Boundaries  []Boundary
	Callouts    []Callout
	Errors      []ErrorProbe
	RMSDev      float64
	MaxAbsDev   float64
	MeanAbsDev  float64
	MeanConf    float64
	KindArea    map[mesh.Feature]float64
	KindCount   map[mesh.Feature]int
	BoundaryAvg float64
	DatumValid  bool
	DatumNormal mesh.Vec3
	DatumOffset float64
}

type localFit struct {
	fit   cylinderFit
	valid bool
}

func Classify(m *mesh.Mesh) Result {
	t := buildTopology(m)
	n := len(m.Tris)
	regionID := make([]int, n)
	for i := range regionID {
		regionID[i] = -1
	}
	locals := make([]localFit, n)
	for i := 0; i < n; i++ {
		faces := t.smoothPatch(i, 2)
		if len(faces) > 96 {
			faces = faces[:96]
		}
		if len(faces) < 3 {
			continue
		}
		f := fitCylinder(t, faces)
		locals[i] = localFit{fit: f, valid: f.valid && f.normalRMS < 10.5}
	}

	var regions []Region
	curvedSeeds := make([]int, 0, n)
	for i := 0; i < n; i++ {
		if locals[i].valid {
			curvedSeeds = append(curvedSeeds, i)
		}
	}
	for _, seed := range curvedSeeds {
		if regionID[seed] >= 0 {
			continue
		}
		faces := growCurved(t, seed, locals, regionID, len(regions))
		if len(faces) == 0 {
			continue
		}
		cf := fitCylinder(t, faces)
		if !cf.valid {
			for _, f := range faces {
				regionID[f] = -1
			}
			continue
		}
		kind := mesh.Fillet
		if cf.span >= 150 {
			kind = mesh.Cylinder
			if cf.signed < -0.08 {
				kind = mesh.Hole
			}
		}
		reg := Region{ID: len(regions), Kind: kind, Faces: faces, Cylinder: cf}
		reg.Confidence = curvedConfidence(cf)
		for _, f := range faces {
			regionID[f] = reg.ID
		}
		regions = append(regions, reg)
	}

	for pass := 0; pass < 3; pass++ {
		absorbCurvedFringe(t, regionID, regions)
	}

	for i := 0; i < n; i++ {
		if regionID[i] >= 0 {
			continue
		}
		faces := growPlane(t, i, regionID, len(regions))
		if len(faces) == 0 {
			continue
		}
		pf := fitPlane(t, faces)
		reg := Region{ID: len(regions), Kind: mesh.Plane, Faces: faces, Plane: pf}
		reg.Confidence = planeConfidence(t, pf, faces)
		for _, f := range faces {
			regionID[f] = reg.ID
		}
		regions = append(regions, reg)
	}

	absolveUnknown(t, regionID, &regions)
	regions, regionID = mergePlaneRegions(t, regionID, regions)
	finalizeRegions(t, regionID, regions)
	res := Result{
		Faces:     make([]FaceResult, n),
		Regions:   regions,
		KindArea:  map[mesh.Feature]float64{},
		KindCount: map[mesh.Feature]int{},
	}
	datum := datumPlane(regions)
	if datum != nil {
		res.DatumValid = true
		res.DatumNormal = datum.Plane.normal
		res.DatumOffset = datum.Plane.offset
	}
	var confSum, devSq, devSum, areaSum, maxDev float64
	for i, f := range m.Tris {
		r := regions[regionID[i]]
		fr := FaceResult{Kind: r.Kind, RegionID: r.ID, Confidence: r.Confidence}
		if r.Kind == mesh.Plane {
			fr.Deviation = r.Plane.normal.Dot(f.Center) - r.Plane.offset
		} else {
			d := f.Center.Sub(r.Cylinder.center)
			d = d.Sub(r.Cylinder.axis.Mul(d.Dot(r.Cylinder.axis)))
			fr.Deviation = d.Len() - r.Cylinder.radius
		}
		if datum != nil && r.Kind == mesh.Plane &&
			angleDeg(f.Normal, datum.Plane.normal) < 25 &&
			math.Abs(datum.Plane.normal.Dot(f.Center)-datum.Plane.offset) < 1.8 {
			fr.Deviation = datum.Plane.normal.Dot(f.Center) - datum.Plane.offset
		}
		res.Faces[i] = fr
		res.KindArea[r.Kind] += f.Area
		if r.Kind != mesh.Unknown {
			res.KindCount[r.Kind]++
		}
		confSum += f.Area * fr.Confidence
		areaSum += f.Area
		devSq += f.Area * fr.Deviation * fr.Deviation
		devSum += f.Area * math.Abs(fr.Deviation)
		maxDev = math.Max(maxDev, math.Abs(fr.Deviation))
	}
	if areaSum > 0 {
		res.MeanConf = confSum / areaSum
		res.RMSDev = math.Sqrt(devSq / areaSum)
		res.MeanAbsDev = devSum / areaSum
		res.MaxAbsDev = maxDev
	}
	res.Boundaries = findBoundaries(t, m, regionID, regions, res.Faces)
	res.Callouts = makeCallouts(res.Boundaries)
	res.Errors = errorProbes(m, res.Faces)
	computeBoundaryAverage(&res)
	return res
}

func datumPlane(regions []Region) *Region {
	var best *Region
	for i := range regions {
		r := &regions[i]
		if r.Kind != mesh.Plane {
			continue
		}
		if best == nil || r.Area > best.Area {
			best = r
		}
	}
	return best
}

func growCurved(t topology, seed int, locals []localFit, ids []int, id int) []int {
	seen := map[int]bool{seed: true}
	var faces []int
	queue := []int{seed}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		ids[cur] = id
		faces = append(faces, cur)
		base := locals[cur].fit
		for _, nb := range t.adj[cur] {
			n := nb.face
			if seen[n] || ids[n] >= 0 || !locals[n].valid || nb.angle > 42 {
				continue
			}
			other := locals[n].fit
			if angleDeg(base.axis, other.axis) > 19 {
				continue
			}
			if math.Abs(base.radius-other.radius) > math.Max(1.8, 0.32*base.radius) {
				continue
			}
			d := other.center.Sub(base.center)
			d = d.Sub(base.axis.Mul(d.Dot(base.axis)))
			if d.Len() > 2.2 {
				continue
			}
			seen[n] = true
			queue = append(queue, n)
		}
	}
	return faces
}

func growPlane(t topology, seed int, ids []int, id int) []int {
	if ids[seed] >= 0 {
		return nil
	}
	n0 := t.normal[seed]
	p0 := t.center[seed]
	seen := map[int]bool{seed: true}
	queue := []int{seed}
	var faces []int
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		ids[cur] = id
		faces = append(faces, cur)
		for _, nb := range t.adj[cur] {
			n := nb.face
			if seen[n] || ids[n] >= 0 {
				continue
			}
			angle := angleDeg(t.normal[n], n0)
			if angle > 2.0 {
				continue
			}
			if math.Abs(t.normal[n].Dot(t.center[n].Sub(p0))) > 0.32 {
				continue
			}
			seen[n] = true
			queue = append(queue, n)
		}
	}
	return faces
}

func planeConfidence(t topology, pf planeFit, faces []int) float64 {
	var spread, total float64
	for _, f := range faces {
		w := t.area[f]
		spread += w * math.Pow(angleDeg(t.normal[f], pf.normal)/2.4, 2)
		total += w
	}
	if total <= 0 {
		return 0.2
	}
	spread = math.Sqrt(spread / total)
	dev := math.Pow(pf.rms/0.16, 2)
	support := math.Min(1, float64(len(faces))/9)
	conf := math.Exp(-spread-dev) * (0.72 + 0.28*support)
	return clamp01(conf)
}

func curvedConfidence(cf cylinderFit) float64 {
	nr := cf.normalRMS / 12
	rr := cf.radialRMS / math.Max(0.12, cf.radius*0.04)
	spanMargin := math.Min(1, cf.span/70)
	conf := math.Exp(-(nr*nr)-(rr*rr)) * (0.70 + 0.30*spanMargin)
	return clamp01(conf)
}

func clamp01(v float64) float64 {
	return math.Max(0.15, math.Min(0.995, v))
}
