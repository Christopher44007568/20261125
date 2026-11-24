package camera

import (
	"testing"

	"handlate/internal/vec3"
)

func TestOccludedConservativeBias(t *testing.T) {
	head := vec3.Pose{Pos: vec3.V{X: 0, Y: 1.5, Z: 0}}
	panel := vec3.NewAABB(vec3.V{X: 0, Y: 1.5, Z: 1.0}, vec3.V{X: 0.3, Y: 0.3, Z: 0.02})

	// Point 0.5 cm behind the panel face is inside the bias window and hidden.
	justBehind := vec3.V{X: 0, Y: 1.5, Z: 1.025}
	if !Occluded(head, justBehind, []vec3.AABB{panel}) {
		t.Fatal("point within depth bias of an occluder must be hidden")
	}

	// Point 5 cm in front of the panel is visible.
	inFront := vec3.V{X: 0, Y: 1.5, Z: 0.93}
	if Occluded(head, inFront, []vec3.AABB{panel}) {
		t.Fatal("point clearly in front of the panel must stay visible")
	}
}

func TestRayBoxMiss(t *testing.T) {
	head := vec3.Pose{Pos: vec3.V{X: 0, Y: 1.5, Z: 0}}
	panel := vec3.NewAABB(vec3.V{X: 1.0, Y: 1.5, Z: 1.0}, vec3.V{X: 0.1, Y: 0.1, Z: 0.02})
	if _, ok := RayBox(head.Pos, vec3.V{X: 0, Y: 0, Z: 1}, panel); ok {
		t.Fatal("off-axis ray must miss the box")
	}
}
