import { Renderer } from "./renderer.js";

const FEATURE_META = {
  plane: { short: "平面", en: "PLANE", color: "#58ccf2" },
  cylinder: { short: "圆柱", en: "CYLINDER", color: "#f4b942" },
  hole: { short: "孔", en: "HOLE", color: "#ff5d8f" },
  fillet: { short: "圆角", en: "FILLET", color: "#58e0b0" },
  unknown: { short: "未定", en: "UNKNOWN", color: "#a18cd1" },
};

const CASE_GLYPHS = {
  "adjacent-fillets": `<svg viewBox="0 0 52 32"><path d="M2 7h14c10 0 2 18 12 18s2-18 12-18h10"/><path d="M4 26h44"/><path d="M17 7v-3M29 25v3"/></svg>`,
  "oblique-hole": `<svg viewBox="0 0 52 32"><path d="M3 24 43 8l6 7-38 15z"/><ellipse cx="22" cy="20" rx="9" ry="3" transform="rotate(-20 22 20)"/><path d="M15 18 23 8l4 16"/></svg>`,
  "near-planar": `<svg viewBox="0 0 52 32"><path d="M3 19 14 13l13 5 12-9 10 4"/><path d="M3 19v8h43v-8"/><path d="M14 13v8M27 18v9M39 9v8"/></svg>`,
};

const dom = {
  caseList: document.getElementById("caseList"),
  stageCode: document.getElementById("stageCode"),
  stageSeverity: document.getElementById("stageSeverity"),
  stageIndex: document.getElementById("stageIndex"),
  stageTitle: document.getElementById("stageTitle"),
  stageSubtitle: document.getElementById("stageSubtitle"),
  viewTriangleCount: document.getElementById("viewTriangleCount"),
  renderModeChip: document.getElementById("renderModeChip"),
  viewportLegend: document.getElementById("viewportLegend"),
  footerBoundary: document.getElementById("footerBoundary"),
  footerError: document.getElementById("footerError"),
  footerRegions: document.getElementById("footerRegions"),
  inspectorCode: document.getElementById("inspectorCode"),
  severityChip: document.getElementById("severityChip"),
  inspectorTitle: document.getElementById("inspectorTitle"),
  inspectorDescription: document.getElementById("inspectorDescription"),
  confidenceRing: document.getElementById("confidenceRing"),
  meanConfidence: document.getElementById("meanConfidence"),
  boundaryConfidence: document.getElementById("boundaryConfidence"),
  lowBoundaryConfidence: document.getElementById("lowBoundaryConfidence"),
  regionCount: document.getElementById("regionCount"),
  triangleSummary: document.getElementById("triangleSummary"),
  featureRows: document.getElementById("featureRows"),
  boundaryCount: document.getElementById("boundaryCount"),
  boundaryRows: document.getElementById("boundaryRows"),
  rmsError: document.getElementById("rmsError"),
  maxError: document.getElementById("maxError"),
  errorMarker: document.getElementById("errorMarker"),
  auditNote: document.getElementById("auditNote"),
  findingRows: document.getElementById("findingRows"),
  loadingCard: document.getElementById("loadingCard"),
  errorCard: document.getElementById("errorCard"),
  overlaySlider: document.getElementById("overlaySlider"),
  overlayValue: document.getElementById("overlayValue"),
  boundarySwitch: document.getElementById("boundarySwitch"),
  contourSwitch: document.getElementById("contourSwitch"),
  toggleBounds: document.getElementById("toggleBounds"),
  toggleRotate: document.getElementById("toggleRotate"),
  resetView: document.getElementById("resetView"),
  auditShortcut: document.getElementById("auditShortcut"),
};

let renderer = null;
let sceneMap = new Map();
let metadata = [];
let currentScene = null;
let currentMode = "class";
let boundariesEnabled = true;

