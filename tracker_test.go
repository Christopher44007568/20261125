package texturelife

import (
	"errors"
	"testing"
)

func mustCreate(t *testing.T, tr *Tracker, frame Frame, desc Desc) TextureID {
	t.Helper()
	id, err := tr.Create(frame, desc)
	if err != nil {
		t.Fatalf("Create(%v, %+v): %v", frame, desc, err)
	}
	return id
}

func mustAcquire(t *testing.T, tr *Tracker, frame Frame, id TextureID) {
	t.Helper()
	if err := tr.Acquire(frame, id); err != nil {
		t.Fatalf("Acquire(%d, %d): %v", frame, id, err)
	}
}

func mustRetire(t *testing.T, tr *Tracker, frame Frame, id TextureID) {
	t.Helper()
	if err := tr.Retire(frame, id); err != nil {
		t.Fatalf("Retire(%d, %d): %v", frame, id, err)
	}
}

func TestDeferredReleaseKeepsNewFrameReferenceAlive(t *testing.T) {
	tr := NewTracker()
	id := mustCreate(t, tr, 1, Desc{Kind: KindColor, Eye: EyeLeft, Size: Size{Width: 64, Height: 32}})
	mustAcquire(t, tr, 1, id)
	mustRetire(t, tr, 2, id)
	mustAcquire(t, tr, 3, id)

	if err := tr.ReleaseAt(1, 3, id); err != nil {
		t.Fatalf("Release old frame: %v", err)
	}
	texture, ok := tr.Texture(id)
	if !ok {
		t.Fatal("texture metadata disappeared")
	}
	if !texture.Alive() {
		t.Fatalf("texture released while frame 3 still references it: %+v", texture)
	}
	if texture.RefCount != 1 || texture.PeakRefCount != 2 {
		t.Fatalf("refs = %d, peak refs = %d; want 1, 2", texture.RefCount, texture.PeakRefCount)
	}
	if texture.RetireFrame != 2 || texture.LastUsedFrame != 3 {
		t.Fatalf("retire/use frame = %d/%d; want 2/3", texture.RetireFrame, texture.LastUsedFrame)
	}

	if err := tr.ReleaseAt(3, 3, id); err != nil {
		t.Fatalf("Release new frame: %v", err)
	}
	texture, _ = tr.Texture(id)
	if texture.Alive() || texture.ReleasedAt != 3 {
		t.Fatalf("texture state after final release = %+v; want released at 3", texture)
	}
	if texture.RefCount != 0 {
		t.Fatalf("released texture refs = %d; want 0", texture.RefCount)
	}

	report := tr.FinalAudit()
	if !report.Healthy() {
		t.Fatalf("completed lifecycle is unhealthy: %+v", report.Issues)
	}
	if report.LiveCount != 0 || report.LiveBytes != 0 {
		t.Fatalf("live after release = %d/%d; want 0/0", report.LiveCount, report.LiveBytes)
	}
}

func TestTextureMetadataIsRetainedAfterRelease(t *testing.T) {
	tr := NewTracker()
	size := Size{Width: 320, Height: 180}
	id := mustCreate(t, tr, 4, Desc{Kind: KindDepth, Eye: EyeRight, Size: size})
	mustAcquire(t, tr, 4, id)
	mustRetire(t, tr, 5, id)
	if err := tr.ReleaseAt(4, 5, id); err != nil {
		t.Fatalf("Release: %v", err)
	}

	texture, ok := tr.Texture(id)
	if !ok {
		t.Fatal("released texture metadata was discarded")
	}
	if texture.Kind != KindDepth || texture.Eye != EyeRight || texture.Size != size {
		t.Fatalf("identity metadata changed: %+v", texture)
	}
	if texture.Bytes != size.Bytes(KindDepth) || texture.Bytes != 230400 {
		t.Fatalf("bytes = %d; want 230400", texture.Bytes)
	}
	if texture.CreatedFrame != 4 || texture.LastUsedFrame != 4 || texture.RetireFrame != 5 || texture.ReleasedAt != 5 {
		t.Fatalf("timestamps changed: %+v", texture)
	}
}

