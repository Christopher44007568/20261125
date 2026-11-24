package stereorollback

import (
	"sync"
	"testing"
)

func TestPoolConcurrentLeasesReturnToBaseline(t *testing.T) {
	pool := NewResourcePool()
	if _, err := pool.Create(ResourceDesc{ID: ResSharedAtlas, Kind: KindSharedTexture, Shared: true}); err != nil {
		t.Fatal(err)
	}
	anchor, err := pool.Acquire(ResSharedAtlas, OwnerRecorder)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				lease, err := pool.Acquire(ResSharedAtlas, "eye:left")
				if err != nil {
					t.Errorf("acquire: %v", err)
					return
				}
				if err := pool.Release(lease); err != nil {
					t.Errorf("release: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	if got := pool.RefCount(ResSharedAtlas); got != 1 {
		t.Fatalf("after churn refcount should be the anchor only, got %d", got)
	}
	if err := pool.Release(anchor); err != nil {
		t.Fatal(err)
	}
	if got := pool.RefCount(ResSharedAtlas); got != 0 {
		t.Fatalf("after anchor release refcount should be zero, got %d", got)
	}
}

func TestExecuteEyeDirectlyRollsBackOnlyThatEye(t *testing.T) {
	p, err := NewStandardPipeline(Script{EyeFaults: map[uint64]map[Eye]FaultType{
		9: {EyeLeft: FaultAttachmentCreate},
	}})
	if err != nil {
		t.Fatal(err)
	}

	surface, failure := p.ExecuteEye(9, EyeLeft)
	if failure == nil || failure.FailureStage != StageAttachment {
		t.Fatalf("expected attachment failure, got %+v", failure)
	}
	if surface.Status != StatusPlaceholder || surface.Success {
		t.Fatalf("expected non-success placeholder, got %+v", surface)
	}
	if len(failure.Rollback.DestroyedPrivate) != 1 {
		t.Fatalf("partial attachment should be destroyed: %+v", failure.Rollback.DestroyedPrivate)
	}
	if _, alive := p.Pool().Info("attach.L.f9"); alive {
		t.Fatal("partial attachment must not survive rollback")
	}
	if p.Pool().RefCount(ResSharedAtlas) != 4 {
		t.Fatalf("shared atlas users must be intact, refcount=%d", p.Pool().RefCount(ResSharedAtlas))
	}

	right, rightFailure := p.ExecuteEye(9, EyeRight)
	if rightFailure != nil || right.Status != StatusRendered {
		t.Fatalf("right eye should render independently: surface=%+v failure=%+v", right, rightFailure)
	}
}
