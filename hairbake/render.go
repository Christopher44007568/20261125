package main

import "math"

// Camera is a simple pinhole camera looking at the head center.
type Camera struct {
	Pos, Target, Up        Vec3
	Focal                  float64
	View                   Vec3
	BasisX, BasisY, BasisZ Vec3
}

var (
	keyLight  = v3(-0.55, 0.75, 0.45).Norm()
	fillLight = v3(0.4, 0.25, -0.6).Norm()
	viewViews = []struct {
		Name string
		Az   float64
		El   float64
	}{
		{"front", 0, 12},
		{"left", -90, 8},
		{"right", 90, 8},
		{"back", 180, 10},
		{"top", 20, 72},
	}
)

func makeCamera(azDeg, elDeg, dist, focal float64) Camera {
	az, el := azDeg*math.Pi/180, elDeg*math.Pi/180
	pos := v3(
		dist*math.Cos(el)*math.Sin(az),
		scalpCenterY+dist*math.Sin(el),
		dist*math.Cos(el)*math.Cos(az),
	)
	target := v3(0, scalpCenterY-0.5, 0)
	view := target.Sub(pos).Norm()
	up := v3(0, 1, 0)
	bx := up.Cross(view).Norm()
	if bx.Len() < 1e-9 {
		bx = v3(1, 0, 0)
	}
	by := view.Cross(bx).Norm()
	return Camera{Pos: pos, Target: target, Up: up, Focal: focal, View: view, BasisX: bx, BasisY: by, BasisZ: view}
}

// project maps a world point to screen coordinates (y down) and camera depth.
func (cam Camera) project(p Vec3, w, h int) (float64, float64, float64) {
	d := p.Sub(cam.Pos)
	cx := d.Dot(cam.BasisX)
	cy := d.Dot(cam.BasisY)
	cz := d.Dot(cam.BasisZ)
	if cz < 0.05 {
		return -1, -1, cz
	}
	sx := float64(w)/2 + cam.Focal*cx/cz
	sy := float64(h)/2 - cam.Focal*cy/cz
	return sx, sy, cz
}

// hairShade implements a Kajiya-Kay style anisotropic term plus fill light.
func hairShade(tangent, view, light Vec3, specAmt float64) float64 {
	t := tangent.Norm()
	v := view.Norm()
	l := light.Norm()
	ttl := t.Dot(l)
	sinTL := math.Sqrt(math.Max(0, 1-ttl*ttl))
	spec := math.Pow(clampf(sinTL, 0, 1), 26) * specAmt
	diff := 0.32 + 0.5*clampf(0.5+0.5*ttl, 0, 1)

	ttf := t.Dot(fillLight)
	sinTF := math.Sqrt(math.Max(0, 1-ttf*ttf))
	fill := 0.18 * math.Pow(clampf(sinTF, 0, 1), 20)
	rim := math.Pow(1-clampf(math.Abs(t.Dot(v)), 0, 1), 3) * 0.12
	return diff + spec*0.9 + fill + rim
}

// renderHead draws the scalp ellipsoid as a lit disc behind the hair.
func renderHead(im *RGBA, cam Camera, e Ellipsoid) {
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			ndcX := (float64(x) - float64(im.W)/2) / cam.Focal
			ndcY := (float64(im.H)/2 - float64(y)) / cam.Focal
			ray := cam.BasisX.Scale(ndcX).Add(cam.BasisY.Scale(ndcY)).Add(cam.BasisZ).Norm()
			if hit, n, ok := rayEllipsoid(cam.Pos, ray, e); ok {
				shade := 0.25 + 0.6*clampf(n.Dot(keyLight), 0, 1) + 0.2*clampf(n.Dot(fillLight), 0, 1)
				skin := [3]float64{0.55 * shade, 0.42 * shade, 0.34 * shade}
				im.set(x, y, [4]float64{skin[0], skin[1], skin[2], 1})
				_ = hit
			}
		}
	}
}

func rayEllipsoid(origin, dir Vec3, e Ellipsoid) (Vec3, Vec3, bool) {
	oc := origin.Sub(e.Center)
	rr := e.Radii
	scaled := v3(dir.X/rr.X, dir.Y/rr.Y, dir.Z/rr.Z)
	ocs := v3(oc.X/rr.X, oc.Y/rr.Y, oc.Z/rr.Z)
	a := scaled.Dot(scaled)
	b := 2 * ocs.Dot(scaled)
	c := ocs.Dot(ocs) - 1
	disc := b*b - 4*a*c
	if disc < 0 {
		return Vec3{}, Vec3{}, false
	}
	t := (-b - math.Sqrt(disc)) / (2 * a)
	if t < 0 {
		t = (-b + math.Sqrt(disc)) / (2 * a)
	}
	if t < 0 {
		return Vec3{}, Vec3{}, false
	}
	hit := origin.Add(dir.Scale(t))
	n := e.surfaceNormal(hit)
	return hit, n, true
}
