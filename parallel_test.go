package geomlife

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestParallelDoubleFrame(t *testing.T) {
	system, _ := newTestSystem(t, 512, 0)
	started := make(chan uint64, 2)
	proceed := make(chan struct{})
	errs := make(chan error, 8)

	var wg sync.WaitGroup
	for frameID := uint64(1); frameID <= 2; frameID++ {
		wg.Add(1)
		go func(frameID uint64) {
			defer wg.Done()

			submission, tickets, err := system.Submit(SubmitRequest{
				FrameID: frameID,
				Sizes:   []int{64, 32},
			})
			if err != nil {
				errs <- err
				return
			}
			started <- frameID
			<-proceed

			if frameID == 1 {
				_, err = system.Complete(submission.ID)
			} else {
				_, err = system.Cancel(submission.ID, ReasonCanceled)
			}
			if err != nil {
				errs <- err
				return
			}

			_, releaseErr := system.Release(tickets[0], submission.ID, ReasonCompleted)
			if !errors.Is(releaseErr, ErrDuplicateRelease) {
				errs <- fmt.Errorf("frame %d repeat Release() error = %v, want duplicate", frameID, releaseErr)
			}
		}(frameID)
	}

	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for parallel frame submission")
		}
	}

	submissions := system.Submissions()
	if len(submissions) != 2 {
		t.Fatalf("submissions = %d, want 2", len(submissions))
	}
	for _, submission := range submissions {
		if submission.State != StateSubmitted {
			t.Fatalf("submission %d state = %q, want submitted before barrier opens", submission.ID, submission.State)
		}
	}

	close(proceed)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("parallel frames did not finish")
	}

	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
	if err := system.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentLifecycleAndTimeoutReaper(t *testing.T) {
	system, err := New(Config{
		Capacity:          4096,
		SubmissionTimeout: 2 * time.Millisecond,
		Clock:             SystemClock{},
	})
	if err != nil {
		t.Fatal(err)
	}

	stopReaper := make(chan struct{})
	reaperDone := make(chan struct{})
	go func() {
		defer close(reaperDone)
		for {
			select {
			case <-stopReaper:
				return
			default:
				_ = system.ExpireTimeouts()
				time.Sleep(time.Millisecond)
			}
		}
	}()

	errs := make(chan error, 64)
	var workers sync.WaitGroup
	for worker := 0; worker < 32; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()

			submission, tickets, submitErr := system.Submit(SubmitRequest{
				FrameID: uint64(worker + 1),
				Sizes:   []int{16, 32},
			})
			if submitErr != nil {
				errs <- submitErr
				return
			}

			time.Sleep(time.Duration(worker%4) * time.Millisecond)
			var finishErr error
			if worker%5 == 0 {
				_, finishErr = system.Cancel(submission.ID, ReasonCanceled)
			} else {
				_, finishErr = system.Complete(submission.ID)
			}
			if finishErr != nil && !errors.Is(finishErr, ErrAlreadyTerminal) {
				errs <- finishErr
			}

			for _, ticket := range tickets {
				_, releaseErr := system.Release(ticket, submission.ID, ReasonExplicit)
				if releaseErr != nil &&
					!errors.Is(releaseErr, ErrDuplicateRelease) &&
					!errors.Is(releaseErr, ErrStaleTicket) {
					errs <- releaseErr
				}
			}
		}(worker)
	}

	workersDone := make(chan struct{})
	go func() {
		workers.Wait()
		close(workersDone)
	}()
	select {
	case <-workersDone:
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent lifecycle test timed out")
	}
	close(stopReaper)
	<-reaperDone
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}

	_ = system.ExpireTimeouts()
	if err := system.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}
