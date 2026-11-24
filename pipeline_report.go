package eyebw

type EyeFoveation struct {
	Eye EyeID `json:"eye"`
	FoveationSummary
}

type RepaintReport struct {
	TotalTexels            uint64 `json:"total_texels"`
	FoveationUpgradeTexels uint64 `json:"foveation_upgrade_texels"`
	DisocclusionTexels     uint64 `json:"disocclusion_texels"`
	DynamicMotionTexels    uint64 `json:"dynamic_motion_texels"`
}

type PipelineReport struct {
	ID         PipelineID `json:"id"`
	Label      string     `json:"label_zh"`
	FrameIndex int        `json:"frame_index"`
	Aggregate
	Hotspots           []Hotspot      `json:"hotspots"`
	TopHotspot         *Hotspot       `json:"top_hotspot,omitempty"`
	Budget             BudgetReport   `json:"budget"`
	Quality            QualityReport  `json:"quality"`
	FoveationByEye     []EyeFoveation `json:"foveation_by_eye"`
	Repaint            RepaintReport  `json:"repaint"`
	CacheResidentBytes uint64         `json:"cache_resident_bytes"`
	Notes              []string       `json:"notes,omitempty"`
	Accesses           []Access       `json:"-"`
}

type FrameReport struct {
	Index     int               `json:"index"`
	Gazes     [2]Gaze           `json:"gazes"`
	Motion    float64           `json:"motion"`
	Pipelines []*PipelineReport `json:"pipelines"`
}

func (f *FrameReport) Get(id PipelineID) *PipelineReport {
	if f == nil {
		return nil
	}
	for _, report := range f.Pipelines {
		if report != nil && report.ID == id {
			return report
		}
	}
	return nil
}

func (f *FrameReport) BudgetExceededPipelines() []PipelineID {
	var out []PipelineID
	for _, report := range f.Pipelines {
		if report != nil && report.Budget.Exceeded {
			out = append(out, report.ID)
		}
	}
	return out
}

func (f *FrameReport) QualityFailedPipelines() []PipelineID {
	var out []PipelineID
	for _, report := range f.Pipelines {
		if report != nil && !report.Quality.Passed {
			out = append(out, report.ID)
		}
	}
	return out
}
