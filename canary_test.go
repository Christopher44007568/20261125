package geomlife

import "testing"

func TestGeometryCanaryDistinguishesUnsafeReuse(t *testing.T) {
	report := RunGeometryCanary()
	if !report.OK {
		t.Fatalf("canary report not clean: %+v", report)
	}
	if !report.UnsafeDrawCorrupted {
		t.Fatal("offset-only unsafe reuse should corrupt the old vertex pattern")
	}
	if !report.SafeDrawIntact {
		t.Fatal("safe arena should preserve the submitted geometry pattern")
	}
}
