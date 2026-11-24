package geomlife

import (
	"fmt"
	"sort"
	"strings"
)

type span struct {
	offset int
	size   int
	kind   string
	seq    uint64
}

func (s *System) CheckInvariants() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	problems := make([]string, 0)
	if uint64(len(s.objects)) != s.nextAllocationSeq {
		problems = append(problems, fmt.Sprintf(
			"object history length %d does not match next seq %d",
			len(s.objects), s.nextAllocationSeq,
		))
	}

	spans := make([]span, 0, len(s.free)+len(s.live))
	for _, freeRange := range s.free {
		if freeRange.Offset < 0 || freeRange.Size <= 0 || freeRange.Offset+freeRange.Size > s.capacity {
			problems = append(problems, fmt.Sprintf("invalid free interval %+v", freeRange))
		}
		spans = append(spans, span{offset: freeRange.Offset, size: freeRange.Size, kind: "free"})
	}

	for seq, record := range s.live {
		if record.info.AllocationSeq != seq || record.info.State != ObjectHeld {
			problems = append(problems, fmt.Sprintf("live map contains bad record seq=%d", seq))
		}
		spans = append(spans, span{
			offset: record.info.Offset,
			size:   record.info.Size,
			kind:   "live",
			seq:    seq,
		})
	}

	sort.Slice(spans, func(i, j int) bool {
		if spans[i].offset != spans[j].offset {
			return spans[i].offset < spans[j].offset
		}
		return spans[i].size < spans[j].size
	})

	cursor := 0
	for _, current := range spans {
		if current.offset != cursor {
			problems = append(problems, fmt.Sprintf(
				"interval does not tile arena at %d: %s at offset %d",
				cursor, current.kind, current.offset,
			))
			break
		}
		cursor += current.size
	}
	if cursor != s.capacity {
		problems = append(problems, fmt.Sprintf("intervals end at %d, capacity is %d", cursor, s.capacity))
	}

	for index, record := range s.objects {
		expectedSeq := uint64(index + 1)
		info := record.info
		if info.AllocationSeq != expectedSeq {
			problems = append(problems, fmt.Sprintf("object index %d has seq %d", index, info.AllocationSeq))
		}

		if info.State == ObjectHeld {
			if _, ok := s.live[info.AllocationSeq]; !ok {
				problems = append(problems, fmt.Sprintf("held seq=%d missing from live map", info.AllocationSeq))
			}
		} else {
			if info.ReleaseReason == "" || info.ReleaseReason == ReasonPending {
				problems = append(problems, fmt.Sprintf("reclaimed seq=%d has no release reason", info.AllocationSeq))
			}
			if info.ReleasedAt == nil {
				problems = append(problems, fmt.Sprintf("reclaimed seq=%d has no release time", info.AllocationSeq))
			}
			if _, ok := s.live[info.AllocationSeq]; ok {
				problems = append(problems, fmt.Sprintf("reclaimed seq=%d remains in live map", info.AllocationSeq))
			}
			if len(record.claims) != 0 {
				problems = append(problems, fmt.Sprintf("reclaimed seq=%d still has claims", info.AllocationSeq))
			}
		}

		for submissionID := range record.claims {
			submission, ok := s.submissions[submissionID]
			if !ok {
				problems = append(problems, fmt.Sprintf("seq=%d claimed by missing submission %d", info.AllocationSeq, submissionID))
				continue
			}
			if submission.info.State != StateSubmitted {
				problems = append(problems, fmt.Sprintf(
					"seq=%d still claimed by terminal submission %d (%s)",
					info.AllocationSeq, submissionID, submission.info.State,
				))
			}
			if !containsUint(submission.info.AllocationSeqs, info.AllocationSeq) {
				problems = append(problems, fmt.Sprintf(
					"seq=%d claim is missing from submission %d allocation list",
					info.AllocationSeq, submissionID,
				))
			}
		}
	}

	for _, submission := range s.submissions {
		seen := make(map[uint64]struct{})
		for _, seq := range submission.info.AllocationSeqs {
			if _, duplicate := seen[seq]; duplicate {
				problems = append(problems, fmt.Sprintf("submission %d lists seq=%d twice", submission.info.ID, seq))
			}
			seen[seq] = struct{}{}

			record := s.objectBySeqLocked(seq)
			if record == nil {
				problems = append(problems, fmt.Sprintf("submission %d references missing seq=%d", submission.info.ID, seq))
				continue
			}
			_, claimed := record.claims[submission.info.ID]
			if submission.info.State == StateSubmitted && !claimed {
				problems = append(problems, fmt.Sprintf("active submission %d lost claim on seq=%d", submission.info.ID, seq))
			}
			if submission.info.State != StateSubmitted && claimed {
				problems = append(problems, fmt.Sprintf("terminal submission %d retained claim on seq=%d", submission.info.ID, seq))
			}
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("%w:\n  - %s", ErrInvariantViolation, strings.Join(problems, "\n  - "))
	}
	return nil
}

func containsUint(values []uint64, target uint64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
