package binocular

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDisconnectScopesReleaseEverythingAndReconnectClean(t *testing.T) {
	scopes := []DisconnectScope{DisconnectLeftEye, DisconnectRightEye, DisconnectDevice}
	for _, scope := range scopes {
		t.Run(string(scope), func(t *testing.T) {
			scenario, err := RunScenario(scope, ScenarioOptions{Cycles: 2, SettleTimeout: 2 * time.Second})
			if err != nil {
				t.Fatalf("RunScenario: %v", err)
			}
			if !scenario.Clean {
				t.Fatalf("scenario is not clean: %+v", scenario)
			}
			if len(scenario.Cycles) != 2 {
				t.Fatalf("cycles = %d, want 2", len(scenario.Cycles))
			}

			var tokens []string
			for _, cycle := range scenario.Cycles {
				assertRelease(t, cycle.Release)
				if len(cycle.Unreleased) != 0 {
					t.Fatalf("cycle %d unreleased: %v", cycle.Cycle, cycle.Unreleased)
				}
				if !cycle.PrivateState.Clean || len(cycle.PrivateState.Inherited) != 0 {
					t.Fatalf("cycle %d inherited private state: %+v", cycle.Cycle, cycle.PrivateState)
				}
				if cycle.PrivateState.OldToken == cycle.PrivateState.NewToken {
					t.Fatalf("cycle %d reused private token", cycle.Cycle)
				}
				tokens = append(tokens, cycle.PrivateState.NewToken)
				if !cycle.FirstFrame.Equal || cycle.FirstFrame.DiffBytes != 0 {
					t.Fatalf("cycle %d first frame differs: %+v", cycle.Cycle, cycle.FirstFrame)
				}
				for _, side := range cycle.FirstFrame.Sides {
					if !side.Equal || side.DiffBytes != 0 || side.DiffPixels != 0 {
						t.Fatalf("cycle %d %s first frame differs: %+v", cycle.Cycle, side.Eye, side)
					}
				}
				if len(cycle.StaleWrites) != 20 {
					t.Fatalf("cycle %d stale checks = %d, want 20", cycle.Cycle, len(cycle.StaleWrites))
				}
				for _, stale := range cycle.StaleWrites {
					if stale.Accepted {
						t.Fatalf("cycle %d accepted stale write: %+v", cycle.Cycle, stale)
					}
				}
				if cycle.Reconnect.TotalNS <= 0 {
					t.Fatalf("cycle %d reconnect latency = %d", cycle.Cycle, cycle.Reconnect.TotalNS)
				}
			}
			if tokens[0] == tokens[1] {
				t.Fatalf("double reconnect reused private token: %q", tokens[0])
			}
		})
	}
}

func assertRelease(t *testing.T, report ReleaseReport) {
	t.Helper()
	if report.Scope == "" {
		t.Fatal("release report has no scope")
	}
	if !report.Complete || len(report.Unreleased) != 0 {
		t.Fatalf("release incomplete: %+v", report)
	}
	if report.Tasks.Submitted != 10 || report.Tasks.Running != 8 || report.Tasks.Queued != 2 {
		t.Fatalf("unexpected task layout: %+v", report.Tasks)
	}
	if report.Tasks.Stopped != 8 || report.Tasks.QueuedCancel != 2 ||
		report.Tasks.StillRunning != 0 || report.Tasks.Failed != 0 {
		t.Fatalf("tasks not fully stopped: %+v", report.Tasks)
	}
	want := map[string][3]int{
		"textures":    {4, 4, 0},
		"depth":       {2, 2, 0},
		"history":     {2, 2, 0},
		"queue_tasks": {10, 10, 0},
	}
	for name, counts := range want {
		category, ok := report.Category(name)
		if !ok {
			t.Fatalf("missing release category %q", name)
		}
		if category.Discovered != counts[0] || category.Released != counts[1] ||
			category.Failed != counts[2] || !category.OK {
			t.Fatalf("category %s = %+v, want discovered=%d released=%d failed=%d",
				name, category, counts[0], counts[1], counts[2])
		}
	}
	cache, ok := report.Category("cache_keys")
	if !ok || !cache.OK || cache.Discovered < 4 || cache.Released != cache.Discovered {
		t.Fatalf("cache release = %+v", cache)
	}
	if len(cache.Keys) != cache.Discovered {
		t.Fatalf("cache keys = %v, discovered=%d", cache.Keys, cache.Discovered)
	}
	if report.LeasesRevoked != 10 {
		t.Fatalf("revoked leases = %d, want 10", report.LeasesRevoked)
	}
}

