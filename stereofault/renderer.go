package stereofault

import "fmt"

// EventKind enumerates recorded timeline events.
type EventKind string

const (
	EventFailure   EventKind = "failure"
	EventFallback  EventKind = "fallback"
	EventRecovery  EventKind = "recovery"
	EventInvariant EventKind = "invariant-violation"
)

// Event is one audit entry.
type Event struct {
	Frame int
	Eye   Eye // -1-ish sentinel: Left when event is not eye-specific
	Kind  EventKind
	Layer LayerID
	Fault Fault
	Note  string
}

func (e Event) eyeLabel() string {
	if e.Kind == EventFallback {
		return e.Eye.String()
	}
	return "-"
}

// LayerSnapshot is one layer's state at frame end.
type LayerSnapshot struct {
	Layer             LayerID
	Status            LayerStatus
	Fault             Fault
	HealthyFrames     int
	PlaceholderFrames int
	HiddenFrames      int
}

// FrameSnapshot captures the full auditable outcome of one frame.
type FrameSnapshot struct {
	Frame       int
	Layers      []LayerSnapshot
	Commands    int // shared-buffer commands appended this frame (both eyes)
	Black       bool
	GhostPixels int
	Violations  []string
}

// Renderer runs the stereo composite and records the timeline.
type Renderer struct {
	layers map[LayerID]*Layer
	order  []LayerID
	eyes   []Eye
	target map[Eye]*Framebuffer
	cb     *CommandBuffer

	Frame    int
	Events   []Event
	Timeline []FrameSnapshot
}

func NewRenderer(w, h int, layers ...*Layer) *Renderer {
	r := &Renderer{
		layers: map[LayerID]*Layer{},
		eyes:   []Eye{Left, Right},
		target: map[Eye]*Framebuffer{
			Left:  NewFramebuffer(w, h),
			Right: NewFramebuffer(w, h),
		},
		cb: NewCommandBuffer(),
	}
	for _, l := range layers {
		r.layers[l.ID] = l
		r.order = append(r.order, l.ID)
	}
	return r
}

func (r *Renderer) Layer(id LayerID) *Layer { return r.layers[id] }

func (r *Renderer) CommandBuffer() *CommandBuffer { return r.cb }

func (r *Renderer) Target(e Eye) *Framebuffer { return r.target[e] }

func (r *Renderer) record(ev Event) {
	r.Events = append(r.Events, ev)
}

// RenderFrame advances the simulation by one frame.
func (r *Renderer) RenderFrame() FrameSnapshot {
	r.Frame++
	frame := r.Frame
	cmdsBefore := r.cb.Len()

	// 1. Resolve each layer's service state once; recovery is re-validated.
	for _, id := range r.order {
		l := r.layers[id]
		if !l.Healthy {
			if err := l.Validate(); err == nil {
				l.markRecovered()
				r.record(Event{Frame: frame, Eye: Left, Kind: EventRecovery,
					Layer: id, Note: "resource restored; layer back in service"})
			}
		}
		if l.Healthy {
			if err := l.Validate(); err != nil {
				l.markFailed()
				r.record(Event{Frame: frame, Eye: Left, Kind: EventFailure,
					Layer: id, Fault: l.Fault, Note: err.Error()})
			}
		}
	}

	// 2. Composite each eye independently onto a cleared target.
	for _, eye := range r.eyes {
		fb := r.target[eye]
		fb.Clear(frame)

		for _, id := range r.order {
			l := r.layers[id]
			if l.Healthy {
				// Validated path: may use the shared command buffer.
				l.Draw(r.cb, eye, fb, frame)
				l.recordOutcome(StatusHealthy)
				continue
			}

			// Broken path: never record into the shared buffer. Choose the
			// only declared fallback.
			switch l.Policy {
			case PolicyPlaceholder:
				if err := l.BlitPlaceholder(fb, frame); err != nil {
					// No declared placeholder means the layer must hide.
					fb.fill(l.Region, "", "", frame)
					l.recordOutcome(StatusHidden)
					r.record(Event{Frame: frame, Eye: eye, Kind: EventFallback,
						Layer: id, Fault: l.Fault, Note: "hidden (no placeholder declared): " + err.Error()})
					continue
				}
				l.recordOutcome(StatusPlaceholder)
				r.record(Event{Frame: frame, Eye: eye, Kind: EventFallback,
					Layer: id, Fault: l.Fault, Note: "declared placeholder via isolated path"})
			case PolicyHide:
				// Region stays cleared: nothing to ghost.
				fb.fill(l.Region, "", "", frame)
				l.recordOutcome(StatusHidden)
				r.record(Event{Frame: frame, Eye: eye, Kind: EventFallback,
					Layer: id, Fault: l.Fault, Note: "hidden per policy"})
			}
		}
	}

	snap := r.snapshot(frame, r.cb.Len()-cmdsBefore)
	r.Timeline = append(r.Timeline, snap)
	return snap
}

