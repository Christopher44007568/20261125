package main

import "time"

const displayInterval = time.Millisecond * 1000 / 90

type LayerName string

const (
	World LayerName = "world"
	UI    LayerName = "ui"
	Hand  LayerName = "hand"
)

type FrameStatus string

const (
	StatusFresh      FrameStatus = "fresh"
	StatusReused     FrameStatus = "reused"
	StatusDuplicate  FrameStatus = "duplicate"
	StatusPaused     FrameStatus = "paused"
	StatusLate       FrameStatus = "late"
	StatusLatePaused FrameStatus = "late-paused"
)

type Frame struct {
	Layer       LayerName
	Sequence    int
	ProducedAt  time.Duration
	AvailableAt time.Duration
	ExpectedAt  time.Duration
	Late        bool
	Duplicate   bool
	Digest      string
}

func (frame Frame) Age(at time.Duration) time.Duration {
	return at - frame.AvailableAt
}

type LayerSource struct {
	Layer       LayerName
	Sequence    int
	ProducedAt  time.Duration
	AvailableAt time.Duration
	Age         time.Duration
	Status      FrameStatus
	Digest      string
}

type CompositeFrame struct {
	DisplayTick int
	At          time.Duration
	Layers      map[LayerName]LayerSource
	Digest      string
}

type MissedTick struct {
	DisplayTick int
	At          time.Duration
	WaitingOn   []LayerName
}

type SimulationReport struct {
	Name         string
	Frames       []CompositeFrame
	MissedTicks  []MissedTick
	VisualDeltas []VisualDelta
}

type VisualDelta struct {
	FromTick int
	ToTick   int
	Changed  []LayerName
	Digest   string
}

type submission struct {
	at    time.Duration
	frame Frame
}
