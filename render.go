package main

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"strings"
)

const (
	screenW = 960
	screenH = 572
)

var (
	colBg      = color.RGBA{10, 15, 29, 255}
	colHeader  = color.RGBA{14, 23, 44, 255}
	colPanel   = color.RGBA{15, 26, 48, 255}
	colHist    = color.RGBA{16, 24, 42, 255}
	colFooter  = color.RGBA{8, 12, 22, 255}
	colBorder  = color.RGBA{43, 58, 85, 255}
	colGrid    = color.RGBA{34, 61, 104, 255}
	colText    = color.RGBA{230, 238, 251, 255}
	colDim     = color.RGBA{140, 158, 196, 255}
	colLine    = color.RGBA{210, 224, 248, 255}
	colCyan    = color.RGBA{84, 200, 232, 255}
	colCyanDim = color.RGBA{44, 110, 140, 255}
	colGreen   = color.RGBA{46, 204, 113, 255}
	colGreenTx = color.RGBA{64, 224, 140, 255}
	colAmber   = color.RGBA{242, 163, 60, 255}
	colRed     = color.RGBA{255, 84, 76, 255}
	colMagenta = color.RGBA{224, 86, 154, 255}
)

var enHeadlines = map[string]string{
	"01_false_positive_spike#2":          "ALL CAPABILITIES NOMINAL",
	"02_budget_sampling_recover#4":       "FRAME BUDGET: SAMPLING REDUCED",
	"03_continuous_mono_partial#6":       "FLOOR: MONO + MIN SAMPLE, STILL OVER BUDGET",
	"03_continuous_mono_partial#12":      "MONOCULAR: ONE EYE STAYS ACTIVE",
	"04_frustum_failure#3":               "REPROJ PAUSED: DIRECT RENDER + HISTORY",
	"05_pose_over_limit#3":               "REPROJ PAUSED: DIRECT RENDER + HISTORY",
	"06_partial_reprojection_recovery#3": "LOW SAMPLE + NO REPROJECTION",
	"08_invalidated_false_positive#2":    "REPROJ PAUSED: DIRECT RENDER + HISTORY",
	"09_safety_interlocks#2":             "SAFETY: ZERO-EYE / SILENT-HISTORY REJECTED",
}

// RenderShotPNG renders the retro status screen for one shot frame.
func RenderShotPNG(sr *ScenarioReport, frameNo int) ([]byte, error) {
	var fr *FrameReport
	for _, f := range sr.Frames {
		if f.Frame == frameNo {
			fr = f
			break
		}
	}
	if fr == nil && len(sr.Frames) > 0 {
		fr = sr.Frames[len(sr.Frames)-1]
	}
	c := newCanvas(screenW, screenH, colBg)
	accent := colGreen
	if fr.StatusLevel == "warn" {
		accent = colAmber
	} else if fr.StatusLevel == "critical" {
		accent = colRed
	}
	drawHeader(c, sr, fr, frameNo, accent)
	drawViewports(c, fr)
	drawHistory(c, fr)
	drawStatusLines(c, sr, fr, frameNo)
	drawFooter(c)

	var buf bytes.Buffer
	if err := png.Encode(&buf, c.img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawHeader(c *cv, sr *ScenarioReport, fr *FrameReport, frameNo int, accent color.RGBA) {
	c.fill(0, 0, screenW, 88, colHeader)
	c.fill(0, 0, 7, 85, accent)
	c.fill(0, 84, screenW, 87, accent)

	micro := strings.ToUpper(sr.ID) + " / FRAME " + pad2(frameNo)
	c.text(24, 12, fontTerm12, colDim, micro)

	headline := enHeadlines[sr.ID+"#"+itoa(frameNo)]
	if headline == "" {
		headline = enHeadlineFromStatus(fr)
	}
	c.text(20, 32, fontBold24, colText, headline)

	m := fr.Metrics
	c.textRight(940, 12, fontTerm16, colLine,
		f1(m.FrameTimeMS)+"MS / BUD "+f1(m.BudgetMS)+"MS")
	fpsCol := colGreenTx
	if m.FrameTimeMS > m.BudgetMS {
		fpsCol = colRed
	}
	c.textRight(940, 38, fontBold20, fpsCol, itoa(int(math.Round(m.FPS)))+" FPS")
}

func enHeadlineFromStatus(fr *FrameReport) string {
	if fr.StatusDetail.FloorWarning {
		return "FLOOR: MONO + MIN SAMPLE, STILL OVER BUDGET"
	}
	has := func(mode string) bool {
		for _, m := range fr.StatusDetail.ActiveModes {
			if m == mode {
				return true
			}
		}
		return false
	}
	low, mono, rep := has(modeLowSampling), has(modeMonocular), has(modeNoReproj)
	switch {
	case fr.StatusLevel == "critical":
		return "SAFETY: ZERO-EYE / SILENT-HISTORY REJECTED"
	case low && rep:
		return "LOW SAMPLE + NO REPROJECTION"
	case mono && low:
		return "CONTINUOUS DEGRADE: MONO + LOW SAMPLE"
	case mono:
		return "MONOCULAR: ONE EYE STAYS ACTIVE"
	case low:
		return "FRAME BUDGET: SAMPLING REDUCED"
	case rep:
		return "REPROJ PAUSED: DIRECT RENDER + HISTORY"
	default:
		return "ALL CAPABILITIES NOMINAL"
	}
}

func drawFooter(c *cv) {
	c.fill(0, 528, screenW, screenH, colFooter)
	c.fill(0, 528, screenW, 530, colBorder)
	c.text(22, 548, fontTerm16, colDim,
		"[B]BUDGET  [F]FRUSTUM  [P]POSE  [T]TRIM  [SPACE]STEP  [R]RESET")
	c.textRight(940, 548, fontTerm16, colGreenTx, "INPUT ACTIVE")
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func f1(v float64) string {
	return fmt.Sprintf("%.1f", v)
}