func (r *Renderer) snapshot(frame, commandsAdded int) FrameSnapshot {
	snap := FrameSnapshot{Frame: frame, Commands: commandsAdded}
	for _, id := range r.order {
		l := r.layers[id]
		status := StatusHealthy
		if !l.Healthy {
			if l.Policy == PolicyPlaceholder && l.HasPlaceholder {
				status = StatusPlaceholder
			} else {
				status = StatusHidden
			}
		}
		snap.Layers = append(snap.Layers, LayerSnapshot{
			Layer:             id,
			Status:            status,
			Fault:             l.Fault,
			HealthyFrames:     l.HealthyFrames,
			PlaceholderFrames: l.PlaceholderFrames,
			HiddenFrames:      l.HiddenFrames,
		})
	}

	black := true
	for _, eye := range r.eyes {
		fb := r.target[eye]
		if !fb.Empty(frame) {
			black = false
		}
		snap.GhostPixels += r.countGhosts(fb, frame)
	}
	snap.Black = black
	snap.Violations = r.checkInvariants(frame, snap.GhostPixels, black)
	for _, v := range snap.Violations {
		r.record(Event{Frame: frame, Eye: Left, Kind: EventInvariant, Note: v})
	}
	return snap
}

// countGhosts finds cells whose owner layer is out of service yet still
// carries drawable content written this frame. Cleared placeholder/hidden
// regions never count because their content is empty.
func (r *Renderer) countGhosts(fb *Framebuffer, frame int) int {
	ghosts := 0
	for y := 0; y < fb.H; y++ {
		for x := 0; x < fb.W; x++ {
			p := fb.At(x, y)
			if p.Frame != frame || p.Content == "" || p.Owner == "" {
				continue
			}
			if l, ok := r.layers[p.Owner]; ok && !l.Healthy && p.Content == "layer" {
				ghosts++
			}
		}
	}
	return ghosts
}

// checkInvariants encodes the guarantees the demo and tests assert on.
func (r *Renderer) checkInvariants(frame, ghosts int, black bool) []string {
	var violations []string

	// A broken layer must never reach the shared command buffer.
	for _, id := range r.order {
		l := r.layers[id]
		if !l.Healthy && r.cb.ContainsLayer(id, frame) {
			violations = append(violations,
				fmt.Sprintf("frame %d: broken layer %s polluted the shared command buffer", frame, id))
		}
	}

	// Hidden/placeholder layers must leave no stale drawable pixels.
	if ghosts != 0 {
		violations = append(violations,
			fmt.Sprintf("frame %d: %d ghost pixels from failed layers", frame, ghosts))
	}

	// Placeholder policy without a declared placeholder must not synthesize one.
	for _, id := range r.order {
		l := r.layers[id]
		if !l.Healthy && l.Policy == PolicyPlaceholder && !l.HasPlaceholder {
			for _, eye := range r.eyes {
				fb := r.target[eye]
				for y := 0; y < fb.H; y++ {
					for x := 0; x < fb.W; x++ {
						p := fb.At(x, y)
						if p.Frame == frame && p.Owner == id {
							violations = append(violations,
								fmt.Sprintf("frame %d: layer %s drew without a declared placeholder", frame, id))
						}
					}
				}
			}
		}
	}

	// Total blackout only happens if every layer is simultaneously hidden.
	if black {
		allHidden := true
		anyExists := false
		for _, id := range r.order {
			l := r.layers[id]
			anyExists = true
			if l.Healthy || (l.Policy == PolicyPlaceholder && l.HasPlaceholder) {
				allHidden = false
			}
		}
		if anyExists && !allHidden {
			violations = append(violations,
				fmt.Sprintf("frame %d: stereo target went fully black although a drawable layer exists", frame))
		}
	}
	return violations
}
