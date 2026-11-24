package stereorollback

import (
	"context"
	"errors"
	"testing"
)

func TestLeftEyeFailurePreservesRightEye(t *testing.T) {
	sc, err := RunLeftEyeFailure()
	if err != nil {
		t.Fatal(err)
	}
	if !sc.Clean {
		for _, c := range sc.InvariantChecks {
			if !c.OK {
				t.Errorf("invariant %s failed: %s", c.Name, c.Detail)
			}
		}
	}

	badFrames := 0
	for _, fr := range sc.Frames {
		if fr.Failure == nil {
			continue
		}
		badFrames++
		if fr.Failure.Eye != EyeLeft || fr.Failure.Scope != ScopeLeftEye {
			t.Fatalf("failure not scoped to left eye: %+v", fr.Failure)
		}
		if fr.Right.Status != StatusRendered || !fr.Right.Success {
			t.Fatalf("right eye not preserved on frame %d: %+v", fr.Index, fr.Right)
		}
		if fr.Left.Status != StatusPlaceholder || !fr.Left.Placeholder || fr.Left.Success {
			t.Fatalf("left eye must be a non-success placeholder on frame %d: %+v", fr.Index, fr.Left)
		}
		if fr.FallbackFrame == nil || fr.FallbackFrame.Right.Marker != fr.Right.Marker {
			t.Fatalf("fallback frame does not carry preserved right surface on frame %d", fr.Index)
		}
	}
	if badFrames != 3 {
		t.Fatalf("expected 3 left-eye failure frames, got %d", badFrames)
	}
}

func TestRightEyeFailurePreservesLeftEye(t *testing.T) {
	sc, err := RunRightEyeFailure()
	if err != nil {
		t.Fatal(err)
	}
	if !sc.Clean {
		for _, c := range sc.InvariantChecks {
			if !c.OK {
				t.Errorf("invariant %s failed: %s", c.Name, c.Detail)
			}
		}
	}
	for _, fr := range sc.Frames {
		if fr.Failure == nil {
			continue
		}
		if fr.Failure.Eye != EyeRight || fr.Failure.Scope != ScopeRightEye {
			t.Fatalf("failure not scoped to right eye: %+v", fr.Failure)
		}
		if fr.Left.Status != StatusRendered || !fr.Left.Success {
			t.Fatalf("left eye not preserved on frame %d: %+v", fr.Index, fr.Left)
		}
	}
}

func TestInjectedFaultsRecordCorrectStages(t *testing.T) {
	sc, err := RunLeftEyeFailure()
	if err != nil {
		t.Fatal(err)
	}
	want := map[uint64]struct {
		stage Stage
		fault FaultType
	}{
		2: {StageAttachment, FaultAttachmentCreate},
		3: {StageMaterial, FaultMaterialParameter},
		4: {StageReprojection, FaultReprojectionTimeout},
	}
	for _, fr := range sc.Frames {
		w, ok := want[fr.Index]
		if !ok {
			if fr.Failure != nil {
				t.Fatalf("unexpected failure on frame %d", fr.Index)
			}
			continue
		}
		if fr.Failure == nil {
			t.Fatalf("expected failure on frame %d", fr.Index)
		}
		if fr.Failure.FailureStage != w.stage || fr.Failure.Fault != w.fault {
			t.Fatalf("frame %d got stage=%s fault=%s, want %s/%s",
				fr.Index, fr.Failure.FailureStage, fr.Failure.Fault, w.stage, w.fault)
		}
	}
}

func TestReprojectionTimeoutWrapsDeadlineExceeded(t *testing.T) {
	sc, err := RunLeftEyeFailure()
	if err != nil {
		t.Fatal(err)
	}
	var timeout *FailureRecord
	for _, fr := range sc.Frames {
		if fr.Failure != nil && fr.Failure.Fault == FaultReprojectionTimeout {
			timeout = fr.Failure
		}
	}
	if timeout == nil || timeout.Rollback == nil {
		t.Fatal("missing reprojection timeout record")
	}
	if timeout.ElapsedMS <= timeout.BudgetMS {
		t.Fatalf("expected over-budget timing, got %dms vs budget %dms", timeout.ElapsedMS, timeout.BudgetMS)
	}
	if !errors.Is(ErrReprojectionTimeout, context.DeadlineExceeded) {
		t.Fatal("timeout sentinel must wrap context.DeadlineExceeded")
	}
}
