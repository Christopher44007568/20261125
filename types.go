package eyebw

import (
	"fmt"
	"sort"
)

// DataClass identifies the semantic payload carried by an access.
type DataClass string

const (
	ClassColor   DataClass = "color"
	ClassDepth   DataClass = "depth"
	ClassHistory DataClass = "history"
	ClassRepaint DataClass = "repaint"
)

func (c DataClass) Valid() bool {
	switch c {
	case ClassColor, ClassDepth, ClassHistory, ClassRepaint:
		return true
	default:
		return false
	}
}

func (c DataClass) LabelZH() string {
	switch c {
	case ClassColor:
		return "颜色"
	case ClassDepth:
		return "深度"
	case ClassHistory:
		return "历史"
	case ClassRepaint:
		return "补绘"
	default:
		return string(c)
	}
}

func AllClasses() []DataClass {
	return []DataClass{ClassColor, ClassDepth, ClassHistory, ClassRepaint}
}

// Direction is the logical direction requested by a pipeline stage.
type Direction string

const (
	DirRead  Direction = "read"
	DirWrite Direction = "write"
)

type PipelineID string

const (
	PipelineSeparateEyes PipelineID = "separate_eye_buffers"
	PipelineAtlas        PipelineID = "binocular_atlas"
	PipelineSingleReuse  PipelineID = "single_eye_reuse"
)

func AllPipelines() []PipelineID {
	return []PipelineID{PipelineSeparateEyes, PipelineAtlas, PipelineSingleReuse}
}

func (p PipelineID) LabelZH() string {
	switch p {
	case PipelineSeparateEyes:
		return "独立 eye-buffer"
	case PipelineAtlas:
		return "双眼图集"
	case PipelineSingleReuse:
		return "单眼复用"
	default:
		return string(p)
	}
}

type EyeID string

const (
	EyeLeft   EyeID = "left"
	EyeRight  EyeID = "right"
	EyeShared EyeID = "shared"
	EyeAtlas  EyeID = "atlas"
)

type StageID string

const (
	StageDepthPrepass StageID = "depth_prepass"
	StageOpaquePass   StageID = "opaque_color_pass"
	StageResolve      StageID = "msaa_resolve"
	StageDepthResolve StageID = "depth_resolve"
	StageHistory      StageID = "history_update"
	StageReproject    StageID = "reprojection_warp"
	StageReuse        StageID = "cross_eye_reuse"
	StageRepaint      StageID = "repaint_patches"
	StageScanout      StageID = "scanout"
)

func AllStages() []StageID {
	return []StageID{
		StageDepthPrepass,
		StageOpaquePass,
		StageResolve,
		StageDepthResolve,
		StageHistory,
		StageReproject,
		StageReuse,
		StageRepaint,
		StageScanout,
	}
}

// Access is one instrumented stage request. TransferBytes is the amount that
// actually moves over the bus/memory path. A served cache hit has zero transfer.
type Access struct {
	Frame         int        `json:"frame"`
	Pipeline      PipelineID `json:"pipeline"`
	Stage         StageID    `json:"stage"`
	Eye           EyeID      `json:"eye"`
	Class         DataClass  `json:"class"`
	Direction     Direction  `json:"direction"`
	Key           string     `json:"key,omitempty"`
	Reason        string     `json:"reason,omitempty"`
	LogicalBytes  uint64     `json:"logical_bytes"`
	TransferBytes uint64     `json:"transfer_bytes"`
	CacheHit      bool       `json:"cache_hit"`
	Cacheable     bool       `json:"cacheable"`
}

// Totals separates logical stage traffic from real transfer traffic.
type Totals struct {
	LogicalReadBytes   uint64  `json:"logical_read_bytes"`
	LogicalWriteBytes  uint64  `json:"logical_write_bytes"`
	TransferReadBytes  uint64  `json:"transfer_read_bytes"`
	TransferWriteBytes uint64  `json:"transfer_write_bytes"`
	CacheHitBytes      uint64  `json:"cache_hit_bytes"`
	CacheMissBytes     uint64  `json:"cache_miss_bytes"`
	NonCacheableBytes  uint64  `json:"non_cacheable_bytes"`
	AccessCount        int     `json:"access_count"`
	CacheableReadBytes uint64  `json:"cacheable_read_bytes"`
	CacheHitRatioBytes float64 `json:"cache_hit_ratio_bytes"`
}

