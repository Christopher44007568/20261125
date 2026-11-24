package normalcone

import "math"

type Scenario struct {
	Name          string
	Description   string
	Normals       []Vec3
	Reference     Cone
	WrongAxis     Vec3
	SurfacePoint  Vec3
	CameraSamples []Vec3
	BoundaryViews []Vec3
	Degenerate    bool
}

func BuildScenarios(samples int) []Scenario {
	planar := planarScenario(samples)
	curved := curvedScenario(samples)
	mirrored := mirroredScenario(samples, curved)
	degenerate := degenerateScenario(samples)
	return []Scenario{planar, curved, mirrored, degenerate}
}

func planarScenario(samples int) Scenario {
	normal := Vec3{X: 0, Y: 0, Z: 1}
	normals := []Vec3{normal, normal, normal, normal}
	origin := Vec3{}
	cameras := fibonacciCameras(origin, samples)
	return Scenario{
		Name:          "planar",
		Description:   "flat patch whose four vertex normals are identical",
		Normals:       normals,
		Reference:     ReferenceCone(normals),
		WrongAxis:     normal,
		SurfacePoint:  origin,
		CameraSamples: cameras,
		BoundaryViews: boundaryViews(normal, deg90, origin),
	}
}

func curvedScenario(samples int) Scenario {
	const steps = 7
	const span = math.Pi / 3
	normals := make([]Vec3, steps)
	for index := range normals {
		angle := -span/2 + span*float64(index)/float64(steps-1)
		normals[index] = Vec3{X: math.Sin(angle), Y: 0, Z: math.Cos(angle)}
	}

	reference := ReferenceCone(normals)
	origin := Vec3{}
	return Scenario{
		Name:          "curved_patch",
		Description:   "symmetric curved patch spanning a 60 degree normal fan",
		Normals:       normals,
		Reference:     reference,
		WrongAxis:     normals[0],
		SurfacePoint:  origin,
		CameraSamples: fibonacciCameras(origin, samples),
		BoundaryViews: boundaryViews(reference.Axis, deg90+reference.HalfAngle, origin),
	}
}

func mirroredScenario(samples int, source Scenario) Scenario {
	reflection := Reflection(Vec3{X: 1, Y: 0, Z: 0})
	normals := make([]Vec3, len(source.Normals))
	for index, normal := range source.Normals {
		transformed, ok := reflection.TransformNormal(normal)
		if !ok {
			panic("mirror transform produced a degenerate normal")
		}
		normals[index] = transformed
	}

	reference := ReferenceCone(normals)
	naive, _ := reflection.MulVec(source.Reference.Axis).Normalize()
	origin := Vec3{}
	return Scenario{
		Name:          "mirrored_patch",
		Description:   "curved patch after an orientation-reversing mirror transform",
		Normals:       normals,
		Reference:     reference,
		WrongAxis:     naive,
		SurfacePoint:  origin,
		CameraSamples: fibonacciCameras(origin, samples),
		BoundaryViews: boundaryViews(reference.Axis, deg90+reference.HalfAngle, origin),
	}
}

func degenerateScenario(samples int) Scenario {
	normals := []Vec3{
		{X: 0, Y: 0, Z: 1},
		{},
		{X: 0, Y: 0, Z: -1},
	}
	origin := Vec3{}
	return Scenario{
		Name:          "degenerate_normal",
		Description:   "patch containing a zero normal from a collapsed triangle",
		Normals:       normals,
		Reference:     ReferenceCone(normals),
		WrongAxis:     Vec3{X: 0, Y: 0, Z: 1},
		SurfacePoint:  origin,
		CameraSamples: fibonacciCameras(origin, samples),
		BoundaryViews: boundaryViews(Vec3{X: 0, Y: 0, Z: 1}, deg90, origin),
		Degenerate:    true,
	}
}

func fibonacciCameras(origin Vec3, count int) []Vec3 {
	if count <= 0 {
		count = 1
	}
	cameras := make([]Vec3, count)
	goldenAngle := math.Pi * (3 - math.Sqrt(5))
	for index := range cameras {
		y := 1 - 2*(float64(index)+0.5)/float64(count)
		radius := math.Sqrt(math.Max(0, 1-y*y))
		angle := goldenAngle * float64(index)
		direction := Vec3{
			X: math.Cos(angle) * radius,
			Y: y,
			Z: math.Sin(angle) * radius,
		}
		cameras[index] = origin.Add(direction)
	}
	return cameras
}

func boundaryViews(axis Vec3, angle float64, origin Vec3) []Vec3 {
	basisA := orthonormalBasis(axis)
	direction := axis.Scale(math.Cos(angle)).Add(basisA.Scale(math.Sin(angle)))
	negative := axis.Scale(math.Cos(angle)).Sub(basisA.Scale(math.Sin(angle)))
	return []Vec3{origin.Add(direction), origin.Add(negative)}
}

func orthonormalBasis(axis Vec3) Vec3 {
	reference := Vec3{X: 0, Y: 1, Z: 0}
	if math.Abs(axis.Dot(reference)) > 0.9 {
		reference = Vec3{X: 1, Y: 0, Z: 0}
	}
	basisA, _ := axis.Cross(reference).Normalize()
	return basisA
}