async function boot() {
  try {
    const listResponse = await fetch("/api/scenes");
    if (!listResponse.ok) throw new Error(`Scene list failed (${listResponse.status})`);
    metadata = await listResponse.json();
    const payloads = await Promise.all(metadata.map(async (meta) => {
      const response = await fetch(`/api/scene/${encodeURIComponent(meta.id)}`);
      if (!response.ok) throw new Error(`Scene ${meta.id} failed (${response.status})`);
      return response.json();
    }));
    for (const scene of payloads) sceneMap.set(scene.id, scene);
    buildCaseList();
    renderer = new Renderer(document.getElementById("sceneCanvas"), document.getElementById("calloutLayer"));
    const initial = hashID() || metadata[0]?.id;
    const initialMode = new URLSearchParams(location.search).get("mode");
    if (["class", "audit", "split"].includes(initialMode)) currentMode = initialMode;
    if (initial && sceneMap.has(initial)) selectScene(initial, false);
    else if (metadata[0]) selectScene(metadata[0].id, false);
    bindControls();
    setMode(currentMode);
  } catch (error) {
    showError(error);
  }
}

function hashID() {
  return location.hash ? decodeURIComponent(location.hash.slice(1)) : "";
}

function showError(error) {
  dom.loadingCard.classList.add("hidden");
  dom.errorCard.hidden = false;
  dom.errorCard.textContent = `无法载入网格：${error.message}`;
  console.error(error);
}

function buildCaseList() {
  dom.caseList.replaceChildren();
  for (const meta of metadata) {
    const button = document.createElement("button");
    button.className = "case-card";
    button.dataset.id = meta.id;
    button.type = "button";

    const top = document.createElement("span");
    top.className = "case-top";
    const number = document.createElement("span");
    number.className = "case-number";
    number.textContent = `CASE ${meta.index}`;
    const state = document.createElement("span");
    state.className = "case-state";
    state.textContent = meta.severity;
    top.append(number, state);

    const name = document.createElement("span");
    name.className = "case-name";
    name.textContent = meta.titleCN;
    const english = document.createElement("small");
    english.textContent = meta.title;
    name.append(english);

    const glyph = document.createElement("span");
    glyph.className = "case-glyph";
    glyph.innerHTML = CASE_GLYPHS[meta.id] || "";

    button.append(top, name, glyph);
    button.addEventListener("click", () => selectScene(meta.id, true));
    dom.caseList.append(button);
  }
}

function selectScene(id, updateHash = true) {
  const scene = sceneMap.get(id);
  if (!scene || !renderer) return;
  currentScene = scene;
  renderer.setScene(scene);
  renderer.setMode(currentMode);
  renderer.setOverlay(Number(dom.overlaySlider.value) / 100);
  renderer.setBoundaries(boundariesEnabled);
  renderer.setContours(dom.contourSwitch.classList.contains("active"));
  updateSceneUI(scene);
  for (const card of dom.caseList.querySelectorAll(".case-card")) {
    card.classList.toggle("active", card.dataset.id === id);
  }
  dom.loadingCard.classList.add("hidden");
  dom.errorCard.hidden = true;
  if (updateHash) history.replaceState(null, "", `#${encodeURIComponent(id)}`);
}

function updateSceneUI(scene) {
  dom.stageCode.textContent = scene.code;
  dom.stageSeverity.textContent = scene.severity;
  dom.stageSeverity.style.color = severityColor(scene.severity);
  dom.stageIndex.textContent = scene.index;
  dom.stageTitle.textContent = scene.title;
  dom.stageSubtitle.textContent = scene.subtitle;
  dom.viewTriangleCount.textContent = `${formatInt(scene.summary.triangles)} TRI`;
  dom.inspectorCode.textContent = scene.code;
  dom.severityChip.textContent = scene.severity;
  dom.severityChip.className = `severity-chip ${severityClass(scene.severity)}`;
  dom.inspectorTitle.textContent = scene.titleCN;
  dom.inspectorDescription.textContent = scene.description;

  const mean = scene.summary.meanConfidence;
  dom.meanConfidence.textContent = mean.toFixed(2);
  dom.confidenceRing.style.setProperty("--value", Math.max(0, Math.min(1, mean)));
  dom.confidenceRing.style.setProperty("--ring-color", confidenceColor(mean));
  dom.boundaryConfidence.textContent = scene.summary.boundaryConfidence.toFixed(2);
  dom.lowBoundaryConfidence.textContent = lowestBoundary(scene).toFixed(2);
  dom.regionCount.textContent = String(scene.summary.regions);
  dom.triangleSummary.textContent = `${formatInt(scene.summary.triangles)} FACETS`;
  dom.boundaryCount.textContent = `${formatInt(scene.boundaries.length)} SEGMENTS`;

  dom.footerBoundary.textContent = scene.summary.boundaryConfidence.toFixed(2);
  dom.footerError.textContent = `${scene.summary.rmsError.toFixed(3)} mm`;
  dom.footerRegions.textContent = String(scene.summary.regions).padStart(2, "0");
  dom.rmsError.innerHTML = `${scene.summary.rmsError.toFixed(3)}<em> mm</em>`;
  dom.maxError.innerHTML = `${scene.summary.maxError.toFixed(3)}<em> mm</em>`;
  dom.auditNote.textContent = scene.auditNote;

  const marker = 50 + 38 * Math.min(1, scene.summary.maxError / (scene.summary.maxError + 0.055));
  dom.errorMarker.style.left = `${marker}%`;

  buildFeatureRows(scene);
  buildLegend(scene);
  buildBoundaryRows(scene);
  buildFindings(scene);
  document.title = `${scene.titleCN} · MESHSCOPE`;
}

