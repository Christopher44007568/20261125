package geomlife

import (
	"bytes"
	"encoding/hex"
	"errors"
)

type CanaryReport struct {
	OK                       bool   `json:"ok"`
	UnsafeDrawCorrupted      bool   `json:"unsafe_draw_corrupted"`
	SafeDrawIntact           bool   `json:"safe_draw_intact"`
	ActiveReleaseBlocked     bool   `json:"active_release_blocked"`
	OldPatternHex            string `json:"old_pattern_hex"`
	UnsafeObservedPatternHex string `json:"unsafe_observed_pattern_hex"`
	SafeObservedPatternHex   string `json:"safe_observed_pattern_hex"`
	Explanation              string `json:"explanation"`
}

func RunGeometryCanary() CanaryReport {
	oldPattern := bytes.Repeat([]byte{0x01, 0x02, 0x03, 0x04}, 4)
	newPattern := bytes.Repeat([]byte{0xAA, 0xBB, 0xCC, 0xDD}, 4)

	unsafeMemory := make([]byte, 32)
	copy(unsafeMemory[0:16], oldPattern)
	copy(unsafeMemory[0:16], newPattern)
	unsafeObserved := append([]byte(nil), unsafeMemory[0:16]...)
	unsafeCorrupted := !bytes.Equal(unsafeObserved, oldPattern)

	arena, err := New(Config{Capacity: 32})
	report := CanaryReport{
		OldPatternHex:            hex.EncodeToString(oldPattern),
		UnsafeObservedPatternHex: hex.EncodeToString(unsafeObserved),
		Explanation:              "vertex pattern stands in for one temporary triangle range",
	}
	if err != nil {
		report.Explanation = err.Error()
		return report
	}

	oldSubmission, oldTickets, err := arena.Submit(SubmitRequest{FrameID: 1, Sizes: []int{16}})
	if err != nil {
		report.Explanation = err.Error()
		return report
	}
	newSubmission, newTickets, err := arena.Submit(SubmitRequest{FrameID: 2, Sizes: []int{16}})
	if err != nil {
		report.Explanation = err.Error()
		return report
	}

	safeMemory := make([]byte, arena.Capacity())
	oldTicket := oldTickets[0]
	newTicket := newTickets[0]
	copy(safeMemory[oldTicket.Offset:oldTicket.Offset+oldTicket.Size], oldPattern)
	copy(safeMemory[newTicket.Offset:newTicket.Offset+newTicket.Size], newPattern)

	_, releaseErr := arena.Release(oldTicket, oldSubmission.ID, ReasonExplicit)
	report.ActiveReleaseBlocked = errors.Is(releaseErr, ErrInvalidState)

	safeObserved := append([]byte(nil), safeMemory[oldTicket.Offset:oldTicket.Offset+oldTicket.Size]...)
	report.SafeObservedPatternHex = hex.EncodeToString(safeObserved)
	report.SafeDrawIntact = bytes.Equal(safeObserved, oldPattern)
	report.UnsafeDrawCorrupted = unsafeCorrupted
	report.OK = unsafeCorrupted && report.SafeDrawIntact && report.ActiveReleaseBlocked

	_, _ = arena.Complete(oldSubmission.ID)
	_, _ = arena.Complete(newSubmission.ID)
	_ = arena.CheckInvariants()
	return report
}
