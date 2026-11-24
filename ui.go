package main

type TextStyle struct {
	HeightM float64 `json:"heightMeters"`
	Bold    bool    `json:"bold"`
}

type TextSpec struct {
	ID             string    `json:"id"`
	Text           string    `json:"text"`
	X              float64   `json:"xRelativeMeters"`
	Top            float64   `json:"topRelativeMeters"`
	Style          TextStyle `json:"style"`
	Critical       bool      `json:"critical"`
	CheckSharpness bool      `json:"checkSharpness,omitempty"`
}

type RectSpec struct {
	ID     string  `json:"id"`
	X      float64 `json:"xRelativeMeters"`
	Top    float64 `json:"topRelativeMeters"`
	Width  float64 `json:"widthMeters"`
	Height float64 `json:"heightMeters"`
	Button bool    `json:"button"`
}

type PanelSpec struct {
	ID     string  `json:"id"`
	Center Vec3    `json:"centerMeters"`
	Width  float64 `json:"widthMeters"`
	Height float64 `json:"heightMeters"`
}

type Layer struct {
	ID         string     `json:"id"`
	NameZH     string     `json:"nameZh"`
	DepthClass string     `json:"depthClass"`
	Panel      PanelSpec  `json:"panel"`
	Rects      []RectSpec `json:"rects"`
	Texts      []TextSpec `json:"texts"`
	AccentRGB  [3]uint8   `json:"accentRgb"`
}

func DefaultLayers() []Layer {
	return []Layer{
		{
			ID:         "near_plane",
			NameZH:     "近平面",
			DepthClass: "near",
			Panel: PanelSpec{
				ID:     "near-panel",
				Center: Vec3{X: 0, Y: 0.02, Z: 0.72},
				Width:  0.68,
				Height: 0.46,
			},
			AccentRGB: [3]uint8{94, 234, 212},
			Rects: []RectSpec{
				{ID: "near-button", X: -0.29, Top: -0.075, Width: 0.18, Height: 0.070, Button: true},
			},
			Texts: []TextSpec{
				{ID: "near-title", Text: "NEAR", X: -0.29, Top: 0.175, Style: TextStyle{HeightM: 0.064, Bold: true}, Critical: true},
				{ID: "near-distance", Text: "0.65 M", X: -0.29, Top: 0.075, Style: TextStyle{HeightM: 0.034}, Critical: true},
				{ID: "near-digits", Text: "0123456789", X: -0.29, Top: 0.018, Style: TextStyle{HeightM: 0.034}, Critical: true},
				{ID: "near-ok", Text: "OK", X: -0.232, Top: -0.090, Style: TextStyle{HeightM: 0.038}, Critical: false},
			},
		},
		{
			ID:         "mid_distance",
			NameZH:     "中距离",
			DepthClass: "mid",
			Panel: PanelSpec{
				ID:     "mid-panel",
				Center: Vec3{X: 0, Y: 0.05, Z: 2.45},
				Width:  1.70,
				Height: 1.05,
			},
			AccentRGB: [3]uint8{147, 197, 253},
			Rects: []RectSpec{
				{ID: "mid-button", X: -0.72, Top: -0.24, Width: 0.42, Height: 0.16, Button: true},
			},
			Texts: []TextSpec{
				{ID: "mid-title", Text: "MID", X: -0.72, Top: 0.39, Style: TextStyle{HeightM: 0.150, Bold: true}, Critical: true},
				{ID: "mid-distance", Text: "2.20 M", X: -0.72, Top: 0.17, Style: TextStyle{HeightM: 0.080}, Critical: true},
				{ID: "mid-label", Text: "STEREO", X: -0.72, Top: 0.030, Style: TextStyle{HeightM: 0.080}, Critical: true},
				{ID: "mid-ok", Text: "OK", X: -0.578, Top: -0.279, Style: TextStyle{HeightM: 0.085}, Critical: false},
			},
		},
		{
			ID:         "system_layer",
			NameZH:     "系统层",
			DepthClass: "system",
			Panel: PanelSpec{
				ID:     "system-panel",
				Center: Vec3{X: 0, Y: 0.08, Z: 8.0},
				Width:  4.00,
				Height: 1.55,
			},
			AccentRGB: [3]uint8{52, 211, 153},
			Texts: []TextSpec{
				{ID: "system-title", Text: "SYSTEM", X: -1.72, Top: 0.55, Style: TextStyle{HeightM: 0.260, Bold: true}, Critical: true, CheckSharpness: true},
				{ID: "system-edge", Text: "EDGE 0123456789", X: -1.72, Top: 0.18, Style: TextStyle{HeightM: 0.150}, Critical: true, CheckSharpness: true},
				{ID: "system-status", Text: "SHARP OK", X: -1.72, Top: -0.18, Style: TextStyle{HeightM: 0.130}, Critical: true},
			},
		},
	}
}

func (l Layer) worldPoint(x, top float64) Vec3 {
	return Vec3{
		X: l.Panel.Center.X + x,
		Y: l.Panel.Center.Y + top,
		Z: l.Panel.Center.Z,
	}
}
