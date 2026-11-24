package eyestage

import "sort"

func mergeIntervals(intervals []Interval) []Interval {
	if len(intervals) == 0 {
		return nil
	}
	sorted := append([]Interval(nil), intervals...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].StartNS == sorted[j].StartNS {
			return sorted[i].EndNS < sorted[j].EndNS
		}
		return sorted[i].StartNS < sorted[j].StartNS
	})
	merged := []Interval{sorted[0]}
	for _, current := range sorted[1:] {
		last := &merged[len(merged)-1]
		if current.StartNS <= last.EndNS {
			if current.EndNS > last.EndNS {
				last.EndNS = current.EndNS
			}
			continue
		}
		merged = append(merged, current)
	}
	return merged
}

func intersectLength(left, right []Interval) int64 {
	var total int64
	i, j := 0, 0
	for i < len(left) && j < len(right) {
		start := maxInt64(left[i].StartNS, right[j].StartNS)
		end := minInt64(left[i].EndNS, right[j].EndNS)
		if end > start {
			total += end - start
		}
		if left[i].EndNS < right[j].EndNS {
			i++
		} else {
			j++
		}
	}
	return total
}

func subtractIntervals(base, cut []Interval) []Interval {
	result := make([]Interval, 0)
	for _, source := range base {
		pieces := []Interval{source}
		for _, blocker := range cut {
			if blocker.EndNS <= source.StartNS {
				continue
			}
			if blocker.StartNS >= source.EndNS {
				break
			}
			next := make([]Interval, 0, len(pieces)+1)
			for _, piece := range pieces {
				if blocker.EndNS <= piece.StartNS || blocker.StartNS >= piece.EndNS {
					next = append(next, piece)
					continue
				}
				if blocker.StartNS > piece.StartNS {
					next = append(next, Interval{StartNS: piece.StartNS, EndNS: blocker.StartNS})
				}
				if blocker.EndNS < piece.EndNS {
					next = append(next, Interval{StartNS: blocker.EndNS, EndNS: piece.EndNS})
				}
			}
			pieces = next
		}
		result = append(result, pieces...)
	}
	return result
}

func intervalLength(intervals []Interval) int64 {
	var total int64
	for _, interval := range intervals {
		if interval.EndNS > interval.StartNS {
			total += interval.EndNS - interval.StartNS
		}
	}
	return total
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func minInt64(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}
