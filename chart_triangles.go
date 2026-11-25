package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteTriangleChart(path string, metrics []FrameMetric) error {
	if len(metrics) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	const width = 1040
	const height = 430
	const left = 76
	const top = 58
	const plotW = 932
	const plotH = 292
	bottom := top + plotH

	maxTris := float64(metrics[0].LOD0Triangles) * 1.10
	yFor := func(v float64) float64 {
		return top + plotH - clampFloat(v/maxTris, 0, 1)*plotH
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, width, height, width, height)
	b.WriteString(`<rect width="100%" height="100%" fill="#ffffff"/>`)
	b.WriteString(`<style>text{font-family:Inter,Segoe UI,Arial,sans-serif;fill:#1f2937}.title{font-size:22px;font-weight:700}.label{font-size:13px}.tick{font-size:12px;fill:#6b7280}.legend{font-size:13px;font-weight:600}</style>`)
	fmt.Fprintf(&b, `<text x="%d" y="30" class="title">Hair triangle counts — deterministic LOD collapse</text>`, left)

	for t := 0.0; t <= 1.0001; t += 0.2 {
		v := maxTris * t
		y := yFor(v)
		fmt.Fprintf(&b, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="#e5e7eb"/>`, left, y, left+plotW, y)
		fmt.Fprintf(&b, `<text x="%d" y="%.1f" text-anchor="end" class="tick">%s</text>`, left-10, y+4, formatTriangleTick(v))
	}

	n := len(metrics)
	slot := plotW / float64(n)
	barW := slot * 0.66
	for i, m := range metrics {
		x := left + float64(i)*slot + (slot-barW)/2
		y := yFor(float64(m.HairTriangles))
		h := float64(bottom) - y
		fill := "#f97316"
		if m.Blend > 0 && m.LODFrom != m.LODTo {
			fill = "#f59e0b"
		}
		fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="1.5" fill="%s"/>`, x, y, barW, h, fill)
	}

	drawCountReference(&b, "LOD0 asset", metrics[0].LOD0Triangles, yFor, left, plotW, "#9a3412")
	drawCountReference(&b, "LOD1 asset", metrics[0].LOD1Triangles, yFor, left, plotW, "#b45309")
	drawCountReference(&b, "LOD2 asset", metrics[0].LOD2Triangles, yFor, left, plotW, "#15803d")

	for i := 0; i < n; i += chartTickStep(n) {
		x := left + float64(i)*slot + slot/2
		fmt.Fprintf(&b, `<text x="%.1f" y="%d" text-anchor="middle" class="tick">%03d</text>`, x, bottom+22, metrics[i].Frame)
	}
	fmt.Fprintf(&b, `<text x="%d" y="%d" text-anchor="middle" class="label">frame</text>`, left+plotW/2, height-16)
	fmt.Fprintf(&b, `<text transform="translate(20 %d) rotate(-90)" text-anchor="middle" class="label">hair triangles per frame</text>`, top+plotH/2)

	fmt.Fprintf(&b, `<rect x="%d" y="42" width="14" height="14" fill="#f97316"/><text x="%d" y="54" class="legend">stable LOD</text>`, left+520, left+540)
	fmt.Fprintf(&b, `<rect x="%d" y="42" width="14" height="14" fill="#f59e0b"/><text x="%d" y="54" class="legend">geomorph window</text>`, left+650, left+670)

	b.WriteString(`</svg>`)
	_, err = f.WriteString(b.String())
	return err
}

func drawCountReference(b *strings.Builder, label string, count int,
	yFor func(float64) float64, left, plotW int, color string) {
	y := yFor(float64(count))
	fmt.Fprintf(b, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="%s" stroke-width="1.6" stroke-dasharray="6 5"/>`, left, y, left+plotW, y, color)
	fmt.Fprintf(b, `<text x="%d" y="%.1f" text-anchor="end" class="tick" fill="%s">%s: %d</text>`, left+plotW-8, y-5, color, label, count)
}

func formatTriangleTick(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.1fk", v/1000)
	}
	return fmt.Sprintf("%.0f", v)
}
