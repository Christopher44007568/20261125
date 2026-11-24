package main

import (
	"fmt"
	"sort"
	"time"
)

type Scenario struct {
	Ticks       int
	Intervals   map[LayerName]time.Duration
	submissions []submission
}

func buildScenario() Scenario {
	intervals := map[LayerName]time.Duration{
		World: displayInterval,
		UI:    2 * displayInterval,
		Hand:  3 * displayInterval,
	}
	scenario := Scenario{Ticks: 12, Intervals: intervals}

	for tick := 0; tick <= 12; tick++ {
		at := time.Duration(tick) * displayInterval
		scenario.add(Frame{
			Layer:       World,
			Sequence:    tick,
			ProducedAt:  at,
			AvailableAt: at,
			ExpectedAt:  at,
			Digest:      digestFor(World, tick),
		})
	}

	uiSequence := 0
	for tick := 0; tick <= 12; tick += 2 {
		at := time.Duration(tick) * displayInterval
		frame := Frame{
			Layer:       UI,
			Sequence:    uiSequence,
			ProducedAt:  at,
			AvailableAt: at,
			ExpectedAt:  at,
			Digest:      digestFor(UI, uiSequence),
		}
		if tick == 12 {
			frame.Sequence = uiSequence - 1
			frame.ProducedAt = 10 * displayInterval
			frame.ExpectedAt = 10 * displayInterval
			frame.Late = true
			frame.Digest = digestFor(UI, frame.Sequence)
		}
		if tick != 8 && tick != 10 {
			scenario.add(frame)
		}
		uiSequence++
	}

	for tick := 0; tick <= 12; tick += 3 {
		at := time.Duration(tick) * displayInterval
		sequence := tick / 3
		frame := Frame{
			Layer:       Hand,
			Sequence:    sequence,
			ProducedAt:  at,
			AvailableAt: at,
			ExpectedAt:  at,
			Digest:      digestFor(Hand, sequence),
		}
		if tick == 6 {
			frame.Sequence = 1
			frame.ProducedAt = 3 * displayInterval
			frame.ExpectedAt = 3 * displayInterval
			frame.Duplicate = true
			frame.Digest = digestFor(Hand, 1)
		}
		scenario.add(frame)
	}

	scenario.add(Frame{
		Layer:       Hand,
		Sequence:    1,
		ProducedAt:  3 * displayInterval,
		AvailableAt: 4 * displayInterval,
		ExpectedAt:  3 * displayInterval,
		Late:        true,
		Digest:      digestFor(Hand, 1),
	})

	sort.SliceStable(scenario.submissions, func(i, j int) bool {
		if scenario.submissions[i].at == scenario.submissions[j].at {
			return scenario.submissions[i].frame.Layer < scenario.submissions[j].frame.Layer
		}
		return scenario.submissions[i].at < scenario.submissions[j].at
	})
	return scenario
}

func (scenario *Scenario) add(frame Frame) {
	scenario.submissions = append(scenario.submissions, submission{
		at:    frame.AvailableAt,
		frame: frame,
	})
}

func (scenario Scenario) submissionsAt(at time.Duration) []Frame {
	var frames []Frame
	for _, item := range scenario.submissions {
		if item.at == at {
			frames = append(frames, item.frame)
		}
	}
	return frames
}

func digestFor(layer LayerName, sequence int) string {
	return fmt.Sprintf("%s:%d", layer, sequence)
}
