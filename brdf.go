package main

import "math"

const sampleCount = 4096

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func dot(a, b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func normalize(v Vec3) Vec3 {
	length := math.Sqrt(dot(v, v))
	return Vec3{v.X / length, v.Y / length, v.Z / length}
}

func reflect(i, n Vec3) Vec3 {
	d := 2 * dot(i, n)
	return Vec3{i.X - d*n.X, i.Y - d*n.Y, i.Z - d*n.Z}
}

func clamp01(x float64) float64 {
	return math.Max(0, math.Min(1, x))
}

func schlick(cosTheta, f0 float64) float64 {
	cosTheta = clamp01(cosTheta)
	return f0 + (1-f0)*math.Pow(1-cosTheta, 5)
}

func ggxAlpha(roughness float64) float64 {
	return math.Max(roughness*roughness, 1e-4)
}

func ggxDistribution(normalDotHalf, alpha float64) float64 {
	denominator := normalDotHalf*normalDotHalf*(alpha*alpha-1) + 1
	return alpha * alpha / (math.Pi * denominator * denominator)
}

func ggxGeometry1(cosTheta, alpha float64) float64 {
	cosTheta = clamp01(cosTheta)
	if cosTheta <= 0 {
		return 0
	}
	tanSquared := (1 - cosTheta*cosTheta) / (cosTheta * cosTheta)
	lambda := (math.Sqrt(1+alpha*alpha*tanSquared) - 1) / 2
	return 1 / (1 + lambda)
}

func ggxGeometry(viewCos, lightCos, alpha float64) float64 {
	return ggxGeometry1(viewCos, alpha) * ggxGeometry1(lightCos, alpha)
}

func ggxBRDF(view, light, normal Vec3, f0, alpha float64) float64 {
	viewCos := clamp01(dot(view, normal))
	lightCos := clamp01(dot(light, normal))
	if viewCos <= 0 || lightCos <= 0 {
		return 0
	}
	half := normalize(Vec3{
		X: view.X + light.X,
		Y: view.Y + light.Y,
		Z: view.Z + light.Z,
	})
	halfCos := clamp01(dot(half, normal))
	if halfCos <= 0 {
		return 0
	}
	d := ggxDistribution(halfCos, alpha)
	g := ggxGeometry(viewCos, lightCos, alpha)
	f := schlick(clamp01(dot(light, half)), f0)
	return d * g * f / (4 * viewCos * lightCos)
}

func ggxSample(view Vec3, alpha float64, index int) (Vec3, float64) {
	grid := int(math.Sqrt(float64(sampleCount)))
	u := (float64(index/grid) + 0.5) / float64(grid)
	v := (float64(index%grid) + 0.5) / float64(grid)

	phi := 2 * math.Pi * v
	tanSquared := alpha * alpha * u / (1 - u)
	normalCos := 1 / math.Sqrt(1+tanSquared)
	planeRadius := math.Sqrt(math.Max(0, 1-normalCos*normalCos))
	half := normalize(Vec3{
		X: planeRadius * math.Cos(phi),
		Y: planeRadius * math.Sin(phi),
		Z: normalCos,
	})

	light := reflect(Vec3{-view.X, -view.Y, -view.Z}, half)
	if light.Z <= 0 {
		return light, 0
	}
	halfCos := clamp01(half.Z)
	pdf := ggxDistribution(halfCos, alpha) * halfCos /
		(4 * clamp01(dot(view, half)))
	return light, pdf
}

func cosineHemisphereSample(index int) Vec3 {
	grid := int(math.Sqrt(float64(sampleCount)))
	u := (float64(index/grid) + 0.5) / float64(grid)
	v := (float64(index%grid) + 0.5) / float64(grid)
	phi := 2 * math.Pi * v
	normalCos := math.Sqrt(1 - u)
	planeRadius := math.Sqrt(u)
	return Vec3{
		X: planeRadius * math.Cos(phi),
		Y: planeRadius * math.Sin(phi),
		Z: normalCos,
	}
}
