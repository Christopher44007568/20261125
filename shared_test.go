package stereorollback

import (
	"errors"
	"testing"
)

func TestPlaceholderNeverCountedAsSuccess(t *testing.T) {
	p := NewPipeline(Config{Script: Script{EyeFaults: map[uint64]map[Eye]FaultType{
		1: {EyeRight: FaultMaterialParameter},
	}}})
	if err := p.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	counters := NewCounters()
	fr := p.RunFrame(1)
	counters.RecordFrame(fr)

	if counters.SuccessfulFrames != 1 {
		t.Fatalf("only the left eye render is a success, got %d", counters.SuccessfulFrames)
	}
	if counters.PlaceholderEyeFrames != 1 {
		t.Fatalf("expected one placeholder, got %d", counters.PlaceholderEyeFrames)
	}
	if counters.EyeSuccessFrames["left"] != 1 || counters.EyeSuccessFrames["right"] != 0 {
		t.Fatalf("bad per-eye success counts: %v", counters.EyeSuccessFrames)
	}
	if counters.PartialFrames != 1 || counters.FullSuccessFrames != 0 {
		t.Fatalf("expected one partial frame: %+v", counters)
	}
}

func TestDestroyRefusesWhileOtherUsersHoldRefs(t *testing.T) {
	pool := NewResourcePool()
	if _, err := pool.Create(ResourceDesc{ID: "tex.x", Kind: KindSharedTexture, Shared: true}); err != nil {
		t.Fatal(err)
	}
	left, err := pool.Acquire("tex.x", OwnerEye(EyeLeft))
	if err != nil {
		t.Fatal(err)
	}
	right, err := pool.Acquire("tex.x", OwnerEye(EyeRight))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Destroy("tex.x"); !errors.Is(err, ErrResourceBusy) {
		t.Fatalf("expected ErrResourceBusy with two holders, got %v", err)
	}
	if err := pool.Release(left); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Destroy("tex.x"); !errors.Is(err, ErrResourceBusy) {
		t.Fatalf("right still holds a ref, expected busy, got %v", err)
	}
	if err := pool.Release(right); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Destroy("tex.x"); err != nil {
		t.Fatalf("destroy after all releases should succeed, got %v", err)
	}
}

func TestDoubleReleaseLeaseIsRejected(t *testing.T) {
	pool := NewResourcePool()
	_, _ = pool.Create(ResourceDesc{ID: "tex.y", Kind: KindSharedTexture})
	lease, err := pool.Acquire("tex.y", "eye:left")
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Release(lease); err != nil {
		t.Fatal(err)
	}
	if err := pool.Release(lease); !errors.Is(err, ErrUnknownLease) {
		t.Fatalf("double release must be ErrUnknownLease, got %v", err)
	}
}

func TestSharedTextureCorruptionHandledSeparately(t *testing.T) {
	sc, err := RunSharedTextureCorruption()
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

	var incident *FrameRecord
	for i := range sc.Frames {
		if sc.Frames[i].SharedCorruption != nil {
			incident = &sc.Frames[i]
		}
	}
	if incident == nil {
		t.Fatal("missing shared corruption frame")
	}
	if incident.Failure != nil {
		t.Fatalf("shared corruption must not be reported as an eye failure: %+v", incident.Failure)
	}
	if incident.Left.Status != StatusSharedFallback || incident.Right.Status != StatusSharedFallback {
		t.Fatalf("both eyes should use shared fallback: L=%s R=%s", incident.Left.Status, incident.Right.Status)
	}
	corr := incident.SharedCorruption
	if len(corr.DroppedUsers) != 0 || len(corr.MigratedUsers) != 4 {
		t.Fatalf("users must all be retained: affected=%v migrated=%v dropped=%v",
			corr.AffectedUsers, corr.MigratedUsers, corr.DroppedUsers)
	}

	after := sc.Frames[len(sc.Frames)-1]
	if !after.FullSuccess {
		t.Fatal("pipeline should recover after shared texture restore")
	}
}

func TestRunAllScenariosClean(t *testing.T) {
	report, err := RunAll()
	if err != nil {
		t.Fatal(err)
	}
	if !report.Summary.AllInvariantsHold {
		for _, sc := range report.Scenarios {
			for _, c := range sc.InvariantChecks {
				if !c.OK {
					t.Errorf("%s/%s: %s", sc.ID, c.Name, c.Detail)
				}
			}
		}
	}
	if report.Summary.LeftFailures != 3 || report.Summary.RightFailures != 3 {
		t.Fatalf("unexpected failure totals: %+v", report.Summary)
	}
	if report.Summary.SharedCorruptions != 1 {
		t.Fatalf("expected one shared corruption, got %d", report.Summary.SharedCorruptions)
	}
}
