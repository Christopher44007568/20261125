package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

type box struct{ x0, y0, x1, y1 int }

func drawViewports(c *cv, fr *FrameReport) {
	left := box{20, 94, 470, 348}
	right := box{490, 94, 940, 348}
	drawEyePanel(c, left, "L / PRIMARY", fr, true)
	if fr.State.EyeCount >= 2 {
		drawEyePanel(c, right, "R EYE", fr, false)
	} else {
		drawSuspendedEye(c, right)
	}
}

func drawEyePanel(c *cv, b box, label string, fr *FrameReport, primary bool) {
	c.fill(b.x0, b.y0, b.x1, b.y1, colPanel)
	c.strokeRect(b.x0, b.y0, b.x1, b.y1, 2, colBorder)

	tw := textWidth(fontTerm16, label)
	cx0, cy0 := b.x0+14, b.y0+12
	c.strokeRect(cx0-8, cy0-4, cx0+tw+10, cy0+22, 1, colCyanDim)
	c.text(cx0, cy0, fontTerm16, colCyan, label)

	c.withClip(image.Rect(b.x0+2, b.y0+28, b.x1-2, b.y1-2), func() {
		drawWireScene(c, b)
	})

	pct := scalePercent(fr.State.SamplingScale)
	txt := fmt.Sprintf("SAMPLE %d%%", pct)
	sc := colGreenTx
	if fr.State.SampleLevel > 0 {
		sc = colAmber
	}
	w := textWidth(fontTerm16, txt)
	bx1 := b.x1 - 16
	bx0 := bx1 - w - 24
	by0, by1 := b.y1-36, b.y1-12
	c.fill(bx0+2, by0+2, bx1-2, by1-2, colHeader)
	c.strokeRect(bx0, by0, bx1, by1, 2, sc)
	c.text(bx0+12, by0+5, fontTerm16, sc, txt)
}

func drawSuspendedEye(c *cv, b box) {
	c.fill(b.x0, b.y0, b.x1, b.y1, colBg)
	c.dashedRect(b.x0+2, b.y0+2, b.x1-2, b.y1-2, 12, 8, 2, colAmber)
	c.textCentered((b.x0+b.x1)/2, b.y0+92, fontBold20, colAmber, "R EYE SUSPENDED")
	c.textCentered((b.x0+b.x1)/2, b.y0+140, fontTerm12, colDim,
		"MONOCULAR MODE - NOT SILENT")
	c.textCentered((b.x0+b.x1)/2, b.y0+160, fontTerm12, colDim,
		"ONE EYE ALWAYS RENDERED")
}

func drawWireScene(c *cv, b box) {
	vpX := (b.x0 + b.x1) / 2
	vpY := b.y0 + 108
	bottom := b.y1 - 10

	for x := b.x0 - 40; x <= b.x1+40; x += 46 {
		c.line(vpX, vpY, x, bottom, colGrid)
	}
	for _, t := range []float64{0.16, 0.3, 0.48, 0.68, 1} {
		y := vpY + int(float64(bottom-vpY)*t*t)
		xl := vpX - int(float64(vpX-(b.x0+8))*t)
		xr := vpX + int(float64((b.x1-8)-vpX)*t)
		c.line(xl, y, xr, y, colGrid)
	}

	drawCube(c, vpX-72, b.y1-60, 56, colCyan)
	drawCube(c, vpX-12, b.y1-108, 38, colMagenta)
	drawCube(c, vpX+48, b.y1-58, 48, colAmber)
}

func drawCube(c *cv, cx, baseY, s int, col color.RGBA) {
	x := cx - s/2
	y := baseY - s
	ox := int(float64(s) * 0.38)
	oy := -int(float64(s) * 0.38)

	c.strokeRect(x, y, x+s, y+s, 2, col)
	bx, by := x+ox, y+oy
	c.strokeRect(bx, by, bx+s, by+s, 1, col)
	c.line(x, y, bx, by, col)
	c.line(x+s, y, bx+s, by, col)
	c.line(x, y+s, bx, by+s, col)
	c.line(x+s, y+s, bx+s, by+s, col)
}

func drawHistory(c *cv, fr *FrameReport) {
	b := box{20, 356, 940, 424}
	c.fill(b.x0, b.y0, b.x1, b.y1, colHist)
	c.strokeRect(b.x0, b.y0, b.x1, b.y1, 1, colBorder)
	c.text(36, 366, fontTerm16, colLine, "HISTORY BUFFER")

	n := fr.Frame
	if n > 16 {
		n = 16
	}
	for i := 0; i < n; i++ {
		x := 36 + i*21
		c.fill(x, 392, x+15, 407, colGreenTx)
	}

	reproj := "ON"
	reprojCol := colGreenTx
	if !fr.State.Reprojection {
		reproj = "OFF"
		reprojCol = colAmber
	}
	c.text(752, 366, fontTerm16, reprojCol, "REPROJ: "+reproj)
	c.text(752, 392, fontTerm16, colGreenTx, "HIST: ON")
}

func drawStatusLines(c *cv, sr *ScenarioReport, fr *FrameReport, frameNo int) {
	x, y, pitch := 22, 430, 21
	st := fr.State
	stereo := "ON"
	if !st.Stereo {
		stereo = "OFF(MONO)"
	}
	stateLine := fmt.Sprintf("EYES=%d  STEREO=%s  SAMPLE=%d%%  REPROJ=%s  HIST=%s",
		st.EyeCount, stereo, scalePercent(st.SamplingScale),
		onOff(st.Reprojection), onOff(st.History))
	c.text(x, y, fontTerm16, colText, stateLine)
	y += pitch

	for _, e := range sr.Episodes {
		if frameNo < e.StartFrame || frameNo > e.EndFrame {
			continue
		}
		line := fmt.Sprintf("%s %s %s DUR=%dF", e.ID,
			strings.ToUpper(e.Mode), strings.ToUpper(e.ReasonCode),
			frameNo-e.StartFrame+1)
		c.text(x, y, fontTerm16, colLine, line)
		y += pitch
	}

	for _, rp := range fr.StatusDetail.RecoveryProgress {
		line := fmt.Sprintf("RECOVERY %s %d/%d",
			strings.ToUpper(rp.Kind), rp.Streak, rp.Need)
		c.text(x, y, fontTerm16, colLine, line)
		y += pitch
	}

	for i := len(sr.SafetyEvents) - 1; i >= 0; i-- {
		ev := sr.SafetyEvents[i]
		c.text(x, y, fontTerm16, colRed, "SAFETY REJECT: "+strings.ToUpper(ev.Code))
		y += pitch
	}
}

func onOff(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}
