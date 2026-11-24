package stereofault

import "errors"

// Layer is one independently fault-isolated compositing layer.
type Layer struct {
	ID        LayerID
	Region    Rect
	Policy    FailurePolicy
	Resources Resources

	// HasPlaceholder declares whether the layer owns a fallback texture.
	// A hidden layer sets this false; it may never synthesize one on failure.
	HasPlaceholder bool
	Placeholder    string

	// pristine is the known-good snapshot used to repair injected faults.
	pristine Resources

	// state
	Fault      Fault
	Healthy    bool
	FailFrames int

	// cumulative counters
	HealthyFrames     int
	PlaceholderFrames int
	HiddenFrames      int
}

func newLayer(id LayerID, region Rect, policy FailurePolicy, res Resources, placeholder string) *Layer {
	return &Layer{
		ID:             id,
		Region:         region,
		Policy:         policy,
		Resources:      res,
		HasPlaceholder: placeholder != "",
		Placeholder:    placeholder,
		pristine:       res,
		Healthy:        true,
	}
}

func goodResources() Resources {
	return Resources{
		TextureOK:   true,
		DepthFormat: DepthFormatValid,
		Projection:  Identity(),
	}
}

func NewWorldLayer(region Rect) *Layer {
	return newLayer(World, region, PolicyPlaceholder, goodResources(), "PH-WORLD")
}

func NewUILayer(region Rect) *Layer {
	return newLayer(UI, region, PolicyPlaceholder, goodResources(), "PH-UI")
}

func NewHandLayer(region Rect) *Layer {
	// The hand layer deliberately declares no placeholder: on failure its
	// only legal option is to hide.
	return newLayer(Hand, region, PolicyHide, goodResources(), "")
}

// Validate checks only the resource class this layer depends on. A damaged
// resource is rejected before any command for the layer is recorded.
func (l *Layer) Validate() error {
	switch l.ID {
	case World:
		if !l.Resources.TextureOK {
			return errors.New("world texture is missing")
		}
	case UI:
		if l.Resources.DepthFormat != DepthFormatValid {
			return errors.New("ui depth buffer format is unsupported: " + l.Resources.DepthFormat)
		}
	case Hand:
		if l.Resources.Projection.Det() == 0 {
			return errors.New("hand projection matrix is singular")
		}
	}
	return nil
}

// Inject corrupts the resource field mapped to the given fault class.
func (l *Layer) Inject(f Fault) {
	l.Fault = f
	switch f {
	case FaultMissingTexture:
		l.Resources.TextureOK = false
	case FaultBadDepthFormat:
		l.Resources.DepthFormat = "UNKNOWN_FORMAT"
	case FaultBadProjection:
		zero := Mat4{}
		l.Resources.Projection = zero
	}
}

// Repair restores the known-good resource snapshot and clears the fault.
func (l *Layer) Repair() {
	l.Resources = l.pristine
	l.Fault = FaultNone
}

// markFailed records one failed frame and flips the layer out of service.
func (l *Layer) markFailed() {
	l.Healthy = false
	l.FailFrames++
}

// markRecovered returns the layer to normal service.
func (l *Layer) markRecovered() {
	l.Healthy = true
}

// recordOutcome accumulates the per-frame policy counters.
func (l *Layer) recordOutcome(status LayerStatus) {
	switch status {
	case StatusHealthy:
		l.HealthyFrames++
	case StatusPlaceholder:
		l.PlaceholderFrames++
	case StatusHidden:
		l.HiddenFrames++
	}
}

// BlitPlaceholder draws the declared fallback through a dedicated safe path.
// It writes directly to the eye target and never appends to a command buffer,
// so a broken layer cannot poison shared recording state.
func (l *Layer) BlitPlaceholder(fb *Framebuffer, frame int) error {
	if !l.HasPlaceholder || l.Placeholder == "" {
		return errors.New("layer " + string(l.ID) + " has no declared placeholder")
	}
	fb.fill(l.Region, l.ID, "placeholder:"+l.Placeholder, frame)
	return nil
}

// Draw records a normal layer command into the shared command buffer and
// rasterizes it. It is only reachable after Validate has passed.
func (l *Layer) Draw(cb *CommandBuffer, eye Eye, fb *Framebuffer, frame int) {
	cb.append(command{
		frame: frame,
		eye:   eye,
		layer: l.ID,
		kind:  "draw",
	})
	fb.fill(l.Region, l.ID, "layer", frame)
}
