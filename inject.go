package notify

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// Inject accepts one notification or world-frame state at update.At.
//
// It never overwrites a newer revision with an older one. Inputs at or before
// the open frame deadline target that frame; all later inputs target a later
// frame and are kept out of the current committed snapshot.
func (p *Pipeline) Inject(update Update) (*InputRecord, error) {
	if update.Source == "" {
		update.Source = SourceNotification
	}
	if update.At < 0 {
		return nil, fmt.Errorf("update time must not be negative: %s", update.At)
	}
	if err := update.State.validate(); err != nil {
		return nil, err
	}

	p.seq++
	id := update.State.ID
	previous, known := p.latest[id]
	record := &InputRecord{
		Seq:    p.seq,
		Update: update,
		Change: DiffStates(previous, update.State),
	}

	if p.open && update.At <= p.schedule.Deadline {
		record.Timing = TimingBeforeDeadline
		record.TargetFrame = p.schedule.Number
	} else {
		record.Timing = TimingLate
		if p.open {
			record.TargetFrame = p.schedule.Number + 1
		} else {
			record.TargetFrame = p.expectedFrame
		}
	}

	if known && update.State.Revision <= previous.Revision {
		record.Outcome = OutcomeStale
		if update.State.Revision < previous.Revision {
			record.Reason = fmt.Sprintf(
				"revision %d is older than latest revision %d",
				update.State.Revision, previous.Revision,
			)
		} else {
			record.Reason = fmt.Sprintf("revision %d is duplicate", update.State.Revision)
		}
		p.recordEvent(record)
		return record, nil
	}

	p.latest[id] = update.State
	record.Outcome = OutcomeAccepted

	if record.Timing == TimingBeforeDeadline {
		if oldRecord := p.stagedRec[id]; oldRecord != nil {
			oldRecord.Outcome = OutcomeSuperseded
			oldRecord.Reason = fmt.Sprintf("superseded by seq %d/revision %d", record.Seq, update.State.Revision)
		}
		p.staged[id] = update.State
		p.stagedRec[id] = record
	} else {
		if oldRecord := p.queuedRec[id]; oldRecord != nil {
			oldRecord.Outcome = OutcomeSuperseded
			oldRecord.Reason = fmt.Sprintf("superseded by seq %d/revision %d", record.Seq, update.State.Revision)
		}
		p.queued[id] = update.State
		p.queuedRec[id] = record
	}

	p.recordEvent(record)
	return record, nil
}

func (p *Pipeline) recordEvent(record *InputRecord) {
	p.events = append(p.events, record)
	switch {
	case p.open:
		p.arrivals = append(p.arrivals, record)
	case p.last != nil:
		p.last.Arrivals = append(p.last.Arrivals, record)
	}
}

// Commit creates the immutable frame snapshot for the open frame.
func (p *Pipeline) Commit() (*Frame, error) {
	if !p.open {
		return nil, errors.New("cannot commit: no open frame")
	}

	frame := &Frame{
		Number:        p.schedule.Number,
		Deadline:      p.schedule.Deadline,
		CommitAt:      p.schedule.CommitAt,
		ScanoutAt:     p.schedule.ScanoutAt,
		States:        cloneStates(p.base),
		Diffs:         make(map[string]Diff),
		WorldReceipts: append([]*WorldReceipt(nil), p.frameReceipts...),
	}

	for id, state := range p.staged {
		old := frame.States[id]
		frame.States[id] = state
		frame.Diffs[id] = DiffStates(old, state)

		if record := p.stagedRec[id]; record != nil {
			record.Outcome = OutcomeCommitted
			record.Reason = ""
			frame.Inputs = append(frame.Inputs, record)
		}
	}
	sort.Slice(frame.Inputs, func(i, j int) bool {
		return frame.Inputs[i].Seq < frame.Inputs[j].Seq
	})
	frame.Arrivals = append([]*InputRecord(nil), p.arrivals...)

	p.frames = append(p.frames, frame)
	p.base = cloneStates(frame.States)
	p.last = frame
	p.expectedFrame = frame.Number + 1
	p.open = false
	p.arrivals = nil
	p.frameReceipts = nil

	return p.Scanout(frame.Number)
}

// Scanout returns a defensive copy of the committed frame shown at scanout.
func (p *Pipeline) Scanout(number int) (*Frame, error) {
	for _, frame := range p.frames {
		if frame.Number == number {
			return cloneFrame(frame), nil
		}
	}
	return nil, fmt.Errorf("frame %d has not been committed", number)
}

// Latest returns the newest accepted revision, including input not committed.
func (p *Pipeline) Latest(id string) (State, bool) {
	state, ok := p.latest[id]
	return state, ok
}

// Events returns all injection records in injection order.
func (p *Pipeline) Events() []*InputRecord {
	return append([]*InputRecord(nil), p.events...)
}

// Frames returns defensive copies of all committed frames.
func (p *Pipeline) Frames() []*Frame {
	frames := make([]*Frame, 0, len(p.frames))
	for _, frame := range p.frames {
		frames = append(frames, cloneFrame(frame))
	}
	return frames
}

func cloneFrame(frame *Frame) *Frame {
	if frame == nil {
		return nil
	}
	copyFrame := *frame
	copyFrame.States = cloneStates(frame.States)
	copyFrame.Diffs = make(map[string]Diff, len(frame.Diffs))
	for id, diff := range frame.Diffs {
		copyFrame.Diffs[id] = diff
	}
	copyFrame.Inputs = append([]*InputRecord(nil), frame.Inputs...)
	copyFrame.Arrivals = append([]*InputRecord(nil), frame.Arrivals...)
	copyFrame.WorldReceipts = append([]*WorldReceipt(nil), frame.WorldReceipts...)
	return &copyFrame
}

// Millis is a small helper for readable durations in scenario construction.
func Millis(value int64) time.Duration {
	return time.Duration(value) * time.Millisecond
}
