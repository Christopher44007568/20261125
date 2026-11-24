package sorter

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestFinalOrderUsesLogicalFrameNotCompletionOrder(t *testing.T) {
	base := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	submissions := []Submission{
		{ID: "frame12-pass-right", Frame: 12, Queue: QueuePassthrough, Eye: EyeRight, SubmittedAt: base.Add(time.Second), CompletedAt: base.Add(1 * time.Second)},
		{ID: "frame12-world-left", Frame: 12, Queue: QueueWorld, Eye: EyeLeft, SubmittedAt: base, CompletedAt: base.Add(10 * time.Second)},
		{ID: "frame12-world-right", Frame: 12, Queue: QueueWorld, Eye: EyeRight, SubmittedAt: base, CompletedAt: base.Add(3 * time.Second)},
		{ID: "frame11-hand-both", Frame: 11, Queue: QueueHand, Eye: EyeBoth, SubmittedAt: base, CompletedAt: base.Add(2 * time.Second)},
		{ID: "frame10-ui-left", Frame: 10, Queue: QueueUI, Eye: EyeLeft, SubmittedAt: base, CompletedAt: base.Add(20 * time.Second)},
	}

	report, err := Sort(submissions, Config{ReorderWindow: 2})
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}
	if report.LatestFrame != 12 {
		t.Fatalf("LatestFrame = %d, want 12", report.LatestFrame)
	}
	if got, want := len(report.Adopted), 5; got != want {
		t.Fatalf("len(Adopted) = %d, want %d; rejected = %#v", got, want, report.Rejected)
	}

	wantOrder := []string{
		"frame10-ui-left",
		"frame11-hand-both",
		"frame12-world-left",
		"frame12-world-right",
		"frame12-pass-right",
	}
	for index, want := range wantOrder {
		got := report.Adopted[index]
		if got.ID != want {
			t.Errorf("Adopted[%d].ID = %q, want %q", index, got.ID, want)
		}
		if got.Order != index+1 {
			t.Errorf("Adopted[%d].Order = %d, want %d", index, got.Order, index+1)
		}
	}

	if !report.Adopted[0].OutOfOrder {
		t.Error("last-arriving frame 10 should be marked out of order")
	}
	if !report.Adopted[1].OutOfOrder {
		t.Error("frame 11 completed after frame 12 should be marked out of order")
	}
	if report.Counts.OutOfOrderAdopted != 2 {
		t.Errorf("OutOfOrderAdopted = %d, want 2", report.Counts.OutOfOrderAdopted)
	}
	if report.Adopted[0].Eye != EyeLeft {
		t.Errorf("frame 10 eye = %s, want left", report.Adopted[0].Eye)
	}
}

func TestDuplicateSelectionAndPermutationDeterminism(t *testing.T) {
	base := time.Date(2026, 11, 25, 11, 0, 0, 0, time.UTC)
	submissions := []Submission{
		{ID: "late-copy", Frame: 7, Queue: QueueWorld, Eye: EyeLeft, SubmittedAt: base.Add(5 * time.Second), CompletedAt: base.Add(6 * time.Second)},
		{ID: "early-copy", Frame: 7, Queue: QueueWorld, Eye: EyeLeft, SubmittedAt: base.Add(2 * time.Second), CompletedAt: base.Add(3 * time.Second)},
		{ID: "early-copy", Frame: 7, Queue: QueueWorld, Eye: EyeLeft, SubmittedAt: base.Add(2 * time.Second), CompletedAt: base.Add(3 * time.Second)},
		{ID: "right-eye", Frame: 7, Queue: QueueWorld, Eye: EyeRight, SubmittedAt: base.Add(4 * time.Second), CompletedAt: base.Add(5 * time.Second)},
		{ID: "ui-frame", Frame: 8, Queue: QueueUI, Eye: EyeBoth, SubmittedAt: base.Add(7 * time.Second), CompletedAt: base.Add(8 * time.Second)},
	}

	baseline, err := Sort(submissions, Config{ReorderWindow: 2})
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}
	if len(baseline.Adopted) != 3 {
		t.Fatalf("len(Adopted) = %d, want 3; rejected = %#v", len(baseline.Adopted), baseline.Rejected)
	}
	if baseline.Adopted[0].ID != "early-copy" || baseline.Adopted[1].ID != "right-eye" {
		t.Fatalf("unexpected canonical entries: %#v", baseline.Adopted)
	}
	if baseline.Counts.Duplicates != 2 {
		t.Fatalf("Duplicates = %d, want 2", baseline.Counts.Duplicates)
	}
	for _, rejected := range baseline.Rejected {
		if rejected.Reason != ReasonDuplicate {
			t.Errorf("rejected reason = %s, want duplicate", rejected.Reason)
		}
		if rejected.Eye != EyeLeft {
			t.Errorf("duplicate eye = %s, want left", rejected.Eye)
		}
	}

	want := reportJSON(t, baseline)
	permute(submissions, func(permutation []Submission) {
		got, err := Sort(permutation, Config{ReorderWindow: 2})
		if err != nil {
			t.Fatalf("Sort() error = %v", err)
		}
		if encoded := reportJSON(t, got); encoded != want {
			t.Fatalf("report changed with input permutation\nwant: %s\ngot:  %s", want, encoded)
		}
	})
}

