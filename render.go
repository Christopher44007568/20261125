package main

import (
	"fmt"
	"math"
	"os"
)

type RGB struct {
	R, G, B uint8
}

type Image struct {
	Width  int
	Height int
	Pixels []RGB
}

func newImage(width, height int) *Image {
	pixels := make([]RGB, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			t := float64(y) / float64(height-1)
			value := uint8(18 + 18*t)
			pixels[y*width+x] = RGB{R: value, G: value + 3, B: value + 9}
		}
	}
	return &Image{Width: width, Height: height, Pixels: pixels}
}

func clampByte(value float64) uint8 {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(math.Round(value))
}

func tierColor(tier uint8, composite bool) RGB {
	if !composite {
		return RGB{R: 76, G: 142, B: 196}
	}
	switch tier {
	case tierNear:
		return RGB{R: 72, G: 168, B: 118}
	case tierMid:
		return RGB{R: 218, G: 170, B: 64}
	case tierFar:
		return RGB{R: 185, G: 94, B: 82}
	default:
		return RGB{R: 132, G: 108, B: 190}
	}
}

func triangleNormal(mesh *Mesh, tri Tri) Vec3 {
	a := mesh.Vertices[tri.Vertices[0]]
	b := mesh.Vertices[tri.Vertices[1]]
	c := mesh.Vertices[tri.Vertices[2]]
	return normalize(cross(sub(b, a), sub(c, a)))
}

func renderMesh(surface *Surface, mesh *Mesh, camera Camera, composite bool) *Image {
	image := newImage(camera.Width, camera.Height)
	depth := make([]float64, camera.Width*camera.Height)
	for i := range depth {
		depth[i] = math.Inf(1)
	}

	for _, tri := range mesh.Tris {
		p0 := camera.Project(mesh.Vertices[tri.Vertices[0]])
		p1 := camera.Project(mesh.Vertices[tri.Vertices[1]])
		p2 := camera.Project(mesh.Vertices[tri.Vertices[2]])
		if p0.Z <= 0 || p1.Z <= 0 || p2.Z <= 0 {
			continue
		}

		minX := max(0, int(math.Floor(math.Min(p0.X, math.Min(p1.X, p2.X)))))
		maxX := min(camera.Width-1, int(math.Ceil(math.Max(p0.X, math.Max(p1.X, p2.X)))))
		minY := max(0, int(math.Floor(math.Min(p0.Y, math.Min(p1.Y, p2.Y)))))
		maxY := min(camera.Height-1, int(math.Ceil(math.Max(p0.Y, math.Max(p1.Y, p2.Y)))))

		denom := (p1.Y-p2.Y)*(p0.X-p2.X) + (p2.X-p1.X)*(p0.Y-p2.Y)
		if math.Abs(denom) < 1e-12 {
			continue
		}

		normal := triangleNormal(mesh, tri)
		if dot(normal, normalize(sub(camera.Position, mesh.Vertices[tri.Vertices[0]]))) < 0 {
			normal = scale(normal, -1)
		}
		light := normalize(Vec3{X: -0.45, Y: -0.35, Z: 1})
		shade := 0.48 + 0.52*math.Max(0, dot(normal, light))
		base := tierColor(tri.Tier, composite)
		color := RGB{
			R: clampByte(float64(base.R) * shade),
			G: clampByte(float64(base.G) * shade),
			B: clampByte(float64(base.B) * shade),
		}

		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				px := float64(x) + 0.5
				py := float64(y) + 0.5
				w0 := ((p1.Y-p2.Y)*(px-p2.X) + (p2.X-p1.X)*(py-p2.Y)) / denom
				w1 := ((p2.Y-p0.Y)*(px-p2.X) + (p0.X-p2.X)*(py-p2.Y)) / denom
				w2 := 1 - w0 - w1
				if w0 < -1e-5 || w1 < -1e-5 || w2 < -1e-5 {
					continue
				}
				w0 = math.Max(0, w0)
				w1 = math.Max(0, w1)
				w2 = math.Max(0, w2)
				weightSum := w0 + w1 + w2
				w0 /= weightSum
				w1 /= weightSum
				w2 /= weightSum
				z := w0*p0.Z + w1*p1.Z + w2*p2.Z
				index := y*camera.Width + x
				if z < depth[index] {
					depth[index] = z
					image.Pixels[index] = color
				}
			}
		}
	}

	if composite {
		renderWireframe(mesh, camera, image, depth, composite)
	}
	return image
}

func renderWireframe(mesh *Mesh, camera Camera, image *Image, depth []float64, composite bool) {
	edges := make(map[[2]int32]bool)
	for _, tri := range mesh.Tris {
		for i := 0; i < 3; i++ {
			a := tri.Vertices[i]
			b := tri.Vertices[(i+1)%3]
			edges[edgeKey(a, b)] = tri.Boundary[i]
		}
	}

	for key, boundary := range edges {
		if composite && !boundary {
			continue
		}
		a := camera.Project(mesh.Vertices[key[0]])
		b := camera.Project(mesh.Vertices[key[1]])
		color := RGB{R: 58, G: 70, B: 86}
		if composite && boundary {
			color = RGB{R: 255, G: 58, B: 34}
		}
		drawLine(image, depth, a, b, color, boundary)
	}
}

func drawLine(image *Image, depth []float64, a, b Projected, color RGB, thick bool) {
	steps := int(math.Ceil(math.Max(math.Abs(a.X-b.X), math.Abs(a.Y-b.Y)))) + 1
	radius := 0
	if thick {
		radius = 1
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		p := Projected{
			X: a.X + (b.X-a.X)*t,
			Y: a.Y + (b.Y-a.Y)*t,
			Z: a.Z + (b.Z-a.Z)*t,
		}
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				x := int(math.Round(p.X)) + dx
				y := int(math.Round(p.Y)) + dy
				if x < 0 || x >= image.Width || y < 0 || y >= image.Height {
					continue
				}
				index := y*image.Width + x
				if p.Z <= depth[index]+1e-7 {
					image.Pixels[index] = color
				}
			}
		}
	}
}

func writePPM(path string, image *Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, "P6\n%d %d\n255\n", image.Width, image.Height); err != nil {
		return err
	}
	buffer := make([]byte, 0, image.Width*3)
	for y := 0; y < image.Height; y++ {
		buffer = buffer[:0]
		for x := 0; x < image.Width; x++ {
			pixel := image.Pixels[y*image.Width+x]
			buffer = append(buffer, pixel.R, pixel.G, pixel.B)
		}
		if _, err := file.Write(buffer); err != nil {
			return err
		}
	}
	return nil
}