func TestStaleTaskCannotWriteNewBufferOrCache(t *testing.T) {
	device := NewDevice()
	old := device.CurrentSession()
	workload, err := old.PrepareWorkload()
	if err != nil {
		t.Fatal(err)
	}
	if !old.WaitUntilReady(8, 2, 4, 2*time.Second) {
		t.Fatal("workload did not start")
	}
	var textureLease, cacheLease *Lease
	for _, lease := range workload.Leases {
		if textureLease == nil && lease.Kind() == ResourceTexture {
			textureLease = lease
		}
		if lease.Kind() == ResourceCache {
			cacheLease = lease
		}
	}
	if textureLease == nil || cacheLease == nil {
		t.Fatal("workload leases missing")
	}

	if _, err := device.Disconnect(DisconnectLeftEye); err != nil {
		t.Fatal(err)
	}
	next, _, err := device.Reconnect()
	if err != nil {
		t.Fatal(err)
	}
	before, err := next.RenderFrame()
	if err != nil {
		t.Fatal(err)
	}

	if err := next.Write(textureLease, 0, []byte{1, 2, 3, 4}); !errors.Is(err, ErrStaleLease) {
		t.Fatalf("new session stale write error = %v, want ErrStaleLease", err)
	}
	if err := next.PutCache(cacheLease, "stale", "value"); !errors.Is(err, ErrStaleLease) {
		t.Fatalf("new session stale cache error = %v, want ErrStaleLease", err)
	}
	if err := old.Write(textureLease, 0, []byte{1, 2, 3, 4}); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("old session write error = %v, want ErrSessionClosed", err)
	}
	if next.res.cacheSize() != 0 {
		t.Fatalf("stale cache key landed in new session")
	}

	after, err := next.RenderFrame()
	if err != nil {
		t.Fatal(err)
	}
	diff := CompareFrames(before, after)
	if !diff.Equal || diff.DiffBytes != 0 {
		t.Fatalf("stale write changed new buffer: %+v", diff)
	}
	if got := device.UnreleasedObjects(); len(got) != 0 {
		t.Fatalf("old session leaked: %v", got)
	}
}

func TestCloseWaitsForCancellationAndIsIdempotent(t *testing.T) {
	session := newSession(1, 1, CleanFrame())
	started := make(chan struct{})
	task, err := session.Submit(EyeLeft, ResourceTexture, func(ctx context.Context, _ *Task) error {
		close(started)
		<-ctx.Done()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("task did not start")
	}

	first := session.Close(DisconnectDevice)
	second := session.Close(DisconnectDevice)
	if task.State() != TaskStopped {
		t.Fatalf("task state = %s, want stopped", task.State())
	}
	if first.Tasks.StillRunning != 0 || !first.Complete {
		t.Fatalf("first close incomplete: %+v", first)
	}
	if second.Tasks.Submitted != first.Tasks.Submitted || len(second.Unreleased) != 0 {
		t.Fatalf("second close was not idempotent: first=%+v second=%+v", first, second)
	}
	if _, err := session.Submit(EyeRight, ResourceDepth, func(context.Context, *Task) error { return nil }); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("submit after close = %v, want ErrSessionClosed", err)
	}
	if got := session.UnreleasedObjects(); len(got) != 0 {
		t.Fatalf("closed session leaked: %v", got)
	}
}
