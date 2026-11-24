package xrpath

import (
	"bytes"
	"image/png"
)

func encodeRenderedEye(frame int, eye *RenderedEye) (ImageRef, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, eye.Image); err != nil {
		return ImageRef{}, err
	}
	eye.PNGBytes = append([]byte(nil), buf.Bytes()...)
	ref := ImageRef{
		Eye:   eye.Eye,
		Path:  formatFrameName("images/"+eye.Eye, frame),
		Width: eye.Image.Bounds().Dx(), Height: eye.Image.Bounds().Dy(),
		PixelSHA256: bytesSHA256(eye.Image.Pix),
		PNGSHA256:   bytesSHA256(buf.Bytes()),
	}
	return ref, nil
}
