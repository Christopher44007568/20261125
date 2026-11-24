package stereorollback

import "strings"

// EyeSurface is what is presented for one eye in one frame.
type EyeSurface struct {
	Eye         Eye        `json:"eye"`
	Status      EyeStatus  `json:"status"`
	Success     bool       `json:"success"`
	Placeholder bool       `json:"placeholder"`
	Marker      string     `json:"marker"`
	Checksum    string     `json:"checksum,omitempty"`
	Resource    ResourceID `json:"resource_id,omitempty"`
	Width       int        `json:"width,omitempty"`
	Height      int        `json:"height,omitempty"`
	Note        string     `json:"note,omitempty"`
}

func renderedSurface(eye Eye, img *PixelImage, target ResourceID) EyeSurface {
	return EyeSurface{
		Eye:      eye,
		Status:   StatusRendered,
		Success:  true,
		Marker:   img.Marker,
		Checksum: img.Checksum,
		Resource: target,
		Width:    img.Width,
		Height:   img.Height,
	}
}

func placeholderSurface(eye Eye, img *PixelImage, stage Stage, note string) EyeSurface {
	if note == "" {
		note = "declared placeholder after " + string(stage) + " failure"
	}
	return EyeSurface{
		Eye:         eye,
		Status:      StatusPlaceholder,
		Success:     false,
		Placeholder: true,
		Marker:      img.Marker,
		Checksum:    img.Checksum,
		Width:       img.Width,
		Height:      img.Height,
		Note:        note,
	}
}

func sharedFallbackSurface(eye Eye, img *PixelImage, note string) EyeSurface {
	return EyeSurface{
		Eye:      eye,
		Status:   StatusSharedFallback,
		Success:  false,
		Marker:   img.Marker + "-" + eye.Label(),
		Checksum: img.Checksum,
		Width:    img.Width,
		Height:   img.Height,
		Note:     note,
	}
}

// FallbackFrame is the presented stereo picture when something did not render
// cleanly. The healthy eye keeps its real surface while the failed eye shows
// its declared placeholder.
type FallbackFrame struct {
	Frame          uint64       `json:"frame"`
	Kind           string       `json:"kind"`
	Scope          FailureScope `json:"scope"`
	FailedEye      Eye          `json:"failed_eye,omitempty"`
	Left           EyeSurface   `json:"left"`
	Right          EyeSurface   `json:"right"`
	ComposedMarker string       `json:"composed_marker"`
	PreviewPNG     string       `json:"preview_png,omitempty"`
	Note           string       `json:"note,omitempty"`
}

func composeMarker(left, right EyeSurface) string {
	var b strings.Builder
	b.WriteString("L[")
	b.WriteString(left.Marker)
	b.WriteString("]|R[")
	b.WriteString(right.Marker)
	b.WriteString("]")
	return b.String()
}

func newEyeFallbackFrame(frame uint64, failed Eye, left, right EyeSurface) FallbackFrame {
	return FallbackFrame{
		Frame:          frame,
		Kind:           "failed_eye_placeholder",
		Scope:          ScopeForEye(failed),
		FailedEye:      failed,
		Left:           left,
		Right:          right,
		ComposedMarker: composeMarker(left, right),
		Note:           "healthy eye preserved; failed eye shows its declared placeholder",
	}
}

func newSharedFallbackFrame(frame uint64, scope FailureScope, left, right EyeSurface) FallbackFrame {
	return FallbackFrame{
		Frame:          frame,
		Kind:           "shared_texture_fallback",
		Scope:          scope,
		Left:           left,
		Right:          right,
		ComposedMarker: composeMarker(left, right),
		Note:           "shared texture unavailable; both eyes use the shared emergency image",
	}
}
