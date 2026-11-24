package main

import (
	"fmt"
	"strings"
)

// CoverageText 渲染终端版缺陷覆盖图：非流形边清单、面覆盖强度、薄片覆盖。
func CoverageText(m *Mesh, t *Topology) string {
	var b strings.Builder
	rule := strings.Repeat("=", 78)
	thin := strings.Repeat("-", 78)
	fmt.Fprintf(&b, "缺陷覆盖图 · %s\n%s\n", m.Name, rule)
	fmt.Fprintf(&b, "拓扑: %d 顶点 / %d 面 / %d 拓扑边\n",
		len(m.Vertices), len(m.Faces), len(t.Edges))
	fmt.Fprintf(&b, "边覆盖: 边界=%d  流形=%d  非流形=%d\n\n",
		t.BoundaryCount, t.ManifoldCount, t.NonManifoldCount)
	b.WriteString("非流形边覆盖\n" + thin + "\n")
	if len(t.NonManifoldEdges) == 0 {
		b.WriteString("（无）网格拓扑闭合，未发现非流形边。\n")
	}
	for _, e := range t.NonManifoldEdges {
		fmt.Fprintf(&b, "%s  v%d--v%d  %d 个面  ->  %s\n",
			e.ID, e.U, e.V, e.Count, strings.Join(e.Faces, ", "))
		for _, fi := range e.FaceIndices {
			p := m.Faces[fi].Provenance
			fmt.Fprintf(&b, "    %s  <-  %s/%s  [%s]\n",
				m.Faces[fi].ID, p.SourceID, p.SourceFace, p.Sheet)
		}
	}
	b.WriteString("\n面覆盖强度\n" + thin + "\n")
	for _, fc := range t.FaceCoverage {
		fmt.Fprintf(&b, "%-4s [%s] %-10s %-10s %s\n",
			fc.FaceID, coverageBar(fc.Coverage), fc.Coverage, fc.Sheet,
			strings.Join(fc.DirectNonManifoldEdges, ","))
	}
	b.WriteString("\n薄片覆盖\n" + thin + "\n")
	for _, sc := range t.SheetCoverage {
		fmt.Fprintf(&b, "%-8s faces=%-14s primary=%-14s edges=%s\n",
			sc.Sheet, strings.Join(sc.FaceIDs, ","),
			strings.Join(sc.PrimaryFaceIDs, ","),
			strings.Join(sc.NonManifoldEdgeIDs, ","))
	}
	b.WriteString("\n修复策略: ")
	if t.NonManifoldCount == 0 {
		b.WriteString("无需修复。\n")
	} else {
		fmt.Fprintf(&b, "按薄片连通分量拆分非流形脊边；0 个面被删除。\n")
	}
	return b.String()
}

func coverageBar(coverage string) string {
	switch coverage {
	case "direct":
		return strings.Repeat("█", 10)
	case "same-sheet":
		return strings.Repeat("█", 4) + strings.Repeat("░", 6)
	default:
		return strings.Repeat("░", 10)
	}
}
