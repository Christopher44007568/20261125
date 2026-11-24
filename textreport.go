package stereorollback

import (
	"fmt"
	"strings"
)

// RenderText produces a plain-text audit summary of the report.
func RenderText(r *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Stereo single-eye rollback report  (%s, %s)\n", r.GoVersion, r.GeneratedAt)
	fmt.Fprintf(&b, "scenarios=%d all_invariants_hold=%v\n\n", r.Summary.ScenarioCount, r.Summary.AllInvariantsHold)

	for _, sc := range r.Scenarios {
		fmt.Fprintf(&b, "=== %s : %s ===\n", sc.ID, sc.Title)
		fmt.Fprintf(&b, "%s\n", sc.Description)
		fmt.Fprintf(&b, "frames=%d successful_eye_frames=%d full_success=%d partial=%d placeholder=%d shared_fallback=%d\n",
			sc.Counters.TotalStereoFrames, sc.Counters.SuccessfulFrames,
			sc.Counters.FullSuccessFrames, sc.Counters.PartialFrames,
			sc.Counters.PlaceholderEyeFrames, sc.Counters.SharedFallbackEyeFrames)

		for _, fr := range sc.Frames {
			fmt.Fprintf(&b, "  f%-2d L=%-16s R=%-16s", fr.Index, fr.Left.Marker, fr.Right.Marker)
			if fr.Failure != nil {
				fmt.Fprintf(&b, " FAIL[%s/%s] retained=%d destroyed=%d",
					fr.Failure.FailureStage, fr.Failure.Fault,
					len(fr.Failure.Rollback.RetainedResources),
					len(fr.Failure.Rollback.DestroyedPrivate))
			}
			if fr.SharedCorruption != nil {
				fmt.Fprintf(&b, " SHARED-CORRUPT gen->%d users=%d",
					fr.SharedCorruption.NewGeneration, len(fr.SharedCorruption.MigratedUsers))
			}
			fmt.Fprintln(&b)
		}

		for _, chk := range sc.InvariantChecks {
			mark := "PASS"
			if !chk.OK {
				mark = "FAIL"
			}
			fmt.Fprintf(&b, "  [%s] %s — %s\n", mark, chk.Name, chk.Detail)
		}
		fmt.Fprintln(&b)
	}

	s := r.Summary
	fmt.Fprintf(&b, "SUMMARY frames=%d successful_eye_frames=%d placeholders=%d shared_fallback=%d L_fail=%d R_fail=%d shared_corrupt=%d clean=%v\n",
		s.TotalStereoFrames, s.TotalSuccessfulFrames, s.TotalPlaceholderFrames,
		s.TotalSharedFallback, s.LeftFailures, s.RightFailures, s.SharedCorruptions, s.AllInvariantsHold)
	return b.String()
}
