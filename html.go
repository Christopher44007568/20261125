package eyestage

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

type htmlBucket struct {
	Label string
	Count int
	Width int
}

type htmlStat struct {
	StageStat
	EyeText   string
	StageText string
	Buckets   []htmlBucket
}

type htmlView struct {
	*Report
	Stats []htmlStat
}

var htmlReportTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"eye":   eyeLabel,
	"stage": stageLabel,
	"ns":    formatNS,
	"pct":   func(value float64) string { return fmt.Sprintf("%.1f%%", value*100) },
	"wait": func(value *int64) string {
		if value == nil {
			return "-"
		}
		return formatNS(*value)
	},
	"dur": func(value *int64) string {
		if value == nil {
			return "-"
		}
		return formatNS(*value)
	},
	"diagClass": func(code string) string {
		switch code {
		case "missing_stage", "missing_eye", "unsupported_timestamp":
			return "bad"
		case "timestamp_regression", "invalid_duration":
			return "warn"
		default:
			return "info"
		}
	},
	"diagText": func(diagnostic Diagnostic) string {
		items := make([]string, 0, 4)
		if diagnostic.Frame != "" {
			items = append(items, "frame="+diagnostic.Frame)
		}
		if diagnostic.Eye != "" {
			items = append(items, "eye="+diagnostic.Eye)
		}
		if diagnostic.Stage != "" {
			items = append(items, "stage="+diagnostic.Stage)
		}
		if diagnostic.FallbackSource != "" {
			items = append(items, "fallback="+diagnostic.FallbackSource)
		}
		return strings.Join(items, " ")
	},
}).Parse(htmlReportSource))

func WriteHTML(w io.Writer, report *Report) error {
	if err := validateReport(report); err != nil {
		return err
	}
	view := htmlView{Report: report, Stats: buildHTMLStats(report.StageStats)}
	return htmlReportTemplate.Execute(w, view)
}

func buildHTMLStats(stats []StageStat) []htmlStat {
	result := make([]htmlStat, 0, len(stats))
	for _, stat := range stats {
		maxCount := 0
		for _, bucket := range stat.Histogram {
			if bucket.Count > maxCount {
				maxCount = bucket.Count
			}
		}
		entry := htmlStat{
			StageStat: stat,
			EyeText:   eyeLabel(stat.Eye),
			StageText: stageLabel(stat.Stage),
		}
		for _, bucket := range stat.Histogram {
			width := 0
			if maxCount > 0 && bucket.Count > 0 {
				width = bucket.Count * 100 / maxCount
				if width < 2 {
					width = 2
				}
			}
			label := formatNS(bucket.LowerNS) + " - " + formatNS(bucket.UpperNS)
			if bucket.UpperNS <= bucket.LowerNS+1 {
				label = formatNS(bucket.LowerNS)
			}
			entry.Buckets = append(entry.Buckets, htmlBucket{
				Label: label, Count: bucket.Count, Width: width,
			})
		}
		result = append(result, entry)
	}
	return result
}
