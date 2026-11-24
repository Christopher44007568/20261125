package stereofault

import "testing"

func newTestRenderer() (*Renderer, *Layer, *Layer, *Layer) {
	const w, h = 10, 8
	world := NewWorldLayer(Rect{X0: 0, Y0: 0, X1: w, Y1: h})
	ui := NewUILayer(Rect{X0: 0, Y0: 0, X1: w, Y1: 2})
	hand := NewHandLayer(Rect{X0: 6, Y0: 4, X1: w, Y1: h})
	// Back-to-front: world, hand, UI.
	r := NewRenderer(w, h, world, hand, ui)
	return r, world, ui, hand
}

func TestBackToFrontCompositingOrder(t *testing.T) {
	r, _, _, _ := newTestRenderer()
	snap := r.RenderFrame()

	// UI is drawn last and must own the overlapping top band on both eyes.
	for _, eye := range []Eye{Left, Right} {
		p := r.Target(eye).At(1, 1)
		if p.Owner != UI || p.Content != "layer" || p.Frame != snap.Frame {
			t.Fatalf("eye %s top-band owner = %+v, want UI layer content", eye, p)
		}
	}
}

func statusOf(s FrameSnapshot, id LayerID) LayerStatus {
	for _, ls := range s.Layers {
		if ls.Layer == id {
			return ls.Status
		}
	}
	return -1
}

func TestFaultIsolationAndRecovery(t *testing.T) {
	r, world, ui, hand := newTestRenderer()

	s1 := r.RenderFrame()
	if s1.Black {
		t.Fatal("frame 1 must not be black when every layer is healthy")
	}

	// Frame 2: world loses its texture.
	world.Inject(FaultMissingTexture)
	s2 := r.RenderFrame()
	if got := statusOf(s2, World); got != StatusPlaceholder {
		t.Fatalf("world = %v, want placeholder", got)
	}
	if statusOf(s2, UI) != StatusHealthy || statusOf(s2, Hand) != StatusHealthy {
		t.Fatal("unaffected layers must stay healthy")
	}
	if s2.Black {
		t.Fatal("a single broken layer must not black out the stereo picture")
	}

	// Frame 3: both world and UI are down.
	ui.Inject(FaultBadDepthFormat)
	s3 := r.RenderFrame()
	if statusOf(s3, World) != StatusPlaceholder || statusOf(s3, UI) != StatusPlaceholder {
		t.Fatal("world and UI must show declared placeholders")
	}
	if statusOf(s3, Hand) != StatusHealthy {
		t.Fatal("hand must keep rendering")
	}
	if s3.Black {
		t.Fatal("healthy hand layer must keep the picture non-black")
	}

	// Frame 4: all three layers are down; hand has no placeholder and hides.
	hand.Inject(FaultBadProjection)
	s4 := r.RenderFrame()
	if statusOf(s4, World) != StatusPlaceholder || statusOf(s4, UI) != StatusPlaceholder {
		t.Fatal("world/UI placeholders must remain active")
	}
	if statusOf(s4, Hand) != StatusHidden {
		t.Fatalf("hand = %v, want hidden (no placeholder declared)", statusOf(s4, Hand))
	}
	if s4.Black {
		t.Fatal("declared placeholders keep the picture visible even when all layers are down")
	}

	// Recover world first.
	world.Repair()
	s5 := r.RenderFrame()
	if statusOf(s5, World) != StatusHealthy {
		t.Fatal("world must recover after its resource is restored")
	}
	if statusOf(s5, Hand) != StatusHidden {
		t.Fatal("hand must remain hidden until its own resource is repaired")
	}
	if s5.Black {
		t.Fatal("recovered world must bring the picture back")
	}

	// Recover the remaining two layers.
	ui.Repair()
	hand.Repair()
	s6 := r.RenderFrame()
	for _, id := range []LayerID{World, UI, Hand} {
		if statusOf(s6, id) != StatusHealthy {
			t.Fatalf("layer %s must be healthy after repair", id)
		}
	}

	// No frame may ever violate an invariant.
	for _, s := range r.Timeline {
		if len(s.Violations) != 0 {
			t.Fatalf("frame %d invariants violated: %v", s.Frame, s.Violations)
		}
		if s.GhostPixels != 0 {
			t.Fatalf("frame %d has %d ghost pixels", s.Frame, s.GhostPixels)
		}
	}

	// Cumulative counters must cover every frame for every layer.
	const frames = 6
	for _, l := range []*Layer{world, ui, hand} {
		total := l.HealthyFrames + l.PlaceholderFrames + l.HiddenFrames
		if total != frames*2 { // two eyes per frame
			t.Fatalf("layer %s counted %d eye-frames, want %d", l.ID, total, frames*2)
		}
	}
	if world.PlaceholderFrames != 3*2 { // frames 2,3,4
		t.Fatalf("world placeholder eye-frames = %d, want 6", world.PlaceholderFrames)
	}
	if hand.HiddenFrames != 2*2 { // frames 4,5
		t.Fatalf("hand hidden eye-frames = %d, want 4", hand.HiddenFrames)
	}
}