func (t Totals) LogicalTotal() uint64 {
	return t.LogicalReadBytes + t.LogicalWriteBytes
}

func (t Totals) TransferTotal() uint64 {
	return t.TransferReadBytes + t.TransferWriteBytes
}

func (t Totals) String() string {
	return fmt.Sprintf("logical=%s transfer=%s hit=%s (%.1f%%)",
		FormatBytes(t.LogicalTotal()), FormatBytes(t.TransferTotal()),
		FormatBytes(t.CacheHitBytes), t.CacheHitRatioBytes*100)
}

func (t *Totals) addAccess(a Access) {
	t.AccessCount++
	if a.Direction == DirRead {
		t.LogicalReadBytes += a.LogicalBytes
		t.TransferReadBytes += a.TransferBytes
	} else {
		t.LogicalWriteBytes += a.LogicalBytes
		t.TransferWriteBytes += a.TransferBytes
	}
	if a.CacheHit {
		t.CacheHitBytes += a.LogicalBytes
	}
	if a.Direction == DirRead {
		if a.Cacheable {
			t.CacheableReadBytes += a.LogicalBytes
			if a.CacheHit {
				// Already represented by CacheHitBytes.
			} else {
				t.CacheMissBytes += a.TransferBytes
			}
		} else {
			t.NonCacheableBytes += a.TransferBytes
		}
	}
	if t.CacheableReadBytes > 0 {
		t.CacheHitRatioBytes = float64(t.CacheHitBytes) / float64(t.CacheableReadBytes)
	}
}

func (t Totals) add(other Totals) Totals {
	var out Totals
	out.AccessCount = t.AccessCount + other.AccessCount
	out.LogicalReadBytes = t.LogicalReadBytes + other.LogicalReadBytes
	out.LogicalWriteBytes = t.LogicalWriteBytes + other.LogicalWriteBytes
	out.TransferReadBytes = t.TransferReadBytes + other.TransferReadBytes
	out.TransferWriteBytes = t.TransferWriteBytes + other.TransferWriteBytes
	out.CacheHitBytes = t.CacheHitBytes + other.CacheHitBytes
	out.CacheMissBytes = t.CacheMissBytes + other.CacheMissBytes
	out.NonCacheableBytes = t.NonCacheableBytes + other.NonCacheableBytes
	out.CacheableReadBytes = t.CacheableReadBytes + other.CacheableReadBytes
	if out.CacheableReadBytes > 0 {
		out.CacheHitRatioBytes = float64(out.CacheHitBytes) / float64(out.CacheableReadBytes)
	}
	return out
}

type NamedTotal struct {
	Name  string `json:"name"`
	Label string `json:"label,omitempty"`
	Totals
}

type Hotspot struct {
	Stage              StageID `json:"stage"`
	Label              string  `json:"label_zh"`
	TransferBytes      uint64  `json:"transfer_bytes"`
	LogicalBytes       uint64  `json:"logical_bytes"`
	TransferReadBytes  uint64  `json:"transfer_read_bytes"`
	TransferWriteBytes uint64  `json:"transfer_write_bytes"`
	CacheHitBytes      uint64  `json:"cache_hit_bytes"`
	CacheHitRatio      float64 `json:"cache_hit_ratio_bytes"`
	ShareOfTransfer    float64 `json:"share_of_transfer"`
	Rank               int     `json:"rank"`
}

func sortedNamed(m map[string]Totals) []NamedTotal {
	out := make([]NamedTotal, 0, len(m))
	for name, totals := range m {
		out = append(out, NamedTotal{Name: name, Totals: totals})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
