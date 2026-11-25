package main

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"strings"
)

type MetricsFile struct {
	Generator    string           `json:"generator"`
	Pairing      string           `json:"pairing"`
	WorldScaleMM float64          `json:"world_scale_mm"`
	Scenarios    []ScenarioReport `json:"scenarios"`
}

func writeMetrics(path string, runs []ScenarioRun) error {
	metrics := MetricsFile{
		Generator:    "groominterp (Go standard library only)",
		Pairing:      "Hilbert scalp atlas quantile resampling; no random pairing",
		WorldScaleMM: worldScaleMM,
	}
	for _, run := range runs {
		metrics.Scenarios = append(metrics.Scenarios, run.Report)
	}
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeReport(path string, runs []ScenarioRun) error {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	b.WriteString(`<title>Groom 中间姿态插值</title><style>`)
	b.WriteString(`*{box-sizing:border-box}body{margin:0;background:#f7f4ed;color:#26313b;font-family:'Noto Sans CJK SC','Source Han Sans SC','Microsoft YaHei',Arial,sans-serif;line-height:1.5}`)
	b.WriteString(`main{max-width:1320px;margin:0 auto;padding:38px 28px 70px}h1{font-size:31px;margin:0 0 8px}h2{font-size:22px;margin:42px 0 12px}h3{font-size:15px;margin:20px 0 8px}`)
	b.WriteString(`.lede{max-width:940px;color:#5d6874;margin:0 0 18px}.grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:18px}.card{background:#fffdfa;border:1px solid #d9d2c5;border-radius:14px;padding:18px}`)
	b.WriteString(`.badge{display:inline-block;background:#e7f4ed;color:#166534;border:1px solid #b6dec6;border-radius:999px;padding:3px 10px;font-size:12px;font-weight:700;margin:3px 5px 3px 0}`)
	b.WriteString(`.metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;margin:14px 0 20px}.metric{background:#fffdfa;border:1px solid #d9d2c5;border-radius:12px;padding:13px 14px}.metric b{display:block;font-size:20px}.metric span{font-size:11px;color:#68737e}`)
	b.WriteString(`.scenario{background:#fffdfa;border:1px solid #d9d2c5;border-radius:16px;padding:20px;margin:22px 0}.scenario img{display:block;width:100%;height:auto;border-radius:10px;border:1px solid #e2dbcf;background:#f7f4ed}`)
	b.WriteString(`.row{display:grid;grid-template-columns:minmax(0,1.45fr) minmax(300px,.7fr);gap:18px;align-items:start}table{width:100%;border-collapse:collapse;font-size:12px}th,td{text-align:left;padding:6px 8px;border-bottom:1px solid #e6dfd3}th{color:#65717d;font-weight:600}code{font-family:'JetBrains Mono','SFMono-Regular',Consolas,monospace;font-size:11px;background:#f0ece3;padding:2px 5px;border-radius:4px}`)
	b.WriteString(`a{color:#0f6f68}.note{font-size:12px;color:#66717c}.hash{word-break:break-all}.legend{display:flex;gap:14px;flex-wrap:wrap;font-size:12px;color:#5f6b76;margin-top:8px}`)
	b.WriteString(`@media(max-width:900px){.grid,.row,.metrics{grid-template-columns:1fr}}`)
	b.WriteString(`</style></head><body><main>`)
	b.WriteString(`<h1>两套 groom 的中间姿态插值</h1>`)
	b.WriteString(`<p class="lede">发丝数量可以不同。两套 groom 的根部在 Hilbert scalp atlas 上排序，按分位重采样到中间帧的发丝数量；只混合 atlas 中相邻发丝，不使用随机索引配对。每帧在中间姿态下插值，并以颜色表示控制点从 A 到 B 的迁移量。</p>`)
	b.WriteString(`<span class="badge">纯 Go 标准库</span><span class="badge">确定性配对</span><span class="badge">端点精确</span><span class="badge">尖端凸包约束</span>`)
	b.WriteString(`<div class="metrics">`)
	for _, run := range runs {
		fmt.Fprintf(&b, `<div class="metric"><b>%d → %d</b><span>%s 发丝数量 / 混合帧 %d</span></div>`,
			run.Report.StrandsA, run.Report.StrandsB, html.EscapeString(run.Report.Title), run.Report.BlendStrands)
	}
	for _, run := range runs {
		fmt.Fprintf(&b, `<div class="metric"><b>%d</b><span>%s 尖端越界帧数</span></div>`,
			countTipViolations(run.Report), html.EscapeString(run.Report.Title))
	}
	b.WriteString(`</div>`)
	b.WriteString(`<h2>结果</h2>`)
	for _, run := range runs {
		writeScenarioSection(&b, run)
	}
	b.WriteString(`<h2>方法与验收</h2><div class="grid">`)
	b.WriteString(`<div class="card"><h3>确定性配对</h3><p class="note">每根发丝的根部映射到半球面的 Hilbert 单元。两套 groom 分别按 atlas 顺序排列；当数量不同时，以 <code>u=j/(K-1)</code> 在 atlas 中相邻的两根发丝之间线性重采样。中间帧数量为 <code>round((1-t)nA+tnB)</code>，不使用 <code>math/rand</code>。</p></div>`)
	b.WriteString(`<div class="card"><h3>尖端不发散</h3><p class="note">输出控制点始终是两个对应源控制点的凸组合，因此每个控制点相对根部的距离、每段长度和尖端半径都不超过两端对应量的线性包络。测试以 1/128 步长扫描中间帧，越界阈值 1e-8。</p></div>`)
	b.WriteString(`<div class="card"><h3>中间姿态</h3><p class="note">角度使用最短角差插值；两套 groom 的局部发丝几何在各自头皮坐标系中完成对应，再统一施加中间姿态。</p></div>`)
	b.WriteString(`<div class="card"><h3>颜色</h3><p class="note">蓝 → 青 → 黄 → 橙 → 红表示对应控制点 A→B 的位移由小到大，单位为毫米。发丝上的圆点是控制点；迁移细节图中的虚线箭头是抽样控制点轨迹。</p></div>`)
	b.WriteString(`</div>`)
	b.WriteString(`</main></body></html>`)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func countTipViolations(report ScenarioReport) int {
	total := 0
	for _, frame := range report.Frames {
		total += frame.TipBoundViolations
	}
	return total
}

func writeScenarioSection(b *strings.Builder, run ScenarioRun) {
	report := run.Report
	b.WriteString(`<section class="scenario">`)
	fmt.Fprintf(b, `<h2 style="margin-top:0">%s</h2>`, html.EscapeString(report.Title))
	fmt.Fprintf(b, `<p class="note">A: %s · %d 根　B: %s · %d 根　混合 t=0.50: %d 根　最大迁移 %.1f mm</p>`,
		html.EscapeString(report.GroomA), report.StrandsA,
		html.EscapeString(report.GroomB), report.StrandsB,
		report.BlendStrands, report.MigrationScaleMM)
	fmt.Fprintf(b, `<img src="%s/three_view.svg" alt="%s 三视图">`, html.EscapeString(report.ID), html.EscapeString(report.Title))
	b.WriteString(`<h3>三视图细节</h3><div class="grid">`)
	fmt.Fprintf(b, `<div class="card"><img src="%s/control_migration.svg" alt="控制点迁移"></div>`, html.EscapeString(report.ID))
	fmt.Fprintf(b, `<div class="card"><img src="%s/motion_sheet.svg" alt="中间帧"></div>`, html.EscapeString(report.ID))
	b.WriteString(`</div>`)
	b.WriteString(`<h3>逐帧验收</h3><table><thead><tr><th>t</th><th>发丝</th><th>尖端包络</th><th>尖端越界</th><th>控制点越界</th><th>分段越界</th><th>最大迁移</th></tr></thead><tbody>`)
	for _, index := range []int{0, 8, 16, 24, 32} {
		frame := report.Frames[index]
		fmt.Fprintf(b, `<tr><td>%.3f</td><td>%d</td><td>%.4f</td><td>%d</td><td>%d</td><td>%d</td><td>%.1f mm</td></tr>`,
			frame.T, frame.StrandCount, frame.MaxTipRadius, frame.TipBoundViolations,
			frame.ControlViolations, frame.SpanViolations, frame.MaxMigrationMM)
	}
	b.WriteString(`</tbody></table>`)
	fmt.Fprintf(b, `<p class="note hash">确定性摘要：<code>%s</code></p>`, html.EscapeString(report.DeterminismHash))
	fmt.Fprintf(b, `<p class="note"><a href="%s/three_view.svg">三视图 SVG</a> · <a href="%s/control_migration.svg">控制点迁移 SVG</a> · <a href="%s/motion_sheet.svg">中间帧 SVG</a></p>`,
		html.EscapeString(report.ID), html.EscapeString(report.ID), html.EscapeString(report.ID))
	b.WriteString(`</section>`)
}
