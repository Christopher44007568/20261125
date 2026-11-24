package main

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
)

func writeBundle(
	outDir string,
	report Report,
	reference map[Eye]*image.RGBA,
	isolated *RunResult,
	fault *RunResult,
) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, eye := range []Eye{EyeLeft, EyeRight} {
		if err := writePNG(filepath.Join(outDir, imageFilename(eye)), isolated.Images[eye]); err != nil {
			return fmt.Errorf("write isolated %s image: %w", eye, err)
		}
		if err := writePNG(filepath.Join(outDir, faultFilename(eye)), fault.Images[eye]); err != nil {
			return fmt.Errorf("write fault %s image: %w", eye, err)
		}
		if err := WritePollutionDiff(filepath.Join(outDir, diffFilename(eye)), reference[eye], fault.Images[eye]); err != nil {
			return fmt.Errorf("write %s pollution diff: %w", eye, err)
		}
	}
	if err := writeJSON(filepath.Join(outDir, "report.json"), report); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "report.md"), []byte(renderMarkdown(report)), 0o644); err != nil {
		return err
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func renderMarkdown(report Report) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# Eye dynamic-state isolation audit\n\n")
	fmt.Fprintf(&out, "- Schema: `%s`\n", report.Schema)
	fmt.Fprintf(&out, "- Program: `%s` (`%s`)\n", report.Program.ID, report.Program.SourceHash)
	fmt.Fprintf(&out, "- Static layout: `%s`, %d panels, %dx%d\n",
		report.Shared.StaticLayoutHash, report.Shared.PanelCount,
		report.Shared.Viewport[0], report.Shared.Viewport[1])
	fmt.Fprintf(&out, "- Isolated result: **%s**\n\n", passLabel(report.Isolation.Clean))

	out.WriteString("## Shared and isolated state\n\n")
	fmt.Fprintf(&out, "- Shared shader code/layout: `%t`\n", report.Shared.CodeAndLayoutShared)
	fmt.Fprintf(&out, "- Shared static layout: `%t`\n", report.Shared.StaticLayoutShared)
	fmt.Fprintf(&out, "- Dynamic state shared across eyes: `%t`\n", report.Shared.DynamicStateShared)
	fmt.Fprintf(&out, "- Cameras distinct: `%t`\n", report.Isolation.CamerasDistinct)
	fmt.Fprintf(&out, "- Material states distinct: `%t`\n", report.Isolation.MaterialStatesDistinct)
	fmt.Fprintf(&out, "- Alpha overrides distinct: `%t`\n", report.Isolation.AlphaOverridesDistinct)
	fmt.Fprintf(&out, "- Material constants distinct: `%t`\n", report.Isolation.MaterialConstantsDistinct)
	fmt.Fprintf(&out, "- Eye constants distinct: `%t`\n", report.Isolation.EyeConstantsDistinct)
	fmt.Fprintf(&out, "- Snapshot hashes stable after render: `%t`\n", report.Isolation.SnapshotsStable)
	fmt.Fprintf(&out, "- Isolated equals uncached reference: `%t`\n\n", report.Isolation.IsolatedMatchesReference)

	out.WriteString("## Per-eye parameter snapshots\n\n")
	out.WriteString("| Eye | Instance | Roughness | Alpha override | Gain | Spec | Ring | Tint RGB |\n")
	out.WriteString("|---|---|---:|---:|---:|---:|---:|---|\n")
	for _, eye := range report.Eyes {
		for _, material := range eye.ParameterSnapshot.Materials {
			alpha := "inherit"
			if material.AlphaOverride != nil {
				alpha = fmt.Sprintf("%.3f", *material.AlphaOverride)
			}
			fmt.Fprintf(&out, "| %s | %s | %.3f | %s | %.3f | %.3f | %.3f | %.2f, %.2f, %.2f |\n",
				eye.Eye, material.Instance, material.Roughness, alpha,
				material.Constants["gain"], material.Constants["spec"], material.Constants["ring"],
				material.Constants["tint_r"], material.Constants["tint_g"], material.Constants["tint_b"])
		}
	}
	out.WriteString("\n| Eye | Camera | Snapshot hash | Image hash | Isolated pollution |\n")
	out.WriteString("|---|---|---|---|---:|\n")
	for _, eye := range report.Eyes {
		camera := eye.ParameterSnapshot.Camera
		fmt.Fprintf(&out, "| %s | pos=(%.3f, %.3f, %.3f), yaw=%.3f, fov=%.2f | `%s` | `%s` | %d |\n",
			eye.Eye, camera.Position.X, camera.Position.Y, camera.Position.Z,
			camera.YawRadians, camera.VerticalFOVDegree, eye.SnapshotHash, eye.ImageHash,
			eye.PollutedPixels)
	}

	out.WriteString("\n## Shuffled batch and cache stress\n\n")
	fmt.Fprintf(&out, "- Seed: `%d`\n", report.Pipeline.Config.Seed)
	fmt.Fprintf(&out, "- Rounds: `%d`\n", report.Pipeline.Config.Rounds)
	fmt.Fprintf(&out, "- Batches: `%d`\n", report.Pipeline.BatchCount)
	fmt.Fprintf(&out, "- Cache capacity: `%d`\n", report.Pipeline.Cache.Capacity)
	fmt.Fprintf(&out, "- Cache hits/misses/evictions: `%d` / `%d` / `%d`\n",
		report.Pipeline.Cache.Hits, report.Pipeline.Cache.Misses, report.Pipeline.Cache.Evictions)
	fmt.Fprintf(&out, "- Final resident entries: `%d`\n\n", report.Pipeline.Cache.Entries)

	out.WriteString("## Fault injection: program-keyed dynamic cache\n\n")
	fmt.Fprintf(&out, "- %s\n", report.FaultInjection.Description)
	fmt.Fprintf(&out, "- Broken key: `%s`\n", strings.Join(report.FaultInjection.BrokenKeyFields, " + "))
	fmt.Fprintf(&out, "- Omitted: `%s`\n", strings.Join(report.FaultInjection.OmittedDynamicKeys, ", "))
	fmt.Fprintf(&out, "- Batches receiving a foreign dynamic hash: `%d`\n", report.FaultInjection.HashMismatchBatches)
	fmt.Fprintf(&out, "- Fault cache hits/misses/evictions: `%d` / `%d` / `%d`\n\n",
		report.FaultInjection.Cache.Hits, report.FaultInjection.Cache.Misses, report.FaultInjection.Cache.Evictions)
	out.WriteString("| Eye | Image hash | First polluted pixel | Polluted pixels | Reference RGBA | Actual RGBA |\n")
	out.WriteString("|---|---|---|---:|---|---|\n")
	for _, eye := range report.FaultInjection.Eyes {
		pixel := "none"
		if eye.FirstPollutedPixel != nil {
			pixel = fmt.Sprintf("(%d, %d)", eye.FirstPollutedPixel.X, eye.FirstPollutedPixel.Y)
		}
		fmt.Fprintf(&out, "| %s | `%s` | %s | %d | %s | %s |\n",
			eye.Eye, eye.ImageHash, pixel, eye.PollutedPixels,
			formatRGBA(eye.ReferenceRGBA), formatRGBA(eye.ActualRGBA))
	}
	out.WriteString("\nThe diff PNGs mark all polluted pixels in magenta and the first mismatch with a white cross.\n")
	return out.String()
}

func formatRGBA(value *[4]uint8) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("(%d, %d, %d, %d)", value[0], value[1], value[2], value[3])
}

func passLabel(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
