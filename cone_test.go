package normalcone

import (
	"math"
	"testing"
)

func TestReferenceCones(t *testing.T) {
	scenarios := BuildScenarios(256)
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if scenario.Degenerate {
				if scenario.Reference.Valid {
					t.Fatal("degenerate normal set produced a valid cone")
				}
				return
			}

			if !scenario.Reference.Valid {
				t.Fatalf("reference cone is invalid: %s", scenario.Reference.Reason)
			}
			for _, normal := range scenario.Normals {
				angle := math.Acos(clamp(scenario.Reference.Axis.Dot(normal), -1, 1))
				if angle > scenario.Reference.HalfAngle+1e-12 {
					t.Fatalf("normal angle %.12f exceeds half-angle %.12f", angle, scenario.Reference.HalfAngle)
				}
			}
		})
	}
}

func TestConservativeHasNoFalseCulls(t *testing.T) {
	for _, report := range Evaluate(BuildScenarios(1024)) {
		if report.Conservative.FalseCulls != 0 {
			t.Fatalf("%s: conservative culled %d visible samples", report.Scenario.Name, report.Conservative.FalseCulls)
		}
	}
}

func TestBoundaryIsDeterministicAndRetained(t *testing.T) {
	for _, scenario := range BuildScenarios(256) {
		for _, camera := range scenario.BoundaryViews {
			if conservativeDecision(scenario, camera) {
				t.Fatalf("%s: boundary camera was culled", scenario.Name)
			}
		}

		stats := evaluatePolicy(scenario, conservativeDecision)
		if !stats.BoundaryStable {
			t.Fatalf("%s: boundary decisions were not stable: %v", scenario.Name, stats.BoundaryDecisions)
		}
	}
}

func TestExactConservativeBoundaryRetains(t *testing.T) {
	axis := Vec3{X: 0, Y: 0, Z: 1}
	halfAngle := math.Pi / 6
	cone := Cone{Axis: axis, HalfAngle: halfAngle, Valid: true}
	edgeView := math.Cos(deg90 + halfAngle)
	camera := Vec3{X: math.Sin(deg90 + halfAngle), Y: 0, Z: edgeView}

	if ConservativeCull(cone, Vec3{}, camera) {
		t.Fatal("camera exactly on the cone boundary must be retained")
	}

	inside := camera.Add(Vec3{Z: 1e-10})
	if ConservativeCull(cone, Vec3{}, inside) {
		t.Fatal("camera just inside the boundary must be retained")
	}

	outside := camera.Add(Vec3{Z: -1e-10})
	if !ConservativeCull(cone, Vec3{}, outside) {
		t.Fatal("camera strictly beyond the boundary must be culled")
	}
}

func TestFullSphereConeNeverCulls(t *testing.T) {
	cone := Cone{HalfAngle: math.Pi, Valid: true}
	if ConservativeCull(cone, Vec3{}, Vec3{X: 1, Y: 2, Z: 3}) {
		t.Fatal("full-space cone must never cull")
	}
}

func TestOppositeNormalsProduceFullSphereCone(t *testing.T) {
	cone := ReferenceCone([]Vec3{{X: 0, Y: 0, Z: 1}, {X: 0, Y: 0, Z: -1}})
	if !cone.Valid {
		t.Fatal("valid opposite normals should produce a valid full-space cone")
	}
	if cone.HalfAngle != math.Pi || cone.Axis != (Vec3{}) {
		t.Fatalf("unexpected full-space cone: %+v", cone)
	}
}

func TestMirroredNormalOrientation(t *testing.T) {
	mirror := Reflection(Vec3{X: 1, Y: 0, Z: 0})
	if !IsReflection(mirror) {
		t.Fatal("reflection matrix was not detected as orientation reversing")
	}

	original := Vec3{X: 0.5, Y: 0, Z: math.Sqrt(3) / 2}
	transformed, ok := mirror.TransformNormal(original)
	if !ok {
		t.Fatal("normal transform failed")
	}
	expected := Vec3{X: 0.5, Y: 0, Z: -math.Sqrt(3) / 2}
	assertVec(t, transformed, expected, 1e-12)
}

func TestWrongAxisCausesFalseCullOnCurvedPatch(t *testing.T) {
	scenario := curvedScenario(1024)
	stats := evaluatePolicy(scenario, wrongAxisDecision)
	if stats.FalseCulls == 0 {
		t.Fatal("expected wrong-axis aggressive culling to remove a visible sample")
	}
	if stats.FalseCullRate < 0.2 || stats.FalseCullRate > 0.8 {
		t.Fatalf("unexpected wrong-axis false-kill rate: %f", stats.FalseCullRate)
	}
}

func TestDegenerateConeFailsClosed(t *testing.T) {
	cone := ReferenceCone([]Vec3{{X: 0, Y: 0, Z: 1}, {}})
	if cone.Valid {
		t.Fatal("zero normal must invalidate the reference cone")
	}
	if ConservativeCull(cone, Vec3{}, Vec3{Z: 1}) {
		t.Fatal("invalid cone must not cull")
	}
}

func TestAggressiveFirstNormalFalseCullsDegeneratePatch(t *testing.T) {
	scenario := degenerateScenario(1024)
	stats := evaluatePolicy(scenario, firstNormalDecision)
	if stats.FalseCulls == 0 {
		t.Fatal("expected aggressive first-normal culling to remove visible samples")
	}
	if math.Abs(stats.FalseCullRate-0.5) > 0.02 {
		t.Fatalf("unexpected degenerate false-kill rate: %f", stats.FalseCullRate)
	}
}

func assertVec(t *testing.T, got, want Vec3, tolerance float64) {
	t.Helper()
	if math.Abs(got.X-want.X) > tolerance || math.Abs(got.Y-want.Y) > tolerance || math.Abs(got.Z-want.Z) > tolerance {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