function buildLegend(scene) {
  dom.viewportLegend.replaceChildren();
  for (const feature of scene.features) {
    const meta = FEATURE_META[feature.key] || FEATURE_META.unknown;
    const chip = document.createElement("span");
    chip.className = `legend-chip${feature.triangles === 0 ? " zero" : ""}`;
    const dot = document.createElement("i");
    dot.style.color = meta.color;
    dot.style.background = meta.color;
    const label = document.createElement("span");
    label.textContent = `${meta.short} ${feature.triangles ? formatCompact(feature.triangles) : "—"}`;
    chip.append(dot, label);
    dom.viewportLegend.append(chip);
  }
}

function buildFeatureRows(scene) {
  dom.featureRows.replaceChildren();
  for (const feature of scene.features) {
    const meta = FEATURE_META[feature.key] || FEATURE_META.unknown;
    const row = document.createElement("div");
    row.className = `feature-row${feature.triangles === 0 ? " zero" : ""}`;

    const name = document.createElement("div");
    name.className = "feature-name";
    const dot = document.createElement("i");
    dot.style.color = meta.color;
    dot.style.background = meta.color;
    const zh = document.createElement("span");
    zh.textContent = meta.short;
    const en = document.createElement("em");
    en.textContent = meta.en;
    name.append(dot, zh, en);

    const track = document.createElement("div");
    track.className = "feature-track";
    track.style.color = meta.color;
    const fill = document.createElement("i");
    fill.style.width = `${Math.max(2, feature.confidence * 100)}%`;
    track.append(fill);

    const stat = document.createElement("div");
    stat.className = "feature-stat";
    const conf = document.createElement("b");
    conf.textContent = feature.triangles ? feature.confidence.toFixed(2) : "—";
    const count = document.createElement("small");
    count.textContent = `${formatCompact(feature.triangles)} tri`;
    stat.append(conf, count);

    row.append(name, track, stat);
    dom.featureRows.append(row);
  }
}

function buildBoundaryRows(scene) {
  dom.boundaryRows.replaceChildren();
  if (!scene.callouts.length) {
    const empty = document.createElement("div");
    empty.className = "no-boundary";
    empty.textContent = "当前视角没有可标记的边界。";
    dom.boundaryRows.append(empty);
    return;
  }
  for (const callout of scene.callouts) {
    const row = document.createElement("div");
    row.className = "boundary-row";
    const pair = document.createElement("span");
    pair.className = "pair";
    pair.append(kindDot(callout.kindA), kindText(callout.kindA), kindArrow(), kindDot(callout.kindB), kindText(callout.kindB));
    const value = document.createElement("span");
    value.className = "confidence";
    value.textContent = Number(callout.confidence).toFixed(2);
    value.style.color = confidenceColor(callout.confidence);
    const risk = document.createElement("span");
    risk.className = "risk";
    risk.style.background = confidenceColor(callout.confidence);
    row.append(pair, value, risk);
    dom.boundaryRows.append(row);
  }
}

function buildFindings(scene) {
  dom.findingRows.replaceChildren();
  for (const finding of scene.findings) {
    const row = document.createElement("div");
    row.className = "finding-row";
    const tone = document.createElement("span");
    tone.className = `tone ${finding.tone || "good"}`;
    const title = document.createElement("span");
    title.className = "title";
    title.textContent = finding.title;
    const value = document.createElement("span");
    value.className = "value";
    value.textContent = finding.value;
    row.append(tone, title, value);
    dom.findingRows.append(row);
  }
}

