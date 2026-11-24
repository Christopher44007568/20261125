package stereorollback

import (
	"fmt"
	"strings"
)

func addCheck(checks *[]InvariantCheck, name string, ok bool, detail string) {
	*checks = append(*checks, InvariantCheck{Name: name, OK: ok, Detail: detail})
}

func poolIndex(resources []ResourceInfo) map[ResourceID]ResourceInfo {
	out := make(map[ResourceID]ResourceInfo, len(resources))
	for _, r := range resources {
		out[r.ID] = r
	}
	return out
}

func checkEyeScenario(report *ScenarioReport, p *Pipeline, failed Eye) []InvariantCheck {
	checks := make([]InvariantCheck, 0, 10)
	other := failed.Other()
	c := report.Counters

	addCheck(&checks, "all_three_fault_stages_recorded",
		hasStage(c, StageAttachment) && hasStage(c, StageMaterial) && hasStage(c, StageReprojection),
		fmt.Sprintf("stages=%v", c.FailuresByStage))

	wrongEye := c.EyePlaceholderFrames[string(other)]
	addCheck(&checks, "only_injected_eye_uses_placeholder",
		c.EyePlaceholderFrames[string(failed)] == 3 && wrongEye == 0,
		fmt.Sprintf("placeholder left=%d right=%d", c.EyePlaceholderFrames["left"], c.EyePlaceholderFrames["right"]))

	preservedOK := true
	preservedCount := 0
	for _, fr := range report.Frames {
		if fr.Failure == nil {
			continue
		}
		surface, ok := fr.HealthyEye()
		if !ok || surface.Eye != other || surface.Status != StatusRendered || !surface.Success {
			preservedOK = false
			continue
		}
		preservedCount++
	}
	addCheck(&checks, "other_eye_preserved_on_each_failure",
		preservedOK && preservedCount == 3,
		fmt.Sprintf("preserved real frames=%d/3 for %s eye", preservedCount, other))

	expectedSuccess := c.EyeSuccessFrames["left"] + c.EyeSuccessFrames["right"]
	addCheck(&checks, "placeholder_not_counted_as_success",
		c.SuccessfulFrames == expectedSuccess &&
			c.SuccessfulFrames == 7 &&
			c.PlaceholderEyeFrames == 3 &&
			c.EyeSuccessFrames[string(failed)] == 2 &&
			c.EyeSuccessFrames[string(other)] == 5,
		fmt.Sprintf("successful_eye_frames=%d (left=%d right=%d), placeholders=%d",
			c.SuccessfulFrames, c.EyeSuccessFrames["left"], c.EyeSuccessFrames["right"], c.PlaceholderEyeFrames))

	retainedOK := true
	retainedDetail := ""
	for _, fr := range report.Frames {
		if fr.Failure == nil || fr.Failure.Rollback == nil {
			continue
		}
		var atlas *RetainedResource
		for i := range fr.Failure.Rollback.RetainedResources {
			if fr.Failure.Rollback.RetainedResources[i].ID == ResSharedAtlas {
				atlas = &fr.Failure.Rollback.RetainedResources[i]
			}
		}
		if atlas == nil {
			retainedOK = false
			continue
		}
		want := map[string]bool{OwnerEye(other): false, OwnerReprojector: false, OwnerRecorder: false}
		for _, u := range atlas.OtherUsers {
			if _, ok := want[u]; ok {
				want[u] = true
			}
		}
		for owner, found := range want {
			if !found {
				retainedOK = false
				retainedDetail = "missing surviving user " + owner
			}
		}
	}
	addCheck(&checks, "rollback_retains_shared_atlas_with_other_users", retainedOK,
		firstNonEmpty(retainedDetail, "atlas retained on all three rollbacks with opposite eye/reprojector/recorder"))

	idx := poolIndex(report.PoolResources)
	_, atlasAlive := idx[ResSharedAtlas]
	_, depthAlive := idx[ResSharedDepth]
	_, materialAlive := idx[ResSharedMaterial]
	_, historyAlive := idx[ResSharedHistory]
	atlasUsers := idx[ResSharedAtlas].Users
	addCheck(&checks, "whole_shared_set_survives",
		atlasAlive && depthAlive && materialAlive && historyAlive,
		fmt.Sprintf("atlas=%v depth=%v material=%v history=%v", atlasAlive, depthAlive, materialAlive, historyAlive))

	addCheck(&checks, "other_users_not_cleared",
		idx[ResSharedAtlas].RefCount == 4 && len(atlasUsers) == 4,
		fmt.Sprintf("atlas ref_count=%d users=%v", idx[ResSharedAtlas].RefCount, atlasUsers))

	leftover := []string{}
	for _, r := range report.PoolResources {
		if strings.HasPrefix(string(r.ID), "attach.") {
			leftover = append(leftover, string(r.ID))
		}
	}
	addCheck(&checks, "failed_eye_private_attachments_cleaned", len(leftover) == 0,
		"leftover attachments: "+strings.Join(leftover, ","))

	addCheck(&checks, "scoped_failure_counts",
		c.LeftFailures == map[Eye]int{EyeLeft: 3, EyeRight: 0}[failed] &&
			c.RightFailures == map[Eye]int{EyeLeft: 0, EyeRight: 3}[failed],
		fmt.Sprintf("left=%d right=%d", c.LeftFailures, c.RightFailures))

	return checks
}

