package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

func WriteTable(w io.Writer, rows []Row) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "SCENARIO\tMODE\tSTATUS\tLAYER\tITERS\tBINARY\tDEPTH\tERR\tUV_X\tUV_Y"); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\t%d\t%.6f\t%.3e\t%.6f\t%.6f\n",
			row.Scenario, row.Mode, row.Status, row.HitLayer, row.Iterations,
			row.BinarySteps, row.Depth, row.DepthError, row.UV.X, row.UV.Y); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func WriteJSON(w io.Writer, rows []Row) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rows)
}

func WriteCSV(w io.Writer, rows []Row) error {
	cw := csv.NewWriter(w)
	header := []string{
		"scenario", "mode", "status", "hit_layer", "iterations",
		"binary_steps", "depth", "expected_depth", "depth_error",
		"residual", "uv_x", "uv_y", "fallback", "reason",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		record := []string{
			row.Scenario,
			string(row.Mode),
			string(row.Status),
			fmt.Sprintf("%d", row.HitLayer),
			fmt.Sprintf("%d", row.Iterations),
			fmt.Sprintf("%d", row.BinarySteps),
			fmt.Sprintf("%.12f", row.Depth),
			fmt.Sprintf("%.12f", row.ExpectedDepth),
			fmt.Sprintf("%.12f", row.DepthError),
			fmt.Sprintf("%.12f", row.Residual),
			fmt.Sprintf("%.12f", row.UV.X),
			fmt.Sprintf("%.12f", row.UV.Y),
			fmt.Sprintf("%t", row.Fallback),
			row.Reason,
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
