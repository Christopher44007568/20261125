package binocular

import (
	"fmt"
)

// CheckPrivateState proves that the replacement session did not inherit any
// mutable private state from the session it replaced.
func CheckPrivateState(oldSession, newSession *Session) PrivateStateCheck {
	oldSnap := oldSession.snapshotPrivate()
	newSnap := newSession.snapshotPrivate()
	inherited := []string{}
	if newSnap.Token == oldSnap.Token {
		inherited = append(inherited, "private token was reused")
	}
	if newSnap.ParentSessionID != 0 {
		inherited = append(inherited, fmt.Sprintf("parent session id is %d", newSnap.ParentSessionID))
	}
	if newSnap.Epoch != 0 {
		inherited = append(inherited, fmt.Sprintf("private epoch is %d", newSnap.Epoch))
	}
	if newSnap.HistoryFrames != 0 {
		inherited = append(inherited, fmt.Sprintf("history frame count is %d", newSnap.HistoryFrames))
	}
	if newSnap.CacheEntries != 0 {
		inherited = append(inherited, fmt.Sprintf("cache entry count is %d", newSnap.CacheEntries))
	}
	return PrivateStateCheck{
		Clean:              len(inherited) == 0,
		Inherited:          inherited,
		OldToken:           oldSnap.Token,
		NewToken:           newSnap.Token,
		NewEpoch:           newSnap.Epoch,
		NewParentSessionID: newSnap.ParentSessionID,
		NewHistoryFrames:   newSnap.HistoryFrames,
		NewCacheEntries:    newSnap.CacheEntries,
	}
}

// CheckStaleWrites exercises every lease held by the old workload against both
// the closed session and the replacement session. At least one write is aimed
// at the first pixel so an accepted stale write would immediately show up in
// the first-frame comparison.
func CheckStaleWrites(oldSession, newSession *Session, workload *Workload) []StaleWriteCheck {
	if workload == nil {
		return nil
	}
	var checks []StaleWriteCheck
	for _, lease := range workload.Leases {
		checks = append(checks,
			attemptWrite("old_session", oldSession, lease),
			attemptWrite("new_session", newSession, lease),
		)
	}
	return checks
}

func attemptWrite(path string, session *Session, lease *Lease) StaleWriteCheck {
	var err error
	if lease.Kind() == ResourceCache {
		err = session.PutCache(lease, fmt.Sprintf("stale:%d", lease.ID()), "must-not-land")
	} else {
		err = session.Write(lease, 0, []byte{0xde, 0xad, 0xbe, 0xef})
	}
	check := StaleWriteCheck{
		Path:     fmt.Sprintf("%s:lease=%d:%s:%s", path, lease.ID(), lease.Eye(), lease.Kind()),
		Accepted: err == nil,
	}
	if err != nil {
		check.Error = err.Error()
	}
	return check
}

func CheckFirstFrame(baseline Frame, session *Session) (FirstFrameDiff, error) {
	got, err := session.RenderFrame()
	if err != nil {
		return FirstFrameDiff{}, err
	}
	return CompareFrames(baseline, got), nil
}

func anyStaleAccepted(checks []StaleWriteCheck) bool {
	for _, check := range checks {
		if check.Accepted {
			return true
		}
	}
	return false
}

func releaseOK(report ReleaseReport) bool {
	if !report.Complete || report.Tasks.StillRunning != 0 || len(report.Unreleased) != 0 {
		return false
	}
	for _, category := range report.Categories {
		if !category.OK || category.Failed != 0 {
			return false
		}
	}
	return true
}