func TestPeaksTrackBytesEyesAndKinds(t *testing.T) {
	tr := NewTracker()
	size := Size{Width: 100, Height: 100}
	color := mustCreate(t, tr, 1, Desc{Kind: KindColor, Eye: EyeLeft, Size: size})
	depth := mustCreate(t, tr, 1, Desc{Kind: KindDepth, Eye: EyeRight, Size: size})
	mustAcquire(t, tr, 1, color)
	mustAcquire(t, tr, 1, depth)

	history := mustCreate(t, tr, 2, Desc{Kind: KindHistory, Eye: EyeLeft, Size: size})
	tr.ReleaseFrame(1)
	mustRetire(t, tr, 2, color)
	mustRetire(t, tr, 2, depth)
	mustAcquire(t, tr, 2, history)

	report := tr.Snapshot()
	if report.Peak.Count != 3 || report.Peak.Bytes != 120000 || report.Peak.CountFrame != 2 {
		t.Fatalf("peak = %+v; want count=3 bytes=120000 frame=2", report.Peak)
	}
	if report.Peak.RefCount != 2 || report.Peak.RefFrame != 1 {
		t.Fatalf("peak refs = %d at frame %d; want 2 at frame 1", report.Peak.RefCount, report.Peak.RefFrame)
	}
	if got := report.PeakByEye[EyeLeft]; got.Count != 2 || got.Bytes != 80000 {
		t.Fatalf("left peak = %+v; want count=2 bytes=80000", got)
	}
	if got := report.PeakByEye[EyeRight]; got.Count != 1 || got.Bytes != 40000 {
		t.Fatalf("right peak = %+v; want count=1 bytes=40000", got)
	}
	for _, kind := range []Kind{KindColor, KindDepth, KindHistory} {
		if got := report.PeakByKind[kind]; got.Count != 1 || got.Bytes != 40000 {
			t.Fatalf("%s peak = %+v; want count=1 bytes=40000", kind, got)
		}
	}
	if len(report.Dangling) != 0 {
		t.Fatalf("unexpected dangling references: %+v", report.Dangling)
	}

	if err := tr.ReleaseAt(2, 2, history); err != nil {
		t.Fatalf("Release history: %v", err)
	}
	mustRetire(t, tr, 3, history)
	if report := tr.FinalAudit(); !report.Healthy() {
		t.Fatalf("final audit is unhealthy: %+v", report.Issues)
	}
}

func TestFinalAuditReportsLeaksAndInvalidOperations(t *testing.T) {
	tr := NewTracker()
	id := mustCreate(t, tr, 1, Desc{Kind: KindColor, Eye: EyeLeft, Size: Size{Width: 10, Height: 10}})

	report := tr.FinalAudit()
	if report.Healthy() || len(report.Leaks) != 1 || report.Leaks[0].ID != id {
		t.Fatalf("leak not reported: %+v", report)
	}
	foundLeakIssue := false
	for _, issue := range report.Issues {
		if issue.Code == IssueLeak && issue.TextureID == id {
			foundLeakIssue = true
		}
	}
	if !foundLeakIssue {
		t.Fatalf("leak issue missing: %+v", report.Issues)
	}
	if err := tr.Release(1, id); !errors.Is(err, ErrNotReferenced) {
		t.Fatalf("Release without reference error = %v; want ErrNotReferenced", err)
	}
	if err := tr.Acquire(1, id+100); !errors.Is(err, ErrTextureAbsent) {
		t.Fatalf("Acquire missing error = %v; want ErrTextureAbsent", err)
	}
	if dangling := tr.Snapshot().Dangling; len(dangling) != 0 {
		t.Fatalf("invalid operations created dangling refs: %+v", dangling)
	}
}

