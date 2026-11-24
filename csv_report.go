package eyebw

import (
	"encoding/csv"
	"fmt"
	"os"
)

func createCSV(path string, header []string) (*os.File, *csv.Writer, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	w := csv.NewWriter(f)
	if err := w.Write(header); err != nil {
		f.Close()
		return nil, nil, err
	}
	return f, w, nil
}

func closeCSV(f *os.File, w *csv.Writer) error {
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func WriteFramesCSV(path string, report *Report) error {
	f, w, err := createCSV(path, []string{
		"scenario", "frame", "pipeline",
		"logical_read_bytes", "logical_write_bytes",
		"transfer_read_bytes", "transfer_write_bytes", "transfer_total_bytes",
		"cache_hit_bytes", "cache_miss_bytes", "cache_hit_ratio",
		"budget_bytes", "budget_exceeded", "budget_overage_bytes",
		"quality_passed", "top_hotspot", "top_hotspot_transfer_bytes",
		"repaint_texels", "motion",
	})
	if err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, frame := range scenario.FrameReports {
			for _, p := range frame.Pipelines {
				hotspot := ""
				var hotspotBytes uint64
				if p.TopHotspot != nil {
					hotspot = string(p.TopHotspot.Stage)
					hotspotBytes = p.TopHotspot.TransferBytes
				}
				err := w.Write([]string{
					scenario.Name,
					fmt.Sprint(frame.Index),
					string(p.ID),
					fmt.Sprint(p.Totals.LogicalReadBytes),
					fmt.Sprint(p.Totals.LogicalWriteBytes),
					fmt.Sprint(p.Totals.TransferReadBytes),
					fmt.Sprint(p.Totals.TransferWriteBytes),
					fmt.Sprint(p.TransferTotal()),
					fmt.Sprint(p.Totals.CacheHitBytes),
					fmt.Sprint(p.Totals.CacheMissBytes),
					fmt.Sprintf("%.6f", p.Totals.CacheHitRatioBytes),
					fmt.Sprint(p.Budget.BudgetBytes),
					fmt.Sprint(p.Budget.Exceeded),
					fmt.Sprint(p.Budget.OverageBytes),
					fmt.Sprint(p.Quality.Passed),
					hotspot,
					fmt.Sprint(hotspotBytes),
					fmt.Sprint(p.Repaint.TotalTexels),
					fmt.Sprintf("%.4f", frame.Motion),
				})
				if err != nil {
					f.Close()
					return err
				}
			}
		}
	}
	return closeCSV(f, w)
}

func WriteStagesCSV(path string, report *Report) error {
	f, w, err := createCSV(path, []string{
		"scenario", "pipeline", "rank", "stage", "stage_zh",
		"logical_bytes", "transfer_bytes", "transfer_read_bytes", "transfer_write_bytes",
		"cache_hit_bytes", "cache_hit_ratio", "transfer_share",
	})
	if err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, pipeline := range scenario.Pipelines {
			for _, hotspot := range pipeline.Hotspots {
				if err := w.Write([]string{
					scenario.Name,
					string(pipeline.ID),
					fmt.Sprint(hotspot.Rank),
					string(hotspot.Stage),
					hotspot.Label,
					fmt.Sprint(hotspot.LogicalBytes),
					fmt.Sprint(hotspot.TransferBytes),
					fmt.Sprint(hotspot.TransferReadBytes),
					fmt.Sprint(hotspot.TransferWriteBytes),
					fmt.Sprint(hotspot.CacheHitBytes),
					fmt.Sprintf("%.6f", hotspot.CacheHitRatio),
					fmt.Sprintf("%.6f", hotspot.ShareOfTransfer),
				}); err != nil {
					f.Close()
					return err
				}
			}
		}
	}
	return closeCSV(f, w)
}

func WriteClassesCSV(path string, report *Report) error {
	f, w, err := createCSV(path, []string{
		"scenario", "pipeline", "data_class", "class_zh",
		"logical_read_bytes", "logical_write_bytes",
		"transfer_read_bytes", "transfer_write_bytes", "transfer_total_bytes",
		"cache_hit_bytes", "cache_hit_ratio",
	})
	if err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, pipeline := range scenario.Pipelines {
			for _, class := range pipeline.Classes {
				total := class.TransferReadBytes + class.TransferWriteBytes
				if err := w.Write([]string{
					scenario.Name,
					string(pipeline.ID),
					class.Name,
					class.Label,
					fmt.Sprint(class.LogicalReadBytes),
					fmt.Sprint(class.LogicalWriteBytes),
					fmt.Sprint(class.TransferReadBytes),
					fmt.Sprint(class.TransferWriteBytes),
					fmt.Sprint(total),
					fmt.Sprint(class.CacheHitBytes),
					fmt.Sprintf("%.6f", class.CacheHitRatioBytes),
				}); err != nil {
					f.Close()
					return err
				}
			}
		}
	}
	return closeCSV(f, w)
}

func WriteEventsCSV(path string, report *Report) error {
	f, w, err := createCSV(path, []string{
		"scenario", "frame", "pipeline", "stage", "eye", "class", "direction",
		"key", "reason", "logical_bytes", "transfer_bytes", "cache_hit", "cacheable",
	})
	if err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, frame := range scenario.FrameReports {
			for _, pipeline := range frame.Pipelines {
				for _, a := range pipeline.Accesses {
					if err := w.Write([]string{
						scenario.Name,
						fmt.Sprint(a.Frame),
						string(a.Pipeline),
						string(a.Stage),
						string(a.Eye),
						string(a.Class),
						string(a.Direction),
						a.Key,
						a.Reason,
						fmt.Sprint(a.LogicalBytes),
						fmt.Sprint(a.TransferBytes),
						fmt.Sprint(a.CacheHit),
						fmt.Sprint(a.Cacheable),
					}); err != nil {
						f.Close()
						return err
					}
				}
			}
		}
	}
	return closeCSV(f, w)
}
