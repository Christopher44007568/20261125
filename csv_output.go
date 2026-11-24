package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
)

func writeParallaxCSV(outDir string, report *Report) error {
	file, err := os.Create(filepath.Join(outDir, "parallax.csv"))
	if err != nil {
		return err
	}
	defer file.Close()
	w := csv.NewWriter(file)
	header := []string{
		"scenario_id", "layer_id", "element_id", "glyph_index", "char",
		"left_x_px", "left_y_px", "right_x_px", "right_y_px",
		"horizontal_disparity_left_minus_right_px", "vertical_disparity_px",
		"angular_horizontal_disparity_arcmin", "angular_vertical_disparity_arcmin",
		"naive_same_coordinate_copy_error_px",
		"coordinates_copied", "projection_path",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, layer := range scenario.Layers {
			for _, glyph := range layer.Glyphs {
				row := []string{
					scenario.ID, layer.LayerID, glyph.ElementID,
					fmt.Sprintf("%d", glyph.Index), glyph.Char,
					fmt.Sprintf("%.3f", glyph.Left.View.Origin.X),
					fmt.Sprintf("%.3f", glyph.Left.View.Origin.Y),
					fmt.Sprintf("%.3f", glyph.Right.View.Origin.X),
					fmt.Sprintf("%.3f", glyph.Right.View.Origin.Y),
					fmt.Sprintf("%.3f", glyph.Parallax.HorizontalPx),
					fmt.Sprintf("%.3f", glyph.Parallax.VerticalPx),
					fmt.Sprintf("%.3f", glyph.Parallax.AngularArcMin),
					fmt.Sprintf("%.3f", glyph.Parallax.VerticalAngularArcMin),
					fmt.Sprintf("%.3f", glyph.Parallax.NaiveCopyErrorPx),
					fmt.Sprintf("%t", glyph.Parallax.CoordinatesCopied),
					glyph.Parallax.ProjectionPath,
				}
				if err := w.Write(row); err != nil {
					return err
				}
			}
		}
	}
	w.Flush()
	return w.Error()
}

func writeBoundaryCSV(outDir string, report *Report) error {
	file, err := os.Create(filepath.Join(outDir, "boundary.csv"))
	if err != nil {
		return err
	}
	defer file.Close()
	w := csv.NewWriter(file)
	header := []string{
		"scenario_id", "layer_id", "element_id", "eye", "status",
		"min_image_margin_px", "required_safe_inset_px", "visible_area_ratio",
		"margin_left_px", "margin_right_px", "margin_top_px", "margin_bottom_px",
		"bounds",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, layer := range scenario.Layers {
			for _, element := range layer.Elements {
				for _, eye := range []EyeID{LeftEye, RightEye} {
					b := element.Boundary[eye]
					row := []string{
						scenario.ID, layer.LayerID, element.Spec.ID, string(eye), b.Status,
						fmt.Sprintf("%.2f", b.MinImageMarginPx),
						fmt.Sprintf("%.2f", b.RequiredSafeInset),
						fmt.Sprintf("%.4f", b.VisibleAreaRatio),
						fmt.Sprintf("%.2f", b.MarginLeftPx),
						fmt.Sprintf("%.2f", b.MarginRightPx),
						fmt.Sprintf("%.2f", b.MarginTopPx),
						fmt.Sprintf("%.2f", b.MarginBottomPx),
						fmt.Sprintf("[%.1f,%.1f,%.1f,%.1f]", b.Bounds.MinX, b.Bounds.MinY, b.Bounds.MaxX, b.Bounds.MaxY),
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