func TestValidationAndReleaseFrame(t *testing.T) {
	tr := NewTracker()
	if _, err := tr.Create(0, Desc{Kind: KindColor, Eye: EyeLeft, Size: Size{Width: 1, Height: 1}}); !errors.Is(err, ErrInvalidFrame) {
		t.Fatalf("zero frame error = %v", err)
	}
	if _, err := tr.Create(1, Desc{Kind: KindColor, Eye: 0, Size: Size{Width: 1, Height: 1}}); !errors.Is(err, ErrInvalidEye) {
		t.Fatalf("bad eye error = %v", err)
	}
	if _, err := tr.Create(1, Desc{Kind: 0, Eye: EyeLeft, Size: Size{Width: 1, Height: 1}}); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("bad kind error = %v", err)
	}
	if _, err := tr.Create(1, Desc{Kind: KindColor, Eye: EyeLeft, Size: Size{}}); !errors.Is(err, ErrInvalidSize) {
		t.Fatalf("bad size error = %v", err)
	}

	id := mustCreate(t, tr, 1, Desc{Kind: KindHistory, Eye: EyeLeft, Size: Size{Width: 8, Height: 8}})
	mustAcquire(t, tr, 1, id)
	mustAcquire(t, tr, 1, id)
	tr.ReleaseFrame(1)
	tr.ReleaseFrame(1)
	if texture, _ := tr.Texture(id); texture.RefCount != 0 {
		t.Fatalf("ReleaseFrame left refs = %d; want 0", texture.RefCount)
	}
	mustRetire(t, tr, 2, id)
	if report := tr.FinalAudit(); !report.Healthy() {
		t.Fatalf("ReleaseFrame cleanup unhealthy: %+v", report.Issues)
	}
}

func TestReleaseFrameAtRecordsEventTime(t *testing.T) {
	tr := NewTracker()
	id := mustCreate(t, tr, 2, Desc{Kind: KindHistory, Eye: EyeRight, Size: Size{Width: 16, Height: 16}})
	mustAcquire(t, tr, 2, id)
	mustRetire(t, tr, 3, id)
	tr.ReleaseFrameAt(2, 7)

	texture, _ := tr.Texture(id)
	if texture.Alive() || texture.ReleasedAt != 7 {
		t.Fatalf("release event time = %d alive=%v; want 7/false", texture.ReleasedAt, texture.Alive())
	}
	if report := tr.FinalAudit(); !report.Healthy() {
		t.Fatalf("release event audit unhealthy: %+v", report.Issues)
	}
}

func TestAuditDetectsDanglingReference(t *testing.T) {
	tr := NewTracker()
	id := mustCreate(t, tr, 1, Desc{Kind: KindColor, Eye: EyeLeft, Size: Size{Width: 4, Height: 4}})
	mustAcquire(t, tr, 1, id)

	// Corrupt the tracker state to emulate a stale frame edge surviving a
	// physical release. The public API deliberately prevents this transition.
	rec := tr.records[id]
	rec.ReleasedAt = 2
	tr.liveCount--
	tr.liveBytes -= rec.Bytes

	report := tr.FinalAudit()
	if len(report.Dangling) != 1 {
		t.Fatalf("dangling refs = %+v; want exactly one", report.Dangling)
	}
	dangling := report.Dangling[0]
	if dangling.Frame != 1 || dangling.TextureID != id || dangling.Count != 1 {
		t.Fatalf("dangling ref = %+v; want frame=1 texture=%d count=1", dangling, id)
	}
	foundReleasedWithRefs := false
	foundDanglingIssue := false
	for _, issue := range report.Issues {
		if issue.Code == IssueReleasedWithRefs && issue.TextureID == id {
			foundReleasedWithRefs = true
		}
		if issue.Code == IssueDanglingReference && issue.TextureID == id {
			foundDanglingIssue = true
		}
	}
	if !foundReleasedWithRefs || !foundDanglingIssue {
		t.Fatalf("expected released-with-refs and dangling issues: %+v", report.Issues)
	}
}
