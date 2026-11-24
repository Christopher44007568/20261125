package stereorollback

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strconv"
)

// PixelImage is a tiny deterministic RGBA image used as a stand-in for a real
// eye target or a declared placeholder/fallback picture.
type PixelImage struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Marker   string `json:"marker"`
	Checksum string `json:"checksum"`
	RGBA     []byte `json:"-"`
}

// PNG encodes the image with the standard library only.
func (img *PixelImage) PNG() ([]byte, error) {
	rect := image.Rect(0, 0, img.Width, img.Height)
	rgba := image.NewRGBA(rect)
	if len(img.RGBA) == img.Width*img.Height*4 {
		copy(rgba.Pix, img.RGBA)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func newPatternImage(width, height int, marker string, base, stripe color.RGBA, frame uint64) *PixelImage {
	if width <= 0 {
		width = 16
	}
	if height <= 0 {
		height = 16
	}
	pix := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := base
			if ((x+y)/4)%2 == 0 {
				c = stripe
			}
			if frame > 0 {
				c.R = byte(int(c.R) + int(frame)*7)
				c.G = byte(int(c.G) + int(frame)*3)
			}
			i := (y*width + x) * 4
			pix[i+0] = c.R
			pix[i+1] = c.G
			pix[i+2] = c.B
			pix[i+3] = 255
		}
	}
	return &PixelImage{
		Width:    width,
		Height:   height,
		Marker:   marker,
		Checksum: checksumBytes(append(pix, []byte(marker)...)),
		RGBA:     pix,
	}
}

func eyeImage(eye Eye, frame uint64) *PixelImage {
	marker := "EYE-" + eye.Label() + "-F" + strconv.FormatUint(frame, 10)
	if eye == EyeLeft {
		return newPatternImage(16, 16, marker,
			color.RGBA{R: 40, G: 90, B: 200, A: 255},
			color.RGBA{R: 90, G: 150, B: 255, A: 255}, frame)
	}
	return newPatternImage(16, 16, marker,
		color.RGBA{R: 20, G: 140, B: 120, A: 255},
		color.RGBA{R: 70, G: 210, B: 180, A: 255}, frame)
}

func placeholderImage(eye Eye) *PixelImage {
	marker := "PH-" + string(eye)
	return newPatternImage(16, 16, marker,
		color.RGBA{R: 200, G: 170, B: 30, A: 255},
		color.RGBA{R: 40, G: 32, B: 0, A: 255}, 0)
}

func sharedFallbackImage() *PixelImage {
	marker := "PH-SHARED-TEXTURE"
	return newPatternImage(16, 16, marker,
		color.RGBA{R: 150, G: 40, B: 170, A: 255},
		color.RGBA{R: 235, G: 200, B: 255, A: 255}, 0)
}
