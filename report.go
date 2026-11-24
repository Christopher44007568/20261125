package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
)

type SweepSample struct {
	OffsetMM              float64 `json:"offset_mm"`
	SignedVolumeMM3       float64 `json:"signed_volume_mm3"`
	VolumeMM3             float64 `json:"volume_mm3"`
	Vertices              int     `json:"vertices"`
	Triangles             int     `json:"triangles"`
	InvertedSectionTris   int     `json:"inverted_section_triangles"`
	SelfIntersections     int     `json:"self_intersections"`
	SelfIntersectionLoops int     `json:"self_intersection_loops"`
}

type Continuity struct {
	FromMM             float64 `json:"from_mm"`
	ToMM               float64 `json:"to_mm"`
	StepMM             float64 `json:"step_mm"`
	Samples            int     `json:"samples"`
	MaxFirstDifference float64 `json:"max_first_difference_mm3"`
	MaxThirdDifference float64 `json:"max_third_difference_mm3"`
	TopologyConstant   bool    `json:"topology_constant"`
	NoRegionDeletion   bool    `json:"no_region_deletion"`
}

type Files struct {
	InnerSurface string `json:"inner_surface"`
	OuterSurface string `json:"outer_surface"`
	ReportJSON   string `json:"report_json"`
	Section      string `json:"section"`
	VolumeCSV    string `json:"volume_csv"`
}

type Report struct {
	Unit                      string        `json:"unit"`
	RequestedOffsetMM         float64       `json:"requested_offset_mm"`
	Parameters                Params        `json:"parameters"`
	Topology                  Topology      `json:"topology"`
	InvertedSectionTriangles  int           `json:"inverted_section_triangles"`
	SelfIntersections         int           `json:"self_intersections"`
	SelfIntersectionLoops     int           `json:"self_intersection_loops"`
	RedPixelsInSection        int           `json:"red_pixels_in_section"`
	VolumeMM3                 float64       `json:"volume_mm3"`
	SignedVolumeMM3           float64       `json:"signed_volume_mm3"`
	AnalyticRetainedVolumeMM3 float64       `json:"analytic_retained_volume_mm3"`
	Continuity                Continuity    `json:"continuity"`
	Sweep                     []SweepSample `json:"sweep"`
	Files                     Files         `json:"files"`
	Policy                    []string      `json:"policy"`
}

func sweep(p Params, from, to, step float64) []SweepSample {
	if step <= 0 || to < from {
		return nil
	}
	count := int(math.Floor((to-from)/step+1e-9)) + 1
	out := make([]SweepSample, 0, count)
	for i := 0; i < count; i++ {
		d := from + float64(i)*step
		q := p
		q.Offset = d
		profile := buildProfile(q)
		mesh := buildMesh(profile)
		diag := analyzeSection(profile)
		signed := mesh.SignedVolume()
		if math.Abs(signed) < 1e-9 {
			signed = 0
		}
		out = append(out, SweepSample{
			OffsetMM:              d,
			SignedVolumeMM3:       signed,
			VolumeMM3:             math.Abs(signed),
			Vertices:              len(mesh.Vertices),
			Triangles:             len(mesh.Triangles),
			InvertedSectionTris:   diag.Inverted,
			SelfIntersections:     len(diag.Intersections),
			SelfIntersectionLoops: diag.LoopCount,
		})
	}
	return out
}

func continuityFrom(samples []SweepSample, from, to, step float64) Continuity {
	cont := Continuity{
		FromMM: from, ToMM: to, StepMM: step, Samples: len(samples),
		TopologyConstant: true, NoRegionDeletion: true,
	}
	if len(samples) == 0 {
		return cont
	}
	vertices, triangles := samples[0].Vertices, samples[0].Triangles
	vals := make([]float64, len(samples))
	for i, s := range samples {
		vals[i] = s.SignedVolumeMM3
		if s.Vertices != vertices || s.Triangles != triangles {
			cont.TopologyConstant = false
		}
	}
	for i := 1; i < len(vals); i++ {
		d1 := math.Abs(vals[i] - vals[i-1])
		if d1 > cont.MaxFirstDifference {
			cont.MaxFirstDifference = d1
		}
	}
	for i := 3; i < len(vals); i++ {
		d3 := math.Abs(vals[i] - 3*vals[i-1] + 3*vals[i-2] - vals[i-3])
		if d3 > cont.MaxThirdDifference {
			cont.MaxThirdDifference = d3
		}
	}
	cont.NoRegionDeletion = cont.TopologyConstant
	return cont
}

func writeSweepCSV(w io.Writer, samples []SweepSample) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"offset_mm", "signed_volume_mm3", "volume_mm3", "vertices",
		"triangles", "inverted_section_triangles", "self_intersections", "loops",
	}); err != nil {
		return err
	}
	for _, s := range samples {
		row := []string{
			fmt.Sprintf("%.4f", s.OffsetMM),
			fmt.Sprintf("%.9f", s.SignedVolumeMM3),
			fmt.Sprintf("%.9f", s.VolumeMM3),
			strconv.Itoa(s.Vertices),
			strconv.Itoa(s.Triangles),
			strconv.Itoa(s.InvertedSectionTris),
			strconv.Itoa(s.SelfIntersections),
			strconv.Itoa(s.SelfIntersectionLoops),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeReport(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
