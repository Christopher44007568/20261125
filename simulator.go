package main

import (
	"fmt"
	"hash/fnv"
	"sort"
	"time"
)

var layers = []LayerName{World, UI, Hand}

func simulateIndependent(scenario Scenario) SimulationReport {
	report := SimulationReport{Name: "independent-refresh"}
	current := map[LayerName]Frame{}

	for tick := 0; tick <= scenario.Ticks; tick++ {
		at := time.Duration(tick) * displayInterval
		for _, frame := range scenario.submissionsAt(at) {
			current[frame.Layer] = frame
		}
		if len(current) < len(layers) {
			report.MissedTicks = append(report.MissedTicks, missingTick(tick, at, current))
			continue
		}
		sources := map[LayerName]LayerSource{}
		for _, layer := range layers {
			frame := current[layer]
			sources[layer] = LayerSource{
				Layer:       layer,
				Sequence:    frame.Sequence,
				ProducedAt:  frame.ProducedAt,
				AvailableAt: frame.AvailableAt,
				Age:         frame.Age(at),
				Status:      sourceStatus(frame, at, scenario.Intervals[layer]),
				Digest:      frame.Digest,
			}
		}
		report.Frames = append(report.Frames, compositeFrame(tick, at, sources))
	}
	report.VisualDeltas = visualDeltas(report.Frames)
	return report
}

func simulateUnifiedWait(scenario Scenario) SimulationReport {
	report := SimulationReport{Name: "unified-wait"}
	current := map[LayerName]Frame{}
	lastSeen := map[LayerName]int{}
	initialized := false

	for tick := 0; tick <= scenario.Ticks; tick++ {
		at := time.Duration(tick) * displayInterval
		for _, frame := range scenario.submissionsAt(at) {
			current[frame.Layer] = frame
		}
		if len(current) < len(layers) {
			report.MissedTicks = append(report.MissedTicks, missingTick(tick, at, current))
			continue
		}
		waiting := []LayerName(nil)
		if initialized {
			waiting = waitingLayers(current, lastSeen)
		}
		if len(waiting) > 0 {
			report.MissedTicks = append(report.MissedTicks, MissedTick{
				DisplayTick: tick,
				At:          at,
				WaitingOn:   waiting,
			})
			continue
		}
		sources := map[LayerName]LayerSource{}
		for _, layer := range layers {
			frame := current[layer]
			lastSeen[layer] = frame.Sequence
			sources[layer] = sourceFromFrame(frame, at, scenario.Intervals[layer])
		}
		initialized = true
		report.Frames = append(report.Frames, compositeFrame(tick, at, sources))
	}
	report.VisualDeltas = visualDeltas(report.Frames)
	return report
}

func simulateExpiredReuse(scenario Scenario) SimulationReport {
	report := SimulationReport{Name: "expired-layer-reuse"}
	current := map[LayerName]Frame{}
	consumed := map[LayerName]int{}
	initialized := false

	for tick := 0; tick <= scenario.Ticks; tick++ {
		at := time.Duration(tick) * displayInterval
		for _, frame := range scenario.submissionsAt(at) {
			current[frame.Layer] = frame
		}
		if len(current) < len(layers) {
			report.MissedTicks = append(report.MissedTicks, missingTick(tick, at, current))
			continue
		}
		waiting := []LayerName(nil)
		if initialized {
			waiting = waitingLayers(current, consumed)
		}
		hasFreshLayer := len(waiting) < len(layers)
		hasDuplicateSubmission := false
		for _, frame := range scenario.submissionsAt(at) {
			if frame.Duplicate {
				hasDuplicateSubmission = true
			}
		}
		hasExpiredLayer := false
		for _, layer := range waiting {
			if current[layer].Age(at) >= scenario.Intervals[layer] {
				hasExpiredLayer = true
			}
		}
		if initialized && !hasFreshLayer && !hasDuplicateSubmission && !hasExpiredLayer {
			report.MissedTicks = append(report.MissedTicks, MissedTick{
				DisplayTick: tick,
				At:          at,
				WaitingOn:   waiting,
			})
			continue
		}
		sources := map[LayerName]LayerSource{}
		for _, layer := range layers {
			frame := current[layer]
			consumed[layer] = frame.Sequence
			sources[layer] = sourceFromFrame(frame, at, scenario.Intervals[layer])
		}
		initialized = true
		report.Frames = append(report.Frames, compositeFrame(tick, at, sources))
	}
	report.VisualDeltas = visualDeltas(report.Frames)
	return report
}

func waitingLayers(current map[LayerName]Frame, seen map[LayerName]int) []LayerName {
	var waiting []LayerName
	for _, layer := range layers {
		if current[layer].Sequence > seen[layer] {
			continue
		}
		waiting = append(waiting, layer)
	}
	return waiting
}

func missingTick(tick int, at time.Duration, current map[LayerName]Frame) MissedTick {
	var waiting []LayerName
	for _, layer := range layers {
		if _, ok := current[layer]; !ok {
			waiting = append(waiting, layer)
		}
	}
	return MissedTick{DisplayTick: tick, At: at, WaitingOn: waiting}
}

func sourceFromFrame(frame Frame, at time.Duration, interval time.Duration) LayerSource {
	return LayerSource{
		Layer:       frame.Layer,
		Sequence:    frame.Sequence,
		ProducedAt:  frame.ProducedAt,
		AvailableAt: frame.AvailableAt,
		Age:         frame.Age(at),
		Status:      sourceStatus(frame, at, interval),
		Digest:      frame.Digest,
	}
}

func sourceStatus(frame Frame, at time.Duration, interval time.Duration) FrameStatus {
	if frame.AvailableAt == at {
		switch {
		case frame.Duplicate:
			return StatusDuplicate
		case frame.Late:
			return StatusLate
		default:
			return StatusFresh
		}
	}
	if frame.Age(at) > interval {
		if frame.Late {
			return StatusLatePaused
		}
		return StatusPaused
	}
	return StatusReused
}

func compositeFrame(tick int, at time.Duration, sources map[LayerName]LayerSource) CompositeFrame {
	return CompositeFrame{
		DisplayTick: tick,
		At:          at,
		Layers:      sources,
		Digest:      compositeDigest(sources),
	}
}

func compositeDigest(sources map[LayerName]LayerSource) string {
	hash := fnv.New64a()
	for _, layer := range layers {
		_, _ = hash.Write([]byte(string(layer)))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(sources[layer].Digest))
		_, _ = hash.Write([]byte{0})
	}
	return fmt.Sprintf("%016x", hash.Sum64())
}

func visualDeltas(frames []CompositeFrame) []VisualDelta {
	var deltas []VisualDelta
	for index := 1; index < len(frames); index++ {
		previous := frames[index-1]
		current := frames[index]
		var changed []LayerName
		for _, layer := range layers {
			if previous.Layers[layer].Digest != current.Layers[layer].Digest {
				changed = append(changed, layer)
			}
		}
		deltas = append(deltas, VisualDelta{
			FromTick: previous.DisplayTick,
			ToTick:   current.DisplayTick,
			Changed:  changed,
			Digest:   current.Digest,
		})
	}
	return deltas
}

func runAll(scenario Scenario) []SimulationReport {
	reports := []SimulationReport{
		simulateIndependent(scenario),
		simulateUnifiedWait(scenario),
		simulateExpiredReuse(scenario),
	}
	for _, report := range reports {
		sort.Slice(report.MissedTicks, func(i, j int) bool {
			return report.MissedTicks[i].DisplayTick < report.MissedTicks[j].DisplayTick
		})
	}
	return reports
}
