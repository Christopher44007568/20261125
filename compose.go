package stereorollback

// ComposeFallbackImage rebuilds the presented stereo fallback picture as a
// side-by-side RGBA image: real pixels for the healthy eye and the declared
// placeholder/shared image for the failing side.
func ComposeFallbackImage(fb FallbackFrame) *PixelImage {
	const w, h = 16, 16
	canvas := make([]byte, w*2*h*4)
	copySide(canvas, w*2, 0, pixelsForSurface(fb.Left, fb.Frame))
	copySide(canvas, w*2, w, pixelsForSurface(fb.Right, fb.Frame))
	return &PixelImage{
		Width:    w * 2,
		Height:   h,
		Marker:   fb.ComposedMarker,
		RGBA:     canvas,
		Checksum: checksumBytes(append(append([]byte(nil), canvas...), []byte(fb.ComposedMarker)...)),
	}
}

func pixelsForSurface(s EyeSurface, frame uint64) []byte {
	switch s.Status {
	case StatusRendered:
		return eyeImage(s.Eye, frame).RGBA
	case StatusPlaceholder:
		return placeholderImage(s.Eye).RGBA
	case StatusSharedFallback:
		return sharedFallbackImage().RGBA
	default:
		return placeholderImage(s.Eye).RGBA
	}
}

func copySide(canvas []byte, canvasWidth, xOffset int, src []byte) {
	const w, h = 16, 16
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcI := (y*w + x) * 4
			dstI := (y*canvasWidth + xOffset + x) * 4
			copy(canvas[dstI:dstI+4], src[srcI:srcI+4])
		}
	}
}
