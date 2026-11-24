package xrpath

import (
	"image"
	"math"
	"sort"
)

type hitAccumulator struct {
	object WorldObject
	depth  float64
	pixels int
	minX   int
	minY   int
	maxX   int
	maxY   int
}

func RenderStereo(pose Pose, params Parameters, world World, missingResources map[string]bool) *StereoImage {
	return &StereoImage{
		Left:  renderEye("left", -1, pose, params, world, missingResources),
		Right: renderEye("right", 1, pose, params, world, missingResources),
	}
}

func renderEye(eye string, eyeSign float64, head Pose, params Parameters, world World, missing map[string]bool) *RenderedEye {
	w, h := params.Width, params.Height
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	depth := make([]float64, w*h)
	ids := make([]string, w*h)
	accums := make(map[string]*hitAccumulator)

	for i := range depth {
		depth[i] = math.Inf(1)
	}

	forward := head.Orientation.Rotate(V(0, 0, -1)).Normalize()
	right := head.Orientation.Rotate(V(1, 0, 0)).Normalize()
	up := head.Orientation.Rotate(V(0, 1, 0)).Normalize()
	origin := head.Position.Add(right.Scale(eyeSign * params.IPD / 2))
	tanHalfFov := math.Tan(params.FOVDeg * math.Pi / 360)
	aspect := float64(w) / float64(h)
	lightDir := V(0.38, 0.86, -0.34).Normalize()

	for y := 0; y < h; y++ {
		ndcY := 1 - 2*(float64(y)+0.5)/float64(h)
		for x := 0; x < w; x++ {
			ndcX := 2*(float64(x)+0.5)/float64(w) - 1
			dir := forward.
				Add(right.Scale(ndcX * tanHalfFov * aspect)).
				Add(up.Scale(ndcY * tanHalfFov)).
				Normalize()

			bestT := math.Inf(1)
			bestObj := -1
			bestNormal := V(0, 1, 0)
			for oi, obj := range world.Objects {
				t, normal, ok := rayAABB(origin, dir, obj.Min, obj.Max)
				if !ok || t < params.Near || t > params.Far || t >= bestT {
					continue
				}
				bestT = t
				bestObj = oi
				bestNormal = normal
			}

			idx := y*w + x
			rgbaIdx := idx * 4
			if bestObj < 0 {
				img.Pix[rgbaIdx+0] = 18
				img.Pix[rgbaIdx+1] = 22
				img.Pix[rgbaIdx+2] = 32
				img.Pix[rgbaIdx+3] = 255
				continue
			}

			obj := world.Objects[bestObj]
			hit := origin.Add(dir.Scale(bestT))
			col := shadeObject(obj, hit, bestNormal, lightDir, params.Exposure, missing)
			img.Pix[rgbaIdx+0] = col[0]
			img.Pix[rgbaIdx+1] = col[1]
			img.Pix[rgbaIdx+2] = col[2]
			img.Pix[rgbaIdx+3] = 255
			depth[idx] = bestT
			ids[idx] = obj.ID

			acc := accums[obj.ID]
			if acc == nil {
				acc = &hitAccumulator{
					object: obj, depth: bestT,
					minX: x, minY: y, maxX: x + 1, maxY: y + 1,
				}
				accums[obj.ID] = acc
			}
			acc.pixels++
			if bestT < acc.depth {
				acc.depth = bestT
			}
			if x < acc.minX {
				acc.minX = x
			}
			if y < acc.minY {
				acc.minY = y
			}
			if x+1 > acc.maxX {
				acc.maxX = x + 1
			}
			if y+1 > acc.maxY {
				acc.maxY = y + 1
			}
		}
	}

	hits := make([]VisibleObject, 0, len(accums))
	for _, acc := range accums {
		center := acc.object.Center()
		hits = append(hits, VisibleObject{
			ID: acc.object.ID, ResourceID: acc.object.MeshResource,
			Center: center, HalfExtents: acc.object.HalfExtents(),
			Depth: acc.depth, Pixels: acc.pixels,
			Screen: ScreenRect{
				MinX: acc.minX, MinY: acc.minY, MaxX: acc.maxX, MaxY: acc.maxY,
				CenterX: float64(acc.minX+acc.maxX) / 2,
				CenterY: float64(acc.minY+acc.maxY) / 2,
			},
		})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].ID < hits[j].ID })

	return &RenderedEye{Eye: eye, Image: img, Depth: depth, ObjectIDs: ids, Hits: hits}
}

func shadeObject(obj WorldObject, hit, normal, light Vec3, exposure float64, missing map[string]bool) [3]byte {
	if missing[obj.MeshResource] || missing[obj.TextureResource] {
		stripe := (int(math.Floor(hit.X*18)) + int(math.Floor(hit.Y*18)) + int(math.Floor(hit.Z*18))) & 3
		if stripe < 2 {
			return [3]byte{255, 0, 255}
		}
		return [3]byte{20, 20, 20}
	}

	diffuse := math.Max(0, normal.Dot(light))
	shade := 0.38 + 0.62*diffuse
	check := 1.0
	if obj.Checkered {
		cx := int(math.Floor(hit.X * 1.15))
		cy := int(math.Floor(hit.Y * 2.1))
		cz := int(math.Floor(hit.Z * 1.15))
		if (cx+cy+cz)&1 == 0 {
			check = 0.72
		}
	}
	return [3]byte{
		clampColor(float64(obj.Color[0]) * shade * check * exposure),
		clampColor(float64(obj.Color[1]) * shade * check * exposure),
		clampColor(float64(obj.Color[2]) * shade * check * exposure),
	}
}

func clampColor(v float64) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(math.Round(v))
}
