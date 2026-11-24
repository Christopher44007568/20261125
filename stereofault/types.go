package stereofault

// Eye identifies one side of the stereo pair.
type Eye int

const (
	Left Eye = iota
	Right
)

func (e Eye) String() string {
	if e == Left {
		return "L"
	}
	return "R"
}

// LayerID identifies one composited layer.
type LayerID string

const (
	World LayerID = "world"
	UI    LayerID = "ui"
	Hand  LayerID = "hand"
)

// Fault is the class of injected resource damage.
type Fault int

const (
	FaultNone Fault = iota
	FaultMissingTexture
	FaultBadDepthFormat
	FaultBadProjection
)

func (f Fault) String() string {
	switch f {
	case FaultMissingTexture:
		return "missing-texture"
	case FaultBadDepthFormat:
		return "bad-depth-format"
	case FaultBadProjection:
		return "bad-projection"
	default:
		return "none"
	}
}

// FailurePolicy decides what a broken layer is allowed to do.
type FailurePolicy int

const (
	// PolicyPlaceholder blits the layer's declared placeholder through a
	// safe path that never touches the shared command buffer.
	PolicyPlaceholder FailurePolicy = iota
	// PolicyHide drops the layer entirely for the frame.
	PolicyHide
)

// LayerStatus is the per-frame outcome for one layer.
type LayerStatus int

const (
	StatusHealthy LayerStatus = iota
	StatusPlaceholder
	StatusHidden
)

func (s LayerStatus) String() string {
	switch s {
	case StatusPlaceholder:
		return "PLACEHOLDER"
	case StatusHidden:
		return "HIDDEN"
	default:
		return "healthy"
	}
}

// Mat4 is a row-major 4x4 projection/transform matrix.
type Mat4 [16]float64

func (m Mat4) at(r, c int) float64 { return m[r*4+c] }

func det3(a, b, c, d, e, f, g, h, i float64) float64 {
	return a*(e*i-f*h) - b*(d*i-f*g) + c*(d*h-e*g)
}

// Det returns the determinant; a singular projection cannot map the scene.
func (m Mat4) Det() float64 {
	var det float64
	for c := 0; c < 4; c++ {
		minor := det3(
			m.at(1, (c+1)%4), m.at(1, (c+2)%4), m.at(1, (c+3)%4),
			m.at(2, (c+1)%4), m.at(2, (c+2)%4), m.at(2, (c+3)%4),
			m.at(3, (c+1)%4), m.at(3, (c+2)%4), m.at(3, (c+3)%4),
		)
		sign := 1.0
		if c%2 == 1 {
			sign = -1
		}
		det += sign * m.at(0, c) * minor
	}
	return det
}

// Identity returns the identity matrix.
func Identity() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Rect is a half-open pixel region [X0,X1) x [Y0,Y1).
type Rect struct {
	X0, Y0, X1, Y1 int
}

func (r Rect) contains(x, y int) bool {
	return x >= r.X0 && x < r.X1 && y >= r.Y0 && y < r.Y1
}

// Resources is the full resource bundle a layer may draw from. Each field
// can be independently corrupted by fault injection.
type Resources struct {
	TextureOK   bool
	DepthFormat string // expected value is DepthFormatValid.
	Projection  Mat4
}

const DepthFormatValid = "D24_UNORM_S8_UINT"

// Pixel tags every cell so ownership and stale data are auditable.
type Pixel struct {
	Owner   LayerID
	Content string // "layer", "placeholder", or "" when cleared
	Frame   int    // frame that last wrote the cell
}

// Framebuffer is one eye target. Clear resets ownership, which is what makes
// ghosting detectable rather than merely visually absent.
type Framebuffer struct {
	W, H   int
	pixels [][]Pixel
}

func NewFramebuffer(w, h int) *Framebuffer {
	rows := make([][]Pixel, h)
	for y := range rows {
		rows[y] = make([]Pixel, w)
	}
	return &Framebuffer{W: w, H: h, pixels: rows}
}

func (fb *Framebuffer) Clear(frame int) {
	for y := range fb.pixels {
		for x := range fb.pixels[y] {
			fb.pixels[y][x] = Pixel{Frame: frame}
		}
	}
}

func (fb *Framebuffer) fill(r Rect, owner LayerID, content string, frame int) {
	for y := r.Y0; y < r.Y1; y++ {
		for x := r.X0; x < r.X1; x++ {
			if y >= 0 && y < fb.H && x >= 0 && x < fb.W {
				fb.pixels[y][x] = Pixel{Owner: owner, Content: content, Frame: frame}
			}
		}
	}
}

func (fb *Framebuffer) At(x, y int) Pixel { return fb.pixels[y][x] }

// Empty reports whether no layer wrote any cell this frame.
func (fb *Framebuffer) Empty(frame int) bool {
	for y := range fb.pixels {
		for x := range fb.pixels[y] {
			if fb.pixels[y][x].Frame == frame && fb.pixels[y][x].Content != "" {
				return false
			}
		}
	}
	return true
}
