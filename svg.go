package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

type SVG struct {
	Width  int
	Height int
	body   bytes.Buffer
	defs   bytes.Buffer
}

func NewSVG(width, height int) *SVG {
	return &SVG{Width: width, Height: height}
}

func f(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}

func (s *SVG) Raw(v string)     { s.body.WriteString(v) }
func (s *SVG) Def(v string)     { s.defs.WriteString(v) }
func (s *SVG) Body() string     { return s.body.String() }
func (s *SVG) Size() (int, int) { return s.Width, s.Height }
func (s *SVG) AddBody(v string) { s.body.WriteString(v) }
func (s *SVG) AddDefs(v string) { s.defs.WriteString(v) }
func (s *SVG) Defs() string     { return s.defs.String() }
func (s *SVG) Write(path string) error {
	var out bytes.Buffer
	fmt.Fprintf(&out, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprintf(&out, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n",
		s.Width, s.Height, s.Width, s.Height)
	if s.defs.Len() > 0 {
		out.WriteString("<defs>\n")
		out.WriteString(s.defs.String())
		out.WriteString("</defs>\n")
	}
	out.WriteString(s.body.String())
	out.WriteString("</svg>\n")
	return os.WriteFile(path, out.Bytes(), 0o644)
}

func styleAttrs(fill, stroke string, sw, opacity, fillOpacity float64) string {
	var b strings.Builder
	if fill == "" {
		fill = "none"
	}
	b.WriteString(` fill="`)
	b.WriteString(fill)
	b.WriteString(`"`)
	if stroke != "" && sw > 0 {
		fmt.Fprintf(&b, ` stroke="%s" stroke-width="%s" stroke-linejoin="round" stroke-linecap="round"`, stroke, f(sw))
	}
	if opacity > 0 && opacity < 1 {
		fmt.Fprintf(&b, ` opacity="%s"`, f(opacity))
	}
	if fillOpacity > 0 && fillOpacity < 1 {
		fmt.Fprintf(&b, ` fill-opacity="%s"`, f(fillOpacity))
	}
	return b.String()
}

func (s *SVG) Rect(x, y, w, h float64, fill string) {
	fmt.Fprintf(&s.body, `<rect x="%s" y="%s" width="%s" height="%s"%s/>`+"\n",
		f(x), f(y), f(w), f(h), styleAttrs(fill, "", 0, 1, 1))
}

func (s *SVG) RectEx(x, y, w, h float64, fill, stroke string, sw, opacity, radius float64) {
	fmt.Fprintf(&s.body, `<rect x="%s" y="%s" width="%s" height="%s" rx="%s"%s/>`+"\n",
		f(x), f(y), f(w), f(h), f(radius), styleAttrs(fill, stroke, sw, opacity, 1))
}

func (s *SVG) Circle(cx, cy, r float64, fill, stroke string, sw, opacity float64) {
	fmt.Fprintf(&s.body, `<circle cx="%s" cy="%s" r="%s"%s/>`+"\n",
		f(cx), f(cy), f(r), styleAttrs(fill, stroke, sw, opacity, 1))
}

func (s *SVG) Line(x1, y1, x2, y2 float64, stroke string, sw, opacity float64) {
	dx, dy := x2-x1, y2-y1
	length := math.Hypot(dx, dy)
	if length < 0.01 || sw <= 0 {
		return
	}
	nx := -dy / length * sw * 0.5
	ny := dx / length * sw * 0.5
	pts := []Vec2{
		{x1 + nx, y1 + ny},
		{x2 + nx, y2 + ny},
		{x2 - nx, y2 - ny},
		{x1 - nx, y1 - ny},
	}
	s.Poly(pts, stroke, "", 0, opacity, true)
}

func (s *SVG) DashedLine(x1, y1, x2, y2 float64, stroke string, sw, opacity float64, dash string) {
	parts := strings.Fields(dash)
	on, off := 5.0, 4.0
	if len(parts) > 0 {
		if v, err := strconv.ParseFloat(parts[0], 64); err == nil {
			on = v
		}
	}
	if len(parts) > 1 {
		if v, err := strconv.ParseFloat(parts[1], 64); err == nil {
			off = v
		}
	}
	dx, dy := x2-x1, y2-y1
	length := math.Hypot(dx, dy)
	if length < 0.01 || on <= 0 {
		return
	}
	ux, uy := dx/length, dy/length
	for d := 0.0; d < length; d += on + off {
		end := math.Min(d+on, length)
		s.Line(x1+ux*d, y1+uy*d, x1+ux*end, y1+uy*end, stroke, sw, opacity)
	}
}

func pointList(points []Vec2) string {
	var b strings.Builder
	for i, p := range points {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(f(p.X))
		b.WriteByte(',')
		b.WriteString(f(p.Y))
	}
	return b.String()
}

func (s *SVG) Poly(points []Vec2, fill, stroke string, sw, opacity float64, closed bool) {
	if len(points) == 0 {
		return
	}
	tag := "polyline"
	if closed {
		tag = "polygon"
	}
	fmt.Fprintf(&s.body, `<%s points="%s"%s/>`+"\n",
		tag, pointList(points), styleAttrs(fill, stroke, sw, opacity, 1))
}

// Segmented draws an open polyline as individual line elements. Some SVG
// rasterizers incorrectly fill long open polylines, so keeping explicit
// segments makes the output robust in both CAD and browser viewers.
func (s *SVG) Segmented(points []Vec2, stroke string, sw, opacity float64) {
	for i := 1; i < len(points); i++ {
		s.Line(points[i-1].X, points[i-1].Y, points[i].X, points[i].Y, stroke, sw, opacity)
	}
}

func (s *SVG) Path(d, fill, stroke string, sw, opacity, fillOpacity float64) {
	fmt.Fprintf(&s.body, `<path d="%s"%s/>`+"\n",
		d, styleAttrs(fill, stroke, sw, opacity, fillOpacity))
}

func (s *SVG) Text(x, y, size float64, fill, weight, anchor, text string) {
	s.text(x, y, size, fill, weight, anchor, "", text)
}

func (s *SVG) Mono(x, y, size float64, fill, weight, anchor, text string) {
	s.text(x, y, size, fill, weight, anchor, "ui-monospace, 'SFMono-Regular', 'DejaVu Sans Mono', monospace", text)
}

func (s *SVG) text(x, y, size float64, fill, weight, anchor, family, text string) {
	if family == "" {
		family = "'DejaVu Sans', 'Segoe UI', sans-serif"
	}
	fmt.Fprintf(&s.body, `<text x="%s" y="%s" font-family="%s" font-size="%s" fill="%s" font-weight="%s" text-anchor="%s">%s</text>`+"\n",
		f(x), f(y), family, f(size), fill, weight, anchor, esc(text))
}

func (s *SVG) TextSpaced(x, y, size float64, fill, weight, anchor, text string, spacing float64) {
	fmt.Fprintf(&s.body, `<text x="%s" y="%s" font-family="'DejaVu Sans', 'Segoe UI', sans-serif" font-size="%s" fill="%s" font-weight="%s" text-anchor="%s" letter-spacing="%s">%s</text>`+"\n",
		f(x), f(y), f(size), fill, weight, anchor, f(spacing), esc(text))
}

func (s *SVG) Save(path string) error { return s.Write(path) }
