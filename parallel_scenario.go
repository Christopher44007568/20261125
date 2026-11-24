package geomlife

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

type ParallelFrame struct {
	FrameID            uint64          `json:"frame_id"`
	SubmissionID       uint64          `json:"submission_id"`
	TerminalState      SubmissionState `json:"terminal_state"`
	Tickets            []Ticket        `json:"tickets"`
	RepeatReleaseError string          `json:"repeat_release_error,omitempty"`
	RepeatReleaseOK    bool            `json:"repeat_release_ok"`
}

type ParallelReport struct {
	OK         bool            `json:"ok"`
	Capacity   int             `json:"capacity"`
	Frames     []ParallelFrame `json:"frames"`
	Invariants Check           `json:"invariants"`
}

func RunParallelScenario() ParallelReport {
	system, err := New(Config{Capacity: 512})
	if err != nil {
		return ParallelReport{OK: false, Invariants: Check{Name: "system_created", OK: false, Detail: err.Error()}}
	}

	ready := make(chan error, 2)
	proceed := make(chan struct{})
	errs := make(chan error, 8)
	frames := make([]ParallelFrame, 2)

	var wait sync.WaitGroup
	for index, frameID := range []uint64{1, 2} {
		wait.Add(1)
		go func(index int, frameID uint64) {
			defer wait.Done()

			submission, tickets, submitErr := system.Submit(SubmitRequest{
				FrameID: frameID,
				Sizes:   []int{64, 32},
			})
			if submitErr != nil {
				ready <- submitErr
				return
			}

			ready <- nil
			<-proceed

			terminal := StateCompleted
			var finishErr error
			if frameID == 2 {
				terminal = StateCanceled
				_, finishErr = system.Cancel(submission.ID, ReasonCanceled)
			} else {
				_, finishErr = system.Complete(submission.ID)
			}
			if finishErr != nil {
				errs <- finishErr
				return
			}

			_, releaseErr := system.Release(tickets[0], submission.ID, ReasonExplicit)
			repeatOK := errors.Is(releaseErr, ErrDuplicateRelease)
			repeatError := ""
			if releaseErr != nil {
				repeatError = releaseErr.Error()
			}
			if !repeatOK {
				if releaseErr == nil {
					errs <- fmt.Errorf("frame %d expected duplicate release error", frameID)
				} else {
					errs <- releaseErr
				}
			}

			frames[index] = ParallelFrame{
				FrameID:            frameID,
				SubmissionID:       submission.ID,
				TerminalState:      terminal,
				Tickets:            tickets,
				RepeatReleaseError: repeatError,
				RepeatReleaseOK:    repeatOK,
			}
		}(index, frameID)
	}

	submitFailed := false
	for i := 0; i < 2; i++ {
		if readyErr := <-ready; readyErr != nil {
			submitFailed = true
			errs <- readyErr
		}
	}
	close(proceed)
	wait.Wait()
	close(errs)

	report := ParallelReport{OK: true, Capacity: system.Capacity(), Frames: frames}
	problems := make([]string, 0)
	for err := range errs {
		if err != nil {
			report.OK = false
			problems = append(problems, err.Error())
		}
	}
	if submitFailed {
		report.OK = false
	}

	invariantErr := system.CheckInvariants()
	report.Invariants = Check{Name: "parallel_arena_invariants", OK: invariantErr == nil}
	if invariantErr != nil {
		report.Invariants.Detail = invariantErr.Error()
		report.OK = false
	}
	if len(problems) > 0 {
		runtimeDetail := strings.Join(problems, "; ")
		if report.Invariants.Detail != "" {
			report.Invariants.Detail += "; " + runtimeDetail
		} else {
			report.Invariants.Detail = runtimeDetail
		}
	}
	for _, frame := range frames {
		if !frame.RepeatReleaseOK || frame.TerminalState == "" {
			report.OK = false
		}
	}
	return report
}
