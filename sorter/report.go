package sorter

import "sort"

func buildReport(
	items []candidate,
	latest uint64,
	queuePriority map[Queue]int,
	eyePriority map[Eye]int,
) Report {
	adoptedIndexes := make([]int, 0, len(items))
	rejectedIndexes := make([]int, 0, len(items))
	for index := range items {
		if items[index].reason == "" {
			adoptedIndexes = append(adoptedIndexes, index)
		} else {
			rejectedIndexes = append(rejectedIndexes, index)
		}
	}

	sort.Slice(adoptedIndexes, func(left, right int) bool {
		a := items[adoptedIndexes[left]]
		b := items[adoptedIndexes[right]]
		return adoptedLess(a, b, queuePriority, eyePriority)
	})
	sort.Slice(rejectedIndexes, func(left, right int) bool {
		a := items[rejectedIndexes[left]]
		b := items[rejectedIndexes[right]]
		return rejectedLess(a, b, queuePriority, eyePriority)
	})

	report := Report{
		LatestFrame: latest,
		Adopted:     make([]Adopted, 0, len(adoptedIndexes)),
		Rejected:    make([]Rejected, 0, len(rejectedIndexes)),
		Counts: Counts{
			Submitted: len(items),
			Adopted:   len(adoptedIndexes),
		},
	}

	for position, index := range adoptedIndexes {
		item := items[index]
		report.Adopted = append(report.Adopted, Adopted{
			Submission:      item.submission,
			Order:           position + 1,
			SubmissionOrder: item.submissionRank,
			CompletionOrder: item.completionRank,
			OutOfOrder:      item.outOfOrder,
		})
		if item.outOfOrder {
			report.Counts.OutOfOrderAdopted++
		}
	}

	for _, index := range rejectedIndexes {
		item := items[index]
		report.Rejected = append(report.Rejected, Rejected{
			Submission:      item.submission,
			Reason:          item.reason,
			DuplicateOf:     item.duplicateOf,
			SubmissionOrder: item.submissionRank,
			CompletionOrder: item.completionRank,
		})
		switch item.reason {
		case ReasonDuplicate:
			report.Counts.Duplicates++
		case ReasonTimeout:
			report.Counts.Timeouts++
		case ReasonHistoricalBackfill:
			report.Counts.HistoricalBackfills++
		case ReasonFutureFrame:
			report.Counts.FutureFrames++
		case ReasonInvalidQueue, ReasonInvalidEye:
			report.Counts.Invalid++
		}
	}

	return report
}

func adoptedLess(
	a, b candidate,
	queuePriority map[Queue]int,
	eyePriority map[Eye]int,
) bool {
	x := a.submission
	y := b.submission
	if x.Frame != y.Frame {
		return x.Frame < y.Frame
	}
	if aRank, bRank := queueRank(queuePriority, x.Queue), queueRank(queuePriority, y.Queue); aRank != bRank {
		return aRank < bRank
	}
	if aRank, bRank := eyeRank(eyePriority, x.Eye), eyeRank(eyePriority, y.Eye); aRank != bRank {
		return aRank < bRank
	}
	if x.ID != y.ID {
		return x.ID < y.ID
	}
	return submissionTotalLess(x, y)
}

func rejectedLess(
	a, b candidate,
	queuePriority map[Queue]int,
	eyePriority map[Eye]int,
) bool {
	x := a.submission
	y := b.submission
	if x.Frame != y.Frame {
		return x.Frame < y.Frame
	}
	if aRank, bRank := queueRank(queuePriority, x.Queue), queueRank(queuePriority, y.Queue); aRank != bRank {
		return aRank < bRank
	}
	if aRank, bRank := eyeRank(eyePriority, x.Eye), eyeRank(eyePriority, y.Eye); aRank != bRank {
		return aRank < bRank
	}
	if a.reason != b.reason {
		return a.reason < b.reason
	}
	if x.ID != y.ID {
		return x.ID < y.ID
	}
	return submissionTotalLess(x, y)
}
