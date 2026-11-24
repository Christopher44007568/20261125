package glass

import (
	"math"
	"reflect"
	"testing"
)

func TestPremultipliedCompositeAndDepthWrites(t *testing.T) {
	scene := Scene{
		Kind: SceneIntersection,
		Materials: []Material{
			{ID: "red", Color: RGB{R: 1}},
			{ID: "blue", Color: RGB{B: 1}},
		},
		Layers: []Layer{
			{
				ID:         "far",
				MaterialID: "red",
				Depth:      map[Eye]float64{EyeLeft: 5, EyeRight: 5},
				Opacity:    0.5,
			},
			{
				ID:         "near",
				MaterialID: "blue",
				Depth:      map[Eye]float64{EyeLeft: 1, EyeRight: 1},
				Opacity:    0.5,
			},
		},
	}

	audit, err := AuditScene(scene, Budget{Left: 2, Right: 2}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := audit.ValidateEvidence(); err != nil {
		t.Fatal(err)
	}

	left := eyeAuditFor(t, audit, EyeLeft)
	if got, want := left.KeptIDs, []string{"near", "far"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("kept order = %v, want %v", got, want)
	}
	if got, want := left.RenderOrder, []string{"far", "near"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("render order = %v, want %v", got, want)
	}
	if got, want := left.Premultiplied, (PremultipliedRGBA{R: 0.25, B: 0.5, A: 0.75}); !nearRGBA(got, want, 1e-15) {
		t.Fatalf("premultiplied result = %+v, want %+v", got, want)
	}
	if left.DepthWriteEnabled {
		t.Fatal("transparent pass must not enable depth writes")
	}
	if len(left.DepthWrites) != 2 {
		t.Fatalf("depth-write records = %d, want 2", len(left.DepthWrites))
	}
	for _, entry := range left.DepthWrites {
		if entry.Enabled || entry.Reason != DepthWriteReason {
			t.Fatalf("unexpected depth-write record: %+v", entry)
		}
	}
}

func TestNearOpaqueAndExtremelyLowOpacity(t *testing.T) {
	base := Scene{
		Kind: SceneIntersection,
		Materials: []Material{
			{ID: "red", Color: RGB{R: 1}},
			{ID: "green", Color: RGB{G: 1}},
		},
	}

	faintScene := base
	faintScene.Layers = []Layer{{
		ID:         "faint",
		MaterialID: "red",
		Depth:      map[Eye]float64{EyeLeft: 1, EyeRight: 1},
		Opacity:    extremelyFaint,
	}}
	faintAudit, err := AuditScene(faintScene, Budget{Left: 1, Right: 1}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	faint := eyeAuditFor(t, faintAudit, EyeLeft)
	if !(faint.Premultiplied.A > 0) {
		t.Fatal("extremely low opacity was incorrectly culled")
	}
	if diff := math.Abs(faint.Premultiplied.A - extremelyFaint); diff > 1e-24 {
		t.Fatalf("faint alpha = %.18g, want %.18g", faint.Premultiplied.A, extremelyFaint)
	}

	opaqueScene := base
	opaqueScene.Layers = []Layer{
		{
			ID:         "faint",
			MaterialID: "green",
			Depth:      map[Eye]float64{EyeLeft: 5, EyeRight: 5},
			Opacity:    extremelyFaint,
		},
		{
			ID:         "near-opaque",
			MaterialID: "red",
			Depth:      map[Eye]float64{EyeLeft: 1, EyeRight: 1},
			Opacity:    nearOpaque,
		},
	}
	opaqueAudit, err := AuditScene(opaqueScene, Budget{Left: 2, Right: 2}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	opaque := eyeAuditFor(t, opaqueAudit, EyeLeft)
	if diff := math.Abs(1 - opaque.Premultiplied.A); diff > 1e-8 {
		t.Fatalf("near-opaque alpha = %.18g, want approximately 1", opaque.Premultiplied.A)
	}
	if opaque.Premultiplied.G > 1e-18 {
		t.Fatalf("near-opaque layer did not suppress the faint layer: %+v", opaque.Premultiplied)
	}
	for _, entry := range opaque.DepthWrites {
		if entry.Enabled {
			t.Fatalf("near-opaque glass enabled depth writes: %+v", entry)
		}
	}
}

func TestBudgetInsufficientAndZero(t *testing.T) {
	scene := Scene{
		Kind:      SceneSharedMaterial,
		Materials: []Material{{ID: "mat", Color: RGB{R: 0.4, G: 0.6, B: 0.8}}},
		Layers: []Layer{
			{ID: "a", MaterialID: "mat", Depth: map[Eye]float64{EyeLeft: 1, EyeRight: 1}, Opacity: 0.2},
			{ID: "b", MaterialID: "mat", Depth: map[Eye]float64{EyeLeft: 2, EyeRight: 2}, Opacity: 0.2},
			{ID: "c", MaterialID: "mat", Depth: map[Eye]float64{EyeLeft: 3, EyeRight: 3}, Opacity: 0.2},
			{ID: "d", MaterialID: "mat", Depth: map[Eye]float64{EyeLeft: 4, EyeRight: 4}, Opacity: 0.2},
		},
	}

	audit, err := AuditScene(scene, Budget{Left: 1, Right: 0}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	left := eyeAuditFor(t, audit, EyeLeft)
	right := eyeAuditFor(t, audit, EyeRight)
	if got, want := left.KeptIDs, []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("left kept = %v, want %v", got, want)
	}
	if got, want := left.UnprocessedIDs, []string{"b", "c", "d"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("left unprocessed = %v, want %v", got, want)
	}
	if left.ProcessedMask != "0x1" || left.UnprocessedMask != "0xe" {
		t.Fatalf("left masks = %s/%s, want 0x1/0xe", left.ProcessedMask, left.UnprocessedMask)
	}
	if len(right.KeptIDs) != 0 || right.ProcessedMask != "0x0" || right.UnprocessedMask != "0xf" {
		t.Fatalf("zero budget was not represented exactly: %+v", right)
	}
	if !nearRGBA(right.Premultiplied, PremultipliedRGBA{}, 0) {
		t.Fatalf("zero budget result = %+v, want transparent background", right.Premultiplied)
	}

	unlimited, err := AuditScene(scene, Budget{Left: 99, Right: 99}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	left = eyeAuditFor(t, unlimited, EyeLeft)
	if len(left.UnprocessedIDs) != 0 || left.ProcessedMask != "0xf" {
		t.Fatalf("over-budget selection should keep all layers: %+v", left)
	}
}

func TestMaskBitOrderUsesSortedIDs(t *testing.T) {
	scene := Scene{
		Kind:      SceneNearCoplanar,
		Materials: []Material{{ID: "mat", Color: RGB{G: 1}}},
		Layers: []Layer{
			{ID: "zeta", MaterialID: "mat", Depth: map[Eye]float64{EyeLeft: 1, EyeRight: 1}, Opacity: 0.5},
			{ID: "alpha", MaterialID: "mat", Depth: map[Eye]float64{EyeLeft: 2, EyeRight: 2}, Opacity: 0.5},
			{ID: "mu", MaterialID: "mat", Depth: map[Eye]float64{EyeLeft: 3, EyeRight: 3}, Opacity: 0.5},
		},
	}
	audit, err := AuditScene(scene, Budget{Left: 1, Right: 1}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := audit.MaskBitOrder, []string{"alpha", "mu", "zeta"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mask bit order = %v, want %v", got, want)
	}
	left := eyeAuditFor(t, audit, EyeLeft)
	if left.ProcessedMask != "0x4" || left.UnprocessedMask != "0x3" {
		t.Fatalf("masks = %s/%s, want 0x4/0x3", left.ProcessedMask, left.UnprocessedMask)
	}
}

func TestDepthBucketProvidesTotalOrder(t *testing.T) {
	scene := Scene{
		Kind:      SceneNearCoplanar,
		Materials: []Material{{ID: "mat", Color: RGB{B: 1}}},
		Layers: []Layer{
			{ID: "b", MaterialID: "mat", Depth: map[Eye]float64{EyeLeft: 2.0, EyeRight: 2.0}, Opacity: 0.4},
			{ID: "a", MaterialID: "mat", Depth: map[Eye]float64{EyeLeft: 2.0000004, EyeRight: 2.0000004}, Opacity: 0.4},
		},
	}
	audit, err := AuditScene(scene, Budget{Left: 1, Right: 1}, Options{DepthEpsilon: DefaultDepthEpsilon})
	if err != nil {
		t.Fatal(err)
	}
	left := eyeAuditFor(t, audit, EyeLeft)
	if got, want := left.KeptIDs, []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("near-coplanar tie break = %v, want %v", got, want)
	}
	if left.DepthOrder[0].DepthBucket != left.DepthOrder[1].DepthBucket {
		t.Fatalf("expected one depth bucket, got %+v", left.DepthOrder)
	}
}

func TestPermutationsProduceStableEvidence(t *testing.T) {
	for _, scene := range BuiltinScenes() {
		t.Run(string(scene.Kind), func(t *testing.T) {
			base, err := AuditScene(scene, DefaultBudget(), Options{})
			if err != nil {
				t.Fatal(err)
			}
			for _, submission := range layerPermutations(scene.Layers) {
				shuffled := scene
				shuffled.Layers = submission
				got, err := AuditScene(shuffled, DefaultBudget(), Options{})
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(base.Eyes, got.Eyes) {
					t.Fatalf("eye evidence changed for submission %v", layerIDs(submission))
				}
				if !reflect.DeepEqual(base.MaskBitOrder, got.MaskBitOrder) {
					t.Fatalf("mask bit order changed for submission %v", layerIDs(submission))
				}
			}
		})
	}
}

func eyeAuditFor(t *testing.T, audit Audit, eye Eye) EyeAudit {
	t.Helper()
	for _, candidate := range audit.Eyes {
		if candidate.Eye == eye {
			return candidate
		}
	}
	t.Fatalf("missing %s-eye audit", eye)
	return EyeAudit{}
}

func nearRGBA(left, right PremultipliedRGBA, tolerance float64) bool {
	return math.Abs(left.R-right.R) <= tolerance &&
		math.Abs(left.G-right.G) <= tolerance &&
		math.Abs(left.B-right.B) <= tolerance &&
		math.Abs(left.A-right.A) <= tolerance
}

func layerPermutations(layers []Layer) [][]Layer {
	var result [][]Layer
	var generate func(int)
	generate = func(index int) {
		if index == len(layers) {
			result = append(result, append([]Layer(nil), layers...))
			return
		}
		for swap := index; swap < len(layers); swap++ {
			layers[index], layers[swap] = layers[swap], layers[index]
			generate(index + 1)
			layers[index], layers[swap] = layers[swap], layers[index]
		}
	}
	generate(0)
	return result
}
