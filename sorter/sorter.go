package sorter

import (
	"fmt"
	"sort"
	"time"
)

type candidate struct {
	submission     Submission
	key            entryKey
	submissionRank int
	completionRank int
	reason         RejectReason
	duplicateOf    string
	outOfOrder     bool
}

// Sort orders a complete snapshot of queue completions. The newest frame is
// inferred as the greatest logical frame number; completion time never makes
// an entry newest.
func Sort(submissions []Submission, cfg Config) (Report, error) {
	return sortWithLatest(submissions, cfg, nil)
}

// SortAt is Sort with an externally supplied authoritative latest frame. An
// entry newer than latest is rejected as a future frame, which prevents a late
// outlier from redefining the presentation frontier.
func SortAt(submissions []Submission, latest uint64, cfg Config) (Report, error) {
	return sortWithLatest(submissions, cfg, &latest)
}

func sortWithLatest(submissions []Submission, cfg Config, latest *uint64) (Report, error) {
	queueOrder, eyeOrder, err := normalizeConfig(cfg)
	if err != nil {
		return Report{}, err
	}

	items := make([]candidate, len(submissions))
	for i, submission := range submissions {
		items[i] = candidate{
			submission: submission,
			key: entryKey{
				Frame: submission.Frame,
				Queue: submission.Queue,
				Eye:   submission.Eye,
			},
		}
	}
	assignRanks(items)

	if latest == nil {
		inferred := inferredLatest(items)
		latest = &inferred
	}

	markInvalid(items)
	markDuplicates(items)
	markTimedOut(items, cfg.Timeout)

	for i := range items {
		item := &items[i]
		if item.reason != "" {
			continue
		}

		if item.submission.Frame > *latest {
			item.reason = ReasonFutureFrame
			continue
		}
		if *latest-item.submission.Frame > cfg.ReorderWindow {
			item.reason = ReasonHistoricalBackfill
		}
	}

	markOutOfOrder(items)

	queueRank, eyeRank := priorityRanks(queueOrder, eyeOrder)
	report := buildReport(items, *latest, queueRank, eyeRank)
	return report, nil
}

func normalizeConfig(cfg Config) ([]Queue, []Eye, error) {
	if cfg.Timeout < 0 {
		return nil, nil, fmt.Errorf("timeout must not be negative")
	}

	queueOrder, err := completeQueueOrder(cfg.QueueOrder)
	if err != nil {
		return nil, nil, err
	}
	eyeOrder, err := completeEyeOrder(cfg.EyeOrder)
	if err != nil {
		return nil, nil, err
	}
	return queueOrder, eyeOrder, nil
}

func completeQueueOrder(configured []Queue) ([]Queue, error) {
	seen := make(map[Queue]bool, len(defaultQueueOrder))
	result := make([]Queue, 0, len(defaultQueueOrder))
	for _, queue := range configured {
		if !validQueue(queue) {
			return nil, fmt.Errorf("invalid queue in queue_order: %s", queue)
		}
		if seen[queue] {
			return nil, fmt.Errorf("duplicate queue in queue_order: %s", queue)
		}
		seen[queue] = true
		result = append(result, queue)
	}
	for _, queue := range defaultQueueOrder {
		if !seen[queue] {
			result = append(result, queue)
		}
	}
	return result, nil
}

func completeEyeOrder(configured []Eye) ([]Eye, error) {
	seen := make(map[Eye]bool, len(defaultEyeOrder))
	result := make([]Eye, 0, len(defaultEyeOrder))
	for _, eye := range configured {
		if !validEye(eye) {
			return nil, fmt.Errorf("invalid eye in eye_order: %s", eye)
		}
		if seen[eye] {
			return nil, fmt.Errorf("duplicate eye in eye_order: %s", eye)
		}
		seen[eye] = true
		result = append(result, eye)
	}
	for _, eye := range defaultEyeOrder {
		if !seen[eye] {
			result = append(result, eye)
		}
	}
	return result, nil
}

func inferredLatest(items []candidate) uint64 {
	var latest uint64
	for _, item := range items {
		if validQueue(item.submission.Queue) &&
			validEye(item.submission.Eye) &&
			item.submission.Frame > latest {
			latest = item.submission.Frame
		}
	}
	return latest
}

func markInvalid(items []candidate) {
	for i := range items {
		switch {
		case !validQueue(items[i].submission.Queue):
			items[i].reason = ReasonInvalidQueue
		case !validEye(items[i].submission.Eye):
			items[i].reason = ReasonInvalidEye
		}
	}
}

func markDuplicates(items []candidate) {
	groups := make(map[entryKey][]int)
	for i := range items {
		if items[i].reason != "" {
			continue
		}
		key := items[i].key
		groups[key] = append(groups[key], i)
	}

	for _, indexes := range groups {
		if len(indexes) < 2 {
			continue
		}
		canonical := indexes[0]
		for _, index := range indexes[1:] {
			if duplicateLess(items[index].submission, items[canonical].submission) {
				canonical = index
			}
		}
		for _, index := range indexes {
			if index == canonical {
				continue
			}
			items[index].reason = ReasonDuplicate
			items[index].duplicateOf = items[canonical].submission.ID
		}
	}
}

func markTimedOut(items []candidate, timeout time.Duration) {
	for i := range items {
		item := &items[i]
		if item.reason != "" {
			continue
		}
		submission := item.submission

		if !submission.Deadline.IsZero() &&
			!submission.CompletedAt.IsZero() &&
			submission.CompletedAt.After(submission.Deadline) {
			item.reason = ReasonTimeout
			continue
		}
		if timeout > 0 &&
			!submission.SubmittedAt.IsZero() &&
			!submission.CompletedAt.IsZero() &&
			submission.CompletedAt.Sub(submission.SubmittedAt) > timeout {
			item.reason = ReasonTimeout
		}
	}
}

func markOutOfOrder(items []candidate) {
	ordered := make([]int, 0, len(items))
	for i := range items {
		if items[i].reason == "" {
			ordered = append(ordered, i)
		}
	}
	sort.Slice(ordered, func(left, right int) bool {
		a := items[ordered[left]]
		b := items[ordered[right]]
		if a.completionRank != b.completionRank {
			return a.completionRank < b.completionRank
		}
		return submissionTotalLess(a.submission, b.submission)
	})

	var newestSeen uint64
	haveNewest := false
	for _, index := range ordered {
		frame := items[index].submission.Frame
		if haveNewest && frame < newestSeen {
			items[index].outOfOrder = true
		}
		if !haveNewest || frame > newestSeen {
			newestSeen = frame
			haveNewest = true
		}
	}
}
