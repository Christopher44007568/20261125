package notify

import (
	"fmt"
	"time"
)

// WorldFrame is a submitted world snapshot. Its notification copies may be
// stale because world simulation and system notification updates are separate.
type WorldFrame struct {
	Number        int           `json:"number"`
	SubmittedAt   time.Duration `json:"submitted_at_ns"`
	Notifications []State       `json:"notifications"`
}

// WorldVerdict explains whether one notification snapshot was allowed to merge.
type WorldVerdict struct {
	ID                string `json:"id"`
	WorldRevision     int64  `json:"world_revision"`
	LatestRevision    int64  `json:"latest_revision"`
	OlderThanLatest   bool   `json:"older_than_latest"`
	EqualToLatest     bool   `json:"equal_to_latest"`
	Applied           bool   `json:"applied"`
	RegressionBlocked bool   `json:"regression_blocked"`
	InputSeq          int    `json:"input_seq"`
	TargetFrame       int    `json:"target_frame"`
	Reason            string `json:"reason"`
}

// WorldReceipt records version guard decisions for a submitted world frame.
type WorldReceipt struct {
	WorldFrameNumber int            `json:"world_frame_number"`
	ContextFrame     int            `json:"context_frame"`
	SubmittedAt      time.Duration  `json:"submitted_at_ns"`
	Verdicts         []WorldVerdict `json:"verdicts"`
}

// SubmitWorldFrame feeds embedded notification snapshots through the same
// monotonic guard and frame deadline path as direct notification updates.
func (p *Pipeline) SubmitWorldFrame(frame WorldFrame) (*WorldReceipt, error) {
	contextFrame := 0
	switch {
	case p.open:
		contextFrame = p.schedule.Number
	case p.last != nil:
		contextFrame = p.last.Number
	default:
		contextFrame = p.expectedFrame
	}

	receipt := &WorldReceipt{
		WorldFrameNumber: frame.Number,
		ContextFrame:     contextFrame,
		SubmittedAt:      frame.SubmittedAt,
		Verdicts:         make([]WorldVerdict, 0, len(frame.Notifications)),
	}

	for _, snapshot := range frame.Notifications {
		record, err := p.Inject(Update{
			At:     frame.SubmittedAt,
			Source: SourceWorldFrame,
			State:  snapshot,
		})
		if err != nil {
			return nil, fmt.Errorf("world frame %d: %w", frame.Number, err)
		}

		latest, _ := p.Latest(snapshot.ID)
		verdict := WorldVerdict{
			ID:                snapshot.ID,
			WorldRevision:     snapshot.Revision,
			LatestRevision:    latest.Revision,
			OlderThanLatest:   snapshot.Revision < latest.Revision,
			EqualToLatest:     snapshot.Revision == latest.Revision,
			Applied:           record.Outcome != OutcomeStale,
			RegressionBlocked: record.Outcome == OutcomeStale && snapshot.Revision < latest.Revision,
			InputSeq:          record.Seq,
			TargetFrame:       record.TargetFrame,
			Reason:            record.Reason,
		}
		if verdict.Applied && verdict.Reason == "" {
			verdict.Reason = "newer snapshot accepted into frame schedule"
		}
		receipt.Verdicts = append(receipt.Verdicts, verdict)
	}

	p.receipts = append(p.receipts, receipt)
	if p.open {
		p.frameReceipts = append(p.frameReceipts, receipt)
	} else if p.last != nil {
		p.last.WorldReceipts = append(p.last.WorldReceipts, receipt)
	}
	return receipt, nil
}

// WorldReceipts returns all world-frame version guard receipts.
func (p *Pipeline) WorldReceipts() []*WorldReceipt {
	return append([]*WorldReceipt(nil), p.receipts...)
}
