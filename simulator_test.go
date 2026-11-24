package main

import (
	"testing"
	"time"
)

func findReport(reports []SimulationReport, name string) SimulationReport {
	for _, report := range reports {
		if report.Name == name {
			return report
		}
	}
	panic("missing report " + name)
}

func findFrame(report SimulationReport, tick int) CompositeFrame {
	for _, frame := range report.Frames {
		if frame.DisplayTick == tick {
			return frame
		}
	}
	panic("missing frame")
}

func findDelta(report SimulationReport, tick int) VisualDelta {
	for _, delta := range report.VisualDeltas {
		if delta.ToTick == tick {
			return delta
		}
	}
	panic("missing delta")
}

func TestIndependentRefreshNeverBlocks(t *testing.T) {
	report := findReport(runAll(buildScenario()), "independent-refresh")

	if len(report.Frames) != 13 {
		t.Fatalf("outputs = %d, want 13", len(report.Frames))
	}
	if len(report.MissedTicks) != 0 {
		t.Fatalf("blocking ticks = %d, want 0", len(report.MissedTicks))
	}

	frame := findFrame(report, 9)
	if frame.Layers[UI].Status != StatusPaused {
		t.Fatalf("ui status = %q, want %q", frame.Layers[UI].Status, StatusPaused)
	}
	if frame.Layers[UI].Age != 3*displayInterval {
		t.Fatalf("ui age = %v, want %v", frame.Layers[UI].Age, 3*displayInterval)
	}

	frame = findFrame(report, 6)
	if frame.Layers[Hand].Status != StatusDuplicate {
		t.Fatalf("hand status = %q, want %q", frame.Layers[Hand].Status, StatusDuplicate)
	}

	frame = findFrame(report, 4)
	if frame.Layers[Hand].Status != StatusLate {
		t.Fatalf("hand status = %q, want %q", frame.Layers[Hand].Status, StatusLate)
	}

	delta := findDelta(report, 9)
	if len(delta.Changed) != 2 || delta.Changed[0] != World || delta.Changed[1] != Hand {
		t.Fatalf("changed layers = %v, want [world hand]", delta.Changed)
	}
}

func TestUnifiedWaitBlocksUntilEveryLayerHasFreshFrame(t *testing.T) {
	report := findReport(runAll(buildScenario()), "unified-wait")

	if len(report.Frames) != 4 {
		t.Fatalf("outputs = %d, want 4", len(report.Frames))
	}
	if len(report.MissedTicks) != 9 {
		t.Fatalf("blocking ticks = %d, want 9", len(report.MissedTicks))
	}

	frame := findFrame(report, 12)
	if frame.Layers[UI].Sequence != 5 {
		t.Fatalf("ui sequence = %d, want 5", frame.Layers[UI].Sequence)
	}
	if frame.Layers[UI].Status != StatusLate {
		t.Fatalf("ui status = %q, want %q", frame.Layers[UI].Status, StatusLate)
	}
	if frame.Layers[Hand].Status != StatusFresh {
		t.Fatalf("hand status = %q, want %q", frame.Layers[Hand].Status, StatusFresh)
	}

	for _, missed := range report.MissedTicks {
		if len(missed.WaitingOn) == 0 {
			t.Fatalf("tick %d blocked without a waiting layer", missed.DisplayTick)
		}
	}
}

func TestExpiredReuseDoesNotBlockOnStaleLayers(t *testing.T) {
	report := findReport(runAll(buildScenario()), "expired-layer-reuse")

	if len(report.Frames) != 13 {
		t.Fatalf("outputs = %d, want 13", len(report.Frames))
	}
	if len(report.MissedTicks) != 0 {
		t.Fatalf("blocking ticks = %d, want 0", len(report.MissedTicks))
	}

	frame := findFrame(report, 9)
	if frame.Layers[UI].Status != StatusPaused {
		t.Fatalf("ui status = %q, want %q", frame.Layers[UI].Status, StatusPaused)
	}
	if frame.Layers[UI].Age != 3*displayInterval {
		t.Fatalf("ui age = %v, want %v", frame.Layers[UI].Age, 3*displayInterval)
	}

	frame = findFrame(report, 10)
	if frame.Layers[Hand].Status != StatusReused {
		t.Fatalf("hand status = %q, want %q", frame.Layers[Hand].Status, StatusReused)
	}

	frame = findFrame(report, 12)
	if frame.Layers[UI].Status != StatusLate {
		t.Fatalf("ui status = %q, want %q", frame.Layers[UI].Status, StatusLate)
	}

	delta := findDelta(report, 10)
	if len(delta.Changed) != 1 || delta.Changed[0] != World {
		t.Fatalf("changed layers = %v, want [world]", delta.Changed)
	}
}

func TestEveryOutputRecordsCompleteProvenance(t *testing.T) {
	for _, report := range runAll(buildScenario()) {
		for _, frame := range report.Frames {
			if len(frame.Layers) != len(layers) {
				t.Fatalf("%s tick %d has %d layers, want %d",
					report.Name, frame.DisplayTick, len(frame.Layers), len(layers))
			}
			if frame.Digest == "" {
				t.Fatalf("%s tick %d has empty digest", report.Name, frame.DisplayTick)
			}
			for _, layer := range layers {
				source := frame.Layers[layer]
				if source.AvailableAt > frame.At || source.Age < 0 {
					t.Fatalf("%s tick %d has invalid age for %s",
						report.Name, frame.DisplayTick, layer)
				}
			}
		}
	}
}

func TestLateFrameCanBecomeLatePaused(t *testing.T) {
	frame := Frame{
		Layer:       Hand,
		Sequence:    1,
		ProducedAt:  3 * displayInterval,
		AvailableAt: 4 * displayInterval,
		Late:        true,
	}

	status := sourceStatus(frame, 8*displayInterval, 3*displayInterval)
	if status != StatusLatePaused {
		t.Fatalf("status = %q, want %q", status, StatusLatePaused)
	}
}

func TestExpiredReuseOutputsWhenAllSubmissionsAreDuplicates(t *testing.T) {
	scenario := Scenario{
		Ticks: 1,
		Intervals: map[LayerName]time.Duration{
			World: displayInterval,
			UI:    2 * displayInterval,
			Hand:  3 * displayInterval,
		},
	}
	for tick := 0; tick <= 1; tick++ {
		for _, layer := range layers {
			scenario.add(Frame{
				Layer:       layer,
				Sequence:    0,
				ProducedAt:  0,
				AvailableAt: time.Duration(tick) * displayInterval,
				ExpectedAt:  0,
				Duplicate:   tick == 1,
				Digest:      digestFor(layer, 0),
			})
		}
	}

	reports := runAll(scenario)
	unified := findReport(reports, "unified-wait")
	expired := findReport(reports, "expired-layer-reuse")

	if len(unified.Frames) != 1 || len(unified.MissedTicks) != 1 {
		t.Fatalf("unified outputs=%d blocked=%d, want 1 and 1",
			len(unified.Frames), len(unified.MissedTicks))
	}
	if len(expired.Frames) != 2 || len(expired.MissedTicks) != 0 {
		t.Fatalf("expired outputs=%d blocked=%d, want 2 and 0",
			len(expired.Frames), len(expired.MissedTicks))
	}

	frame := findFrame(expired, 1)
	for _, layer := range layers {
		if frame.Layers[layer].Status != StatusDuplicate {
			t.Fatalf("%s status = %q, want %q",
				layer, frame.Layers[layer].Status, StatusDuplicate)
		}
	}
}
