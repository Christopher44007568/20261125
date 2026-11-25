package main

import (
	"fmt"
	"sort"
	"strings"
)

type FrameSet struct {
	Name     string
	Stations []Station
	Note     string
}

type Camera struct {
	Right Vec3
	Up    Vec3
	Front Vec3
}

type Fragment struct {
	Depth float64
	SVG   string
}

type Projected struct {
	X, Y, Depth float64
}

func renderSVG(sets []FrameSet, pathLength float64) string {
	camera := Camera{
		Right: Vec3{1, 0, 0},
		Up:    Vec3{0, 1, 0},
		Front: Vec3{0, 0, 1},
	}
	panelWidth := 430.0
	panelHeight := 520.0
	margin := 24.0
	width := margin*2 + panelWidth*float64(len(sets)) + margin*float64(len(sets)-1)
	height := 760.0
	var fragments []Fragment
	var backgrounds []Fragment
	centerY := 330.0

	for setIndex, set := range sets {
		centerX := margin + panelWidth/2 + float64(setIndex)*(panelWidth+margin)
		backgrounds = append(backgrounds, backgroundFragment(centerX, panelHeight, panelWidth, setIndex))
		fragments = append(fragments, tubeFragments(set, camera, centerX, centerY)...)
		fragments = append(fragments, arrowFragments(set, camera, centerX, centerY)...)
	}

	sort.SliceStable(fragments, func(i, j int) bool {
		return fragments[i].Depth > fragments[j].Depth
	})

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">`, width, height, width, height))
	b.WriteString(`<rect width="100%" height="100%" fill="#f8fafc"/>`)
	b.WriteString(`<style>text{font-family:'Droid Sans Fallback','Noto Sans CJK SC',system-ui,-apple-system,sans-serif}.title{font-size:22px;font-weight:700;fill:#0f172a}.note{font-size:13px;fill:#475569}.legend{font-size:12px;fill:#334155}</style>`)
	b.WriteString(fmt.Sprintf(`<text class="title" x="%.1f" y="34">闭合椭圆管扫掠标架对比</text>`, width/2))
	b.WriteString(fmt.Sprintf(`<text class="note" x="%.1f" y="58" text-anchor="middle">路径长度 %.2f；黄色为三条随标架前进的螺旋纹理，红/紫箭头表示椭圆截面两个主轴方向</text>`, width/2, pathLength))
	for _, f := range backgrounds {
		b.WriteString(f.SVG)
	}
	for _, f := range fragments {
		b.WriteString(f.SVG)
	}
	for setIndex, set := range sets {
		centerX := margin + panelWidth/2 + float64(setIndex)*(panelWidth+margin)
		b.WriteString(fmt.Sprintf(`<text class="title" x="%.1f" y="665" text-anchor="middle">%s</text>`, centerX, set.Name))
		b.WriteString(fmt.Sprintf(`<text class="note" x="%.1f" y="690" text-anchor="middle">%s</text>`, centerX, set.Note))
	}
	b.WriteString(`<text class="legend" x="24" y="730">红箭头：E1 / 椭圆长轴　　紫箭头：E2 / 椭圆短轴；箭头按 3D 深度与管体一起排序。</text>`)
	b.WriteString(`</svg>`)
	return b.String()
}

func projectPoint(v Vec3, camera Camera, centerX, centerY float64) Projected {
	const scale = 39.0
	return Projected{
		X:     centerX + camera.Right.Dot(v)*scale,
		Y:     centerY - camera.Up.Dot(v)*scale,
		Depth: camera.Front.Dot(v),
	}
}

func backgroundFragment(centerX, panelHeight, panelWidth float64, index int) Fragment {
	return Fragment{
		Depth: -1000,
		SVG: fmt.Sprintf(`<rect x="%.1f" y="82" width="%.1f" height="%.1f" rx="18" fill="%s" stroke="#e2e8f0"/>`,
			centerX-panelWidth/2, panelWidth, panelHeight, []string{"#eff6ff", "#f0fdf4", "#fff7ed"}[index%3]),
	}
}
