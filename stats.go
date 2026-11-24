package eyestage

import (
	"math"
	"sort"
)

func percentile(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	rank := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if rank < 0 {
		rank = 0
	}
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return sorted[rank]
}

func bounds(values []int64) (minValue, maxValue int64) {
	if len(values) == 0 {
		return 0, 0
	}
	minValue = values[0]
	maxValue = values[0]
	for _, value := range values[1:] {
		if value < minValue {
			minValue = value
		}
		if value > maxValue {
			maxValue = value
		}
	}
	return minValue, maxValue
}

func buildHistogram(values []int64, bucketCount int) []HistogramBucket {
	if len(values) == 0 {
		return nil
	}
	if bucketCount < 1 {
		bucketCount = 1
	}
	minValue, maxValue := bounds(values)
	if minValue == maxValue {
		upper := maxValue + 1
		if upper <= maxValue {
			upper = maxValue
		}
		return []HistogramBucket{{LowerNS: minValue, UpperNS: upper, Count: len(values)}}
	}
	if maxValue < 0 {
		maxValue = 0
	}
	if minValue < 0 {
		minValue = 0
	}
	width := int64(math.Ceil(float64(maxValue-minValue+1) / float64(bucketCount)))
	if width < 1 {
		width = 1
	}
	buckets := make([]HistogramBucket, bucketCount)
	for index := range buckets {
		lower := minValue + int64(index)*width
		upper := lower + width
		if index == bucketCount-1 {
			upper = maxValue + 1
			if upper <= lower {
				upper = lower + 1
			}
		}
		buckets[index] = HistogramBucket{LowerNS: lower, UpperNS: upper}
	}
	for _, value := range values {
		if value < minValue {
			value = minValue
		}
		index := int((value - minValue) / width)
		if index >= bucketCount {
			index = bucketCount - 1
		}
		buckets[index].Count++
	}
	return buckets
}

func summarize(values []int64, percentileValue float64, bucketCount int) StageStat {
	stat := StageStat{Count: len(values)}
	if len(values) == 0 {
		return stat
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	stat.MinNS = sorted[0]
	stat.P50NS = percentile(sorted, 50)
	stat.P95NS = percentile(sorted, 95)
	stat.P99NS = percentile(sorted, 99)
	stat.MaxNS = sorted[len(sorted)-1]
	for _, value := range sorted {
		stat.SumNS += value
	}
	stat.TailCutoff = percentile(sorted, percentileValue)
	stat.Histogram = buildHistogram(sorted, bucketCount)
	return stat
}
