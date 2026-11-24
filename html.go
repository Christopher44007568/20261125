package main

import (
	"fmt"
	"html"
	"strings"
)

const reportCSS = `:root{--ink:#0f172a;--muted:#64748b;--line:#dbe3ee;--paper:#fff;--blue:#2563eb;--red:#dc2626;--teal:#0f766e}
*{box-sizing:border-box}body{margin:0;background:#edf2f7;color:var(--ink);font-family:Inter,"Noto Sans SC",system-ui,sans-serif;line-height:1.55}
main{max-width:1240px;margin:0 auto;padding:34px 24px 64px}.hero{background:linear-gradient(135deg,#0f172a,#1e3a5f);color:#fff;border-radius:24px;padding:34px 38px;box-shadow:0 24px 60px rgba(15,23,42,.18)}
.eyebrow{font-size:12px;letter-spacing:.18em;text-transform:uppercase;color:#93c5fd;font-weight:700}.hero h1{margin:8px 0 10px;font-size:clamp(26px,4vw,44px);line-height:1.12}.hero p{margin:0;color:#dbeafe;max-width:820px}
.chips{display:flex;flex-wrap:wrap;gap:9px;margin-top:22px}.chip{border:1px solid rgba(255,255,255,.24);background:rgba(255,255,255,.09);padding:6px 12px;border-radius:999px;font-size:12.5px}
.chip.ok{background:#dcfce7;color:#166534;border-color:#86efac}.chip.warn{background:#fee2e2;color:#991b1b;border-color:#fca5a5}
.card{background:var(--paper);border:1px solid var(--line);border-radius:20px;padding:24px;margin-top:22px;box-shadow:0 12px 34px rgba(15,23,42,.06)}
.card h2{font-size:20px;margin:0 0 6px}.card h3{font-size:14px;margin:22px 0 10px;color:#334155}.card p.lead{margin:0 0 16px;color:var(--muted);font-size:13px}
.svg-wrap{border:1px solid var(--line);border-radius:16px;overflow:hidden;background:#f8fafc}.svg-wrap svg{display:block;width:100%;height:auto}
.two{display:grid;grid-template-columns:1fr 1fr;gap:22px}
pre{white-space:pre-wrap;margin:0;background:#0f172a;color:#dbeafe;border-radius:14px;padding:18px;font:12px/1.55 ui-monospace,SFMono-Regular,Menlo,monospace;overflow:auto}
table{width:100%;border-collapse:collapse;font-size:12px}th{text-align:left;color:#475569;font-size:11px;letter-spacing:.04em;text-transform:uppercase;background:#f1f5f9;padding:10px}
td{padding:10px;border-top:1px solid #e7edf5;vertical-align:top}td.mono{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.ok{color:#15803d;font-weight:700}.bad{color:#dc2626;font-weight:700}
.badge{display:inline-block;padding:2px 7px;border-radius:999px;background:#e0f2fe;color:#075985;font-size:10.5px;font-weight:700}.badge.red{background:#fee2e2;color:#991b1b}
footer{margin-top:30px;color:#64748b;font-size:12px;text-align:center}@media(max-width:860px){.two{grid-template-columns:1fr}main{padding:18px 12px}.hero{padding:26px 22px}}`

func esc(s string) string { return html.EscapeString(s) }

func chip(text string, ok bool) string {
	cls := "ok"
	if !ok {
		cls = "warn"
	}
	return fmt.Sprintf(`<span class="chip %s">%s</span>`, cls, esc(text))
}

func edgeRows(t *Topology) string {
	var b strings.Builder
	for _, e := range t.Edges {
		cls := "badge"
		if e.Status == StatusNonManifold {
			cls = "badge red"
		}
		fmt.Fprintf(&b, `<tr><td class="mono">%s</td><td class="mono">v%d–v%d</td><td class="mono">%s</td><td><span class="%s">%s</span></td><td class="mono">%.3f</td></tr>`,
			esc(e.ID), e.U, e.V, esc(strings.Join(e.Faces, ", ")), cls, esc(e.Status), e.Length)
	}
	return b.String()
}

func mappingRows(maps []FaceMapping) string {
	var b strings.Builder
	fmtInts := func(xs []int) string {
		var s []string
		for _, x := range xs {
			s = append(s, fmt.Sprintf("v%d", x))
		}
		return strings.Join(s, ",")
	}
	for _, m := range maps {
		flag := func(ok bool) string {
			if ok {
				return `<span class="ok">✓</span>`
			}
			return `<span class="bad">✗</span>`
		}
		fmt.Fprintf(&b, `<tr><td class="mono">%s</td><td class="mono">%s<br>%s</td><td class="mono">%s → %s</td><td>%s%s%s</td></tr>`,
			esc(m.FaceID), esc(m.Provenance.SourceID), esc(m.Provenance.SourceFace),
			esc(fmtInts(m.BeforeVertices)), esc(fmtInts(m.AfterVertices)),
			flag(m.GeometryPreserved), flag(m.SourcePreserved), flag(m.NormalPreserved))
	}
	return b.String()
}

