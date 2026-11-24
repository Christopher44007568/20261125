package texturelife

import (
	"errors"
	"testing"
)

func newInitializedDevice(t *testing.T, frame Frame, size Size) (*Device, *Tracker) {
	t.Helper()
	tr := NewTracker()
	device := NewDevice(tr)
	if err := device.Initialize(frame, size); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	return device, tr
}

func textureOrFail(t *testing.T, tr *Tracker, id TextureID) Texture {
	t.Helper()
	texture, ok := tr.Texture(id)
	if !ok {
		t.Fatalf("texture %d metadata missing", id)
	}
	return texture
}

func TestWindowChangeReplacesBothEyeSets(t *testing.T) {
	oldSize := Size{Width: 100, Height: 100}
	newSize := Size{Width: 200, Height: 100}
	device, tr := newInitializedDevice(t, 1, oldSize)

	leftBefore, _ := device.EyeSet(EyeLeft)
	rightBefore, _ := device.EyeSet(EyeRight)
	oldIDs := append(leftBefore.IDs(), rightBefore.IDs()...)

	if err := device.Resize(2, newSize); err != nil {
		t.Fatalf("Resize: %v", err)
	}
	if device.Frame() != 2 || device.Viewport() != newSize {
		t.Fatalf("device frame/viewport = %d/%v; want 2/%v", device.Frame(), device.Viewport(), newSize)
	}

	for _, id := range oldIDs {
		texture := textureOrFail(t, tr, id)
		if texture.Alive() || texture.ReleasedAt != 2 || texture.RetireFrame != 2 {
			t.Fatalf("old texture not retired at transition: %+v", texture)
		}
		if texture.RefCount != 0 || texture.LastUsedFrame != 1 {
			t.Fatalf("old texture refs/last use = %d/%d; want 0/1", texture.RefCount, texture.LastUsedFrame)
		}
	}

	for _, eye := range []Eye{EyeLeft, EyeRight} {
		set, ok := device.EyeSet(eye)
		if !ok || set.Size != newSize {
			t.Fatalf("%s set = %+v, active=%v", eye, set, ok)
		}
		for _, id := range set.IDs() {
			texture := textureOrFail(t, tr, id)
			if !texture.Alive() || texture.RefCount != 1 || texture.LastUsedFrame != 2 {
				t.Fatalf("new texture state = %+v; want alive ref=1 last=2", texture)
			}
		}
	}

	report := tr.Snapshot()
	if !report.Healthy() || report.LiveCount != 6 || report.RetiringCount != 0 {
		t.Fatalf("post-resize snapshot = live %d retiring %d issues %+v", report.LiveCount, report.RetiringCount, report.Issues)
	}
	if report.Peak.Count != 12 || report.Peak.Bytes != 720000 || report.Peak.CountFrame != 2 {
		t.Fatalf("resize peak = %+v; want count=12 bytes=720000 frame=2", report.Peak)
	}
	if report.Peak.RefCount != 12 || report.Peak.RefFrame != 2 {
		t.Fatalf("resize peak refs = %d/%d; want 12/2", report.Peak.RefCount, report.Peak.RefFrame)
	}

	if err := device.Shutdown(3); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	final := tr.FinalAudit()
	if !final.Healthy() || final.LiveCount != 0 || len(final.Leaks) != 0 || len(final.Dangling) != 0 {
		t.Fatalf("final audit after resize/shutdown: %+v", final)
	}
}

func TestDestroySingleEyeLeavesOtherEyeReferenced(t *testing.T) {
	device, tr := newInitializedDevice(t, 1, Size{Width: 640, Height: 480})
	leftBefore, _ := device.EyeSet(EyeLeft)
	rightBefore, _ := device.EyeSet(EyeRight)

	if err := device.DestroyEye(2, EyeLeft); err != nil {
		t.Fatalf("DestroyEye: %v", err)
	}
	if _, ok := device.EyeSet(EyeLeft); ok {
		t.Fatal("left eye still active")
	}
	if device.Frame() != 2 {
		t.Fatalf("frame = %d; want 2", device.Frame())
	}
	for _, id := range leftBefore.IDs() {
		texture := textureOrFail(t, tr, id)
		if texture.Alive() || texture.ReleasedAt != 2 {
			t.Fatalf("left texture not released with eye: %+v", texture)
		}
	}
	for _, id := range rightBefore.IDs() {
		texture := textureOrFail(t, tr, id)
		if !texture.Alive() || texture.RefCount != 1 || texture.LastUsedFrame != 2 {
			t.Fatalf("right texture was disturbed: %+v", texture)
		}
	}
	if report := tr.Snapshot(); !report.Healthy() || report.LiveCount != 3 {
		t.Fatalf("snapshot after eye destruction: %+v", report)
	}

	if err := device.Shutdown(3); err != nil {
		t.Fatalf("Shutdown surviving eye: %v", err)
	}
	final := tr.FinalAudit()
	if !final.Healthy() || len(final.Leaks) != 0 {
		t.Fatalf("final audit after eye destruction: %+v", final)
	}
}

