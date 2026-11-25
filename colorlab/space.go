package colorlab

import (
	"fmt"
	"strings"
)

// Space identifies a display-referred RGB encoding. Rec.2020, PQ, and HLG use
// the BT.2020 primaries; sRGB uses the IEC 61966-2-1 primaries.
type Space uint8

const (
	SRGB Space = iota
	Rec2020
	PQ
	HLG
)

var allSpaces = [...]Space{SRGB, Rec2020, PQ, HLG}

// RGB is a three-channel color. Values are normalized to [0,1] for PPM output,
// but conversion routines retain out-of-range values where the transfer
// functions are defined so that round trips are not clipped prematurely.
type RGB struct {
	R float64
	G float64
	B float64
}

func (s Space) String() string {
	switch s {
	case SRGB:
		return "sRGB"
	case Rec2020:
		return "Rec.2020"
	case PQ:
		return "PQ"
	case HLG:
		return "HLG"
	default:
		return fmt.Sprintf("Space(%d)", s)
	}
}

// Slug returns a stable, filename-safe representation of the space name.
func (s Space) Slug() string {
	switch s {
	case SRGB:
		return "srgb"
	case Rec2020:
		return "rec2020"
	case PQ:
		return "pq"
	case HLG:
		return "hlg"
	default:
		return "unknown"
	}
}

// ParseSpace accepts the names used by the CLI.
func ParseSpace(name string) (Space, error) {
	switch strings.ToLower(strings.TrimSpace(strings.ReplaceAll(name, ".", ""))) {
	case "srgb":
		return SRGB, nil
	case "rec2020", "bt2020", "2020":
		return Rec2020, nil
	case "pq", "st2084":
		return PQ, nil
	case "hlg", "bt2100hlg":
		return HLG, nil
	default:
		return 0, fmt.Errorf("unknown color space %q", name)
	}
}

// AllSpaces returns the four supported spaces in a stable order.
func AllSpaces() []Space {
	out := make([]Space, len(allSpaces))
	copy(out, allSpaces[:])
	return out
}

// Clamp01 limits each component to the conventional display range. It is
// intentionally only applied when writing integer image samples.
func (c RGB) Clamp01() RGB {
	return RGB{
		R: clamp01(c.R),
		G: clamp01(c.G),
		B: clamp01(c.B),
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Decode converts an encoded color to linear light in its native primaries.
func Decode(c RGB, space Space) RGB {
	switch space {
	case SRGB:
		return RGB{sRGBToLinear(c.R), sRGBToLinear(c.G), sRGBToLinear(c.B)}
	case Rec2020:
		return RGB{rec2020ToLinear(c.R), rec2020ToLinear(c.G), rec2020ToLinear(c.B)}
	case PQ:
		return RGB{pqToLinear(c.R), pqToLinear(c.G), pqToLinear(c.B)}
	case HLG:
		return RGB{hlgToLinear(c.R), hlgToLinear(c.G), hlgToLinear(c.B)}
	default:
		return c
	}
}

// Encode converts linear light in a space's native primaries to its encoded
// representation.
func Encode(c RGB, space Space) RGB {
	switch space {
	case SRGB:
		return RGB{linearToSRGB(c.R), linearToSRGB(c.G), linearToSRGB(c.B)}
	case Rec2020:
		return RGB{linearToRec2020(c.R), linearToRec2020(c.G), linearToRec2020(c.B)}
	case PQ:
		return RGB{linearToPQ(c.R), linearToPQ(c.G), linearToPQ(c.B)}
	case HLG:
		return RGB{linearToHLG(c.R), linearToHLG(c.G), linearToHLG(c.B)}
	default:
		return c
	}
}

// ToLinearRec2020 returns BT.2020 linear light. It is the common working space
// used by every conversion in this package.
func ToLinearRec2020(c RGB, from Space) RGB {
	linear := Decode(c, from)
	if from == SRGB {
		return srgbToRec2020.Apply(linear)
	}
	return linear
}

// FromLinearRec2020 converts BT.2020 linear light to the requested encoded
// space.
func FromLinearRec2020(c RGB, to Space) RGB {
	if to == SRGB {
		return Encode(rec2020ToSRGB.Apply(c), SRGB)
	}
	return Encode(c, to)
}

// Convert performs a forward conversion between any two supported spaces.
func Convert(c RGB, from, to Space) RGB {
	return FromLinearRec2020(ToLinearRec2020(c, from), to)
}

// ConvertPixels returns a new slice with every pixel converted.
func ConvertPixels(pixels []RGB, from, to Space) []RGB {
	out := make([]RGB, len(pixels))
	for i, c := range pixels {
		out[i] = Convert(c, from, to)
	}
	return out
}
