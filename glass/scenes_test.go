package glass

import (
	"reflect"
	"testing"
)

func TestIntersectionEyesChooseDifferentOrders(t *testing.T) {
	scene := intersectionScene()
	audit, err := AuditScene(scene, DefaultBudget(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	left := eyeAuditFor(t, audit, EyeLeft)
	right := eyeAuditFor(t, audit, EyeRight)

	if got, want := depthOrderIDs(left), []string{
		"cross-left", "near-opaque", "middle", "cross-right", "faint",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("left depth order = %v, want %v", got, want)
	}
	if got, want := depthOrderIDs(right), []string{
		"cross-right", "near-opaque", "middle", "cross-left", "faint",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("right depth order = %v, want %v", got, want)
	}
	if got, want := left.KeptIDs, []string{"cross-left", "near-opaque", "middle"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("left kept = %v, want %v", got, want)
	}
	if got, want := right.KeptIDs, []string{"cross-right", "near-opaque"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("right kept = %v, want %v", got, want)
	}
	if left.ProcessedMask != "0x19" || left.UnprocessedMask != "0x06" {
		t.Fatalf("left masks = %s/%s, want 0x19/0x06", left.ProcessedMask, left.UnprocessedMask)
	}
	if right.ProcessedMask != "0x12" || right.UnprocessedMask != "0x0d" {
		t.Fatalf("right masks = %s/%s, want 0x12/0x0d", right.ProcessedMask, right.UnprocessedMask)
	}
	if reflect.DeepEqual(left.DepthOrder, right.DepthOrder) {
		t.Fatal("intersection must be able to produce different left/right orders")
	}
}

func TestNearCoplanarUsesStableIDTieBreak(t *testing.T) {
	scene := nearCoplanarScene()
	audit, err := AuditScene(scene, DefaultBudget(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	left := eyeAuditFor(t, audit, EyeLeft)
	right := eyeAuditFor(t, audit, EyeRight)
	wantOrder := []string{
		"cp-a-near-opaque", "cp-b-faint", "cp-c", "cp-d", "cp-e",
	}
	if got := depthOrderIDs(left); !reflect.DeepEqual(got, wantOrder) {
		t.Fatalf("left order = %v, want %v", got, wantOrder)
	}
	if got := depthOrderIDs(right); !reflect.DeepEqual(got, wantOrder) {
		t.Fatalf("right order = %v, want %v", got, wantOrder)
	}
	for _, entry := range append(left.DepthOrder, right.DepthOrder...) {
		if entry.DepthBucket != left.DepthOrder[0].DepthBucket {
			t.Fatalf("near-coplanar layer escaped its depth bucket: %+v", entry)
		}
	}
	if got, want := left.KeptIDs, wantOrder[:3]; !reflect.DeepEqual(got, want) {
		t.Fatalf("left kept = %v, want %v", got, want)
	}
	if got, want := right.KeptIDs, wantOrder[:2]; !reflect.DeepEqual(got, want) {
		t.Fatalf("right kept = %v, want %v", got, want)
	}
}

func TestSharedMaterialKeepsPerLayerOrder(t *testing.T) {
	scene := sharedMaterialScene()
	audit, err := AuditScene(scene, DefaultBudget(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	left := eyeAuditFor(t, audit, EyeLeft)
	right := eyeAuditFor(t, audit, EyeRight)
	if got, want := depthOrderIDs(left), []string{
		"shared-a", "shared-b", "shared-c", "shared-d", "shared-e",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("left order = %v, want %v", got, want)
	}
	if got, want := depthOrderIDs(right), []string{
		"shared-b", "shared-a", "shared-c", "shared-d", "shared-e",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("right order = %v, want %v", got, want)
	}
	if got, want := left.KeptIDs, []string{"shared-a", "shared-b", "shared-c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("left kept = %v, want %v", got, want)
	}
	if got, want := right.KeptIDs, []string{"shared-b", "shared-a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("right kept = %v, want %v", got, want)
	}
	for _, eye := range audit.Eyes {
		for _, entry := range eye.DepthOrder {
			if entry.MaterialID != "frosted" {
				t.Fatalf("shared material was lost: %+v", entry)
			}
		}
	}
}

func TestAllBuiltinScenesExposeReviewableEvidence(t *testing.T) {
	for _, scene := range BuiltinScenes() {
		t.Run(string(scene.Kind), func(t *testing.T) {
			audit, err := AuditScene(scene, DefaultBudget(), Options{})
			if err != nil {
				t.Fatal(err)
			}
			if err := audit.ValidateEvidence(); err != nil {
				t.Fatal(err)
			}
			if len(audit.MaskBitOrder) != len(scene.Layers) {
				t.Fatalf("mask has %d bits for %d layers", len(audit.MaskBitOrder), len(scene.Layers))
			}
			for _, eye := range audit.Eyes {
				if len(eye.DepthOrder) != len(scene.Layers) {
					t.Fatalf("%s eye has %d order entries, want %d", eye.Eye, len(eye.DepthOrder), len(scene.Layers))
				}
				kept := make(map[string]bool, len(eye.KeptIDs))
				for _, id := range eye.KeptIDs {
					kept[id] = true
				}
				for _, entry := range eye.DepthOrder {
					wantDecision := DecisionOverBudget
					if kept[entry.ID] {
						wantDecision = DecisionKept
					}
					if entry.Decision != wantDecision {
						t.Fatalf("%s %s decision = %q, want %q", eye.Eye, entry.ID, entry.Decision, wantDecision)
					}
				}
			}
		})
	}
}

func depthOrderIDs(audit EyeAudit) []string {
	ids := make([]string, 0, len(audit.DepthOrder))
	for _, entry := range audit.DepthOrder {
		ids = append(ids, entry.ID)
	}
	return ids
}