func TestSimulatedRebuildFailureRollsBackCandidates(t *testing.T) {
	device, tr := newInitializedDevice(t, 1, Size{Width: 128, Height: 72})
	leftBefore, _ := device.EyeSet(EyeLeft)
	rightBefore, _ := device.EyeSet(EyeRight)
	beforeIDs := append(leftBefore.IDs(), rightBefore.IDs()...)

	err := device.SimulateRebuildFailure(2, Size{Width: 256, Height: 144})
	if !errors.Is(err, ErrRebuildFailed) {
		t.Fatalf("failure error = %v; want ErrRebuildFailed", err)
	}
	if device.Frame() != 1 || device.Viewport() != (Size{Width: 128, Height: 72}) {
		t.Fatalf("failed rebuild changed device: frame=%d viewport=%v", device.Frame(), device.Viewport())
	}
	leftAfter, _ := device.EyeSet(EyeLeft)
	rightAfter, _ := device.EyeSet(EyeRight)
	if leftAfter != leftBefore || rightAfter != rightBefore {
		t.Fatalf("failed rebuild replaced eye sets: before=%+v/%+v after=%+v/%+v", leftBefore, rightBefore, leftAfter, rightAfter)
	}
	for _, id := range beforeIDs {
		texture := textureOrFail(t, tr, id)
		if !texture.Alive() || texture.RefCount != 1 || texture.LastUsedFrame != 1 {
			t.Fatalf("old texture damaged by failed rebuild: %+v", texture)
		}
	}

	report := tr.Snapshot()
	if !report.Healthy() || report.LiveCount != 6 || len(report.Textures) != 9 {
		t.Fatalf("snapshot after failed rebuild: live=%d created=%d issues=%+v", report.LiveCount, len(report.Textures), report.Issues)
	}
	releasedCandidates := 0
	for _, texture := range report.Textures {
		if !texture.Alive() {
			releasedCandidates++
			if texture.Eye != EyeLeft || texture.RetireFrame != 2 || texture.ReleasedAt != 2 {
				t.Fatalf("candidate rollback metadata = %+v", texture)
			}
		}
	}
	if releasedCandidates != 3 {
		t.Fatalf("released candidates = %d; want 3", releasedCandidates)
	}
	if report.Peak.Count != 9 || report.Peak.CountFrame != 2 {
		t.Fatalf("failed-attempt peak = %+v; want count=9 frame=2", report.Peak)
	}

	if err := device.Shutdown(3); err != nil {
		t.Fatalf("Shutdown after failed rebuild: %v", err)
	}
	if final := tr.FinalAudit(); !final.Healthy() {
		t.Fatalf("final audit after failed rebuild: %+v", final.Issues)
	}
}

func TestDeviceTransitionGuards(t *testing.T) {
	device := NewDevice(NewTracker())
	if err := device.Resize(1, Size{Width: 10, Height: 10}); !errors.Is(err, ErrDeviceNotReady) {
		t.Fatalf("Resize before initialize = %v; want ErrDeviceNotReady", err)
	}
	if err := device.Initialize(1, Size{Width: 10, Height: 10}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if err := device.Initialize(2, Size{Width: 10, Height: 10}); !errors.Is(err, ErrDeviceInitialized) {
		t.Fatalf("second Initialize = %v; want ErrDeviceInitialized", err)
	}
	if err := device.Resize(1, Size{Width: 20, Height: 10}); !errors.Is(err, ErrFrameNotAdvancing) {
		t.Fatalf("same-frame Resize = %v; want ErrFrameNotAdvancing", err)
	}
	if err := device.DestroyEye(2, Eye(99)); !errors.Is(err, ErrEyeNotActive) {
		t.Fatalf("destroy unknown eye = %v; want ErrEyeNotActive", err)
	}
}

func TestDestroyingLastEyeFinalizesDevice(t *testing.T) {
	device, tr := newInitializedDevice(t, 1, Size{Width: 32, Height: 32})
	if err := device.DestroyEye(2, EyeLeft); err != nil {
		t.Fatalf("destroy left: %v", err)
	}
	if err := device.DestroyEye(3, EyeRight); err != nil {
		t.Fatalf("destroy right: %v", err)
	}
	if device.Viewport() != (Size{}) {
		t.Fatalf("viewport after destroying both eyes = %v; want empty", device.Viewport())
	}
	if err := device.Resize(4, Size{Width: 64, Height: 32}); !errors.Is(err, ErrDeviceNotReady) {
		t.Fatalf("Resize after destroying both eyes = %v; want ErrDeviceNotReady", err)
	}
	if final := tr.FinalAudit(); !final.Healthy() || final.LiveCount != 0 {
		t.Fatalf("final audit after destroying both eyes: %+v", final)
	}

	if err := device.Initialize(4, Size{Width: 64, Height: 32}); err != nil {
		t.Fatalf("reinitialize: %v", err)
	}
	if err := device.Shutdown(5); err != nil {
		t.Fatalf("shutdown reinitialized device: %v", err)
	}
	if final := tr.FinalAudit(); !final.Healthy() {
		t.Fatalf("final audit after reinitialize: %+v", final)
	}
}
