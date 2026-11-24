package geomlife

import "testing"

func TestRunSimulationReportIsClean(t *testing.T) {
	report := RunSimulation()
	if !report.OK {
		for _, check := range report.Canonical.Checks {
			if !check.OK {
				t.Errorf("canonical check failed: %s: %s", check.Name, check.Detail)
			}
		}
		t.Fatalf("parallel report failed: %+v", report.Parallel.Invariants)
	}

	if len(report.Canonical.Snapshot.Objects) == 0 {
		t.Fatal("canonical report contains no objects")
	}
	for _, object := range report.Canonical.Snapshot.Objects {
		if object.AllocationSeq == 0 {
			t.Fatalf("object missing allocation sequence: %+v", object)
		}
		if object.LastUsedSubmissionID == 0 {
			t.Fatalf("object seq=%d missing last used submission", object.AllocationSeq)
		}
		if object.ReleaseReason == "" || object.ReleaseReason == ReasonPending {
			t.Fatalf("object seq=%d missing final release reason", object.AllocationSeq)
		}
	}
}
