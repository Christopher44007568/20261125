package main

import "testing"

func TestEyeSnapshotCloneIsDeep(t *testing.T) {
	scene, err := newScene()
	if err != nil {
		t.Fatal(err)
	}
	frames, err := defaultEyeStates(scene)
	if err != nil {
		t.Fatal(err)
	}
	left := frames[0]
	before := left.SnapshotHash()
	clone := left.Clone()

	clone.Camera.Position.X = 99
	clone.Jitter.X = 99
	clone.Constants["fog"] = 99
	clone.Materials[0].Roughness = 99
	clone.Materials[0].Constants["gain"] = 99
	*clone.Materials[0].AlphaOverride = 0

	if got := left.SnapshotHash(); got != before {
		t.Fatalf("clone mutated source snapshot: before=%s after=%s", before, got)
	}
	if left.Camera.Position.X == 99 || left.Materials[0].Roughness == 99 {
		t.Fatal("scalar fields were not copied")
	}
	if left.Materials[0].Constants["gain"] == 99 || left.Constants["fog"] == 99 {
		t.Fatal("constants maps were not deep-copied")
	}
	if left.Materials[0].AlphaOverride == clone.Materials[0].AlphaOverride {
		t.Fatal("alpha override pointers must not be shared")
	}
}

func TestSameMaterialHasPerEyeDynamicState(t *testing.T) {
	scene, err := newScene()
	if err != nil {
		t.Fatal(err)
	}
	frames, err := defaultEyeStates(scene)
	if err != nil {
		t.Fatal(err)
	}
	left, right, err := orderedFrames(frames)
	if err != nil {
		t.Fatal(err)
	}
	distinct, alphaDistinct, constantsDistinct := compareMaterials(left, right)
	if !distinct || !alphaDistinct || !constantsDistinct {
		t.Fatalf("dynamic material state is not distinct: material=%t alpha=%t constants=%t",
			distinct, alphaDistinct, constantsDistinct)
	}
	for _, material := range left.Materials {
		other, ok := findMaterial(right, material.Instance)
		if !ok {
			t.Fatalf("right eye missing %q", material.Instance)
		}
		if material.BaseMaterial != other.BaseMaterial {
			t.Fatalf("%q does not share a base material", material.Instance)
		}
		if material.Roughness == other.Roughness {
			t.Fatalf("%q roughness was shared across eyes", material.Instance)
		}
		if floatPtrEqual(material.AlphaOverride, other.AlphaOverride) {
			t.Fatalf("%q alpha override was shared across eyes", material.Instance)
		}
		if hashJSON(material.Constants) == hashJSON(other.Constants) {
			t.Fatalf("%q constants were shared across eyes", material.Instance)
		}
	}
	if hashJSON(left.Camera) == hashJSON(right.Camera) {
		t.Fatal("cameras must be per-eye")
	}
	if hashJSON(left.Constants) == hashJSON(right.Constants) {
		t.Fatal("eye-level constants must be per-eye")
	}
}

func TestPreparedCacheKeysShareProgramButNotDynamics(t *testing.T) {
	scene, err := newScene()
	if err != nil {
		t.Fatal(err)
	}
	frames, err := defaultEyeStates(scene)
	if err != nil {
		t.Fatal(err)
	}
	left, right, err := orderedFrames(frames)
	if err != nil {
		t.Fatal(err)
	}
	leftPrepared, err := prepareDraw(scene, left, left.Materials[0])
	if err != nil {
		t.Fatal(err)
	}
	rightPrepared, err := prepareDraw(scene, right, right.Materials[0])
	if err != nil {
		t.Fatal(err)
	}
	if leftPrepared.ProgramID != rightPrepared.ProgramID ||
		leftPrepared.LayoutHash != rightPrepared.LayoutHash {
		t.Fatal("program or static layout was duplicated")
	}
	if leftPrepared.DynamicHash == rightPrepared.DynamicHash {
		t.Fatal("dynamic fingerprints must differ")
	}
	if correctCacheKey(leftPrepared) == correctCacheKey(rightPrepared) {
		t.Fatal("correct cache key aliases left and right eye")
	}
	if faultCacheKey(leftPrepared) != faultCacheKey(rightPrepared) {
		t.Fatal("fault probe should demonstrate a colliding base-material key")
	}
}