func TestCrossEyeInterleavingHasStableOrder(t *testing.T) {
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	submissions := []Submission{
		{ID: "world-right", Frame: 20, Queue: QueueWorld, Eye: EyeRight, SubmittedAt: base, CompletedAt: base.Add(2 * time.Second)},
		{ID: "world-left", Frame: 20, Queue: QueueWorld, Eye: EyeLeft, SubmittedAt: base, CompletedAt: base.Add(1 * time.Second)},
		{ID: "ui-right", Frame: 20, Queue: QueueUI, Eye: EyeRight, SubmittedAt: base, CompletedAt: base.Add(3 * time.Second)},
		{ID: "ui-left", Frame: 20, Queue: QueueUI, Eye: EyeLeft, SubmittedAt: base, CompletedAt: base.Add(4 * time.Second)},
	}
	cfg := Config{
		ReorderWindow: 1,
		QueueOrder:    []Queue{QueueUI, QueueWorld},
		EyeOrder:      []Eye{EyeRight, EyeLeft},
	}
	report, err := Sort(submissions, cfg)
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}
	want := []string{"ui-right", "ui-left", "world-right", "world-left"}
	for index, id := range want {
		if got := report.Adopted[index].ID; got != id {
			t.Errorf("Adopted[%d].ID = %q, want %q", index, got, id)
		}
	}
	for _, adopted := range report.Adopted {
		if adopted.OutOfOrder {
			t.Errorf("%s should not be out of order: all entries are the same frame", adopted.ID)
		}
	}
}

func reportJSON(t *testing.T, report Report) string {
	t.Helper()
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal(Report) error = %v", err)
	}
	return string(encoded)
}

func permute(submissions []Submission, visit func([]Submission)) {
	work := append([]Submission(nil), submissions...)
	var generate func(int)
	generate = func(index int) {
		if index == len(work) {
			visit(append([]Submission(nil), work...))
			return
		}
		for candidate := index; candidate < len(work); candidate++ {
			work[index], work[candidate] = work[candidate], work[index]
			generate(index + 1)
			work[index], work[candidate] = work[candidate], work[index]
		}
	}
	generate(0)
}

func TestReportsAreDeepComparableAcrossRuns(t *testing.T) {
	base := time.Date(2026, 11, 25, 13, 0, 0, 0, time.UTC)
	submissions := []Submission{
		{ID: "a", Frame: 3, Queue: QueueWorld, Eye: EyeLeft, SubmittedAt: base, CompletedAt: base.Add(time.Second)},
	}
	first, err := Sort(submissions, Config{ReorderWindow: 1})
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}
	second, err := Sort(submissions, Config{ReorderWindow: 1})
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("reports differ: %#v vs %#v", first, second)
	}
}
