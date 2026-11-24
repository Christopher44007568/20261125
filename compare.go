package xrpath

import (
	"fmt"
	"sort"
)

type Mismatch struct {
	Frame     int    `json:"frame"`
	Subsystem string `json:"subsystem"`
	Item      string `json:"item,omitempty"`
	Field     string `json:"field,omitempty"`
	Expected  any    `json:"expected,omitempty"`
	Actual    any    `json:"actual,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type Report struct {
	OK                 bool           `json:"ok"`
	FrameCountExpected int            `json:"frame_count_expected"`
	FrameCountActual   int            `json:"frame_count_actual"`
	CheckCount         int            `json:"check_count"`
	MismatchCount      int            `json:"mismatch_count"`
	ChecksBySubsystem  map[string]int `json:"checks_by_subsystem"`
	MismatchesBySystem map[string]int `json:"mismatches_by_subsystem"`
	FirstDivergence    *Mismatch      `json:"first_divergence,omitempty"`
	Mismatches         []Mismatch     `json:"mismatches"`
}

type comparator struct {
	checks    map[string]int
	mismatchN map[string]int
	items     []Mismatch
}

func NewReport() Report {
	return Report{
		OK:                 true,
		ChecksBySubsystem:  map[string]int{},
		MismatchesBySystem: map[string]int{},
		Mismatches:         []Mismatch{},
	}
}

func CompareRecords(expected, actual []FrameRecord) Report {
	report := NewReport()
	report.FrameCountExpected = len(expected)
	report.FrameCountActual = len(actual)
	c := &comparator{checks: report.ChecksBySubsystem, mismatchN: report.MismatchesBySystem}

	if len(expected) != len(actual) {
		c.add(Mismatch{
			Frame: minInt(len(expected), len(actual)), Subsystem: SubsystemSequence,
			Field: "frame_count", Expected: len(expected), Actual: len(actual),
			Detail: "rerun produced a different number of frames",
		})
	}

	n := minInt(len(expected), len(actual))
	for i := 0; i < n; i++ {
		want := expected[i]
		got := actual[i]
		frame := want.Frame
		if got.Frame != want.Frame {
			c.add(Mismatch{
				Frame: frame, Subsystem: SubsystemSequence, Field: "frame_index",
				Expected: want.Frame, Actual: got.Frame,
			})
			frame = minInt(want.Frame, got.Frame)
		}
		compareLifecycle(c, frame, want, got)
		compareParameters(c, frame, want.Parameters, got.Parameters)
		compareTracking(c, frame, want.Tracking, got.Tracking)
		comparePose(c, frame, want.Pose, got.Pose)
		compareResources(c, frame, want.Resources, got.Resources)
		compareVisibility(c, frame, want.Visibility, got.Visibility)
		compareReprojection(c, frame, want.Reprojection, got.Reprojection)
		compareImageRefs(c, frame, want.Images, got.Images)
		compareFrameHash(c, frame, want.FrameSHA256, got.FrameSHA256)
	}

	report.CheckCount = c.totalChecks()
	report.Mismatches = c.items
	report.MismatchCount = len(c.items)
	report.MismatchesBySystem = c.mismatchN
	report.ChecksBySubsystem = c.checks
	if len(c.items) > 0 {
		report.OK = false
		first := c.items[0]
		report.FirstDivergence = &first
	}
	return report
}

func CompareRuns(expected, actual RunResult) Report {
	report := CompareRecords(expected.Records, actual.Records)
	if expected.Images == nil || actual.Images == nil {
		return report
	}

	pixelMismatches := []Mismatch{}
	for frame := 0; frame < minInt(len(expected.Records), len(actual.Records)); frame++ {
		a := expected.Images[frame]
		b := actual.Images[frame]
		if a == nil || b == nil {
			continue
		}
		for _, eyeName := range []string{"Left", "Right"} {
			var ae, be *RenderedEye
			if eyeName == "Left" {
				ae, be = a.Left, b.Left
			} else {
				ae, be = a.Right, b.Right
			}
			if ae == nil || be == nil {
				continue
			}
			subsystem := SubsystemLeftImage
			if eyeName == "Right" {
				subsystem = SubsystemRightImage
			}
			report.CheckCount++
			report.ChecksBySubsystem[subsystem]++
			if string(ae.Image.Pix) != string(be.Image.Pix) {
				pixelMismatches = append(pixelMismatches, Mismatch{
					Frame: frame, Subsystem: subsystem, Item: eyeName,
					Field: "raw_rgba_pixels", Expected: ae.Image.Bounds(), Actual: be.Image.Bounds(),
					Detail: "decoded pixel buffers differ despite any hash comparison",
				})
			}
		}
	}

	if len(pixelMismatches) > 0 {
		existing := report.Mismatches
		existing = append(existing, pixelMismatches...)
		sortMismatches(existing)
		report.Mismatches = existing
		report.MismatchCount = len(existing)
		report.MismatchesBySystem = map[string]int{}
		for _, mismatch := range existing {
			report.MismatchesBySystem[mismatch.Subsystem]++
		}
		first := existing[0]
		report.FirstDivergence = &first
		report.OK = false
	}
	return report
}

func (c *comparator) check(subsystem string) {
	if c.checks == nil {
		c.checks = map[string]int{}
	}
	c.checks[subsystem]++
}

func (c *comparator) add(m Mismatch) {
	if c.mismatchN == nil {
		c.mismatchN = map[string]int{}
	}
	c.mismatchN[m.Subsystem]++
	c.items = append(c.items, m)
}

func (c *comparator) equalString(frame int, subsystem, item, field string, expected, actual string) {
	c.check(subsystem)
	if expected != actual {
		c.add(Mismatch{Frame: frame, Subsystem: subsystem, Item: item, Field: field, Expected: expected, Actual: actual})
	}
}

func (c *comparator) equalInt(frame int, subsystem, item, field string, expected, actual int) {
	c.check(subsystem)
	if expected != actual {
		c.add(Mismatch{Frame: frame, Subsystem: subsystem, Item: item, Field: field, Expected: expected, Actual: actual})
	}
}

func (c *comparator) equalFloat(frame int, subsystem, item, field string, expected, actual, eps float64) {
	c.check(subsystem)
	if !floatApprox(expected, actual, eps) {
		c.add(Mismatch{
			Frame: frame, Subsystem: subsystem, Item: item, Field: field,
			Expected: roundForReport(expected), Actual: roundForReport(actual),
			Detail: fmt.Sprintf("delta=%.9g tolerance=%.0e", actual-expected, eps),
		})
	}
}

func (c *comparator) equalBool(frame int, subsystem, item, field string, expected, actual bool) {
	c.check(subsystem)
	if expected != actual {
		c.add(Mismatch{Frame: frame, Subsystem: subsystem, Item: item, Field: field, Expected: expected, Actual: actual})
	}
}

func (c *comparator) sliceEqual(frame int, subsystem, item, field string, expected, actual []string) {
	c.check(subsystem)
	if !stringSliceEqual(expected, actual) {
		c.add(Mismatch{Frame: frame, Subsystem: subsystem, Item: item, Field: field, Expected: expected, Actual: actual})
	}
}

func (c *comparator) totalChecks() int {
	total := 0
	for _, n := range c.checks {
		total += n
	}
	return total
}

func floatApprox(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func roundForReport(v float64) float64 {
	return float64(int64(v*1e9+sign(v)*0.5)) / 1e9
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

func subsystemOrderIndex(name string) int {
	order := []string{
		SubsystemSequence, SubsystemLifecycle, SubsystemParameters, SubsystemTracking,
		SubsystemPose, SubsystemResource, SubsystemVisibility, SubsystemReprojection,
		SubsystemLeftImage, SubsystemRightImage, SubsystemFrameHash,
	}
	for i, candidate := range order {
		if candidate == name {
			return i
		}
	}
	return len(order)
}

func sortMismatches(items []Mismatch) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Frame != items[j].Frame {
			return items[i].Frame < items[j].Frame
		}
		return subsystemOrderIndex(items[i].Subsystem) < subsystemOrderIndex(items[j].Subsystem)
	})
}
