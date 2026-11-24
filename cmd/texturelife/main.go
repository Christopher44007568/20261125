package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"texturelife"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
}

func run(out io.Writer) error {
	tracker := texturelife.NewTracker()
	device := texturelife.NewDevice(tracker)

	if err := device.Initialize(1, texturelife.Size{Width: 1280, Height: 720}); err != nil {
		return err
	}
	printState(out, "frame 1: initialize stereo 1280x720", device, tracker.Snapshot())

	if err := device.Resize(2, texturelife.Size{Width: 1920, Height: 1080}); err != nil {
		return err
	}
	printState(out, "frame 2: window change to 1920x1080", device, tracker.Snapshot())

	if err := device.DestroyEye(3, texturelife.EyeLeft); err != nil {
		return err
	}
	printState(out, "frame 3: destroy left eye", device, tracker.Snapshot())

	if err := device.SimulateRebuildFailure(4, texturelife.Size{Width: 2560, Height: 1440}); !errors.Is(err, texturelife.ErrRebuildFailed) {
		return fmt.Errorf("expected simulated rebuild failure, got %v", err)
	}
	if device.Viewport() != (texturelife.Size{Width: 1920, Height: 1080}) {
		return fmt.Errorf("failed rebuild changed viewport to %v", device.Viewport())
	}
	printState(out, "frame 4: device rebuild failed and rolled back", device, tracker.Snapshot())

	right, ok := device.EyeSet(texturelife.EyeRight)
	if !ok {
		return errors.New("right eye unexpectedly inactive")
	}
	oldHistory := right.History
	if err := tracker.Acquire(5, oldHistory); err != nil {
		return err
	}
	fmt.Fprintf(out, "\nframe 5: new frame references old right-history texture %d\n", oldHistory)

	if err := device.Resize(6, texturelife.Size{Width: 2560, Height: 1440}); err != nil {
		return err
	}
	held, _ := tracker.Texture(oldHistory)
	if !held.Alive() || held.RefCount != 1 || held.RetireFrame != 6 {
		return fmt.Errorf("old history was released too early: %+v", held)
	}
	fmt.Fprintf(out, "frame 6: viewport rebuilt; old history %d remains alive (refs=%d, retire=%d)\n",
		oldHistory, held.RefCount, held.RetireFrame)

	tracker.ReleaseFrameAt(5, 7)
	held, _ = tracker.Texture(oldHistory)
	if held.Alive() || held.ReleasedAt != 7 {
		return fmt.Errorf("old history release time = %d, alive=%v; want 7, false", held.ReleasedAt, held.Alive())
	}
	fmt.Fprintf(out, "frame 7: final reference to old history released at frame %d\n", held.ReleasedAt)

	if err := device.Shutdown(8); err != nil {
		return err
	}
	report := tracker.FinalAudit()
	printState(out, "frame 8: shutdown", device, report)
	printAudit(out, report)

	if !report.Healthy() {
		return fmt.Errorf("final audit failed with %d issues", len(report.Issues))
	}
	fmt.Fprintln(out, "\nPASS: no leaks, dangling references, or premature releases")
	return nil
}

func printState(out io.Writer, label string, device *texturelife.Device, report texturelife.Report) {
	fmt.Fprintf(out, "\n%s: frame=%d viewport=%v live=%d bytes=%d refs=%d\n",
		label, device.Frame(), device.Viewport(), report.LiveCount, report.LiveBytes, report.LiveReferences)
	if len(report.Issues) > 0 {
		for _, issue := range report.Issues {
			fmt.Fprintf(out, "  issue: %s\n", issue)
		}
	}
}

func printAudit(out io.Writer, report texturelife.Report) {
	fmt.Fprintln(out, "\nTexture lifetime records:")
	fmt.Fprintf(out, "  %-4s %-7s %-6s %-10s %-5s %-6s %-7s %-8s %-6s %s\n",
		"id", "kind", "eye", "size", "refs", "peak", "created", "last-use", "retire", "released")
	for _, texture := range report.Textures {
		released := "-"
		if texture.ReleasedAt != 0 {
			released = fmt.Sprint(texture.ReleasedAt)
		}
		fmt.Fprintf(out, "  %-4d %-7s %-6s %-10s %-5d %-6d %-7d %-8d %-6d %s\n",
			texture.ID, texture.Kind, texture.Eye, texture.Size, texture.RefCount,
			texture.PeakRefCount, texture.CreatedFrame, texture.LastUsedFrame,
			texture.RetireFrame, released)
	}

	fmt.Fprintln(out, "\nPeaks:")
	fmt.Fprintf(out, "  live: resources=%d bytes=%d frame=%d refs=%d at-frame=%d\n",
		report.Peak.Count, report.Peak.Bytes, report.Peak.CountFrame,
		report.Peak.RefCount, report.Peak.RefFrame)
	eyes := make([]int, 0, len(report.PeakByEye))
	for eye := range report.PeakByEye {
		eyes = append(eyes, int(eye))
	}
	sort.Ints(eyes)
	for _, value := range eyes {
		eye := texturelife.Eye(value)
		peak := report.PeakByEye[eye]
		fmt.Fprintf(out, "  eye %-5s: resources=%d bytes=%d refs=%d frame=%d\n",
			eye, peak.Count, peak.Bytes, peak.RefCount, peak.Frame)
	}
	kinds := make([]int, 0, len(report.PeakByKind))
	for kind := range report.PeakByKind {
		kinds = append(kinds, int(kind))
	}
	sort.Ints(kinds)
	for _, value := range kinds {
		kind := texturelife.Kind(value)
		peak := report.PeakByKind[kind]
		fmt.Fprintf(out, "  %-8s: resources=%d bytes=%d refs=%d frame=%d\n",
			kind, peak.Count, peak.Bytes, peak.RefCount, peak.Frame)
	}

	fmt.Fprintln(out, "\nAudit:")
	fmt.Fprintf(out, "  created=%d live=%d retiring=%d refs=%d leaks=%d dangling=%d issues=%d\n",
		report.CreatedCount, report.LiveCount, report.RetiringCount, report.LiveReferences,
		len(report.Leaks), len(report.Dangling), len(report.Issues))
}
