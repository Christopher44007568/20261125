package main

import (
	"fmt"
	"strings"
)

func reportHTML(d reportData) string {
	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>BRep surface fit · sparse cloud audit</title>
<style>
:root{--bg:#061018;--panel:#0b1720;--grid:#1b3540;--text:#e8f3f4;--muted:#7e9aa6;--faint:#4e6b76;--cyan:#58e1d2;--gold:#f2c96c;--orange:#ff9848;--violet:#d276ff;--red:#ff5d5d}
*{box-sizing:border-box}body{margin:0;background:radial-gradient(circle at 70% -10%,#102834 0,var(--bg) 45%);color:var(--text);font-family:'DejaVu Sans','Segoe UI',sans-serif}
body:before{content:"";position:fixed;inset:0;pointer-events:none;background-image:linear-gradient(#7e9aa60b 1px,transparent 1px),linear-gradient(90deg,#7e9aa60b 1px,transparent 1px);background-size:28px 28px}
main{max-width:1580px;margin:0 auto;padding:42px 42px 70px;position:relative}
.top{display:flex;justify-content:space-between;align-items:flex-start;border-bottom:1px solid var(--grid);padding-bottom:24px;margin-bottom:28px}
.eyebrow{color:var(--cyan);font-size:12px;font-weight:700;letter-spacing:.12em}.sub{color:var(--muted);font-size:12px;margin-top:7px;letter-spacing:.05em}
h1{font-size:40px;line-height:1.05;margin:14px 0 8px;font-weight:600;letter-spacing:-.025em}
.lede{color:var(--muted);font-size:14px;max-width:820px;line-height:1.65}
.badge{border:1px solid #24515a;background:#0d242c;color:var(--cyan);padding:10px 15px;border-radius:999px;font:600 10px ui-monospace,monospace;letter-spacing:.1em;white-space:nowrap}
.cards{display:grid;grid-template-columns:repeat(6,1fr);gap:12px;margin:24px 0 30px}
.card{background:#0b1720dd;border:1px solid var(--grid);border-radius:12px;padding:14px 15px}
.card small{display:block;color:var(--muted);font-size:9px;letter-spacing:.08em;font-weight:700;margin-bottom:9px}
.card b{font:500 24px ui-monospace,monospace;color:var(--cyan)}.card b.gold{color:var(--gold)}.card b.orange{color:var(--orange)}
.card span{font:400 9px ui-monospace,monospace;color:var(--faint);margin-left:4px}
.grid{display:grid;grid-template-columns:1fr 1fr;gap:22px}.full{grid-column:1/-1}
figure{margin:0;background:#0b1720;border:1px solid var(--grid);border-radius:16px;overflow:hidden}
figure img{display:block;width:100%;height:auto;background:#071017}
figcaption{display:flex;justify-content:space-between;gap:20px;padding:13px 16px;border-top:1px solid var(--grid);color:var(--muted);font-size:10px;line-height:1.5}
figcaption b{color:var(--text);font-size:11px;font-weight:600}figcaption em{font-style:normal;font-family:ui-monospace,monospace;color:var(--faint);white-space:nowrap}
.decision{margin-top:24px;border:1px solid #4c4b29;background:linear-gradient(100deg,#1b1e17,#0b1720);border-radius:14px;padding:20px 22px;display:flex;justify-content:space-between;gap:28px;align-items:center}
.decision h3{margin:0 0 6px;color:var(--gold);font-size:16px}.decision p{margin:0;color:var(--muted);font-size:12px;line-height:1.65;max-width:980px}
.decision code{font-family:ui-monospace,monospace;color:var(--text)}
table{width:100%;border-collapse:collapse;font-size:11px;background:#0b1720;border:1px solid var(--grid);border-radius:14px;overflow:hidden}
th,td{padding:10px 12px;text-align:left;border-bottom:1px solid #17303a}th{color:var(--faint);font-size:9px;letter-spacing:.07em;font-weight:700}
td{font-family:ui-monospace,monospace;color:#c7dadd}tr.sel td{background:#12323a;color:var(--text)}tr.rej td:last-child{color:var(--orange)}td:last-child{color:var(--muted)}
.note{margin-top:18px;color:var(--faint);font:400 10px ui-monospace,monospace;line-height:1.7}
@media(max-width:980px){.cards{grid-template-columns:repeat(3,1fr)}.grid{grid-template-columns:1fr}.top{display:block}.badge{display:inline-block;margin-top:16px}}
</style></head><body><main>`)

	b.WriteString(`<section class="top"><div><div class="eyebrow">SURFACE / FIT LAB · 20261125</div>
<h1>BRep-style preview from a sparse scan</h1>
<div class="lede">曲面由随机顺序稀疏点云鲁棒拟合。验收不采用控制点残差作为唯一依据，而是同时审计稠密高度、法线、曲率、高频波纹、投影轮廓与边界 Chamfer。噪声点和遮挡孔洞保持独立颜色。</div></div>
<div class="badge">CONVERGED · 12 × 9 CONTROL NET</div></section>`)

	fmt.Fprintf(&b, `<section class="cards">
<div class="card"><small>CONTROL RESIDUAL</small><b>%.3f</b><span>mm</span></div>
<div class="card"><small>DENSE HEIGHT</small><b>%.3f</b><span>mm</span></div>
<div class="card"><small>NORMAL p95</small><b class="gold">%.2f</b><span>deg</span></div>
<div class="card"><small>CURVATURE ERROR</small><b class="gold">%.1f</b><span>%%</span></div>
<div class="card"><small>RIPPLE ENERGY</small><b class="gold">%.1f</b><span>%%</span></div>
<div class="card"><small>SILHOUETTE IoU</small><b>%.5f</b><span>&nbsp;</span></div>
</section>`, d.Fit.InlierRMSmm, d.Audit.DenseRMSEmm, d.Audit.NormalP95deg,
		d.Audit.CurvatureRelPct, d.Audit.RippleRelPct, d.Audit.SilhouetteIoU)

	b.WriteString(`<section class="grid">`)
	fig := func(file, title, note, metric string) {
		fmt.Fprintf(&b, `<figure><a href="%s"><img src="%s" alt="%s"></a><figcaption><div><b>%s</b><br>%s</div><em>%s</em></figcaption></figure>`, file, file, title, title, note, metric)
	}
	fig("01_pointcloud.svg", "Raw sparse point cloud",
		"Clean samples are cyan; injected/robustly rejected noise is orange; H1–H3 are scanner occlusion masks.",
		fmt.Sprintf("%d SAMPLES · %d NOISE · 3 VOIDS", d.Input.Clean+d.Input.Noise, d.Input.Noise))
	fig("02_fitted_mesh.svg", "Fitted mesh / BRep preview",
		"Trimmed cubic B-spline, 56 × 36 rendered quads, four patch quadrants and visible u/v seams.",
		"CONTROL NET 12 × 9 · DEGREE 3")
	fig("03_normal_error.svg", "Per-face normal error",
		"Angular deviation to the withheld dense reference. Histogram and p95 marker document the distribution.",
		fmt.Sprintf("RMS %.2f° · p95 %.2f° · max %.2f°", d.Audit.NormalRMSdeg, d.Audit.NormalP95deg, d.Audit.NormalMaxdeg))
	fig("04_view_a.svg", "Rotation A · front quarter",
		"Solid cyan trim is the fit; dashed gold contour is the audit reference. Noise and occlusion remain separately coloured.",
		fmt.Sprintf("AZ −38° · CHAMFER %.3f mm", d.Audit.SilhouetteChamferMM))
	fig("05_view_b.svg", "Rotation B · rear quarter",
		"Second orbit view stresses the opposite trim and compound-curvature shoulder.",
		fmt.Sprintf("AZ +137° · IoU %.5f", d.Audit.SilhouetteIoU))
	fig("06_convergence.svg", "Resolution ladder",
		"Independent acceptance curves show the plateau at 12 × 9 and why the next refinement is rejected.",
		fmt.Sprintf("DENSE AUDIT · %d SAMPLES", d.Audit.DenseSamples))
	b.WriteString(`</section>`)

	fmt.Fprintf(&b, `<div class="decision"><div><h3>Selection is convergence-based</h3>
<p>Moving from <code>12 × 9</code> to <code>14 × 10</code> lowers dense height RMSE by only %.1f%%, while curvature error changes %+.1f%% and 4 mm ripple energy changes %+.1f%%. Silhouette IoU changes %+.5f. The model is stopped at the plateau rather than hiding surface waves behind a lower control-point residual.</p></div>
<div class="badge">SELECTED %s</div></div>`, d.Selection.HeightGainFromNextPct, d.Selection.CurvatureChangePct, d.Selection.RippleChangePct, d.Selection.SilhouetteChange, d.Selection.Selected)

	b.WriteString(`<h3 style="margin:34px 0 12px;font-size:15px;color:#cfe1e3">Convergence table</h3><table><thead><tr>
<th>CONTROL NET</th><th>INLIER RMS</th><th>DENSE RMSE</th><th>NORMAL p95</th><th>CURVATURE</th><th>RIPPLE</th><th>IoU</th><th>CHAMFER</th><th>DECISION</th>
</tr></thead><tbody>`)
	for _, st := range d.Convergence {
		class := ""
		if st.Selected {
			class = ` class="sel"`
		} else if strings.Contains(st.Decision, "rejected") {
			class = ` class="rej"`
		}
		fmt.Fprintf(&b, `<tr%s><td>%s</td><td>%.3f</td><td>%.3f</td><td>%.2f°</td><td>%.1f%%</td><td>%.1f%%</td><td>%.5f</td><td>%.3f</td><td>%s</td></tr>`,
			class, st.ControlNet, st.InlierRMSmm, st.DenseRMSEmm, st.NormalP95deg,
			st.CurvatureRelPct, st.RippleRelPct, st.SilhouetteIoU, st.ChamferMM, st.Decision)
	}
	b.WriteString(`</tbody></table>
<div class="note">REFERENCE USE · dense reference samples are generated only for post-fit acceptance; the fitting API accepts scan points alone. Occlusion masks are display metadata. Noise colours are shown independently from the smooth surface.</div>
</main></body></html>`)
	return b.String()
}
