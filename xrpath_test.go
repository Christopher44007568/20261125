package xrpath

import (
	"path/filepath"
	"testing"
)

func TestBaselineRerunIsItemIdentical(t *testing.T) {
	baseline := Simulate()
	rerun := Simulate()
	report := CompareRuns(baseline, rerun)

	if !report.OK {
		t.Fatalf("baseline rerun diverged: first=%+v all=%+v", report.FirstDivergence, report.Mismatches[:minInt(5, len(report.Mismatches))])
	}
	if len(baseline.Records) != FrameCount || len(rerun.Records) != FrameCount {
		t.Fatalf("frame count = %d/%d, want %d", len(baseline.Records), len(rerun.Records), FrameCount)
	}
	if report.CheckCount < 25000 {
		t.Fatalf("check count = %d, want exhaustive item checks", report.CheckCount)
	}
}

func TestFixedPathCoversAllRequestedMotions(t *testing.T) {
	run := Simulate()
	records := run.Records

	assertPhaseRange(t, records, "static", 0, 11)
	assertPhaseRange(t, records, "fast_rotation", 12, 19)
	assertPhaseRange(t, records, "forward", 20, 37)
	assertPhaseRange(t, records, "occlusion_reveal", 38, 51)
	assertPhaseRange(t, records, "relocalization", 52, 71)

	for frame := 1; frame <= 11; frame++ {
		if !records[frame].Pose.Position.Approx(records[0].Pose.Position, 1e-9) {
			t.Fatalf("static frame %d moved: %+v", frame, records[frame].Pose.Position)
		}
	}

	maxYawRate := 0.0
	for _, record := range records[12:20] {
		maxYawRate = maxFloat(maxYawRate, absFloat(record.Pose.AngularVelocity.Y))
	}
	if maxYawRate < 20 {
		t.Fatalf("fast rotation peak yaw rate = %.2f, want > 20 rad/s", maxYawRate)
	}

	forwardDistance := records[20].Pose.Position.Z - records[37].Pose.Position.Z
	if forwardDistance < 3 {
		t.Fatalf("forward distance = %.3f, want >= 3m", forwardDistance)
	}

	firstReveal := -1
	for _, record := range records {
		for _, obj := range record.Visibility.Left {
			if obj.ID == "reveal-relic" && obj.Pixels > 0 {
				firstReveal = record.Frame
				break
			}
		}
		if firstReveal >= 0 {
			break
		}
	}
	if firstReveal < 40 || firstReveal > 44 {
		t.Fatalf("reveal-relic first left-eye frame = %d, want 40..44", firstReveal)
	}

	statuses := map[string]bool{}
	for _, record := range records[52:] {
		statuses[record.Tracking.Status] = true
	}
	for _, status := range []string{"lost", "relocalizing", "relocalized", "tracking"} {
		if !statuses[status] {
			t.Fatalf("relocation sequence missing status %q; got %v", status, statuses)
		}
	}
}

func TestPauseResumeAndParameterReload(t *testing.T) {
	records := Simulate().Records

	held := records[24].Pose
	for frame := PauseAtFrame; frame < ResumeAtFrame; frame++ {
		record := records[frame]
		if record.Lifecycle != "paused" {
			t.Fatalf("frame %d lifecycle = %q, want paused", frame, record.Lifecycle)
		}
		if !record.Pose.Position.Approx(held.Position, 1e-9) || record.Pose.Yaw != held.Yaw {
			t.Fatalf("pause frame %d pose was not held", frame)
		}
		if record.Pose.LinearVelocity.Length() > 1e-9 || absFloat(record.Pose.AngularVelocity.Y) > 1e-9 {
			t.Fatalf("pause frame %d retained velocity", frame)
		}
	}
	if records[ResumeAtFrame].Lifecycle != "running" {
		t.Fatalf("resume frame lifecycle = %q", records[ResumeAtFrame].Lifecycle)
	}
	if records[28].Parameters.Revision != 1 || records[29].Parameters.Revision != 2 {
		t.Fatalf("parameter reload revisions = %d then %d", records[28].Parameters.Revision, records[29].Parameters.Revision)
	}

	foundReloadRequest := false
	foundReloadActive := false
	for _, event := range records[ReloadAtFrame].Events {
		if event.Type == "parameter_reload_requested" {
			foundReloadRequest = true
		}
	}
	for _, event := range records[ResumeAtFrame].Events {
		if event.Type == "parameter_reloaded" {
			foundReloadActive = true
		}
	}
	if !foundReloadRequest || !foundReloadActive {
		t.Fatalf("reload events missing: request=%v active=%v", foundReloadRequest, foundReloadActive)
	}
}