function kindDot(key) {
  const dot = document.createElement("i");
  const meta = FEATURE_META[key] || FEATURE_META.unknown;
  dot.style.background = meta.color;
  return dot;
}

function kindText(key) {
  const text = document.createElement("span");
  const meta = FEATURE_META[key] || FEATURE_META.unknown;
  text.textContent = meta.en;
  return text;
}

function kindArrow() {
  const arrow = document.createElement("u");
  arrow.textContent = "↔";
  return arrow;
}

function lowestBoundary(scene) {
  if (!scene.boundaries.length) return 0;
  return scene.boundaries.reduce((min, line) => Math.min(min, line.confidence), 1);
}

function confidenceColor(value) {
  if (value >= 0.78) return "#58e0b0";
  if (value >= 0.58) return "#f4b942";
  return "#ff7657";
}

function severityColor(value) {
  if (value === "NOMINAL") return "#58e0b0";
  if (value === "WATCH") return "#ff7657";
  return "#f4b942";
}

function severityClass(value) {
  if (value === "NOMINAL") return "nominal";
  if (value === "WATCH") return "watch";
  return "";
}

function formatInt(value) {
  return Number(value).toLocaleString("en-US");
}

function formatCompact(value) {
  if (value >= 10000) return `${(value / 1000).toFixed(1)}k`;
  if (value >= 1000) return `${(value / 1000).toFixed(2)}k`;
  return String(value);
}

function setMode(mode) {
  currentMode = mode;
  renderer?.setMode(mode);
  for (const button of document.querySelectorAll(".mode-btn")) {
    const active = button.dataset.mode === mode;
    button.classList.toggle("active", active);
    button.setAttribute("aria-selected", active ? "true" : "false");
  }
  dom.renderModeChip.textContent = {
    class: "CLASSIFICATION",
    audit: "FORM DEVIATION",
    split: "SPLIT LENS",
  }[mode];
  dom.renderModeChip.style.color = mode === "audit" ? "#ffc29c" : "";
}

function setBoundaries(enabled) {
  boundariesEnabled = enabled;
  renderer?.setBoundaries(enabled);
  dom.boundarySwitch.classList.toggle("active", enabled);
  dom.boundarySwitch.setAttribute("aria-checked", enabled ? "true" : "false");
  dom.toggleBounds.classList.toggle("active", enabled);
}

function bindControls() {
  for (const button of document.querySelectorAll(".mode-btn")) {
    button.addEventListener("click", () => setMode(button.dataset.mode));
  }
  dom.auditShortcut.addEventListener("click", () => setMode("audit"));
  dom.overlaySlider.addEventListener("input", () => {
    const value = Number(dom.overlaySlider.value);
    dom.overlayValue.textContent = `${value}%`;
    dom.overlaySlider.style.setProperty("--fill", `${value}%`);
    renderer?.setOverlay(value / 100);
  });
  dom.boundarySwitch.addEventListener("click", () => setBoundaries(!boundariesEnabled));
  dom.toggleBounds.addEventListener("click", () => setBoundaries(!boundariesEnabled));
  dom.contourSwitch.addEventListener("click", () => {
    const enabled = !dom.contourSwitch.classList.contains("active");
    dom.contourSwitch.classList.toggle("active", enabled);
    dom.contourSwitch.setAttribute("aria-checked", enabled ? "true" : "false");
    renderer?.setContours(enabled);
  });
  dom.resetView.addEventListener("click", () => renderer?.resetView());
  dom.toggleRotate.addEventListener("click", () => {
    const enabled = renderer.toggleAutoRotate();
    dom.toggleRotate.classList.toggle("active", enabled);
    dom.toggleRotate.setAttribute("aria-pressed", enabled ? "true" : "false");
  });
  window.addEventListener("hashchange", () => {
    const id = hashID();
    if (id && sceneMap.has(id) && id !== currentScene?.id) selectScene(id, false);
  });
  window.addEventListener("keydown", (event) => {
    if (event.target instanceof HTMLInputElement) return;
    if (event.key === "1") setMode("class");
    if (event.key === "2") setMode("audit");
    if (event.key === "3") setMode("split");
  });
  dom.overlaySlider.style.setProperty("--fill", `${dom.overlaySlider.value}%`);
  setBoundaries(boundariesEnabled);
}

boot();
