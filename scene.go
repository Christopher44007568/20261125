package main

import "math"

var lightDirection = vnorm(Vec3{X: -0.42, Y: 0.76, Z: 0.66})

const (
	HairCapStacks = 12
	HairCapSlices = 28
	// Two triangles per stack/slice quad.
	HairCapTriangles = HairCapStacks * HairCapSlices * 2
)

func BuildScene(cam Camera, model *HairModel, lod LODFrame) []Tri {
	tris := make([]Tri, 0, lod.HairTris+2400)

	shirt := RGB{R: 0.16, G: 0.29, B: 0.34}
	skin := RGB{R: 0.78, G: 0.55, B: 0.42}
	skinShadow := RGB{R: 0.68, G: 0.43, B: 0.33}
	eye := RGB{R: 0.055, G: 0.045, B: 0.040}

	tris = AppendEllipsoid(tris, Vec3{X: 0, Y: 0.70, Z: 0},
		Vec3{X: 0.86, Y: 0.34, Z: 0.50}, 20, 28, shirt, MatCloth, cam)
	tris = AppendEllipsoid(tris, Vec3{X: 0, Y: 1.02, Z: 0.02},
		Vec3{X: 0.17, Y: 0.23, Z: 0.17}, 12, 18, skinShadow, MatSkin, cam)
	tris = AppendEllipsoid(tris, model.Center, model.Radii,
		18, 28, skin, MatSkin, cam)

	tris = AppendEllipsoid(tris, Vec3{X: -0.155, Y: 1.485, Z: 0.435},
		Vec3{X: 0.047, Y: 0.058, Z: 0.026}, 8, 12, eye, MatEye, cam)
	tris = AppendEllipsoid(tris, Vec3{X: 0.155, Y: 1.485, Z: 0.435},
		Vec3{X: 0.047, Y: 0.058, Z: 0.026}, 8, 12, eye, MatEye, cam)
	tris = AppendEllipsoid(tris, Vec3{X: 0, Y: 1.405, Z: 0.455},
		Vec3{X: 0.040, Y: 0.060, Z: 0.050}, 8, 12, skinShadow, MatSkin, cam)

	hairBase := RGB{R: 0.155, G: 0.079, B: 0.047}
	tris = AppendHairCap(tris, model, hairBase, cam)
	tris = AppendHairRibbons(tris, lod.Strips, model.Center, cam)
	return tris
}

func AppendHairCap(out []Tri, model *HairModel, base RGB, cam Camera) []Tri {
	grid := make([][]Vertex, HairCapStacks+1)
	for i := 0; i <= HairCapStacks; i++ {
		grid[i] = make([]Vertex, HairCapSlices+1)
		for j := 0; j <= HairCapSlices; j++ {
			phi := 2 * math.Pi * float64(j) / float64(HairCapSlices)
			thetaMax := hairlineThetaMax(phi)
			theta := thetaMax * float64(i) / float64(HairCapStacks)
			dir := Vec3{
				X: math.Sin(theta) * math.Cos(phi),
				Y: math.Cos(theta),
				Z: math.Sin(theta) * math.Sin(phi),
			}
			shell := 1.035
			pos := Vec3{
				X: model.Center.X + model.Radii.X*shell*dir.X,
				Y: model.Center.Y + model.Radii.Y*shell*dir.Y,
				Z: model.Center.Z + model.Radii.Z*shell*dir.Z,
			}
			n := ellipsoidNormal(pos, model.Center, model.Radii)
			grid[i][j] = Vertex{Pos: pos, Color: shadeSurface(base, n, pos, cam)}
		}
	}

	for i := 0; i < HairCapStacks; i++ {
		for j := 0; j < HairCapSlices; j++ {
			a := grid[i][j]
			b := grid[i+1][j]
			c := grid[i+1][j+1]
			d := grid[i][j+1]
			out = append(out,
				Tri{V: [3]Vertex{a, b, d}, Mat: MatHair},
				Tri{V: [3]Vertex{b, c, d}, Mat: MatHair},
			)
		}
	}
	return out
}

func hairlineThetaMax(phi float64) float64 {
	front := math.Max(0, math.Sin(phi))
	return lerpFloat(2.03, 1.04, math.Pow(front, 1.55))
}

