package stereofault

import "testing"

func TestSharedCommandBufferIsolation(t *testing.T) {
	r, world, ui, hand := newTestRenderer()

	world.Inject(FaultMissingTexture)
	ui.Inject(FaultBadDepthFormat)
	hand.Inject(FaultBadProjection)
	snap := r.RenderFrame()

	// Every layer is down, so the shared buffer must receive nothing.
	if r.CommandBuffer().Len() != 0 {
		t.Fatalf("shared buffer holds %d commands despite all layers failing", r.CommandBuffer().Len())
	}
	if snap.Commands != 0 {
		t.Fatalf("frame appended %d commands, want 0", snap.Commands)
	}
	for _, id := range []LayerID{World, UI, Hand} {
		if r.CommandBuffer().ContainsLayer(id, snap.Frame) {
			t.Fatalf("broken layer %s recorded a shared command", id)
		}
	}

	// Repair only the world: exactly one draw command per eye must appear.
	world.Repair()
	snap2 := r.RenderFrame()
	if snap2.Commands != 2 {
		t.Fatalf("recovered world appended %d commands, want 2 (one per eye)", snap2.Commands)
	}
	if !r.CommandBuffer().ContainsLayer(World, snap2.Frame) {
		t.Fatal("recovered world must record normal draw commands again")
	}
	for _, id := range []LayerID{UI, Hand} {
		if r.CommandBuffer().ContainsLayer(id, snap2.Frame) {
			t.Fatalf("still-broken layer %s must not record commands", id)
		}
	}
}

func TestNoGhostingAfterFailureAndRecovery(t *testing.T) {
	r, _, _, hand := newTestRenderer()

	hand.Inject(FaultBadProjection)
	for range 4 {
		r.RenderFrame()
	}

	// While hidden, the hand region must be fully cleared on both eyes.
	for _, eye := range []Eye{Left, Right} {
		fb := r.Target(eye)
		for y := hand.Region.Y0; y < hand.Region.Y1; y++ {
			for x := hand.Region.X0; x < hand.Region.X1; x++ {
				p := fb.At(x, y)
				if p.Content != "" || p.Owner != "" {
					t.Fatalf("eye %s (%d,%d) retains stale hand content: %+v", eye, x, y, p)
				}
			}
		}
	}

	hand.Repair()
	snap := r.RenderFrame()
	if snap.GhostPixels != 0 {
		t.Fatalf("recovery left %d ghost pixels", snap.GhostPixels)
	}
	// Recovered hand draws normally again on the current frame.
	for _, eye := range []Eye{Left, Right} {
		p := r.Target(eye).At(hand.Region.X0, hand.Region.Y0)
		if p.Owner != Hand || p.Content != "layer" || p.Frame != snap.Frame {
			t.Fatalf("eye %s hand cell after recovery = %+v", eye, p)
		}
	}
}

func TestPlaceholderMustBeDeclared(t *testing.T) {
	const w, h = 6, 6
	// A placeholder-policy layer with no declared fallback must hide rather
	// than synthesize pixels.
	world := newLayer(World, Rect{X0: 0, Y0: 0, X1: w, Y1: h}, PolicyPlaceholder, goodResources(), "")
	r := NewRenderer(w, h, world)

	world.Inject(FaultMissingTexture)
	snap := r.RenderFrame()
	if statusOf(snap, World) != StatusHidden {
		t.Fatalf("layer without placeholder = %v, want hidden", statusOf(snap, World))
	}
	if !snap.Black {
		t.Fatal("a hidden sole layer must leave the target empty")
	}
	if len(snap.Violations) != 0 {
		t.Fatalf("hiding without a placeholder must be legal: %v", snap.Violations)
	}
}

func TestAllHiddenProducesBlackWithoutViolation(t *testing.T) {
	r, world, ui, hand := newTestRenderer()

	// Force every layer into the hidden policy path.
	world.HasPlaceholder = false
	world.Policy = PolicyHide
	ui.HasPlaceholder = false
	ui.Policy = PolicyHide

	world.Inject(FaultMissingTexture)
	ui.Inject(FaultBadDepthFormat)
	hand.Inject(FaultBadProjection)
	snap := r.RenderFrame()

	if !snap.Black {
		t.Fatal("when every layer is hidden, both eye targets must be black")
	}
	if snap.GhostPixels != 0 || len(snap.Violations) != 0 {
		t.Fatalf("clean blackout violated invariants: %v ghosts=%d", snap.Violations, snap.GhostPixels)
	}
	if r.CommandBuffer().Len() != 0 {
		t.Fatal("hidden layers must not touch the shared command buffer")
	}
}

func TestFaultClassesMapToCorrectLayers(t *testing.T) {
	r, world, ui, hand := newTestRenderer()

	world.Inject(FaultMissingTexture)
	ui.Inject(FaultBadDepthFormat)
	hand.Inject(FaultBadProjection)
	snap := r.RenderFrame()

	want := map[LayerID]Fault{World: FaultMissingTexture, UI: FaultBadDepthFormat, Hand: FaultBadProjection}
	for _, ls := range snap.Layers {
		if ls.Fault != want[ls.Layer] {
			t.Fatalf("layer %s fault = %v, want %v", ls.Layer, ls.Fault, want[ls.Layer])
		}
	}
}
