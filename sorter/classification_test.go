package sorter

import (
	"testing"
	"time"
)

func TestTimeoutsUseAbsoluteDeadlineAndRelativeLatency(t *testing.T) {
	base := time.Date(2026, 11, 25, 14, 0, 0, 0, time.UTC)
	submissions := []Submission{
		{
			ID: "absolute-late", Frame: 5, Queue: QueueWorld, Eye: EyeLeft,
			SubmittedAt: base, CompletedAt: base.Add(3 * time.Second),
			Deadline: base.Add(2 * time.Second),
		},
		{
			ID: "relative-late", Frame: 5, Queue: QueueWorld, Eye: EyeRight,
			SubmittedAt: base, CompletedAt: base.Add(5 * time.Second),
		},
		{
			ID: "at-deadline", Frame: 5, Queue: QueueUI, Eye: EyeBoth,
			SubmittedAt: base, CompletedAt: base.Add(2 * time.Second),
			Deadline: base.Add(2 * time.Second),
		},
		{
			ID: "latest", Frame: 6, Queue: QueueHand, Eye: EyeLeft,
			SubmittedAt: base, CompletedAt: base.Add(time.Second),
		},
	}

	report, err := Sort(submissions, Config{
		ReorderWindow: 2,
		Timeout:       2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}
	if report.Counts.Timeouts != 2 {
		t.Fatalf("Timeouts = %d, want 2; rejected = %#v", report.Counts.Timeouts, report.Rejected)
	}
	if len(report.Adopted) != 2 {
		t.Fatalf("len(Adopted) = %d, want 2; rejected = %#v", len(report.Adopted), report.Rejected)
	}

	reasons := reasonsByID(report)
	if reasons["absolute-late"] != ReasonTimeout {
		t.Errorf("absolute-late reason = %q, want timeout", reasons["absolute-late"])
	}
	if reasons["relative-late"] != ReasonTimeout {
		t.Errorf("relative-late reason = %q, want timeout", reasons["relative-late"])
	}
	if _, rejected := reasons["at-deadline"]; rejected {
		t.Errorf("at-deadline should be accepted, rejected reason = %q", reasons["at-deadline"])
	}
}

func TestHistoricalBackfillIsNotMadeNewestByLateArrival(t *testing.T) {
	base := time.Date(2026, 11, 25, 15, 0, 0, 0, time.UTC)
	submissions := []Submission{
		{ID: "f8-last-arrival", Frame: 8, Queue: QueueWorld, Eye: EyeLeft, SubmittedAt: base, CompletedAt: base.Add(100 * time.Second)},
		{ID: "f11-first-arrival", Frame: 11, Queue: QueueUI, Eye: EyeBoth, SubmittedAt: base, CompletedAt: base.Add(time.Second)},
		{ID: "f9-middle", Frame: 9, Queue: QueueHand, Eye: EyeRight, SubmittedAt: base, CompletedAt: base.Add(2 * time.Second)},
		{ID: "f10-middle", Frame: 10, Queue: QueuePassthrough, Eye: EyeLeft, SubmittedAt: base, CompletedAt: base.Add(3 * time.Second)},
	}

	report, err := Sort(submissions, Config{ReorderWindow: 2})
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}
	if report.LatestFrame != 11 {
		t.Fatalf("LatestFrame = %d, want 11", report.LatestFrame)
	}
	if report.Counts.HistoricalBackfills != 1 {
		t.Fatalf("HistoricalBackfills = %d, want 1", report.Counts.HistoricalBackfills)
	}
	reason, rejected := reasonsByID(report)["f8-last-arrival"]
	if !rejected || reason != ReasonHistoricalBackfill {
		t.Fatalf("f8-last-arrival reason = %q, rejected = %v; want historical_backfill", reason, rejected)
	}
	if len(report.Adopted) != 3 || report.Adopted[len(report.Adopted)-1].Frame != 11 {
		t.Fatalf("unexpected adopted order: %#v", report.Adopted)
	}
}

func TestSortAtPreventsOutlierFromRedefiningFrontier(t *testing.T) {
	base := time.Date(2026, 11, 25, 16, 0, 0, 0, time.UTC)
	submissions := []Submission{
		{ID: "valid", Frame: 12, Queue: QueueWorld, Eye: EyeLeft, SubmittedAt: base, CompletedAt: base.Add(2 * time.Second)},
		{ID: "late-outlier", Frame: 99, Queue: QueueWorld, Eye: EyeRight, SubmittedAt: base, CompletedAt: base.Add(1 * time.Second)},
	}
	report, err := SortAt(submissions, 12, Config{ReorderWindow: 1})
	if err != nil {
		t.Fatalf("SortAt() error = %v", err)
	}
	if report.LatestFrame != 12 {
		t.Fatalf("LatestFrame = %d, want 12", report.LatestFrame)
	}
	if report.Counts.FutureFrames != 1 {
		t.Fatalf("FutureFrames = %d, want 1", report.Counts.FutureFrames)
	}
	reason, rejected := reasonsByID(report)["late-outlier"]
	if !rejected || reason != ReasonFutureFrame {
		t.Fatalf("late-outlier reason = %q, rejected = %v; want future_frame", reason, rejected)
	}
}

func TestInvalidQueueAndEyeAreRecorded(t *testing.T) {
	base := time.Date(2026, 11, 25, 17, 0, 0, 0, time.UTC)
	submissions := []Submission{
		{ID: "bad-queue", Frame: 4, Queue: QueueUnknown, Eye: EyeLeft, SubmittedAt: base, CompletedAt: base.Add(time.Second)},
		{ID: "bad-eye", Frame: 4, Queue: QueueWorld, Eye: EyeUnknown, SubmittedAt: base, CompletedAt: base.Add(time.Second)},
		{ID: "good", Frame: 4, Queue: QueueWorld, Eye: EyeBoth, SubmittedAt: base, CompletedAt: base.Add(time.Second)},
	}
	report, err := Sort(submissions, Config{ReorderWindow: 1})
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}
	if report.Counts.Invalid != 2 {
		t.Fatalf("Invalid = %d, want 2; rejected = %#v", report.Counts.Invalid, report.Rejected)
	}
	reasons := reasonsByID(report)
	if reasons["bad-queue"] != ReasonInvalidQueue {
		t.Errorf("bad-queue reason = %q, want invalid_queue", reasons["bad-queue"])
	}
	if reasons["bad-eye"] != ReasonInvalidEye {
		t.Errorf("bad-eye reason = %q, want invalid_eye", reasons["bad-eye"])
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "negative timeout", cfg: Config{Timeout: -time.Second}},
		{name: "duplicate queue", cfg: Config{QueueOrder: []Queue{QueueWorld, QueueWorld}}},
		{name: "unknown queue", cfg: Config{QueueOrder: []Queue{QueueUnknown}}},
		{name: "duplicate eye", cfg: Config{EyeOrder: []Eye{EyeLeft, EyeLeft}}},
		{name: "unknown eye", cfg: Config{EyeOrder: []Eye{EyeUnknown}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Sort(nil, test.cfg); err == nil {
				t.Fatal("Sort() error = nil, want configuration error")
			}
		})
	}
}

func reasonsByID(report Report) map[string]RejectReason {
	reasons := make(map[string]RejectReason, len(report.Rejected))
	for _, rejected := range report.Rejected {
		reasons[rejected.ID] = rejected.Reason
	}
	return reasons
}
