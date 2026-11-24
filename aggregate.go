package eyebw

import "sort"

type Aggregate struct {
	Totals  `json:"totals"`
	Stages  []NamedTotal `json:"stages"`
	Classes []NamedTotal `json:"classes"`
	Eyes    []NamedTotal `json:"eyes"`
}

func (a Aggregate) LogicalTotal() uint64  { return a.Totals.LogicalTotal() }
func (a Aggregate) TransferTotal() uint64 { return a.Totals.TransferTotal() }

func AggregateAccesses(accesses []Access) Aggregate {
	var total Totals
	stages := make(map[StageID]*Totals)
	classes := make(map[DataClass]*Totals)
	eyes := make(map[EyeID]*Totals)

	for _, a := range accesses {
		total.addAccess(a)
		if stages[a.Stage] == nil {
			stages[a.Stage] = &Totals{}
		}
		stages[a.Stage].addAccess(a)
		if classes[a.Class] == nil {
			classes[a.Class] = &Totals{}
		}
		classes[a.Class].addAccess(a)
		if eyes[a.Eye] == nil {
			eyes[a.Eye] = &Totals{}
		}
		eyes[a.Eye].addAccess(a)
	}

	stageNames := make([]NamedTotal, 0, len(stages))
	for id, totals := range stages {
		stageNames = append(stageNames, NamedTotal{
			Name:   string(id),
			Label:  stageLabelZH(id),
			Totals: *totals,
		})
	}
	sort.Slice(stageNames, func(i, j int) bool { return stageNames[i].Name < stageNames[j].Name })

	classNames := make([]NamedTotal, 0, len(AllClasses()))
	for _, class := range AllClasses() {
		totals := classes[class]
		value := Totals{}
		if totals != nil {
			value = *totals
		}
		classNames = append(classNames, NamedTotal{
			Name:   string(class),
			Label:  class.LabelZH(),
			Totals: value,
		})
	}

	eyeNames := make([]NamedTotal, 0, len(eyes))
	for id, totals := range eyes {
		eyeNames = append(eyeNames, NamedTotal{Name: string(id), Totals: *totals})
	}
	sort.Slice(eyeNames, func(i, j int) bool { return eyeNames[i].Name < eyeNames[j].Name })

	return Aggregate{Totals: total, Stages: stageNames, Classes: classNames, Eyes: eyeNames}
}

func stageLabelZH(stage StageID) string {
	switch stage {
	case StageDepthPrepass:
		return "深度预通道"
	case StageOpaquePass:
		return "不透明颜色通道"
	case StageResolve:
		return "MSAA 颜色解析"
	case StageDepthResolve:
		return "深度解析"
	case StageHistory:
		return "历史更新"
	case StageReproject:
		return "重投影/变形"
	case StageReuse:
		return "跨眼复用"
	case StageRepaint:
		return "补绘块"
	case StageScanout:
		return "扫描输出"
	default:
		return string(stage)
	}
}

func BuildHotspots(stages []NamedTotal) []Hotspot {
	out := make([]Hotspot, 0, len(stages))
	var totalTransfer uint64
	for _, stage := range stages {
		totalTransfer += stage.TransferTotal()
	}
	for _, stage := range stages {
		if stage.AccessCount == 0 {
			continue
		}
		share := 0.0
		if totalTransfer > 0 {
			share = float64(stage.TransferTotal()) / float64(totalTransfer)
		}
		out = append(out, Hotspot{
			Stage:              StageID(stage.Name),
			Label:              stage.Label,
			TransferBytes:      stage.TransferTotal(),
			LogicalBytes:       stage.LogicalTotal(),
			TransferReadBytes:  stage.TransferReadBytes,
			TransferWriteBytes: stage.TransferWriteBytes,
			CacheHitBytes:      stage.CacheHitBytes,
			CacheHitRatio:      stage.CacheHitRatioBytes,
			ShareOfTransfer:    share,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TransferBytes != out[j].TransferBytes {
			return out[i].TransferBytes > out[j].TransferBytes
		}
		if out[i].LogicalBytes != out[j].LogicalBytes {
			return out[i].LogicalBytes > out[j].LogicalBytes
		}
		return out[i].Stage < out[j].Stage
	})
	for i := range out {
		out[i].Rank = i + 1
	}
	return out
}

func TopHotspot(h []Hotspot) *Hotspot {
	if len(h) == 0 {
		return nil
	}
	return &h[0]
}
