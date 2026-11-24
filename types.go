package texturelife

import "fmt"

// Frame is a logical frame number. Zero means "not set"; real frames start at 1.
type Frame uint64

// TextureID is unique for the lifetime of a Tracker.
type TextureID uint64

// Eye identifies the eye that owns a texture.
type Eye uint8

const (
	EyeLeft Eye = iota + 1
	EyeRight
)

func (e Eye) String() string {
	switch e {
	case EyeLeft:
		return "left"
	case EyeRight:
		return "right"
	default:
		return fmt.Sprintf("eye(%d)", uint8(e))
	}
}

func (e Eye) valid() bool {
	return e == EyeLeft || e == EyeRight
}

// Kind identifies the role of a viewport texture.
type Kind uint8

const (
	KindColor Kind = iota + 1
	KindDepth
	KindHistory
)

func (k Kind) String() string {
	switch k {
	case KindColor:
		return "color"
	case KindDepth:
		return "depth"
	case KindHistory:
		return "history"
	default:
		return fmt.Sprintf("kind(%d)", uint8(k))
	}
}

func (k Kind) valid() bool {
	return k == KindColor || k == KindDepth || k == KindHistory
}

// BytesPerPixel is the simulated allocation density. Color and history use
// RGBA8; depth uses a 32-bit depth buffer.
func BytesPerPixel(k Kind) int64 {
	if k.valid() {
		return 4
	}
	return 0
}

// Size is a texture size in pixels.
type Size struct {
	Width  int
	Height int
}

func (s Size) valid() bool {
	return s.Width > 0 && s.Height > 0
}

func (s Size) String() string {
	return fmt.Sprintf("%dx%d", s.Width, s.Height)
}

// Pixels returns the pixel count, or zero for an invalid size.
func (s Size) Pixels() int64 {
	if !s.valid() {
		return 0
	}
	return int64(s.Width) * int64(s.Height)
}

// Bytes returns the simulated byte size for a texture kind.
func (s Size) Bytes(k Kind) int64 {
	return s.Pixels() * BytesPerPixel(k)
}

// Desc describes a texture allocation.
type Desc struct {
	Kind Kind
	Eye  Eye
	Size Size
}

// Texture is an immutable snapshot of one resource's lifetime metadata.
// ReleasedAt is the logical frame at which the resource was physically
// released. A zero ReleasedAt means the resource is still alive.
type Texture struct {
	ID            TextureID
	Kind          Kind
	Eye           Eye
	Size          Size
	Bytes         int64
	RefCount      int
	PeakRefCount  int
	CreatedFrame  Frame
	LastUsedFrame Frame
	RetireFrame   Frame
	ReleasedAt    Frame
}

// Alive reports whether the resource has not been physically released.
func (t Texture) Alive() bool {
	return t.ReleasedAt == 0
}

// Retiring reports whether release was requested.
func (t Texture) Retiring() bool {
	return t.RetireFrame != 0
}
