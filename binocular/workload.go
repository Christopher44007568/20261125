package binocular

import (
	"context"
	"fmt"
	"time"
)

type Workload struct {
	Leases []*Lease
	Tasks  []*Task
}

// PrepareWorkload creates long-running tasks for both eyes and a small queue
// backlog. Eight workers run immediately; two extra jobs remain queued.
func (s *Session) PrepareWorkload() (*Workload, error) {
	w := &Workload{}
	leases := make(map[string]*Lease)
	for _, eye := range []Eye{EyeLeft, EyeRight} {
		for _, kind := range []ResourceKind{ResourceTexture, ResourceDepth, ResourceHistory, ResourceCache} {
			lease, err := s.AcquireLease(kind, eye)
			if err != nil {
				return w, err
			}
			leases[string(kind)+":"+string(eye)] = lease
			w.Leases = append(w.Leases, lease)
		}
		if err := s.PutCache(leases["cache:"+string(eye)], "seed:"+string(eye), "session="+fmt.Sprint(s.id)); err != nil {
			return w, err
		}
	}
	s.private.bumpEpoch()

	specs := []struct {
		eye  Eye
		kind ResourceKind
	}{
		{EyeLeft, ResourceTexture}, {EyeRight, ResourceTexture},
		{EyeLeft, ResourceDepth}, {EyeRight, ResourceDepth},
		{EyeLeft, ResourceHistory}, {EyeRight, ResourceHistory},
		{EyeLeft, ResourceCache}, {EyeRight, ResourceCache},
	}
	for i, spec := range specs {
		task, err := s.Submit(spec.eye, spec.kind, s.periodicTask(leases[string(spec.kind)+":"+string(spec.eye)], byte(i)))
		if err != nil {
			return w, err
		}
		w.Tasks = append(w.Tasks, task)
	}
	for _, eye := range []Eye{EyeLeft, EyeRight} {
		lease, err := s.AcquireLease(ResourceTexture, eye)
		if err != nil {
			return w, err
		}
		w.Leases = append(w.Leases, lease)
		task, err := s.Submit(eye, ResourceTexture, s.periodicTask(lease, 0xa5))
		if err != nil {
			return w, err
		}
		w.Tasks = append(w.Tasks, task)
	}
	return w, nil
}

func (s *Session) periodicTask(lease *Lease, marker byte) TaskFunc {
	return func(ctx context.Context, task *Task) error {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		var seq byte
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				seq++
				payload := []byte{marker, seq, byte(task.eye[0]), byte(task.kind[0])}
				var err error
				if task.kind == ResourceCache {
					key := fmt.Sprintf("live:%s:%d", task.eye, lease.id)
					err = s.PutCache(lease, key, fmt.Sprintf("%d", seq))
				} else {
					err = s.Write(lease, int(seq)*BytesPerPx, payload)
					if task.kind == ResourceHistory && err == nil {
						s.private.noteHistory()
					}
				}
				if err != nil {
					if ctx.Err() != nil {
						return nil
					}
					return err
				}
			}
		}
	}
}

// WaitUntilRunning waits for the requested worker/queue layout or timeout.
func (s *Session) WaitUntilRunning(minRunning, minQueued int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, queued, _ := s.TaskCounts()
		if running >= minRunning && queued >= minQueued {
			return true
		}
		time.Sleep(100 * time.Microsecond)
	}
	return false
}

// WaitUntilReady additionally waits for real work from every resource family:
// history has advanced and both cache tasks have produced a live key.
func (s *Session) WaitUntilReady(minRunning, minQueued, minCacheKeys int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, queued, _ := s.TaskCounts()
		private := s.snapshotPrivate()
		if running >= minRunning && queued >= minQueued &&
			private.HistoryFrames > 0 && private.CacheEntries >= minCacheKeys {
			return true
		}
		time.Sleep(100 * time.Microsecond)
	}
	return false
}
