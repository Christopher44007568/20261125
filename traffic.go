package eyebw

import "fmt"

type tileKeys struct {
	DepthMSAA     string
	DepthResolved string
	ColorMSAA     string
	ColorResolved string
	History       string
	Front         string
}

func makeTileKeys(prefix string, eye EyeID, x, y int) tileKeys {
	base := fmt.Sprintf("%s:%s:%d:%d", prefix, eye, x, y)
	return tileKeys{
		DepthMSAA:     base + ":depth-msaa",
		DepthResolved: base + ":depth",
		ColorMSAA:     base + ":color-msaa",
		ColorResolved: base + ":color",
		History:       base + ":history",
		Front:         base + ":front",
	}
}

func texelsForDivisor(tile Tile, divisor int) uint64 {
	if divisor <= 1 {
		return tile.FullPixels
	}
	rw := (tile.Width + divisor - 1) / divisor
	rh := (tile.Height + divisor - 1) / divisor
	return mulSize(rw, rh)
}

type tileBuilder struct {
	cfg          FrameConfig
	meter        *Meter
	history      map[string]int
	repaint      *RepaintReport
	observations *[]CenterObservation
}

func (b *tileBuilder) emitFoveationRepaint(eye EyeID, keys tileKeys, tile Tile, previousDivisor int, hadHistory bool) {
	if !hadHistory || tile.Divisor >= previousDivisor {
		return
	}
	previousTexels := texelsForDivisor(tile, previousDivisor)
	texels := uint64(0)
	fullRepaint := false
	if tile.QualityCenter {
		texels = tile.FullPixels
		fullRepaint = true
	} else if tile.Texels > previousTexels {
		texels = tile.Texels - previousTexels
	}
	if texels == 0 {
		return
	}
	b.emitRepaintBytes(eye, keys, texels, "foveation_upgrade")
	b.repaint.FoveationUpgradeTexels += texels
	if fullRepaint && tile.QualityCenter {
		// The observation is appended by renderedTile; record repair through
		// a dedicated event reason that tests can inspect if needed.
	}
}

func (b *tileBuilder) emitRepaintBytes(eye EyeID, keys tileKeys, texels uint64, reason string) {
	if texels == 0 {
		return
	}
	payloadBytes := texels * uint64(b.cfg.RepaintBytesPerTexel)
	b.meter.ReadReason(StageRepaint, eye, ClassRepaint, keys.Front+":payload", reason+":payload", payloadBytes)
	b.meter.WriteReason(StageRepaint, eye, ClassRepaint, keys.Front+":patch", reason+":patch", payloadBytes)
	b.repaint.TotalTexels += texels
}

func (b *tileBuilder) emitDisocclusion(eye EyeID, keys tileKeys, tile Tile, motion float64) {
	baseTexels := uint64(float64(tile.FullPixels) * b.cfg.SingleEye.DisocclusionFraction)
	motionTexels := uint64(float64(tile.FullPixels) * motion * b.cfg.SingleEye.MotionFraction)
	texels := baseTexels + motionTexels
	if texels == 0 {
		return
	}
	b.emitRepaintBytes(eye, keys, texels, "disocclusion")
	b.repaint.DisocclusionTexels += texels
	b.repaint.DynamicMotionTexels += motionTexels
}

func (b *tileBuilder) renderedTile(eye EyeID, prefix string, tile Tile) tileKeys {
	keys := makeTileKeys(prefix, eye, tile.IndexX, tile.IndexY)
	colorBPP := uint64(b.cfg.ColorBytesPerSample)
	depthBPP := uint64(b.cfg.DepthBytesPerSample)
	historyBPP := uint64(b.cfg.HistoryBytesPerTexel)

	sampleColorBytes := tile.Samples * colorBPP
	sampleDepthBytes := tile.Samples * depthBPP
	resolvedColorBytes := tile.Texels * colorBPP
	resolvedDepthBytes := tile.Texels * depthBPP

	b.meter.Write(StageDepthPrepass, eye, ClassDepth, keys.DepthMSAA, sampleDepthBytes)
	b.meter.Read(StageOpaquePass, eye, ClassDepth, keys.DepthMSAA, sampleDepthBytes)
	b.meter.Write(StageOpaquePass, eye, ClassColor, keys.ColorMSAA, sampleColorBytes)

	if b.cfg.MSAASamples > 1 {
		b.meter.Read(StageResolve, eye, ClassColor, keys.ColorMSAA, sampleColorBytes)
		b.meter.Write(StageResolve, eye, ClassColor, keys.ColorResolved, resolvedColorBytes)
		b.meter.Read(StageDepthResolve, eye, ClassDepth, keys.DepthMSAA, sampleDepthBytes)
		b.meter.Write(StageDepthResolve, eye, ClassDepth, keys.DepthResolved, resolvedDepthBytes)
	} else {
		keys.ColorResolved = keys.ColorMSAA
		keys.DepthResolved = keys.DepthMSAA
	}

	previousDivisor, hadHistory := b.history[keys.History]
	historyValid := false
	if hadHistory {
		previousTexels := texelsForDivisor(tile, previousDivisor)
		b.meter.Read(StageHistory, eye, ClassHistory, keys.History, previousTexels*historyBPP)
		historyValid = previousDivisor == tile.Divisor
	}
	b.meter.Write(StageHistory, eye, ClassHistory, keys.History, tile.Texels*historyBPP)
	b.history[keys.History] = tile.Divisor

	b.meter.Read(StageReproject, eye, ClassColor, keys.ColorResolved, resolvedColorBytes)
	b.meter.Read(StageReproject, eye, ClassDepth, keys.DepthResolved, resolvedDepthBytes)
	frontBytes := tile.FullPixels * colorBPP
	b.meter.Write(StageReproject, eye, ClassColor, keys.Front, frontBytes)
	b.meter.ReadUncached(StageScanout, eye, ClassColor, keys.Front, "display_scanout", frontBytes)

	b.emitFoveationRepaint(eye, keys, tile, previousDivisor, hadHistory)

	if tile.QualityCenter {
		_, upgraded := b.history[keys.History]
		_ = upgraded
		*b.observations = append(*b.observations, CenterObservation{
			Eye:                   eye,
			TileX:                 tile.IndexX,
			TileY:                 tile.IndexY,
			Divisor:               tile.Divisor,
			MSAASamples:           b.cfg.msaa(),
			IndependentlyRendered: true,
			HistoryValid:          historyValid,
			FullRepaint:           hadHistory && tile.Divisor < previousDivisor && tile.QualityCenter,
		})
	}
	return keys
}
