package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteBrightnessChart(path string, metrics []FrameMetric) error {
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
	const left = 68
	const top = 58
	const plotW = 940
	const plotH = 292
	bottom := top + plotH

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, width, height, width, height)
	b.WriteString(`<rect width="100%" height="100%" fill="#ffffff"/>`)
	b.WriteString(`<style>text{font-family:Inter,Segoe UI,Arial,sans-serif;fill:#1f2937}.title{font-size:22px;font-weight:700}.label{font-size:13px}.tick{font-size:12px;fill:#6b7280}.legend{font-size:13px;font-weight:600}</style>`)
	fmt.Fprintf(&b, `<text x="%d" y="30" class="title">Average brightness curve — fixed near-to-far camera path</text>`, left)

	xFor := func(i int) float64 {
		if len(metrics) <= 1 {
			return left
		}
		return left + float64(i)/float64(len(metrics)-1)*plotW
	}
	yFor := func(v float64) float64 {
		return top + plotH - clampFloat(v/255, 0, 1)*plotH
	}

	for v := 0.0; v <= 255; v += 50 {
		y := yFor(v)
		fmt.Fprintf(&b, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="#e5e7eb" stroke-width="1"/>`, left, y, left+plotW, y)
		fmt.Fprintf(&b, `<text x="%d" y="%.1f" text-anchor="end" class="tick">%.0f</text>`, left-10, y+4, v)
	}

	drawTransitionBands(&b, metrics, xFor, plotH, top)

	for i := 0; i < len(metrics); i += chartTickStep(len(metrics)) {
		x := xFor(i)
		fmt.Fprintf(&b, `<line x1="%.1f" y1="%d" x2="%.1f" y2="%d" stroke="#d1d5db"/>`, x, bottom, x, bottom+5)
		fmt.Fprintf(&b, `<text x="%.1f" y="%d" text-anchor="middle" class="tick">%03d</text>`, x, bottom+22, metrics[i].Frame)
	}
	fmt.Fprintf(&b, `<text x="%d" y="%d" text-anchor="middle" class="label">frame</text>`, left+plotW/2, height-16)
	fmt.Fprintf(&b, `<text transform="translate(18 %d) rotate(-90)" text-anchor="middle" class="label">Rec. 709 luma (0–255)</text>`, top+plotH/2)

	meanPoints := make([]string, 0, len(metrics))
	hairPoints := make([]string, 0, len(metrics))
	for i, m := range metrics {
		meanPoints = append(meanPoints, fmt.Sprintf("%.1f,%.1f", xFor(i), yFor(m.MeanLuma)))
		hairPoints = append(hairPoints, fmt.Sprintf("%.1f,%.1f", xFor(i), yFor(m.HairLuma)))
	}
	fmt.Fprintf(&b, `<path d="M %s" fill="none" stroke="#2563eb" stroke-width="3" stroke-linejoin="round" stroke-linecap="round"/>`, strings.Join(meanPoints, " L "))
	fmt.Fprintf(&b, `<path d="M %s" fill="none" stroke="#b45309" stroke-width="2.4" stroke-dasharray="7 5" stroke-linejoin="round" stroke-linecap="round"/>`, strings.Join(hairPoints, " L "))

	fmt.Fprintf(&b, `<line x1="%d" y1="42" x2="%d" y2="42" stroke="#2563eb" stroke-width="3"/>`, left+560, left+592)
	fmt.Fprintf(&b, `<text x="%d" y="47" class="legend">whole thumbnail mean</text>`, left+600)
	fmt.Fprintf(&b, `<line x1="%d" y1="42" x2="%d" y2="42" stroke="#b45309" stroke-width="2.4" stroke-dasharray="7 5"/>`, left+760, left+792)
	fmt.Fprintf(&b, `<text x="%d" y="47" class="legend">hair pixels only</text>`, left+800)

	b.WriteString(`</svg>`)
	_, err = f.WriteString(b.String())
	return err
}

func chartTickStep(frames int) int {
	switch {
	case frames <= 24:
		return 4
	case frames <= 60:
		return 6
	default:
		return 10
	}
}

func drawTransitionBands(b *strings.Builder, metrics []FrameMetric, xFor func(int) float64, plotH, top int) {
	n := len(metrics)
	i := 0
	for i < n {
		if metrics[i].Blend <= 0 || metrics[i].LODFrom == metrics[i].LODTo {
			i++
			continue
		}
		start := i
		for i < n && metrics[i].Blend > 0 && metrics[i].LODFrom == metrics[start].LODFrom {
			i++
		}
		end := i - 1
		step := 0.0
		if n > 1 {
			step = xFor(1) - xFor(0)
		}
		x := xFor(start) - step/2
		w := xFor(end) - xFor(start) + step
		fmt.Fprintf(b, `<rect x="%.1f" y="%d" width="%.1f" height="%d" fill="#f59e0b" opacity="0.16"/>`, x, top, w, plotH)
		midX := x + w/2
		label := fmt.Sprintf("LOD %d→%d stable morph", metrics[start].LODFrom, metrics[start].LODTo)
		fmt.Fprintf(b, `<text x="%.1f" y="%d" text-anchor="middle" class="tick" fill="#92400e">%s</text>`, midX, top+18, label)
	}
}
