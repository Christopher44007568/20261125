// Package report aggregates per-marker verdicts into a frame-level
// diagnostic report and renders it as JSON or human-readable text.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"eyediag/internal/classifier"
	"eyediag/internal/model"
)

// CauseStat summarizes one cause across the whole frame.
type CauseStat struct {
	Cause        model.Cause       `json:"cause"`
	EyeHits      map[model.Eye]int `json:"eyeHits"`
	Stages       []model.Stage     `json:"stages"`
	MarkerIDs    []string          `json:"markerIds"`
	DegradedHits int               `json:"degradedHits"`
}

// Report is the final frame-level output.
type Report struct {
	CaseName      string                     `json:"caseName"`
	MarkerCount   int                        `json:"markerCount"`
	CorrectEyes   int                        `json:"correctEyes"`
	FaultyEyes    int                        `json:"faultyEyes"`
	Asymmetric    bool                       `json:"asymmetric"`
	CauseStats    []CauseStat                `json:"causeStats"`
	Verdicts      []classifier.MarkerVerdict `json:"verdicts"`
	PixelDiffNote string                     `json:"pixelDiffNote"`
}

// Build runs the classifier over every marker and aggregates the results.
func Build(c model.Case, th model.Thresholds) Report {
	th = th.Resolve()
	r := Report{CaseName: c.Name, MarkerCount: len(c.Markers)}
	stats := map[model.Cause]*CauseStat{}

	for _, m := range c.Markers {
		v := classifier.Classify(m, th)
		r.Verdicts = append(r.Verdicts, v)

		faultyThisMarker := 0
		for _, eye := range []model.Eye{model.Left, model.Right} {
			ev, ok := v.Eyes[eye]
			if !ok {
				continue
			}
			if ev.Correct {
				r.CorrectEyes++
			} else {
				r.FaultyEyes++
				faultyThisMarker++
			}
			for _, l := range ev.Labels {
				s := stats[l.Cause]
				if s == nil {
					s = &CauseStat{Cause: l.Cause, EyeHits: map[model.Eye]int{}}
					stats[l.Cause] = s
				}
				s.EyeHits[eye]++
				s.MarkerIDs = appendUnique(s.MarkerIDs, m.ID)
				if len(ev.DegradedBy) > 0 {
					s.DegradedHits++
				}
			}
		}
		// Asymmetric: exactly one eye of a marker is faulty.
		if faultyThisMarker == 1 {
			r.Asymmetric = true
		}
	}

	for _, s := range stats {
		s.Stages = stagesFor(s.Cause)
		r.CauseStats = append(r.CauseStats, *s)
	}
	sort.Slice(r.CauseStats, func(i, j int) bool { return r.CauseStats[i].Cause < r.CauseStats[j].Cause })
	r.PixelDiffNote = "总像素差（totalPixelDiff）仅作差异上下文展示，不参与任何因果判定"
	return r
}

// WriteJSON emits the machine-readable report.
func (r Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteText emits the operator-facing Chinese report.
func (r Report) WriteText(w io.Writer) {
	fmt.Fprintf(w, "诊断报告：%s\n", r.CaseName)
	fmt.Fprintf(w, "标记像素 %d 个；正常眼 %d，异常眼 %d\n", r.MarkerCount, r.CorrectEyes, r.FaultyEyes)
	if r.Asymmetric {
		fmt.Fprintln(w, "检出单眼正确/单眼错误的非对称情形")
	}
	fmt.Fprintln(w, strings.Repeat("-", 48))

	if len(r.CauseStats) == 0 {
		fmt.Fprintln(w, "未检出四类原因，全部标记像素在容差内。")
		return
	}
	for _, s := range r.CauseStats {
		fmt.Fprintf(w, "[原因] %s（定位阶段：%s）\n", causeName(s.Cause), joinStages(s.Stages))
		fmt.Fprintf(w, "  左眼命中 %d 次，右眼命中 %d 次；标记：%s\n",
			s.EyeHits[model.Left], s.EyeHits[model.Right], strings.Join(s.MarkerIDs, ", "))
		if s.DegradedHits > 0 {
			fmt.Fprintf(w, "  其中 %d 次发生在预算降级帧，已附降级因子\n", s.DegradedHits)
		}
	}

	fmt.Fprintln(w, strings.Repeat("-", 48))
	for _, v := range r.Verdicts {
		for _, eye := range []model.Eye{model.Left, model.Right} {
			ev, ok := v.Eyes[eye]
			if !ok {
				continue
			}
			if ev.Correct {
				fmt.Fprintf(w, "标记 %s / %s眼：正常（像素差 %.4f，仅参考）\n",
					v.MarkerID, eyeName(eye), ev.TotalPixelDiff)
				continue
			}
			names := []string{}
			for _, l := range ev.Labels {
				names = append(names, fmt.Sprintf("%s(%.2f)", causeName(l.Cause), l.Score))
			}
			fmt.Fprintf(w, "标记 %s / %s眼：%s\n", v.MarkerID, eyeName(eye), strings.Join(names, " + "))
			for _, l := range ev.Labels {
				for _, e := range l.Evidence {
					fmt.Fprintf(w, "    - %s\n", e)
				}
			}
			for _, d := range ev.DegradedBy {
				fmt.Fprintf(w, "    降级因子：%s\n", d)
			}
		}
	}
	fmt.Fprintf(w, "\n说明：%s。\n", r.PixelDiffNote)
}

func stagesFor(c model.Cause) []model.Stage {
	switch c {
	case model.Parallax:
		return []model.Stage{model.VertexStage}
	case model.Sorting, model.Alpha:
		return []model.Stage{model.LayerStage}
	case model.AA:
		return []model.Stage{model.SampleStage}
	}
	return nil
}

func appendUnique(xs []string, s string) []string {
	for _, x := range xs {
		if x == s {
			return xs
		}
	}
	return append(xs, s)
}

func causeName(c model.Cause) string {
	switch c {
	case model.Sorting:
		return "透明排序错误"
	case model.Parallax:
		return "视差/顶点深度错误"
	case model.Alpha:
		return "Alpha 混合/覆盖率错误"
	case model.AA:
		return "抗锯齿/采样错误"
	}
	return string(c)
}

func joinStages(ss []model.Stage) string {
	names := []string{}
	for _, s := range ss {
		switch s {
		case model.VertexStage:
			names = append(names, "顶点")
		case model.LayerStage:
			names = append(names, "图层")
		case model.SampleStage:
			names = append(names, "采样")
		}
	}
	return strings.Join(names, "、")
}

func eyeName(e model.Eye) string {
	if e == model.Left {
		return "左"
	}
	return "右"
}
