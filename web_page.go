package main

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>降级控制台 · Monocular / Sampling / Reprojection</title>
<style>
  :root { color-scheme: dark; }
  * { box-sizing: border-box; }
  body { margin:0; background:#070b16; color:#dce7ff;
    font:15px/1.5 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace; }
  header { padding:14px 18px; border-bottom:2px solid #22304f;
    display:flex; flex-wrap:wrap; gap:10px; align-items:baseline; }
  header h1 { font-size:18px; margin:0; letter-spacing:1px; }
  header .sub { color:#8fa3c8; font-size:12px; }
  main { max-width:1020px; margin:0 auto; padding:14px; }
  .screenwrap { border:1px solid #26365c; border-radius:6px; overflow:hidden;
    background:#0a0f1d; box-shadow:0 0 0 1px #0c1428, 0 12px 40px #000a; }
  img#screen { display:block; width:100%; height:auto; image-rendering:pixelated; }
  .controls { display:flex; flex-wrap:wrap; gap:8px; margin:14px 0; }
  button { background:#122040; color:#dce7ff; border:1px solid #33507f;
    border-radius:5px; padding:9px 13px; font:inherit; cursor:pointer; }
  button:hover { background:#1a2f5c; border-color:#54c8e8; }
  button.warn { border-color:#8a5a1d; } button.warn:hover { border-color:#f2a33c; }
  button.danger { border-color:#7d2b2b; } button.danger:hover { border-color:#ff544c; }
  .row { display:flex; flex-wrap:wrap; gap:8px; align-items:center; margin:6px 0; }
  input[type=number] { width:96px; background:#0c1428; border:1px solid #33507f;
    color:#dce7ff; border-radius:5px; padding:8px; font:inherit; }
  #live { min-height:44px; border-left:3px solid #2ecc71; padding:6px 12px;
    background:#0d1730; border-radius:0 5px 5px 0; margin:10px 0; }
  #live.warn { border-color:#f2a33c; } #live.critical { border-color:#ff544c; }
  details { margin:10px 0; } pre { background:#0a1122; border:1px solid #22304f;
    border-radius:6px; padding:10px; overflow:auto; max-height:320px; font-size:12px; }
  .hint { color:#8fa3c8; font-size:12px; }
  kbd { background:#16233f; border:1px solid #33507f; border-bottom-width:2px;
    border-radius:4px; padding:1px 6px; font-size:12px; }
</style>
</head>
<body>
<header>
  <h1>降级控制器 · 交互状态画面</h1>
  <span class="sub">Go standard library only · 单目 / 低采样 / 无重投影 · 安全底线不可静默</span>
</header>
<main>
  <div class="screenwrap"><img id="screen" alt="degradation status screen" src="/screen.png"></div>

  <div id="live" aria-live="polite">就绪：按 <kbd>Space</kbd> 步进，或先注入故障。</div>

  <div class="controls">
    <button data-act="step">步进 Space</button>
    <button class="warn" data-inject="budget">[B] 帧预算压力</button>
    <button class="danger" data-inject="severe">严重超标(直接降级)</button>
    <button class="warn" data-inject="frustum">[F] 视锥故障</button>
    <button class="warn" data-inject="pose">[P] 姿态超限</button>
    <button data-act="trim">[T] 裁剪姿态</button>
    <button class="danger" data-act="invalidate">[V] 核验作废</button>
    <button data-act="reset">[R] 重置</button>
  </div>
  <div class="row">
    <input id="ft" type="number" step="0.1" value="13.0" aria-label="frame time ms">
    <button data-act="stepft">按该帧时间步进(ms)</button>
    <button class="danger" data-act="eyes-zero">请求关闭双眼(应被拒)</button>
    <button class="danger" data-act="history-silent">静默关历史(应被拒)</button>
  </div>
  <p class="hint">键盘：<kbd>B</kbd> 预算 <kbd>F</kbd> 视锥 <kbd>P</kbd> 姿态 <kbd>T</kbd> 裁剪 <kbd>Space</kbd> 步进 <kbd>R</kbd> 重置。故障为持续注入，再按一次同类键清除。任何情况下至少保留一只眼睛，历史与交互/状态提示不会被静默关闭。</p>

  <details><summary>API 状态 JSON</summary><pre id="json">…</pre></details>
</main>
<script>
async function post(url, body){
  const opt = {method:'POST', headers:{'Content-Type':'application/json'}};
  if (body !== undefined) opt.body = JSON.stringify(body);
  const r = await fetch(url, opt); return r.json();
}
function show(st){
  const lv = document.getElementById('live');
  lv.className = st.status_level || 'ok';
  lv.textContent = 'F' + st.frame + ' · ' + (st.headline || '') +
    (st.active_episode_ids && st.active_episode_ids.length ? ' · 活动片段 ' + st.active_episode_ids.join(',') : '') +
    (st.safety_events && st.safety_events.length ? ' · 安全事件 ' + st.safety_events.length : '');
  document.getElementById('json').textContent = JSON.stringify(st, null, 2);
  document.getElementById('screen').src = '/screen.png?t=' + Date.now();
}
async function refresh(){ show(await (await fetch('/api/state')).json()); }
document.querySelectorAll('button').forEach(b => b.addEventListener('click', async () => {
  if (b.dataset.inject) { show(await post('/api/inject', {kind:b.dataset.inject})); return; }
  const a = b.dataset.act;
  if (a==='step') show(await post('/api/step', {}));
  if (a==='stepft'){ const v=parseFloat(document.getElementById('ft').value)||13;
    show(await post('/api/step', {frame_time_ms:v})); }
  if (a==='trim') show(await post('/api/trim'));
  if (a==='reset') show(await post('/api/reset'));
  if (a==='invalidate') show(await post('/api/invalidate'));
  if (a==='eyes-zero') show(await post('/api/probe/eyes-zero'));
  if (a==='history-silent') show(await post('/api/probe/history-silent'));
}));
window.addEventListener('keydown', async (e) => {
  if (e.target.tagName === 'INPUT') return;
  const k = e.key.toLowerCase();
  if (e.code === 'Space'){ e.preventDefault(); show(await post('/api/step', {})); }
  else if (k==='b') show(await post('/api/inject',{kind:'budget'}));
  else if (k==='f') show(await post('/api/inject',{kind:'frustum'}));
  else if (k==='p') show(await post('/api/inject',{kind:'pose'}));
  else if (k==='t') show(await post('/api/trim'));
  else if (k==='r') show(await post('/api/reset'));
  else if (k==='v') show(await post('/api/invalidate'));
});
refresh();
</script>
</body>
</html>`
