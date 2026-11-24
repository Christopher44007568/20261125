package geomlife

import (
	"fmt"
	"sort"
	"time"
)

func (s *System) Allocate(size int) (Ticket, error) {
	now := s.clock.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	ticket, _, err := s.allocateLocked(now, size)
	if err != nil {
		return Ticket{}, err
	}
	return ticket, nil
}

func (s *System) allocateLocked(now time.Time, size int) (Ticket, *objectRecord, error) {
	if size <= 0 {
		return Ticket{}, nil, fmt.Errorf("%w: %d", ErrInvalidSize, size)
	}

	chosen := -1
	for i, freeRange := range s.free {
		if freeRange.Size >= size {
			chosen = i
			break
		}
	}
	if chosen == -1 {
		return Ticket{}, nil, fmt.Errorf("%w: size=%d", ErrOutOfMemory, size)
	}

	freeRange := s.free[chosen]
	offset := freeRange.Offset
	if freeRange.Size == size {
		s.free = append(s.free[:chosen], s.free[chosen+1:]...)
	} else {
		s.free[chosen].Offset += size
		s.free[chosen].Size -= size
	}

	s.nextAllocationSeq++
	seq := s.nextAllocationSeq
	record := &objectRecord{
		claims: make(map[uint64]struct{}),
		info: Object{
			AllocationSeq: seq,
			Offset:        offset,
			Size:          size,
			State:         ObjectHeld,
			ReleaseReason: ReasonPending,
		},
	}
	s.objects = append(s.objects, record)
	s.live[seq] = record

	ticket := Ticket{AllocationSeq: seq, Offset: offset, Size: size}
	s.emitLocked(now, Event{
		Kind:          "allocated",
		OK:            true,
		AllocationSeq: seq,
		Offset:        offset,
		Size:          size,
	})
	return ticket, record, nil
}

func (s *System) releaseIntervalLocked(released interval) {
	if released.Size <= 0 {
		return
	}
	s.free = append(s.free, released)
	sort.Slice(s.free, func(i, j int) bool { return s.free[i].Offset < s.free[j].Offset })

	merged := s.free[:0]
	for _, current := range s.free {
		if len(merged) == 0 {
			merged = append(merged, current)
			continue
		}
		last := &merged[len(merged)-1]
		lastEnd := last.Offset + last.Size
		currentEnd := current.Offset + current.Size
		if current.Offset > lastEnd {
			merged = append(merged, current)
			continue
		}
		if currentEnd > lastEnd {
			last.Size = currentEnd - last.Offset
		}
	}
	s.free = merged
}

func (s *System) resolveTicketLocked(ticket Ticket) (*objectRecord, error) {
	record := s.objectBySeqLocked(ticket.AllocationSeq)
	if record == nil {
		return nil, fmt.Errorf("%w: allocation_seq=%d", ErrNotFound, ticket.AllocationSeq)
	}
	if ticket.Offset != record.info.Offset || ticket.Size != record.info.Size {
		return record, fmt.Errorf(
			"%w: ticket={seq=%d offset=%d size=%d} object={offset=%d size=%d}",
			ErrRangeMismatch,
			ticket.AllocationSeq, ticket.Offset, ticket.Size,
			record.info.Offset, record.info.Size,
		)
	}
	return record, nil
}

func (s *System) overlapOwnerLocked(ticket Ticket) *objectRecord {
	if ticket.Size <= 0 {
		return nil
	}
	ticketEnd := ticket.Offset + ticket.Size
	for _, record := range s.live {
		other := record.info
		otherEnd := other.Offset + other.Size
		if ticket.Offset < otherEnd && other.Offset < ticketEnd {
			return record
		}
	}
	return nil
}

func (s *System) maxHolderLocked(record *objectRecord, excluded uint64) uint64 {
	var holder uint64
	for id := range record.claims {
		if id != excluded && id > holder {
			holder = id
		}
	}
	return holder
}

func (s *System) reclaimLocked(now time.Time, record *objectRecord, reason ReleaseReason, releasedBy uint64) {
	if record.info.State != ObjectHeld {
		return
	}
	if reason == "" {
		reason = ReasonExplicit
	}

	record.info.State = ObjectReclaimed
	record.info.ReleaseReason = reason
	releasedAt := now
	record.info.ReleasedAt = &releasedAt
	record.info.ReleasedBySubmissionID = releasedBy
	record.claims = make(map[uint64]struct{})

	delete(s.live, record.info.AllocationSeq)
	s.releaseIntervalLocked(interval{Offset: record.info.Offset, Size: record.info.Size})

	s.emitLocked(now, Event{
		Kind:          "reclaimed",
		OK:            true,
		SubmissionID:  releasedBy,
		AllocationSeq: record.info.AllocationSeq,
		Offset:        record.info.Offset,
		Size:          record.info.Size,
		Reason:        reason,
	})
}