func AppendEllipsoid(out []Tri, center, radii Vec3, stacks, slices int,
	base RGB, mat uint8, cam Camera) []Tri {
	type gridVertex struct {
		pos   Vec3
		color RGB
	}
	grid := make([][]gridVertex, stacks+1)
	for i := 0; i <= stacks; i++ {
		theta := math.Pi * float64(i) / float64(stacks)
		sinTheta := math.Sin(theta)
		cosTheta := math.Cos(theta)
		grid[i] = make([]gridVertex, slices+1)
		for j := 0; j <= slices; j++ {
			phi := 2 * math.Pi * float64(j) / float64(slices)
			dir := Vec3{
				X: sinTheta * math.Cos(phi),
				Y: cosTheta,
				Z: sinTheta * math.Sin(phi),
			}
			pos := Vec3{
				X: center.X + radii.X*dir.X,
				Y: center.Y + radii.Y*dir.Y,
				Z: center.Z + radii.Z*dir.Z,
			}
			n := ellipsoidNormal(pos, center, radii)
			grid[i][j] = gridVertex{pos: pos, color: shadeSurface(base, n, pos, cam)}
		}
	}

	for i := 0; i < stacks; i++ {
		for j := 0; j < slices; j++ {
			a := grid[i][j]
			b := grid[i+1][j]
			c := grid[i+1][j+1]
			d := grid[i][j+1]
			out = append(out,
				Tri{V: [3]Vertex{
					{Pos: a.pos, Color: a.color},
					{Pos: b.pos, Color: b.color},
					{Pos: d.pos, Color: d.color},
				}, Mat: mat},
				Tri{V: [3]Vertex{
					{Pos: b.pos, Color: b.color},
					{Pos: c.pos, Color: c.color},
					{Pos: d.pos, Color: d.color},
				}, Mat: mat},
			)
		}
	}
	return out
}

func shadeSurface(base RGB, normal, pos Vec3, cam Camera) RGB {
	n := vnorm(normal)
	v := vnorm(vsub(cam.Pos, pos))
	ndl := math.Max(0, vdot(n, lightDirection))
	intensity := 0.43 + 0.57*ndl
	color := base.mul(intensity)

	h := vnorm(vadd(lightDirection, v))
	spec := 0.16 * math.Pow(math.Max(0, vdot(n, h)), 30)
	rim := 0.10 * math.Pow(1-math.Max(0, vdot(n, v)), 2.5)
	color.R += spec + rim*0.55
	color.G += spec*0.92 + rim*0.72
	color.B += spec*0.72 + rim
	return color
}

func AppendHairRibbons(out []Tri, strips []Strip, headCenter Vec3, cam Camera) []Tri {
	for _, strip := range strips {
		count := len(strip.Points)
		if count < 2 {
			continue
		}
		left := make([]Vertex, count)
		right := make([]Vertex, count)

		for i, p := range strip.Points {
			prev := strip.Points[maxInt(0, i-1)]
			next := strip.Points[minInt(count-1, i+1)]
			tangent := vnorm(vsub(next, prev))
			toCamera := vnorm(vsub(cam.Pos, p))
			side := vnorm(vcross(tangent, toCamera))
			if vlen(side) < 1e-4 {
				side = vnorm(vcross(tangent, cam.Up))
			}
			radial := vnorm(vsub(p, headCenter))
			normal := vnorm(vadd(vmul(toCamera, 0.78), vmul(radial, 0.22)))
			color := shadeHair(strip.Color, normal, tangent, toCamera, lightDirection)
			halfWidth := strip.Widths[i] * 0.5
			leftPos := vsub(p, vmul(side, halfWidth))
			rightPos := vadd(p, vmul(side, halfWidth))
			left[i] = Vertex{Pos: leftPos, Color: color}
			right[i] = Vertex{Pos: rightPos, Color: color}
		}

		for i := 0; i < count-1; i++ {
			out = append(out,
				Tri{V: [3]Vertex{left[i], right[i], left[i+1]}, Mat: MatHair},
				Tri{V: [3]Vertex{right[i], right[i+1], left[i+1]}, Mat: MatHair},
			)
		}
	}
	return out
}

func shadeHair(base RGB, normal, tangent, view, light Vec3) RGB {
	n := vnorm(normal)
	h := vnorm(vadd(light, view))
	diffuse := 0.40 + 0.48*math.Abs(vdot(n, light))
	color := base.mul(diffuse)

	nh := math.Abs(vdot(n, h))
	spec := 0.18 * math.Pow(nh, 26)
	th := clampFloat(math.Abs(vdot(tangent, h)), 0, 1)
	aniso := 0.075 * math.Pow(1-th*th, 5)

	color.R += spec + aniso
	color.G += spec*0.82 + aniso*0.72
	color.B += spec*0.58 + aniso*0.50
	return color
}
