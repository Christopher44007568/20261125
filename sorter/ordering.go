package sorter

import (
	"sort"
	"time"
)

func validQueue(queue Queue) bool {
	return queue >= QueueWorld && queue <= QueuePassthrough
}

func validEye(eye Eye) bool {
	return eye >= EyeLeft && eye <= EyeBoth
}

func assignRanks(items []candidate) {
	submissionOrder := make([]int, len(items))
	for i := range items {
		submissionOrder[i] = i
	}
	sort.Slice(submissionOrder, func(left, right int) bool {
		a := items[submissionOrder[left]].submission
		b := items[submissionOrder[right]].submission
		return submissionOrderLess(a, b)
	})
	assignDenseRanks(items, submissionOrder, true)

	completionOrder := make([]int, len(items))
	for i := range items {
		completionOrder[i] = i
	}
	sort.Slice(completionOrder, func(left, right int) bool {
		a := items[completionOrder[left]].submission
		b := items[completionOrder[right]].submission
		return completionOrderLess(a, b)
	})
	assignDenseRanks(items, completionOrder, false)
}

func assignDenseRanks(items []candidate, order []int, submission bool) {
	rank := 0
	for position, index := range order {
		if position == 0 || !sameTimelineValue(
			items[order[position-1]].submission,
			items[index].submission,
			submission,
		) {
			rank = position + 1
		}
		if submission {
			items[index].submissionRank = rank
		} else {
			items[index].completionRank = rank
		}
	}
}

func sameTimelineValue(a, b Submission, submission bool) bool {
	if submission {
		return !submissionOrderLess(a, b) && !submissionOrderLess(b, a)
	}
	return !completionOrderLess(a, b) && !completionOrderLess(b, a)
}

func submissionOrderLess(a, b Submission) bool {
	if comparison := compareTimes(a.SubmittedAt, b.SubmittedAt); comparison != 0 {
		return comparison < 0
	}
	return submissionTotalLess(a, b)
}

func completionOrderLess(a, b Submission) bool {
	if comparison := compareTimes(a.CompletedAt, b.CompletedAt); comparison != 0 {
		return comparison < 0
	}
	return submissionTotalLess(a, b)
}

func submissionTotalLess(a, b Submission) bool {
	if a.Frame != b.Frame {
		return a.Frame < b.Frame
	}
	if a.Queue != b.Queue {
		return a.Queue < b.Queue
	}
	if a.Eye != b.Eye {
		return a.Eye < b.Eye
	}
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	if comparison := compareTimes(a.SubmittedAt, b.SubmittedAt); comparison != 0 {
		return comparison < 0
	}
	if comparison := compareTimes(a.CompletedAt, b.CompletedAt); comparison != 0 {
		return comparison < 0
	}
	if comparison := compareTimes(a.Deadline, b.Deadline); comparison != 0 {
		return comparison < 0
	}
	return false
}

func duplicateLess(a, b Submission) bool {
	if comparison := compareTimes(a.SubmittedAt, b.SubmittedAt); comparison != 0 {
		return comparison < 0
	}
	if comparison := compareTimes(a.CompletedAt, b.CompletedAt); comparison != 0 {
		return comparison < 0
	}
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	return submissionTotalLess(a, b)
}

// compareTimes orders known timestamps before unknown ones. A zero timestamp
// is missing metadata, not 0001-01-01 in the timeline.
func compareTimes(a, b time.Time) int {
	aKnown := !a.IsZero()
	bKnown := !b.IsZero()
	switch {
	case aKnown && !bKnown:
		return -1
	case !aKnown && bKnown:
		return 1
	case !aKnown && !bKnown:
		return 0
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}

func priorityRanks(queueOrder []Queue, eyeOrder []Eye) (map[Queue]int, map[Eye]int) {
	queues := make(map[Queue]int, len(queueOrder))
	for index, queue := range queueOrder {
		queues[queue] = index
	}
	eyes := make(map[Eye]int, len(eyeOrder))
	for index, eye := range eyeOrder {
		eyes[eye] = index
	}
	return queues, eyes
}

func queueRank(queueRank map[Queue]int, queue Queue) int {
	if rank, ok := queueRank[queue]; ok {
		return rank
	}
	return len(defaultQueueOrder) + int(queue)
}

func eyeRank(eyeRank map[Eye]int, eye Eye) int {
	if rank, ok := eyeRank[eye]; ok {
		return rank
	}
	return len(defaultEyeOrder) + int(eye)
}