// RenderHTML 汇总覆盖图、修复前后视图与面来源对应表。
func RenderHTML(m *Mesh, rep *Report, coverageText, coverageSVG, beforeSVG, afterSVG string) string {
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"zh-CN\"><head><meta charset=\"utf-8\">")
	b.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	fmt.Fprintf(&b, "<title>%s</title><style>\n%s\n</style></head><body><main>", esc(m.Name), reportCSS)
	b.WriteString(`<header class="hero"><div class="eyebrow">Topology safety report</div>`)
	fmt.Fprintf(&b, "<h1>%s</h1>", esc(m.Name))
	b.WriteString(`<p>先识别共享脊边的三片薄板并发布缺陷覆盖图；经确认后只复制拓扑边端点，不删除、不重绕任何面。下图同时绘制拓扑边与面法线。</p><div class="chips">`)
	b.WriteString(chip("状态 · "+rep.Status, rep.RepairConfirmed))
	fmt.Fprintf(&b, `<span class="chip ok">面保留 %d/%d</span>`, rep.Safety.FaceCountAfter, rep.Safety.FaceCountBefore)
	b.WriteString(chip(fmt.Sprintf("面删除 %d", rep.Safety.FacesDeleted), rep.Safety.FacesDeleted == 0))
	b.WriteString(chip("法线保持", rep.Safety.AllNormalsPreserved))
	if rep.After != nil {
		b.WriteString(chip(fmt.Sprintf("修复后非流形边 %d", rep.After.NonManifoldCount), rep.After.NonManifoldCount == 0))
	} else {
		b.WriteString(chip(fmt.Sprintf("非流形边 %d", rep.Before.NonManifoldCount), rep.Before.NonManifoldCount == 0))
	}
	b.WriteString(`</div></header>`)
	b.WriteString(`<section class="card"><h2>缺陷覆盖图</h2><p class="lead">红色斜纹为直接覆盖非流形边的面，琥珀色为同一薄片上被波及的面。</p><div class="svg-wrap">`)
	b.WriteString(coverageSVG)
	b.WriteString(`</div><h3>终端版覆盖图</h3><pre>`)
	b.WriteString(esc(coverageText))
	b.WriteString(`</pre></section>`)
	b.WriteString(`<section class="card"><h2>修复前后 · 拓扑边与法线</h2><p class="lead">边颜色：蓝=边界，灰=流形，红=非流形；绿色箭头为面法线。修复后视图中紫色点为拆分复制的顶点，错位仅用于展示。</p><div class="two"><div class="svg-wrap">`)
	b.WriteString(beforeSVG)
	b.WriteString(`</div><div class="svg-wrap">`)
	if afterSVG != "" {
		b.WriteString(afterSVG)
	} else {
		b.WriteString(`<div class="empty">修复未确认，无修复后视图</div>`)
	}
	b.WriteString(`</div></div></section>`)
	b.WriteString(`<section class="card"><h2>拓扑边明细</h2><div class="two"><div><h3>修复前</h3><table><tr><th>边</th><th>端点</th><th>挂接面</th><th>状态</th><th>长度</th></tr>`)
	b.WriteString(edgeRowsFromSummary(rep.Before))
	b.WriteString(`</table></div><div><h3>修复后</h3>`)
	if rep.After != nil {
		b.WriteString(`<table><tr><th>边</th><th>端点</th><th>挂接面</th><th>状态</th><th>长度</th></tr>`)
		b.WriteString(edgeRowsFromSummary(rep.After))
		b.WriteString(`</table>`)
	} else {
		b.WriteString(`<div class="empty">尚未修复</div>`)
	}
	b.WriteString(`</div></div></section>`)
	b.WriteString(`<section class="card"><h2>面来源对应</h2><p class="lead">每个面在修复前后的顶点、来源与法线一一对应；勾选列为几何 / 来源 / 法线保持。</p><table><tr><th>面</th><th>来源</th><th>顶点 前→后</th><th>保持</th></tr>`)
	b.WriteString(mappingRows(rep.FaceMappings))
	b.WriteString(`</table></section>`)
	b.WriteString(`<footer>meshfix · 纯 Go 标准库 · 不删除任何面 · 修复需用户确认</footer></main></body></html>`)
	return b.String()
}

func edgeRowsFromSummary(s *TopoSummary) string {
	t := &Topology{Edges: s.Edges}
	return edgeRows(t)
}
