package stereorollback

import "fmt"

// Eye identifies one side of a stereo frame.
type Eye string

const (
	EyeLeft  Eye = "left"
	EyeRight Eye = "right"
)

// Valid reports whether e is one of the two known eyes.
func (e Eye) Valid() bool { return e == EyeLeft || e == EyeRight }

// Other returns the opposite eye. It panics on an invalid Eye; callers that
// parse external input should call Valid first.
func (e Eye) Other() Eye {
	switch e {
	case EyeLeft:
		return EyeRight
	case EyeRight:
		return EyeLeft
	default:
		panic(fmt.Sprintf("stereorollback: invalid eye %q", string(e)))
	}
}

// Label returns the single-glyph marker used in composed fallback frames.
func (e Eye) Label() string {
	switch e {
	case EyeLeft:
		return "L"
	case EyeRight:
		return "R"
	default:
		return "?"
	}
}

// EyeStatus describes the surface presented for one eye in one frame.
type EyeStatus string

const (
	// StatusRendered is a genuine render of the eye.
	StatusRendered EyeStatus = "rendered"
	// StatusPlaceholder is the declared placeholder for a failed eye.
	StatusPlaceholder EyeStatus = "placeholder"
	// StatusSharedFallback is an emergency shared-resource fallback.
	StatusSharedFallback EyeStatus = "shared_fallback"
)
