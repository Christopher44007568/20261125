package geomlife

import (
	"errors"
	"testing"
)

func TestMismatchedTicketCannotRelease(t *testing.T) {
	system, _ := newTestSystem(t, 64, 0)
	ticket, err := system.Allocate(32)
	if err != nil {
		t.Fatal(err)
	}

	badTicket := ticket
	badTicket.Size = 16
	if _, err := system.Release(badTicket, 0, ReasonExplicit); !errors.Is(err, ErrRangeMismatch) {
		t.Fatalf("Release(mismatched) error = %v, want ErrRangeMismatch", err)
	}

	object, ok := system.Object(ticket.AllocationSeq)
	if !ok || object.State != ObjectHeld {
		t.Fatalf("object should remain held after mismatched ticket: %+v ok=%t", object, ok)
	}
	if err := system.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}