func checkSharedScenario(report *ScenarioReport, p *Pipeline) []InvariantCheck {
	checks := make([]InvariantCheck, 0, 8)
	c := report.Counters

	var corr *CorruptionRecord
	for _, fr := range report.Frames {
		if fr.SharedCorruption != nil {
			corr = fr.SharedCorruption
		}
	}
	addCheck(&checks, "exactly_one_shared_corruption_record", corr != nil && c.SharedCorruptions == 1,
		fmt.Sprintf("records=%d", c.SharedCorruptions))
	if corr == nil {
		return checks
	}

	addCheck(&checks, "shared_path_has_no_failed_eye",
		corr.Scope == ScopeSharedResource && len(corr.DroppedUsers) == 0,
		"scope="+string(corr.Scope))

	addCheck(&checks, "all_users_retained_after_restore",
		sameUserSet(corr.AffectedUsers, corr.MigratedUsers) && len(corr.MigratedUsers) == 4,
		fmt.Sprintf("users=%v migrated=%v dropped=%v", corr.AffectedUsers, corr.MigratedUsers, corr.DroppedUsers))

	addCheck(&checks, "texture_generation_bumped",
		corr.NewGeneration == corr.OldGeneration+1 && corr.ReplacementID == ResSharedAtlas,
		fmt.Sprintf("gen %d -> %d", corr.OldGeneration, corr.NewGeneration))

	idx := poolIndex(report.PoolResources)
	depth := idx[ResSharedDepth]
	material := idx[ResSharedMaterial]
	history := idx[ResSharedHistory]
	addCheck(&checks, "unrelated_shared_resources_unchanged",
		depth.Generation == 1 && material.Generation == 1 && history.Generation == 1,
		fmt.Sprintf("depth_gen=%d material_gen=%d history_gen=%d", depth.Generation, material.Generation, history.Generation))

	addCheck(&checks, "shared_fallback_not_counted",
		c.SuccessfulFrames == 6 && c.SharedFallbackEyeFrames == 2 && c.FullSuccessFrames == 3,
		fmt.Sprintf("successful=%d shared_fallback_surfaces=%d full_frames=%d",
			c.SuccessfulFrames, c.SharedFallbackEyeFrames, c.FullSuccessFrames))

	recovered := true
	for i, fr := range report.Frames {
		if i >= 2 && !fr.FullSuccess {
			recovered = false
		}
	}
	addCheck(&checks, "frames_after_incident_recover", recovered, "frames 3 and 4 render both eyes")

	addCheck(&checks, "pool_users_intact_after_run",
		idx[ResSharedAtlas].RefCount == 4 && len(idx[ResSharedAtlas].Users) == 4,
		fmt.Sprintf("atlas ref_count=%d users=%v", idx[ResSharedAtlas].RefCount, idx[ResSharedAtlas].Users))

	return checks
}

func sameUserSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return "retention verified"
}
