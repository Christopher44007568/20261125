package main

import (
	"fmt"
	"math"
)

func tubeFragments(set FrameSet, camera Camera, centerX, centerY float64) []Fragment {
	const radialCount = 30
	const turns = 3
	n := len(set.Stations)
	type Vertex struct {
		Point  Vec3
		Normal Vec3
	}
	vertices := make([][]Vertex, n)
	for i, station := range set.Stations {
		vertices[i] = make([]Vertex, radialCount)
		for j := 0; j < radialCount; j++ {
			theta := tau * float64(j) / radialCount
			offset := station.E1.Scale(semiMajor * math.Cos(theta)).
				Add(station.E2.Scale(semiMinor * math.Sin(theta)))
			normal := station.E1.Scale(math.Cos(theta) / semiMajor).
				Add(station.E2.Scale(math.Sin(theta) / semiMinor)).Normalized()
			vertices[i][j] = Vertex{Point: station.Point.Add(offset), Normal: normal}
		}
	}

	light := Vec3{-0.35, -0.45, 0.82}.Normalized()
	fragments := make([]Fragment, 0, n*radialCount)
	for i := 0; i < n; i++ {
		next := (i + 1) % n
		for j := 0; j < radialCount; j++ {
			jNext := (j + 1) % radialCount
			a := vertices[i][j]
			b := vertices[i][jNext]
			c := vertices[next][jNext]
			d := vertices[next][j]
			theta := tau * float64(j) / radialCount
			phase := theta - turns*tau*float64(i)/float64(n)
			stripe := false
			for k := 0; k < turns; k++ {
				distance := math.Abs(wrapAngle(phase - tau*float64(k)/turns))
				distance = math.Min(distance, tau-distance)
				if distance < 0.105 {
					stripe = true
				}
			}
			normal := a.Normal.Add(b.Normal).Add(c.Normal).Add(d.Normal).Normalized()
			diffuse := 0.55 + 0.45*math.Max(0, normal.Dot(light))
			if normal.Dot(camera.Front) < 0 {
				diffuse *= 0.58
			}
			fill := shadedColor(42, 108, 197, diffuse)
			if stripe {
				fill = shadedColor(245, 196, 73, diffuse)
			}
			pa := projectPoint(a.Point, camera, centerX, centerY)
			pb := projectPoint(b.Point, camera, centerX, centerY)
			pc := projectPoint(c.Point, camera, centerX, centerY)
			pd := projectPoint(d.Point, camera, centerX, centerY)
			depth := (pa.Depth + pb.Depth + pc.Depth + pd.Depth) / 4
			svg := fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="0.35"/>`,
				pa.X, pa.Y, pb.X, pb.Y, pc.X, pc.Y, pd.X, pd.Y, fill, fill)
			fragments = append(fragments, Fragment{Depth: depth, SVG: svg})
		}
	}
	return fragments
}

func arrowFragments(set FrameSet, camera Camera, centerX, centerY float64) []Fragment {
	var fragments []Fragment
	step := len(set.Stations) / 12
	for i := 0; i < len(set.Stations); i += step {
		station := set.Stations[i]
		fragments = append(fragments,
			axisArrow(station.Point, station.E1, semiMajor+0.31, camera, centerX, centerY, "#dc2626"),
			axisArrow(station.Point, station.E2, semiMinor+0.27, camera, centerX, centerY, "#7c3aed"),
		)
	}
	return fragments
}

func axisArrow(base, direction Vec3, length float64, camera Camera, centerX, centerY float64, color string) Fragment {
	tipPoint := base.Add(direction.Scale(length))
	backPoint := tipPoint.Sub(direction.Scale(0.13))
	screenRight := Vec3{-camera.Up.Z, 0, camera.Up.X}
	if screenRight.Norm() < 1e-9 {
		screenRight = camera.Right
	}
	screenRight = screenRight.Normalized()
	bar1 := backPoint.Add(screenRight.Scale(0.055)).Sub(camera.Up.Scale(0.025))
	bar2 := backPoint.Sub(screenRight.Scale(0.055)).Sub(camera.Up.Scale(0.025))

	p0 := projectPoint(base, camera, centerX, centerY)
	p1 := projectPoint(tipPoint, camera, centerX, centerY)
	p2 := projectPoint(bar1, camera, centerX, centerY)
	p3 := projectPoint(bar2, camera, centerX, centerY)
	depth := (p0.Depth + p1.Depth + p2.Depth + p3.Depth) / 4
	svg := fmt.Sprintf(`<g stroke-linecap="round" stroke-linejoin="round"><line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="3"/><polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s"/></g>`,
		p0.X, p0.Y, p1.X, p1.Y, color, p1.X, p1.Y, p2.X, p2.Y, p3.X, p3.Y, color)
	return Fragment{Depth: depth, SVG: svg}
}

func shadedColor(r, g, b int, shade float64) string {
	scale := math.Max(0.35, math.Min(1.12, shade))
	red := int(math.Round(math.Min(255, math.Max(0, float64(r)*scale))))
	green := int(math.Round(math.Min(255, math.Max(0, float64(g)*scale))))
	blue := int(math.Round(math.Min(255, math.Max(0, float64(b)*scale))))
	return fmt.Sprintf("#%02x%02x%02x",
		red, green, blue)
}
