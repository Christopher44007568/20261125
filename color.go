package main

import (
	"fmt"
	"math"
	"strconv"
)

type RGB struct{ R, G, B float64 }

func parseHex(s string) RGB {
	if len(s) != 7 || s[0] != '#' {
		return RGB{}
	}
	r, _ := strconv.ParseInt(s[1:3], 16, 32)
	g, _ := strconv.ParseInt(s[3:5], 16, 32)
	b, _ := strconv.ParseInt(s[5:7], 16, 32)
	return RGB{float64(r), float64(g), float64(b)}
}

func hex(c RGB) string {
	r := int(clamp(math.Round(c.R), 0, 255))
	g := int(clamp(math.Round(c.G), 0, 255))
	b := int(clamp(math.Round(c.B), 0, 255))
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func mixColor(a, b string, t float64) string {
	t = clamp(t, 0, 1)
	ca, cb := parseHex(a), parseHex(b)
	return hex(RGB{
		ca.R + (cb.R-ca.R)*t,
		ca.G + (cb.G-ca.G)*t,
		ca.B + (cb.B-ca.B)*t,
	})
}

func scaleColor(c string, k float64) string {
	v := parseHex(c)
	return hex(RGB{v.R * k, v.G * k, v.B * k})
}

func heatColor(t float64) string {
	t = clamp(t, 0, 1)
	stops := []struct {
		P float64
		C RGB
	}{
		{0.00, RGB{17, 22, 58}},
		{0.16, RGB{33, 77, 156}},
		{0.34, RGB{28, 151, 188}},
		{0.50, RGB{60, 201, 148}},
		{0.67, RGB{218, 216, 77}},
		{0.83, RGB{244, 136, 48}},
		{1.00, RGB{185, 29, 54}},
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].P {
			a, b := stops[i-1], stops[i]
			u := (t - a.P) / (b.P - a.P)
			return hex(RGB{
				a.C.R + (b.C.R-a.C.R)*u,
				a.C.G + (b.C.G-a.C.G)*u,
				a.C.B + (b.C.B-a.C.B)*u,
			})
		}
	}
	return hex(stops[len(stops)-1].C)
}

const (
	colBG      = "#071017"
	colPanel   = "#0b1720"
	colPanel2  = "#0e1e28"
	colGrid    = "#18313c"
	colGrid2   = "#244652"
	colText    = "#e8f3f4"
	colMuted   = "#7e9aa6"
	colFaint   = "#4e6b76"
	colCyan    = "#58e1d2"
	colBlue    = "#5aa7ff"
	colTeal    = "#2d8f92"
	colGold    = "#f2c96c"
	colOrange  = "#ff9848"
	colRed     = "#ff5d5d"
	colMagenta = "#d276ff"
	colPurple  = "#7c5cff"
	colWhite   = "#f6fbfb"
)