func TestEveryFrameContainsAllRequiredRecords(t *testing.T) {
	run := Simulate()
	for _, record := range run.Records {
		if record.Schema != SchemaVersion || record.FrameSHA256 == "" {
			t.Fatalf("frame %d schema/hash incomplete", record.Frame)
		}
		if len(record.Resources) < 15 {
			t.Fatalf("frame %d resources = %d", record.Frame, len(record.Resources))
		}
		if len(record.Visibility.Left) == 0 || len(record.Visibility.Right) == 0 {
			t.Fatalf("frame %d empty visibility", record.Frame)
		}
		if record.Images.Left.PixelSHA256 == "" || record.Images.Right.PNGSHA256 == "" {
			t.Fatalf("frame %d image hashes missing", record.Frame)
		}
		if record.Reprojection[0].SourceImageSHA256 == "" || record.Reprojection[1].WarpMeshSHA256 == "" {
			t.Fatalf("frame %d reprojection source missing", record.Frame)
		}
		for _, eye := range record.Reprojection {
			if eye.SourceFrame < 0 || eye.SourceFrame > record.Frame {
				t.Fatalf("frame %d bad source frame %d", record.Frame, eye.SourceFrame)
			}
		}

		clearHash := record
		clearHash.FrameSHA256 = ""
		if got := canonicalSHA256(clearHash); got != record.FrameSHA256 {
			t.Fatalf("frame %d frame hash mismatch", record.Frame)
		}

		stereo := run.Images[record.Frame]
		if bytesSHA256(stereo.Left.Image.Pix) != record.Images.Left.PixelSHA256 {
			t.Fatalf("frame %d left pixel hash does not match buffer", record.Frame)
		}
		if bytesSHA256(stereo.Left.PNGBytes) != record.Images.Left.PNGSHA256 {
			t.Fatalf("frame %d left PNG hash does not match bytes", record.Frame)
		}
	}
}

func TestSaveLoadAndVerifyImages(t *testing.T) {
	run := Simulate()
	dir := filepath.Join(t.TempDir(), "xrrun")
	if err := SaveRun(dir, run); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadRecords(dir)
	if err != nil {
		t.Fatal(err)
	}
	report := CompareRecords(run.Records, loaded)
	if !report.OK {
		t.Fatalf("JSON roundtrip mismatch: %+v", report.FirstDivergence)
	}
	if mismatches := VerifyStoredImages(dir, loaded); len(mismatches) != 0 {
		t.Fatalf("stored image verification failed: %+v", mismatches)
	}
}

func TestCanonicalFaultsPointToFrameAndSubsystem(t *testing.T) {
	baseline := Simulate()
	suite := RunFaultDrills(baseline)
	if !suite.OK {
		for _, drill := range suite.Drills {
			t.Logf("fault %+v detected=%v first=%+v", drill.Fault, drill.Detected, drill.FirstDivergence)
		}
		t.Fatal("one or more fault injections were not localized to the expected frame/subsystem")
	}
}

func TestIdenticalImageHashesDoNotMaskGeometryOrResources(t *testing.T) {
	baseline := Simulate().Records
	drills, err := RunHashBlindDrills(baseline)
	if err != nil {
		t.Fatal(err)
	}
	for _, drill := range drills {
		if !drill.OK {
			t.Fatalf("hash-blind drill %s failed: %+v", drill.Name, drill.FirstDivergence)
		}
	}

	masked, err := CloneRecords(baseline)
	if err != nil {
		t.Fatal(err)
	}
	masked[35].Pose.Position.Z += 0.4
	masked[35].Resources[0].Status = "ready"
	report := CompareRecords(baseline, masked)
	if report.OK || report.FirstDivergence == nil {
		t.Fatal("geometry mutation with unchanged image hashes was not detected")
	}
	if report.FirstDivergence.Frame != 35 || report.FirstDivergence.Subsystem != SubsystemPose {
		t.Fatalf("first divergence = frame %d subsystem %s, want 35/pose", report.FirstDivergence.Frame, report.FirstDivergence.Subsystem)
	}
	if report.MismatchesBySystem[SubsystemResource] == 0 {
		t.Fatal("resource mutation was not independently compared")
	}
}

func TestRunDemoCreatesVerifiedArtifacts(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "artifacts")
	result, err := RunDemo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("demo verification failed: first=%+v stored=%+v faultOK=%v", result.Report.FirstDivergence, result.StoredImageMismatches, result.FaultArtifact.OK)
	}
}

func assertPhaseRange(t *testing.T, records []FrameRecord, phase string, first, last int) {
	t.Helper()
	var gotFirst, gotLast = -1, -1
	for _, record := range records {
		if record.Phase == phase {
			if gotFirst == -1 {
				gotFirst = record.Frame
			}
			gotLast = record.Frame
		}
	}
	if gotFirst != first || gotLast != last {
		t.Fatalf("phase %s range = %d..%d, want %d..%d", phase, gotFirst, gotLast, first, last)
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
