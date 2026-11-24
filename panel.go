package main

import "fmt"

// renderPanel 绘制右侧拓扑摘要 / 图例 / 修复状态面板。
func renderPanel(s *svg, m *Mesh, opts renderOptions) {
	t := opts.Topo
	s.raw(`<rect x="850" y="104" width="232" height="620" rx="14" fill="#ffffff" stroke="#cbd5e1"/>`)
	s.text(868, 134, `font-family="sans-serif" font-size="15" font-weight="700" fill="#0f172a"`, "拓扑摘要")
	s.line(868, 146, 1064, 146, `stroke="#e2e8f0"`)
	y := 170.0
	row := func(label, value, color string) {
		s.text(868, y, `font-family="sans-serif" font-size="11" fill="#64748b"`, label)
		s.raw(`<text x="1064" y="%.1f" text-anchor="end" font-family="monospace" font-size="11" font-weight="700" fill="%s">%s</text>`,
			y, color, xmlEsc(value))
		y += 25
	}
	row("顶点 / 面", fmt.Sprintf("%d / %d", len(m.Vertices), len(m.Faces)), "#334155")
	row("边界边", fmt.Sprintf("%d", t.BoundaryCount), "#2563eb")
	row("流形边", fmt.Sprintf("%d", t.ManifoldCount), "#334155")
	nmColor := "#15803d"
	if t.NonManifoldCount > 0 {
		nmColor = "#dc2626"
	}
	row("非流形边", fmt.Sprintf("%d", t.NonManifoldCount), nmColor)
	row("面法线", fmt.Sprintf("%d 已绘制", len(m.Faces)), "#0f766e")
	y += 8
	s.line(868, y, 1064, y, `stroke="#e2e8f0"`)
	y += 20
	legend := func(color, label string) {
		s.line(868, y-4, 886, y-4, fmt.Sprintf(`stroke="%s" stroke-width="3" stroke-linecap="round"`, color))
		s.text(895, y, `font-family="sans-serif" font-size="10.5" fill="#475569"`, label)
		y += 23
	}
	legend("#2563eb", "边界边 · 1 面")
	legend("#334155", "流形边 · 2 面")
	legend("#dc2626", "非流形边 · ≥3 面")
	legend("#0f766e", fmt.Sprintf("面法线 N00…N%02d", len(m.Faces)-1))
	if opts.Mode == "coverage" {
		legend("#dc2626", "斜纹 · 直接覆盖")
		legend("#f59e0b", "琥珀 · 同薄片波及")
	}
	if opts.Mode == "after" {
		legend("#7c3aed", "拆分复制的顶点")
	}
	y += 12
	s.line(868, y, 1064, y, `stroke="#e2e8f0"`)
	y += 22
	s.text(868, y, `font-family="sans-serif" font-size="11" font-weight="700" fill="#0f172a"`, "修复状态")
	y += 21
	for i, line := range opts.StatusLines {
		color := "#475569"
		if i == 0 {
			color = opts.StatusColor
		}
		s.text(868, y, fmt.Sprintf(`font-family="sans-serif" font-size="10" fill="%s"`, color), line)
		y += 19
	}
	y += 6
	meshState := "manifold"
	stateColor := "#15803d"
	if t.NonManifoldCount > 0 {
		meshState = "non-manifold detected"
		stateColor = "#dc2626"
	}
	s.text(868, y, `font-family="sans-serif" font-size="10" fill="#64748b"`, "网格 "+meshState)
	_ = stateColor
	// 底部安全约束 / 修复结果信息框
	if opts.Mode == "after" {
		s.raw(`<rect x="868" y="620" width="196" height="86" rx="9" fill="#f0fdf4" stroke="#86efac"/>`)
		s.text(880, 641, `font-family="sans-serif" font-size="10.5" font-weight="700" fill="#166534"`, "修复结果")
		s.text(880, 659, `font-family="sans-serif" font-size="9.5" fill="#166534"`, fmt.Sprintf("拆分非流形边 %d 条", len(opts.Splits)))
		s.text(880, 674, `font-family="sans-serif" font-size="9.5" fill="#166534"`, "删除面 0 · 来源全部保留")
		s.text(880, 689, `font-family="sans-serif" font-size="9.5" fill="#166534"`, "法线与几何保持不变")
	} else {
		s.raw(`<rect x="868" y="620" width="196" height="86" rx="9" fill="#fff7ed" stroke="#fdba74"/>`)
		s.text(880, 641, `font-family="sans-serif" font-size="10.5" font-weight="700" fill="#9a3412"`, "安全约束")
		s.text(880, 659, `font-family="sans-serif" font-size="9.5" fill="#9a3412"`, "不自动删除任何面")
		s.text(880, 674, `font-family="sans-serif" font-size="9.5" fill="#9a3412"`, "先出覆盖图，再确认修复")
		s.text(880, 689, `font-family="sans-serif" font-size="9.5" fill="#9a3412"`, "拆分只复制脊边顶点")
	}
}
