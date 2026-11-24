package eyestage

const htmlReportSource = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>眼别阶段时间归因</title>
<style>
:root{color-scheme:dark;--bg:#0d1117;--panel:#161b22;--line:#30363d;--text:#e6edf3;--muted:#8b949e;--accent:#58a6ff;--ok:#3fb950;--warn:#d29922;--bad:#f85149}
*{box-sizing:border-box}body{margin:0;background:linear-gradient(160deg,#0d1117,#101822 55%,#0d1117);color:var(--text);font:14px/1.55 ui-sans-serif,system-ui,"Segoe UI",sans-serif}
main{max-width:1280px;margin:0 auto;padding:36px 24px 64px}h1{margin:0 0 6px;font-size:30px;letter-spacing:-.02em}h2{font-size:18px;margin:0 0 14px}h3{font-size:14px;margin:0 0 8px}
.sub{color:var(--muted);margin:0 0 26px}.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:12px;margin-bottom:28px}
.card,.panel{background:var(--panel);border:1px solid var(--line);border-radius:12px;padding:16px}
.metric{font-size:24px;font-weight:650;margin-top:4px}.metric small{font-size:12px;color:var(--muted);font-weight:400}.label{color:var(--muted);font-size:12px}
section{margin-top:30px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(420px,1fr));gap:14px}
.stat-head{display:flex;justify-content:space-between;gap:12px;align-items:baseline}.muted{color:var(--muted)}.mono{font-family:ui-monospace,SFMono-Regular,Consolas,monospace}
.bars{margin-top:14px;display:grid;gap:5px}.bar-row{display:grid;grid-template-columns:150px 1fr 42px;gap:8px;align-items:center;font-size:11px}
.track{height:14px;background:#0b1016;border:1px solid #252c35;border-radius:4px;overflow:hidden}.bar{height:100%;background:linear-gradient(90deg,#1f6feb,#58a6ff);border-radius:3px}
table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:9px 10px;border-bottom:1px solid var(--line);vertical-align:top}th{font-size:12px;color:var(--muted);font-weight:600}
tr:last-child td{border-bottom:0}.scroll{overflow:auto}.tag{display:inline-block;padding:2px 7px;border-radius:999px;border:1px solid var(--line);font-size:11px}
.bad{color:var(--bad)}.warn{color:var(--warn)}.info{color:var(--accent)}.empty{color:var(--muted);padding:12px 0}
.frame{border-bottom:1px solid var(--line);padding:14px 0}.frame:last-child{border:0}.frame-head{display:flex;gap:12px;align-items:center;margin-bottom:8px}
.eye-row{display:grid;grid-template-columns:150px 70px repeat(4,minmax(90px,1fr));gap:8px;padding:5px 0;font-size:12px}
@media(max-width:700px){.grid{grid-template-columns:1fr}.eye-row{grid-template-columns:1fr 1fr}.bar-row{grid-template-columns:110px 1fr 32px}h1{font-size:24px}main{padding:24px 14px}}
</style>
</head>
<body><main>
<h1>眼别阶段时间归因</h1>
<p class="sub">逐眼观察采集场景、重投影、畸变与合成；阶段耗时用于归因，帧跨度仅作上下文。</p>
<div class="cards">
<div class="card"><div class="label">帧 / 配对</div><div class="metric">{{.Summary.Frames}} <small>/ {{.Summary.PairedFrames}}</small></div></div>
<div class="card"><div class="label">阶段缺失</div><div class="metric bad">{{.Summary.MissingStages}}</div></div>
<div class="card"><div class="label">时间戳回退/倒退</div><div class="metric warn">{{.Summary.TimestampRegressions}}</div></div>
<div class="card"><div class="label">不可用时间戳</div><div class="metric warn">{{.Summary.UnsupportedTimes}}</div></div>
<div class="card"><div class="label">回退来源</div><div class="metric">{{.Summary.FallbackTimes}}</div></div>
<div class="card"><div class="label">长尾帧 / 阶段</div><div class="metric">{{.Summary.LongTailFrames}} <small>/ {{.Summary.LongTailStages}}</small></div></div>
</div>

<section><h2>逐眼帧概览</h2><div class="panel">
<div class="eye-row muted"><div>帧</div><div>眼别</div><div>阶段耗时</div><div>活跃跨度</div><div>完整阶段</div><div>空等</div></div>
{{range .Frames}}<div class="frame"><div class="frame-head"><strong class="mono">{{.FrameID}}</strong><span class="tag">帧跨度 {{ns .FrameSpanNS}}</span></div>
{{range .Eyes}}<div class="eye-row"><div></div><div>{{eye .Eye}}</div><div>{{ns .StageDurationNS}}</div><div>{{ns .ActiveSpanNS}}</div><div>{{.CompleteStages}}</div><div>{{wait .IdleWaitNS}}</div></div>
{{if .MissingStages}}<div class="eye-row"><div></div><div></div><div class="bad" style="grid-column:2 / -1">缺失: {{range .MissingStages}}{{stage .}} {{end}}</div></div>{{end}}
{{end}}
<details><summary class="muted">逐阶段时间戳</summary><div class="scroll"><table><thead><tr><th>眼别</th><th>阶段</th><th>开始</th><th>结束</th><th>时长</th></tr></thead><tbody>
{{range $eye := .Eyes}}{{range $st := $eye.Stages}}<tr><td>{{eye $eye.Eye}}</td><td>{{stage $st.Stage}}</td><td class="mono">{{if $st.Start}}{{ns $st.Start.NS}} <span class="muted">{{$st.Start.Source}}</span>{{else}}-{{end}}</td><td class="mono">{{if $st.End}}{{ns $st.End.NS}} <span class="muted">{{$st.End.Source}}</span>{{else}}-{{end}}</td><td class="mono">{{dur $st.DurationNS}}</td></tr>{{end}}{{end}}
</tbody></table></div></details>
</div>{{end}}
</div></section>

<section><h2>阶段耗时直方图</h2><div class="grid">
{{range .Stats}}<article class="card"><div class="stat-head"><div><h3>{{.StageText}}</h3><span class="muted">{{.EyeText}}</span></div><span class="tag">n={{.Count}}</span></div>
{{if .Count}}<div class="mono muted" style="font-size:11px">min {{ns .MinNS}} · p50 {{ns .P50NS}} · p95 {{ns .P95NS}} · p99 {{ns .P99NS}} · max {{ns .MaxNS}}</div>
<div class="bars">{{range .Buckets}}<div class="bar-row"><span class="mono muted">{{.Label}}</span><div class="track"><div class="bar" style="width:{{.Width}}%"></div></div><span class="mono">{{.Count}}</span></div>{{end}}</div>
{{else}}<div class="empty">无可用耗时</div>{{end}}</article>{{end}}
</div></section>

<section><h2>长尾帧 <span class="muted" style="font-size:12px">P{{printf "%.0f" .Config.TailPercentile}}，逐阶段归因</span></h2><div class="panel scroll">
{{if .LongTailFrames}}<table><thead><tr><th>帧</th><th>眼别</th><th>阶段</th><th>耗时</th><th>阈值</th></tr></thead><tbody>
{{range $frame := .LongTailFrames}}{{range $i, $entry := $frame.Stages}}<tr>{{if eq $i 0}}<td class="mono" rowspan="{{$frame.StageCount}}">{{$frame.FrameID}}</td>{{end}}<td>{{eye $entry.Eye}}</td><td>{{stage $entry.Stage}}</td><td class="mono warn">{{ns $entry.DurationNS}}</td><td class="mono muted">{{ns $entry.ThresholdNS}}</td></tr>{{end}}{{end}}
</tbody></table>{{else}}<div class="empty">无显著长尾阶段</div>{{end}}
</div></section>

<section><h2>跨眼对应阶段重叠</h2><div class="panel scroll">
<table><thead><tr><th>阶段</th><th>比较帧</th><th>重叠帧</th><th>左眼</th><th>右眼</th><th>重叠</th><th>仅左</th><th>仅右</th><th>重叠率</th></tr></thead><tbody>
{{range .StageOverlaps}}<tr><td>{{stage .Stage}}</td><td>{{.FramesCompared}}</td><td>{{.FramesOverlap}}</td><td class="mono">{{ns .LeftNS}}</td><td class="mono">{{ns .RightNS}}</td><td class="mono">{{ns .OverlapNS}}</td><td class="mono muted">{{ns .LeftOnlyNS}}</td><td class="mono muted">{{ns .RightOnlyNS}}</td><td>{{pct .OverlapRatio}}</td></tr>{{end}}
</tbody></table></div></section>

<section><h2>跨阶段错位重叠</h2><div class="panel scroll">
{{if .CrossStageOverlaps}}<table><thead><tr><th>左眼阶段</th><th>右眼阶段</th><th>重叠</th><th>帧数</th></tr></thead><tbody>
{{range .CrossStageOverlaps}}<tr><td>{{stage .LeftStage}}</td><td>{{stage .RightStage}}</td><td class="mono">{{ns .OverlapNS}}</td><td>{{.FrameCount}}</td></tr>{{end}}
</tbody></table>{{else}}<div class="empty">无跨阶段错位重叠</div>{{end}}
</div></section>

<section><h2>单眼空等</h2><div class="grid">
{{range .IdleWaits}}<article class="card"><div class="stat-head"><h3>{{eye .Eye}}</h3><span class="tag">{{.FramesWithWait}} 帧有等待</span></div>
<div class="cards" style="margin:14px 0 0"><div><div class="label">总等待</div><div class="metric" style="font-size:19px">{{ns .TotalWaitNS}}</div></div><div><div class="label">中位</div><div class="metric" style="font-size:19px">{{ns .MedianWaitNS}}</div></div><div><div class="label">P95</div><div class="metric" style="font-size:19px">{{ns .P95WaitNS}}</div></div><div><div class="label">最大</div><div class="metric" style="font-size:19px">{{ns .MaxWaitNS}}</div></div></div></article>{{end}}
</div></section>

<section><h2>诊断</h2><div class="panel scroll">
{{if .Diagnostics}}<table><thead><tr><th>级别</th><th>代码</th><th>上下文</th><th>说明</th></tr></thead><tbody>
{{range .Diagnostics}}<tr><td class="{{diagClass .Code}}">●</td><td class="mono {{diagClass .Code}}">{{.Code}}</td><td class="mono muted">{{diagText .}}</td><td>{{.Message}}</td></tr>{{end}}
</tbody></table>{{else}}<div class="empty">无诊断</div>{{end}}
</div></section>
</main></body></html>
`
