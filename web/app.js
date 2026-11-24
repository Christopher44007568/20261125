(() => {
  "use strict";

  const SCENE_W = 640;
  const SCENE_H = 360;
  const TAU = Math.PI * 2;
  const PALETTES = {
    "rough-metal": { hot: [255, 238, 194], mid: [255, 174, 72], accent: "#ffca68" },
    "mirror-sphere": { hot: [241, 253, 255], mid: [112, 199, 255], accent: "#9ddfff" },
    "glass": { hot: [236, 255, 247], mid: [80, 255, 176], accent: "#83f7c1" },
  };

  const state = {
    timeline: null,
    eye: 0,
    frame: 0,
    playing: true,
    showHistory: true,
    speed: 1,
    selected: "rough-metal",
    scenes: new Map(),
    dirty: true,
    lastTime: 0,
    accumulator: 0,
  };

  const dom = {
    grid: document.getElementById("scene-grid"),
    ledgerRows: document.getElementById("ledger-rows"),
    ledgerEye: document.getElementById("ledger-eye"),
    gateEye: document.getElementById("gate-eye"),
    gatePeak: document.getElementById("gate-peak"),
    gateEnergy: document.getElementById("gate-energy"),
    gatePeakBar: document.getElementById("gate-peak-bar"),
    gateEnergyBar: document.getElementById("gate-energy-bar"),
    gateHistoryEye: document.getElementById("gate-history-eye"),
    gateCapped: document.getElementById("gate-capped"),
    gateDelta: document.getElementById("gate-delta"),
    trace: document.getElementById("trace"),
    playBtn: document.getElementById("play-btn"),
    timecode: document.getElementById("timecode"),
    scrub: document.getElementById("scrub"),
    frameLabel: document.getElementById("frame-label"),
    motionLabel: document.getElementById("motion-label"),
    historyToggle: document.getElementById("history-toggle"),
    speedBtn: document.getElementById("speed-btn"),
    auditCrossEye: document.getElementById("audit-cross-eye"),
    auditStereo: document.getElementById("audit-stereo"),
    auditCap: document.getElementById("audit-cap"),
    auditWarp: document.getElementById("audit-warp"),
  };

  async function boot() {
    try {
      const response = await fetch("/api/timeline", { cache: "no-store" });
      if (!response.ok) throw new Error(`timeline HTTP ${response.status}`);
      state.timeline = await response.json();
    } catch (error) {
      dom.grid.innerHTML = `<div class="loading-card"><span>TIMELINE UNAVAILABLE · ${escapeHTML(error.message)}</span></div>`;
      return;
    }

    applyDeepLink();
    buildScenes();
    buildLedger();
    bindControls();
    updateAudit();
    updateStaticLabels();
    fitAllCanvases();
    state.dirty = true;
    requestAnimationFrame(tick);
  }

  function buildScenes() {
    dom.grid.innerHTML = "";
    state.timeline.scenes.forEach((scene, index) => {
      const card = document.createElement("article");
      card.className = "scene-card" + (scene.id === state.selected ? " is-selected" : "");
      card.dataset.object = scene.id;
      card.style.setProperty("--accent", scene.accent);
      card.tabIndex = 0;
      card.setAttribute("aria-label", `${scene.name}, ${scene.motion}`);
      card.innerHTML = `
        <div class="scene-canvas-wrap">
          <canvas class="scene-canvas" aria-label="${scene.name} highlight simulation"></canvas>
          <div class="scene-index"><b>0${index + 1}</b><span>${scene.motion}</span></div>
          <div class="scene-eye">EYE L · TRACKED</div>
          <div class="scene-readout">
            <span>P <b data-peak>0.00</b></span>
            <span>E <b data-energy>0.00</b></span>
          </div>
        </div>
        <footer class="scene-foot">
          <div>
            <h3>${scene.name}</h3>
            <p>${scene.material}</p>
          </div>
          <div class="scene-metrics">
            <div><span>CENTER</span><b data-center>0/0</b></div>
            <div><span>STRETCH</span><b data-stretch>0</b></div>
            <div><span>ENERGY</span><b data-energy-foot>0.00</b></div>
          </div>
        </footer>`;

      const choose = () => selectObject(scene.id);
      card.addEventListener("click", choose);
      card.addEventListener("keydown", (event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          choose();
        }
      });
      dom.grid.appendChild(card);

      const canvas = card.querySelector(".scene-canvas");
      const ctx = canvas.getContext("2d");
      state.scenes.set(scene.id, {
        scene,
        card,
        canvas,
        ctx,
        peak: card.querySelector("[data-peak]"),
        energy: card.querySelector("[data-energy]"),
        center: card.querySelector("[data-center]"),
        stretch: card.querySelector("[data-stretch]"),
        energyFoot: card.querySelector("[data-energy-foot]"),
        eye: card.querySelector(".scene-eye"),
      });
    });
  }

  function applyDeepLink() {
    const params = new URLSearchParams(window.location.search);
    const frame = Number(params.get("frame"));
    if (Number.isFinite(frame)) {
      state.frame = Math.max(0, Math.min(state.timeline.frameCount - 1, Math.round(frame)));
    }
    if (params.get("eye")?.toUpperCase() === "R") state.eye = 1;
    if (params.has("paused") || params.get("playing") === "0") state.playing = false;
    const objectID = params.get("object");
    if (state.timeline.scenes.some((scene) => scene.id === objectID)) {
      state.selected = objectID;
    }
  }

  function buildLedger() {
    dom.ledgerRows.innerHTML = "";
    state.timeline.scenes.forEach((scene) => {
      const row = document.createElement("div");
      row.className = "ledger-row" + (scene.id === state.selected ? " is-selected" : "");
      row.dataset.object = scene.id;
      row.style.setProperty("--row-accent", scene.accent);
      row.innerHTML = `
        <div class="ledger-object"><i></i><span>${scene.name}</span></div>
        <div class="ledger-value"><b data-x>0</b> / <b data-y>0</b></div>
        <div class="ledger-value"><b data-stretch>0</b> px</div>
        <div class="ledger-value"><b data-peak>0.00</b></div>
        <div class="energy-cell"><span data-energy>0.00</span><div class="energy-mini"><i data-bar></i></div></div>`;
      row.addEventListener("click", () => selectObject(scene.id));
      dom.ledgerRows.appendChild(row);
      scene.row = row;
      scene.rowX = row.querySelector("[data-x]");
      scene.rowY = row.querySelector("[data-y]");
      scene.rowStretch = row.querySelector("[data-stretch]");
      scene.rowPeak = row.querySelector("[data-peak]");
      scene.rowEnergy = row.querySelector("[data-energy]");
      scene.rowBar = row.querySelector("[data-bar]");
    });
  }

  function bindControls() {
    dom.playBtn.addEventListener("click", () => {
      state.playing = !state.playing;
      state.lastTime = 0;
      state.dirty = true;
      syncPlayButton();
    });

    document.getElementById("step-back").addEventListener("click", () => step(-1));
    document.getElementById("step-forward").addEventListener("click", () => step(1));

    document.querySelectorAll(".eye-btn").forEach((button) => {
      button.addEventListener("click", () => {
        state.eye = Number(button.dataset.eye);
        document.querySelectorAll(".eye-btn").forEach((item) => {
          item.classList.toggle("is-active", item === button);
        });
        state.dirty = true;
        updateStaticLabels();
      });
    });

    dom.scrub.max = String(state.timeline.frameCount - 1);
    dom.scrub.addEventListener("input", () => {
      state.playing = false;
      state.frame = Number(dom.scrub.value);
      state.lastTime = 0;
      state.dirty = true;
      syncPlayButton();
    });

    dom.historyToggle.addEventListener("click", () => {
      state.showHistory = !state.showHistory;
      dom.historyToggle.classList.toggle("is-on", state.showHistory);
      dom.historyToggle.setAttribute("aria-pressed", String(state.showHistory));
      state.dirty = true;
    });

    dom.speedBtn.addEventListener("click", () => {
      const speeds = [0.5, 1, 2];
      const current = speeds.indexOf(state.speed);
      state.speed = speeds[(current + 1) % speeds.length];
      dom.speedBtn.textContent = `${state.speed}×`;
      state.dirty = true;
    });

    window.addEventListener("keydown", (event) => {
      if (event.target instanceof HTMLInputElement) return;
      if (event.code === "Space") {
        event.preventDefault();
        dom.playBtn.click();
      } else if (event.code === "ArrowLeft") {
        event.preventDefault();
        step(-1);
      } else if (event.code === "ArrowRight") {
        event.preventDefault();
        step(1);
      } else if (event.key.toLowerCase() === "h") {
        dom.historyToggle.click();
      }
    });

    window.addEventListener("resize", fitAllCanvases);
    if ("ResizeObserver" in window) {
      const observer = new ResizeObserver(() => fitAllCanvases());
      observer.observe(dom.grid);
      observer.observe(dom.trace.parentElement);
    }
    syncPlayButton();
  }

  function step(delta) {
    state.playing = false;
    state.frame = (state.frame + delta + state.timeline.frameCount) % state.timeline.frameCount;
    state.dirty = true;
    syncPlayButton();
  }

  function syncPlayButton() {
    dom.playBtn.classList.toggle("is-paused", !state.playing);
    dom.playBtn.setAttribute("aria-label", state.playing ? "Pause" : "Play");
  }

  function selectObject(id) {
    state.selected = id;
    state.scenes.forEach((entry, key) => {
      entry.card.classList.toggle("is-selected", key === id);
      entry.row.classList.toggle("is-selected", key === id);
    });
    state.dirty = true;
  }

  function updateStaticLabels() {
    if (!state.timeline) return;
    document.querySelectorAll(".eye-btn").forEach((button) => {
      button.classList.toggle("is-active", Number(button.dataset.eye) === state.eye);
    });
    const label = state.eye === 0 ? "LEFT EYE · L" : "RIGHT EYE · R";
    dom.ledgerEye.textContent = label;
    dom.gateEye.innerHTML = `EYE ${state.eye === 0 ? "L" : "R"} · <span class="status-ok">ISOLATED</span>`;
    state.scenes.forEach((entry) => {
      entry.eye.textContent = `EYE ${state.eye === 0 ? "L" : "R"} · TRACKED`;
    });
    dom.motionLabel.textContent = `${state.timeline.fps} FPS · ${state.timeline.audit.frames} FRAME SAMPLES`;
  }

  function updateAudit() {
    const audit = state.timeline.audit;
    dom.auditCrossEye.textContent = audit.crossEyeSamples;
    dom.auditStereo.textContent =
      `${(audit.eyeDirectionDeltaMax * 180 / Math.PI).toFixed(2)}°`;
    dom.auditCap.textContent = audit.energyCapViolations;
    dom.auditWarp.textContent = audit.fullFrameTimewarps;
  }

  function frameData() {
    return state.timeline.eyes[state.eye].frames[state.frame];
  }

  function trackFor(frame, objectID) {
    return frame.tracks.find((track) => track.objectId === objectID);
  }

  function objectFor(frame, objectID) {
    return frame.objects.find((object) => object.id === objectID);
  }

  function escapeHTML(value) {
    return String(value).replace(/[&<>"']/g, (char) => ({
      "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
    }[char]));
  }

  function tick(now) {
    if (!state.lastTime) state.lastTime = now;
    const delta = Math.min(0.1, (now - state.lastTime) / 1000);
    state.lastTime = now;

    if (state.playing) {
      state.accumulator += delta * state.timeline.fps * state.speed;
      if (state.accumulator >= 1) {
        const advance = Math.floor(state.accumulator);
        state.frame = (state.frame + advance) % state.timeline.frameCount;
        state.accumulator -= advance;
        state.dirty = true;
      }
    }

    if (state.dirty) {
      render();
      state.dirty = false;
    }
    requestAnimationFrame(tick);
  }

  function render() {
    const frame = frameData();
    state.scenes.forEach((entry, id) => {
      const object = objectFor(frame, id);
      const track = trackFor(frame, id);
      drawScene(entry, object, track);
      updateSceneUI(entry, track);
      updateLedger(entry.scene, track);
    });
    updateGate(frame);
    updateTransport(frame);
    drawTrace();
  }

  function updateSceneUI(entry, track) {
    entry.peak.textContent = track.peak.toFixed(2);
    entry.energy.textContent = track.energy.toFixed(2);
    entry.center.textContent = `${track.center.x.toFixed(0)}/${track.center.y.toFixed(0)}`;
    entry.stretch.textContent = `${track.stretch.toFixed(0)} px`;
    entry.energyFoot.textContent = track.energy.toFixed(2);
  }

  function updateLedger(scene, track) {
    scene.rowX.textContent = track.center.x.toFixed(0);
    scene.rowY.textContent = track.center.y.toFixed(0);
    scene.rowStretch.textContent = track.stretch.toFixed(1);
    scene.rowPeak.textContent = track.peak.toFixed(3);
    scene.rowEnergy.textContent = track.energy.toFixed(3);
    scene.rowBar.style.width = `${(track.energy * 100).toFixed(1)}%`;
  }

  function updateGate(frame) {
    const track = trackFor(frame, state.selected);
    const history = track.history;
    dom.gatePeak.textContent = track.peak.toFixed(3);
    dom.gateEnergy.textContent = track.energy.toFixed(3);
    dom.gatePeakBar.style.width = `${Math.min(100, track.peak * 100).toFixed(1)}%`;
    dom.gateEnergyBar.style.width = `${Math.min(100, track.energy * 100).toFixed(1)}%`;
    dom.gateHistoryEye.textContent = history.sameEye ? "SAME EYE" : "REJECTED";
    dom.gateHistoryEye.className = history.sameEye ? "status-ok" : "status-warn";
    dom.gateCapped.textContent = history.energyCapped ? "CLAMPED" : "NONE";
    dom.gateCapped.className = history.energyCapped ? "status-ok" : "status-ok";
    const left = trackFor(state.timeline.eyes[0].frames[state.frame], state.selected);
    const right = trackFor(state.timeline.eyes[1].frames[state.frame], state.selected);
    const dot = Math.max(-1, Math.min(1,
      left.reflection.x * right.reflection.x + left.reflection.y * right.reflection.y));
    const degrees = Math.acos(dot) * 180 / Math.PI;
    dom.gateDelta.textContent = `LOCAL · Δ ${degrees.toFixed(2)}°`;
  }

  function updateTransport(frame) {
    const seconds = frame.time;
    const minutes = Math.floor(seconds / 60);
    const wholeSeconds = Math.floor(seconds % 60);
    const millis = Math.round((seconds % 1) * 1000);
    dom.timecode.textContent =
      `${String(minutes).padStart(2, "0")}:${String(wholeSeconds).padStart(2, "0")}.${String(millis).padStart(3, "0")}`;
    dom.scrub.value = String(frame.index);
    const progress = (frame.index / Math.max(1, state.timeline.frameCount - 1)) * 100;
    dom.scrub.style.setProperty("--progress", `${progress.toFixed(3)}%`);
    dom.frameLabel.textContent =
      `FRAME ${String(frame.index).padStart(3, "0")} / ${String(state.timeline.frameCount - 1).padStart(3, "0")}`;
  }

  function fitAllCanvases() {
    state.scenes.forEach((entry) => fitCanvas(entry.canvas, SCENE_W, SCENE_H));
    fitCanvas(dom.trace, 960, 184);
    state.dirty = true;
  }

  function fitCanvas(canvas, logicalWidth, logicalHeight) {
    const rect = canvas.getBoundingClientRect();
    if (!rect.width || !rect.height) return;
    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    const width = Math.max(1, Math.round(rect.width * dpr));
    const height = Math.max(1, Math.round(rect.height * dpr));
    if (canvas.width !== width || canvas.height !== height) {
      canvas.width = width;
      canvas.height = height;
    }
    canvas.logicalWidth = logicalWidth;
    canvas.logicalHeight = logicalHeight;
  }

  function prepareContext(canvas) {
    const ctx = canvas.getContext("2d");
    const sx = canvas.width / (canvas.logicalWidth || canvas.width);
    const sy = canvas.height / (canvas.logicalHeight || canvas.height);
    ctx.setTransform(sx, 0, 0, sy, 0, 0);
    ctx.clearRect(0, 0, canvas.logicalWidth || canvas.width, canvas.logicalHeight || canvas.height);
    return ctx;
  }

  function drawScene(entry, object, track) {
    const ctx = prepareContext(entry.canvas);
    const palette = PALETTES[object.kind];
    drawBackdrop(ctx, object, palette);
    drawMotionPath(ctx, object, palette);

    if (object.kind === "rough-metal") {
      drawMetal(ctx, object, track, palette);
    } else if (object.kind === "mirror-sphere") {
      drawSphere(ctx, object, track, palette);
    } else {
      drawGlass(ctx, object, track, palette);
    }

    if (state.showHistory && track.history.valid) {
      drawHistoryGhost(ctx, track, palette);
    }
    drawTrackMarker(ctx, track, palette);
    drawVignette(ctx);
  }

  function drawBackdrop(ctx, object, palette) {
    const background = ctx.createLinearGradient(0, 0, 0, SCENE_H);
    background.addColorStop(0, "#0d1518");
    background.addColorStop(0.62, "#090e11");
    background.addColorStop(1, "#06090b");
    ctx.fillStyle = background;
    ctx.fillRect(0, 0, SCENE_W, SCENE_H);

    const ambient = ctx.createRadialGradient(
      object.center.x, object.center.y - 20, 4,
      object.center.x, object.center.y, 195,
    );
    ambient.addColorStop(0, rgba(palette.mid, 0.115));
    ambient.addColorStop(0.45, rgba(palette.mid, 0.03));
    ambient.addColorStop(1, rgba(palette.mid, 0));
    ctx.fillStyle = ambient;
    ctx.fillRect(0, 0, SCENE_W, SCENE_H);

    ctx.save();
    ctx.strokeStyle = "rgba(236,233,223,0.055)";
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(0, 294.5);
    ctx.lineTo(SCENE_W, 294.5);
    ctx.stroke();

    for (let y = 316; y < SCENE_H; y += 14) {
      const alpha = 0.022 + (y - 316) * 0.00035;
      ctx.strokeStyle = `rgba(236,233,223,${alpha.toFixed(3)})`;
      ctx.beginPath();
      ctx.moveTo(0, y + 0.5);
      ctx.lineTo(SCENE_W, y + 0.5);
      ctx.stroke();
    }
    for (let x = -240; x <= 880; x += 48) {
      ctx.strokeStyle = "rgba(236,233,223,0.025)";
      ctx.beginPath();
      ctx.moveTo(SCENE_W * 0.5, 294);
      ctx.lineTo(x, SCENE_H);
      ctx.stroke();
    }
    ctx.restore();

    ctx.fillStyle = "rgba(236,233,223,0.24)";
    ctx.font = "7px " + monoFont();
    ctx.fillText("LOCAL 640 × 360", 16, 343);
    ctx.textAlign = "right";
    ctx.fillText(`+${object.center.x.toFixed(0)}  ${object.center.y.toFixed(0)}`, 624, 343);
    ctx.textAlign = "left";

    drawCorner(ctx, 14, 14, 1, 1);
    drawCorner(ctx, SCENE_W - 14, 14, -1, 1);
  }

  function drawCorner(ctx, x, y, sx, sy) {
    ctx.save();
    ctx.strokeStyle = "rgba(236,233,223,0.16)";
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(x, y + sy * 10);
    ctx.lineTo(x, y);
    ctx.lineTo(x + sx * 10, y);
    ctx.stroke();
    ctx.restore();
  }

  function drawMotionPath(ctx, object, palette) {
    ctx.save();
    ctx.strokeStyle = rgba(palette.mid, 0.28);
    ctx.fillStyle = rgba(palette.mid, 0.65);
    ctx.lineWidth = 1;
    ctx.setLineDash([3, 5]);

    if (object.kind === "rough-metal") {
      ctx.beginPath();
      ctx.arc(object.center.x, object.center.y, object.radius + 21, -0.88, 1.42);
      ctx.stroke();
      ctx.setLineDash([]);
      const endX = object.center.x + Math.cos(1.42) * (object.radius + 21);
      const endY = object.center.y + Math.sin(1.42) * (object.radius + 21);
      ctx.beginPath();
      ctx.moveTo(endX - 5, endY - 2);
      ctx.lineTo(endX, endY);
      ctx.lineTo(endX + 1, endY - 6);
      ctx.stroke();
    } else if (object.kind === "mirror-sphere") {
      const span = 128;
      ctx.beginPath();
      ctx.moveTo(object.center.x - span, object.center.y + object.radius + 21);
      ctx.lineTo(object.center.x + span, object.center.y + object.radius + 21);
      ctx.stroke();
      ctx.setLineDash([]);
      ctx.beginPath();
      ctx.arc(object.center.x, object.center.y + object.radius + 21, 2, 0, TAU);
      ctx.fill();
    } else {
      ctx.beginPath();
      ctx.moveTo(205, 242);
      ctx.bezierCurveTo(286, 246, 363, 112, 467, 148);
      ctx.stroke();
      ctx.setLineDash([]);
      ctx.beginPath();
      ctx.arc(467, 148, 2, 0, TAU);
      ctx.fill();
    }

    ctx.restore();
  }

  function drawHistoryGhost(ctx, track, palette) {
    const history = track.history;
    ctx.save();
    ctx.lineWidth = 1;
    ctx.strokeStyle = rgba(palette.hot, 0.48);
    ctx.setLineDash([3, 4]);
    ctx.beginPath();
    ctx.moveTo(history.center.x, history.center.y);
    ctx.lineTo(track.center.x, track.center.y);
    ctx.stroke();
    ctx.beginPath();
    ctx.ellipse(
      history.center.x, history.center.y,
      history.stretch * 0.5, Math.max(1, history.thickness * 0.5),
      history.direction, 0, TAU,
    );
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.fillStyle = rgba(palette.hot, 0.72);
    ctx.beginPath();
    ctx.arc(history.center.x, history.center.y, 1.6, 0, TAU);
    ctx.fill();

    ctx.font = "7px " + monoFont();
    ctx.fillStyle = rgba(palette.hot, 0.7);
    ctx.textAlign = "center";
    const labelY = history.center.y - Math.max(7, history.thickness * 0.5 + 6);
    ctx.fillText("t−1", history.center.x, labelY);
    ctx.restore();
  }

  function drawTrackMarker(ctx, track, palette) {
    const x = track.center.x;
    const y = track.center.y;
    const dx = Math.cos(track.direction) * track.stretch * 0.61;
    const dy = Math.sin(track.direction) * track.stretch * 0.61;
    ctx.save();
    ctx.strokeStyle = rgba(palette.hot, 0.2);
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(x - dx, y - dy);
    ctx.lineTo(x + dx, y + dy);
    ctx.stroke();
    ctx.strokeStyle = rgba(palette.hot, 0.84);
    ctx.beginPath();
    ctx.moveTo(x - 4, y);
    ctx.lineTo(x - 1.5, y);
    ctx.moveTo(x + 1.5, y);
    ctx.lineTo(x + 4, y);
    ctx.moveTo(x, y - 4);
    ctx.lineTo(x, y - 1.5);
    ctx.moveTo(x, y + 1.5);
    ctx.lineTo(x, y + 4);
    ctx.stroke();
    ctx.fillStyle = rgba(palette.hot, 0.95);
    ctx.beginPath();
    ctx.arc(x, y, 1.15, 0, TAU);
    ctx.fill();
    ctx.restore();
  }

  function drawHighlight(ctx, track, palette) {
    const rx = Math.max(2, track.stretch * 0.5);
    const ry = Math.max(1.2, track.thickness * 0.5);
    const limit = track.history.valid ? track.history.compositePeak : track.peak;
    const level = Math.min(1, Math.max(0.05, limit));
    const coreLevel = Math.sqrt(level);
    ctx.save();
    ctx.globalCompositeOperation = "lighter";
    ctx.translate(track.center.x, track.center.y);
    ctx.rotate(track.direction);
    ctx.scale(rx, ry);
    const glow = ctx.createRadialGradient(0, 0, 0, 0, 0, 1);
    glow.addColorStop(0, rgba(palette.hot, 0.95 * level));
    glow.addColorStop(0.14, rgba(palette.hot, 0.72 * level));
    glow.addColorStop(0.42, rgba(palette.mid, 0.27 * level));
    glow.addColorStop(0.72, rgba(palette.mid, 0.07 * level));
    glow.addColorStop(1, rgba(palette.mid, 0));
    ctx.fillStyle = glow;
    ctx.beginPath();
    ctx.arc(0, 0, 1, 0, TAU);
    ctx.fill();

    const core = ctx.createRadialGradient(0, 0, 0, 0, 0, 0.55);
    core.addColorStop(0, `rgba(255,255,255,${(0.98 * coreLevel).toFixed(3)})`);
    core.addColorStop(0.18, rgba(palette.hot, 0.72 * coreLevel));
    core.addColorStop(1, rgba(palette.hot, 0));
    ctx.fillStyle = core;
    ctx.beginPath();
    ctx.arc(0, 0, 0.55, 0, TAU);
    ctx.fill();
    ctx.restore();
  }

  function drawMetal(ctx, object, track, palette) {
    const { x, y } = object.center;
    const r = object.radius;

    ctx.save();
    ctx.fillStyle = "rgba(0,0,0,0.48)";
    ctx.beginPath();
    ctx.ellipse(x + 8, y + r * 0.9, r * 0.92, 17, 0, 0, TAU);
    ctx.fill();
    ctx.restore();

    ctx.save();
    ctx.beginPath();
    ctx.arc(x, y, r, 0, TAU);
    ctx.clip();
    const body = ctx.createRadialGradient(x - r * 0.29, y - r * 0.35, 3, x, y, r * 1.08);
    body.addColorStop(0, "#c3c8bd");
    body.addColorStop(0.12, "#8b9794");
    body.addColorStop(0.34, "#53605f");
    body.addColorStop(0.62, "#2d383b");
    body.addColorStop(0.86, "#151c1f");
    body.addColorStop(1, "#080b0d");
    ctx.fillStyle = body;
    ctx.fillRect(x - r, y - r, r * 2, r * 2);

    const axis = { x: Math.cos(object.angle), y: Math.sin(object.angle) };
    const normal = { x: -axis.y, y: axis.x };
    for (let offset = -r + 1; offset < r; offset += 3.4) {
      const half = Math.sqrt(Math.max(0, r * r - offset * offset));
      const alpha = 0.025 + 0.035 * (0.5 + 0.5 * Math.sin(offset * 0.73));
      ctx.strokeStyle = `rgba(232,239,228,${alpha.toFixed(3)})`;
      ctx.lineWidth = offset % 2 === 0 ? 0.8 : 1.4;
      ctx.beginPath();
      ctx.moveTo(
        x + axis.x * half + normal.x * offset,
        y + axis.y * half + normal.y * offset,
      );
      ctx.lineTo(
        x - axis.x * half + normal.x * offset,
        y - axis.y * half + normal.y * offset,
      );
      ctx.stroke();
    }

    const oblique = ctx.createLinearGradient(x - r, y + r, x + r, y - r);
    oblique.addColorStop(0, "rgba(255,202,104,0.1)");
    oblique.addColorStop(0.35, "rgba(255,255,255,0.02)");
    oblique.addColorStop(0.6, "rgba(0,0,0,0.18)");
    oblique.addColorStop(1, "rgba(255,255,255,0.08)");
    ctx.fillStyle = oblique;
    ctx.fillRect(x - r, y - r, r * 2, r * 2);

    const edge = ctx.createRadialGradient(x, y, r * 0.62, x, y, r);
    edge.addColorStop(0, "rgba(0,0,0,0)");
    edge.addColorStop(0.82, "rgba(0,0,0,0.1)");
    edge.addColorStop(0.96, "rgba(220,232,222,0.18)");
    edge.addColorStop(1, "rgba(0,0,0,0.5)");
    ctx.fillStyle = edge;
    ctx.fillRect(x - r, y - r, r * 2, r * 2);
    drawHighlight(ctx, track, palette);
    ctx.restore();

    ctx.save();
    ctx.strokeStyle = "rgba(225,232,222,0.27)";
    ctx.lineWidth = 1.2;
    ctx.beginPath();
    ctx.arc(x, y, r - 0.5, 0, TAU);
    ctx.stroke();
    ctx.strokeStyle = "rgba(255,202,104,0.14)";
    ctx.beginPath();
    ctx.arc(x, y, r - 7, 0, TAU);
    ctx.stroke();
    ctx.strokeStyle = "rgba(0,0,0,0.45)";
    ctx.beginPath();
    ctx.arc(x, y, r - 12, 0, TAU);
    ctx.stroke();
    ctx.restore();
  }

  function drawSphere(ctx, object, track, palette) {
    const { x, y } = object.center;
    const r = object.radius;

    ctx.save();
    ctx.fillStyle = "rgba(0,0,0,0.52)";
    ctx.beginPath();
    ctx.ellipse(x + 9, y + r * 0.93, r * 0.9, 15, 0, 0, TAU);
    ctx.fill();
    ctx.restore();

    ctx.save();
    ctx.beginPath();
    ctx.arc(x, y, r, 0, TAU);
    ctx.clip();

    const body = ctx.createRadialGradient(
      x - r * 0.34, y - r * 0.4, r * 0.04,
      x, y, r * 1.03,
    );
    body.addColorStop(0, "#dbe7e7");
    body.addColorStop(0.12, "#aebfc1");
    body.addColorStop(0.31, "#61757a");
    body.addColorStop(0.55, "#29383d");
    body.addColorStop(0.78, "#10181c");
    body.addColorStop(1, "#050809");
    ctx.fillStyle = body;
    ctx.fillRect(x - r, y - r, r * 2, r * 2);

    const room = ctx.createLinearGradient(0, y - r, 0, y + r);
    room.addColorStop(0, "rgba(216,242,250,0.32)");
    room.addColorStop(0.3, "rgba(86,119,128,0.12)");
    room.addColorStop(0.47, "rgba(0,0,0,0.54)");
    room.addColorStop(0.66, "rgba(42,55,55,0.1)");
    room.addColorStop(0.88, "rgba(151,119,79,0.22)");
    room.addColorStop(1, "rgba(226,188,126,0.15)");
    ctx.fillStyle = room;
    ctx.fillRect(x - r, y - r, r * 2, r * 2);

    ctx.save();
    ctx.translate(x - r * 0.44, y - r * 0.16);
    ctx.rotate(-0.08);
    const windowLight = ctx.createLinearGradient(-10, 0, 28, 0);
    windowLight.addColorStop(0, "rgba(230,250,255,0)");
    windowLight.addColorStop(0.5, "rgba(230,250,255,0.48)");
    windowLight.addColorStop(1, "rgba(230,250,255,0)");
    ctx.fillStyle = windowLight;
    ctx.fillRect(-10, -r * 0.5, 38, r * 1.05);
    ctx.restore();

    const rim = ctx.createRadialGradient(x, y, r * 0.7, x, y, r);
    rim.addColorStop(0, "rgba(0,0,0,0)");
    rim.addColorStop(0.8, "rgba(0,0,0,0.03)");
    rim.addColorStop(0.96, "rgba(205,236,245,0.28)");
    rim.addColorStop(1, "rgba(0,0,0,0.38)");
    ctx.fillStyle = rim;
    ctx.fillRect(x - r, y - r, r * 2, r * 2);
    drawHighlight(ctx, track, palette);
    ctx.restore();

    ctx.save();
    ctx.strokeStyle = "rgba(196,231,242,0.3)";
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.arc(x, y, r - 0.5, 0, TAU);
    ctx.stroke();
    ctx.restore();
  }

  function rgba(rgb, alpha) {
    return `rgba(${rgb[0]},${rgb[1]},${rgb[2]},${alpha})`;
  }

  function monoFont() {
    return '"SFMono-Regular","Cascadia Code","Roboto Mono",Consolas,monospace';
  }

  function drawGlass(ctx, object, track, palette) {
    const { x, y } = object.center;
    const halfHeight = object.height * 0.5;

    ctx.save();
    ctx.fillStyle = "rgba(0,0,0,0.44)";
    ctx.beginPath();
    ctx.ellipse(x + 8, y + halfHeight * 0.98, object.width * 0.53, 16, 0, 0, TAU);
    ctx.fill();
    ctx.restore();

    traceGlassPath(ctx, object);
    const glassFill = ctx.createLinearGradient(0, y - halfHeight, 0, y + halfHeight);
    glassFill.addColorStop(0, "rgba(207,255,235,0.13)");
    glassFill.addColorStop(0.18, "rgba(102,197,165,0.055)");
    glassFill.addColorStop(0.62, "rgba(62,133,119,0.08)");
    glassFill.addColorStop(0.9, "rgba(190,255,231,0.15)");
    glassFill.addColorStop(1, "rgba(226,255,243,0.22)");
    ctx.fillStyle = glassFill;
    ctx.fill();

    ctx.save();
    traceGlassPath(ctx, object);
    ctx.clip();
    const tallAxis = { x: Math.sin(object.angle), y: Math.cos(object.angle) };
    const sideAxis = { x: Math.cos(object.angle), y: -Math.sin(object.angle) };
    for (let offset = -0.38; offset <= 0.4; offset += 0.13) {
      const shiftX = sideAxis.x * object.width * offset;
      const shiftY = sideAxis.y * object.width * offset;
      const streak = ctx.createLinearGradient(
        x + shiftX - tallAxis.x * 40, y + shiftY - tallAxis.y * 40,
        x + shiftX + tallAxis.x * 95, y + shiftY + tallAxis.y * 95,
      );
      streak.addColorStop(0, "rgba(225,255,242,0)");
      streak.addColorStop(0.48, "rgba(225,255,242,0.08)");
      streak.addColorStop(0.7, "rgba(255,255,255,0.16)");
      streak.addColorStop(1, "rgba(225,255,242,0)");
      ctx.strokeStyle = streak;
      ctx.lineWidth = 2 + Math.abs(offset) * 4;
      ctx.beginPath();
      ctx.moveTo(
        x + shiftX - tallAxis.x * halfHeight,
        y + shiftY - tallAxis.y * halfHeight,
      );
      ctx.lineTo(
        x + shiftX + tallAxis.x * halfHeight,
        y + shiftY + tallAxis.y * halfHeight,
      );
      ctx.stroke();
    }

    const caustic = ctx.createRadialGradient(
      x + object.width * 0.14, y + halfHeight * 0.63, 2,
      x + object.width * 0.14, y + halfHeight * 0.63, object.width * 0.5,
    );
    caustic.addColorStop(0, rgba(palette.hot, 0.24));
    caustic.addColorStop(0.35, rgba(palette.mid, 0.1));
    caustic.addColorStop(1, rgba(palette.mid, 0));
    ctx.fillStyle = caustic;
    ctx.fillRect(x - object.width, y - object.height, object.width * 2, object.height * 2);
    drawHighlight(ctx, track, palette);
    ctx.restore();

    traceGlassPath(ctx, object);
    ctx.strokeStyle = "rgba(213,255,239,0.47)";
    ctx.lineWidth = 1.15;
    ctx.stroke();

    ctx.save();
    ctx.translate(x - Math.sin(object.angle) * halfHeight * 0.92,
      y - Math.cos(object.angle) * halfHeight * 0.92);
    ctx.rotate(object.angle);
    const rim = ctx.createLinearGradient(-object.width * 0.4, 0, object.width * 0.4, 0);
    rim.addColorStop(0, "rgba(231,255,246,0.18)");
    rim.addColorStop(0.32, "rgba(255,255,255,0.82)");
    rim.addColorStop(0.68, "rgba(225,255,242,0.3)");
    rim.addColorStop(1, "rgba(225,255,242,0.08)");
    ctx.strokeStyle = rim;
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.ellipse(0, 0, object.width * 0.43, 9, 0, 0, TAU);
    ctx.stroke();
    ctx.strokeStyle = "rgba(216,255,240,0.13)";
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.ellipse(0, 3, object.width * 0.38, 6, 0, 0, TAU);
    ctx.stroke();
    ctx.restore();

    ctx.save();
    ctx.strokeStyle = "rgba(255,255,255,0.24)";
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(x - object.width * 0.39, y - halfHeight * 0.69);
    ctx.lineTo(x - object.width * 0.44, y + halfHeight * 0.36);
    ctx.stroke();
    ctx.restore();
  }

  function traceGlassPath(ctx, object) {
    const w = object.width;
    const h = object.height;
    const local = [
      [-0.43 * w, -0.46 * h],
      [0.43 * w, -0.46 * h],
      [0.50 * w, 0.37 * h],
      [0.40 * w, 0.49 * h],
      [0.00, 0.52 * h],
      [-0.40 * w, 0.49 * h],
      [-0.50 * w, 0.37 * h],
    ];
    const points = local.map(([px, py]) => ({
      x: object.center.x + px * Math.cos(object.angle) - py * Math.sin(object.angle),
      y: object.center.y + px * Math.sin(object.angle) + py * Math.cos(object.angle),
    }));
    ctx.beginPath();
    ctx.moveTo(points[0].x, points[0].y);
    ctx.lineTo(points[1].x, points[1].y);
    ctx.lineTo(points[2].x, points[2].y);
    ctx.lineTo(points[3].x, points[3].y);
    const bottom = points[4];
    ctx.quadraticCurveTo(bottom.x, bottom.y + 8, points[5].x, points[5].y);
    ctx.lineTo(points[6].x, points[6].y);
    ctx.closePath();
  }

  function drawVignette(ctx) {
    const vignette = ctx.createRadialGradient(
      SCENE_W * 0.5, SCENE_H * 0.46, 80,
      SCENE_W * 0.5, SCENE_H * 0.5, 390,
    );
    vignette.addColorStop(0, "rgba(0,0,0,0)");
    vignette.addColorStop(0.68, "rgba(0,0,0,0.04)");
    vignette.addColorStop(1, "rgba(0,0,0,0.42)");
    ctx.fillStyle = vignette;
    ctx.fillRect(0, 0, SCENE_W, SCENE_H);

    const lower = ctx.createLinearGradient(0, SCENE_H - 58, 0, SCENE_H);
    lower.addColorStop(0, "rgba(4,6,7,0)");
    lower.addColorStop(1, "rgba(4,6,7,0.4)");
    ctx.fillStyle = lower;
    ctx.fillRect(0, SCENE_H - 58, SCENE_W, 58);
  }

  function drawTrace() {
    const canvas = dom.trace;
    if (!canvas.width || !canvas.height) return;
    const ctx = prepareContext(canvas);
    const width = 960;
    const height = 184;
    const left = 38;
    const right = 12;
    const top = 10;
    const bottom = 24;
    const innerW = width - left - right;
    const innerH = height - top - bottom;
    const frames = state.timeline.eyes[state.eye].frames;

    ctx.strokeStyle = "rgba(236,233,223,0.08)";
    ctx.fillStyle = "rgba(236,233,223,0.32)";
    ctx.font = `7px ${monoFont()}`;
    ctx.lineWidth = 1;
    for (let i = 0; i <= 4; i++) {
      const y = top + innerH * (i / 4);
      ctx.beginPath();
      ctx.moveTo(left, y + 0.5);
      ctx.lineTo(width - right, y + 0.5);
      ctx.stroke();
      if (i === 0 || i === 2 || i === 4) {
        ctx.textAlign = "right";
        ctx.fillText((1 - i / 4).toFixed(1), left - 8, y + 2.5);
      }
    }

    state.timeline.scenes.forEach((scene) => {
      const palette = PALETTES[scene.id];
      const selected = scene.id === state.selected;
      ctx.strokeStyle = rgba(palette.mid, selected ? 0.95 : 0.38);
      ctx.lineWidth = selected ? 1.8 : 1;
      ctx.beginPath();
      frames.forEach((frame, index) => {
        const track = trackFor(frame, scene.id);
        const x = left + innerW * (index / Math.max(1, frames.length - 1));
        const y = top + innerH * (1 - track.energy);
        if (index === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      });
      ctx.stroke();

      if (selected) {
        ctx.save();
        ctx.strokeStyle = rgba(palette.hot, 0.34);
        ctx.setLineDash([2, 5]);
        ctx.lineWidth = 1;
        ctx.beginPath();
        frames.forEach((frame, index) => {
          const history = trackFor(frame, scene.id).history;
          const value = history.valid ? history.sampledEnergy : 0;
          const x = left + innerW * (index / Math.max(1, frames.length - 1));
          const y = top + innerH * (1 - value);
          if (index === 0) ctx.moveTo(x, y);
          else ctx.lineTo(x, y);
        });
        ctx.stroke();
        ctx.restore();
      }
    });

    const playheadX = left + innerW * (state.frame / Math.max(1, frames.length - 1));
    ctx.strokeStyle = "rgba(216,255,99,0.6)";
    ctx.beginPath();
    ctx.moveTo(playheadX + 0.5, top - 2);
    ctx.lineTo(playheadX + 0.5, top + innerH + 3);
    ctx.stroke();
    ctx.fillStyle = "#d8ff63";
    ctx.beginPath();
    ctx.arc(playheadX, top + innerH + 5, 2, 0, TAU);
    ctx.fill();
    ctx.textAlign = "left";
    ctx.fillStyle = "rgba(236,233,223,0.28)";
    ctx.fillText("ENERGY", 0, top + 4);
    ctx.textAlign = "right";
    ctx.fillText("SAMPLE WINDOW", width - right, height - 4);
    ctx.textAlign = "left";
  }

  boot();
})();
