package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeReadabilityCSV(outDir string, report *Report) error {
	file, err := os.Create(filepath.Join(outDir, "readability.csv"))
	if err != nil {
		return err
	}
	defer file.Close()
	w := csv.NewWriter(file)
	header := []string{
		"scenario_id", "layer_id", "element_id", "eye", "status", "pass",
		"cap_height_px", "stroke_width_px", "cap_height_arcmin",
		"pixels_per_degree", "contrast_ratio", "motion_residual_px",
		"persistence_blur_px", "tracking_error_px", "warnings", "failures",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, layer := range scenario.Layers {
			for _, element := range layer.Elements {
				for _, eye := range []EyeID{LeftEye, RightEye} {
					r := element.Readability[eye]
					row := []string{
						scenario.ID, layer.LayerID, element.Spec.ID, string(eye), r.Status,
						fmt.Sprintf("%t", r.Pass),
						fmt.Sprintf("%.2f", r.CapHeightPx),
						fmt.Sprintf("%.2f", r.StrokeWidthPx),
						fmt.Sprintf("%.2f", r.CapHeightArcMin),
						fmt.Sprintf("%.2f", r.PixelsPerDegree),
						fmt.Sprintf("%.2f", r.ContrastRatio),
						fmt.Sprintf("%.2f", r.MotionResidualPx),
						fmt.Sprintf("%.2f", r.PersistenceBlurPx),
						fmt.Sprintf("%.2f", r.TrackingErrorPx),
						strings.Join(r.Warnings, "; "),
						strings.Join(r.Failures, "; "),
					}
					if err := w.Write(row); err != nil {
						return err
					}
				}
			}
		}
	}
	w.Flush()
	return w.Error()
}

func writeSharpnessCSV(outDir string, report *Report) error {
	file, err := os.Create(filepath.Join(outDir, "sharpness.csv"))
	if err != nil {
		return err
	}
	defer file.Close()
	w := csv.NewWriter(file)
	header := []string{
		"scenario_id", "layer_id", "element_id", "eye", "status", "pass",
		"edge_samples", "median_transition_10_to_90_px",
		"p90_transition_px", "median_luminance_step", "failures",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, layer := range scenario.Layers {
			if layer.DepthClass != "system" {
				continue
			}
			for _, element := range layer.Elements {
				for _, eye := range []EyeID{LeftEye, RightEye} {
					s, ok := element.Sharpness[eye]
					if !ok {
						continue
					}
					row := []string{
						scenario.ID, layer.LayerID, element.Spec.ID, string(eye), s.Status,
						fmt.Sprintf("%t", s.Pass),
						fmt.Sprintf("%d", s.EdgeSamples),
						fmt.Sprintf("%.3f", s.MedianTransitionPx),
						fmt.Sprintf("%.3f", s.P90TransitionPx),
						fmt.Sprintf("%.4f", s.MedianLuminanceStep),
						strings.Join(s.Failures, "; "),
					}
					if err := w.Write(row); err != nil {
						return err
					}
				}
			}
		}
	}
	w.Flush()
	return w.Error()
}
