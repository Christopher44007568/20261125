import { clamp, dot, lookAt, multiply, normalize, perspective, sub, transform } from "./glmath.js";
import { lineFragment, lineVertex, meshFragment, meshVertex } from "./shaders.js";

const FEATURE_COLORS = {
  0: [0.62, 0.54, 0.82],
  1: [0.28, 0.77, 0.94],
  2: [0.96, 0.72, 0.25],
  3: [1.0, 0.35, 0.55],
  4: [0.33, 0.88, 0.69],
};

const FEATURE_DOM_COLORS = {
  plane: "#58ccf2",
  cylinder: "#f4b942",
  hole: "#ff5d8f",
  fillet: "#58e0b0",
  unknown: "#a18cd1",
};

function confidenceColor(value) {
  if (value >= 0.78) return "#62e0c0";
  if (value >= 0.58) return "#f4b942";
  return "#ff7657";
}

function compile(gl, type, source) {
  const shader = gl.createShader(type);
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    throw new Error(gl.getShaderInfoLog(shader) || "Shader compilation failed");
  }
  return shader;
}

function program(gl, vertexSource, fragmentSource) {
  const p = gl.createProgram();
  gl.attachShader(p, compile(gl, gl.VERTEX_SHADER, vertexSource));
  gl.attachShader(p, compile(gl, gl.FRAGMENT_SHADER, fragmentSource));
  gl.linkProgram(p);
  if (!gl.getProgramParameter(p, gl.LINK_STATUS)) {
    throw new Error(gl.getProgramInfoLog(p) || "Program link failed");
  }
  return p;
}

function attribute(gl, p, name) {
  return gl.getAttribLocation(p, name);
}

function uniform(gl, p, name) {
  return gl.getUniformLocation(p, name);
}

function emptyColor() {
  return [0.62, 0.54, 0.82];
}

export class Renderer {
  constructor(canvas, calloutLayer) {
    this.canvas = canvas;
    this.calloutLayer = calloutLayer;
    this.gl = canvas.getContext("webgl2", {
      alpha: true,
      antialias: true,
      premultipliedAlpha: false,
      powerPreference: "high-performance",
    });
    if (!this.gl) throw new Error("WebGL2 is not available in this browser.");
    const gl = this.gl;
    this.meshProgram = program(gl, meshVertex, meshFragment);
    this.lineProgram = program(gl, lineVertex, lineFragment);
    this.meshUniforms = {
      projection: uniform(gl, this.meshProgram, "uProjection"),
      view: uniform(gl, this.meshProgram, "uView"),
      camera: uniform(gl, this.meshProgram, "uCameraPosition"),
      resolution: uniform(gl, this.meshProgram, "uResolution"),
      mode: uniform(gl, this.meshProgram, "uMode"),
      overlay: uniform(gl, this.meshProgram, "uOverlay"),
      errorScale: uniform(gl, this.meshProgram, "uErrorScale"),
      contours: uniform(gl, this.meshProgram, "uContours"),
    };
    this.lineUniforms = {
      projection: uniform(gl, this.lineProgram, "uProjection"),
      view: uniform(gl, this.lineProgram, "uView"),
      resolution: uniform(gl, this.lineProgram, "uResolution"),
    };
    this.mode = 0;
    this.overlay = 0.72;
    this.contours = true;
    this.showBoundaries = true;
    this.autoRotate = false;
    this.rotationFrame = 0;
    this.camera = null;
    this.scene = null;
    this.buffers = null;
    this.callouts = [];
    this.dragging = false;
    this.lastPointer = null;
    this.initialCamera = null;
    this.resize();
    this.bindInteractions();
    new ResizeObserver(() => {
      this.resize();
      this.render();
    }).observe(canvas);
  }

  resize() {
    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    const width = Math.max(1, Math.round(this.canvas.clientWidth * dpr));
    const height = Math.max(1, Math.round(this.canvas.clientHeight * dpr));
    if (this.canvas.width !== width || this.canvas.height !== height) {
      this.canvas.width = width;
      this.canvas.height = height;
    }
    this.dpr = dpr;
    this.gl.viewport(0, 0, width, height);
  }

  setScene(scene) {
    this.scene = scene;
    this.camera = {
      target: scene.camera.target.slice(),
      distance: scene.camera.distance,
      azimuth: scene.camera.azimuth,
      elevation: scene.camera.elevation,
      fov: scene.camera.fov || 30,
    };
    this.initialCamera = { ...this.camera, target: scene.camera.target.slice() };
    this.uploadMesh();
    this.uploadBoundaries();
    this.buildCallouts();
    this.resize();
    this.render();
  }

  uploadMesh() {
    const gl = this.gl;
    this.deleteMeshBuffers();
    const mesh = this.scene.mesh;
    const tris = mesh.features.length;
    const colors = new Float32Array(tris * 9);
    const conf = new Float32Array(tris * 3);
    const dev = new Float32Array(tris * 3);
    const feature = new Float32Array(tris * 3);
    const datum = this.scene.datum;
    for (let i = 0; i < tris; i++) {
      const c = FEATURE_COLORS[mesh.features[i]] || emptyColor();
      const cf = mesh.confidence[i] || 0;
      const faceDev = mesh.deviation[i] || 0;
      const isDatumPlane = datum && mesh.features[i] === 1;
      const dn = datum ? datum.normal : [0, 0, 0];
      for (let k = 0; k < 3; k++) {
        const o = i * 9 + k * 3;
        colors[o] = c[0];
        colors[o + 1] = c[1];
        colors[o + 2] = c[2];
        conf[i * 3 + k] = cf;
        let d = faceDev;
        if (isDatumPlane) {
          const px = mesh.positions[o], py = mesh.positions[o + 1], pz = mesh.positions[o + 2];
          const nx = mesh.normals[o], ny = mesh.normals[o + 1], nz = mesh.normals[o + 2];
          const normalDot = nx * dn[0] + ny * dn[1] + nz * dn[2];
          const offset = px * dn[0] + py * dn[1] + pz * dn[2] - datum.offset;
          if (normalDot > 0.906 && Math.abs(offset) < 1.8) d = offset;
        }
        dev[i * 3 + k] = d;
        feature[i * 3 + k] = mesh.features[i];
      }
    }
    const posBuf = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, posBuf);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(mesh.positions), gl.STATIC_DRAW);
    const normBuf = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, normBuf);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(mesh.normals), gl.STATIC_DRAW);
    const colorBuf = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, colorBuf);
    gl.bufferData(gl.ARRAY_BUFFER, colors, gl.STATIC_DRAW);
    const confBuf = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, confBuf);
    gl.bufferData(gl.ARRAY_BUFFER, conf, gl.STATIC_DRAW);
    const devBuf = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, devBuf);
    gl.bufferData(gl.ARRAY_BUFFER, dev, gl.STATIC_DRAW);
    const featureBuf = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, featureBuf);
    gl.bufferData(gl.ARRAY_BUFFER, feature, gl.STATIC_DRAW);
    this.buffers = { posBuf, normBuf, colorBuf, confBuf, devBuf, featureBuf, count: tris * 3 };
  }

  uploadBoundaries() {
    const gl = this.gl;
    if (this.lineBuffer) gl.deleteBuffer(this.lineBuffer);
    const values = [];
    for (const line of this.scene.boundaries) {
      const a = line.a, b = line.b, n = line.n;
      const cf = line.confidence;
      const sameKind = line.kindA === line.kindB;
      const style = !sameKind
        ? 1.0
        : (line.kindA === "fillet" || line.kindA === "cylinder" || line.kindA === "hole")
          ? 0.62
          : 0.10;
      const push = (p, other, side) => {
        values.push(p[0], p[1], p[2], other[0], other[1], other[2], n[0], n[1], n[2], side, cf, style);
      };
      push(a, b, -1);
      push(a, b, 1);
      push(b, a, -1);
      push(a, b, -1);
      push(b, a, -1);
      push(b, a, 1);
    }
    this.lineBuffer = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, this.lineBuffer);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(values), gl.STATIC_DRAW);
    this.lineVertexCount = values.length / 12;
  }

  deleteMeshBuffers() {
    if (!this.buffers) return;
    for (const key of ["posBuf", "normBuf", "colorBuf", "confBuf", "devBuf", "featureBuf"]) {
      this.gl.deleteBuffer(this.buffers[key]);
    }
  }

  cameraMatrices() {
    const cam = this.camera;
    const az = cam.azimuth * Math.PI / 180;
    const el = cam.elevation * Math.PI / 180;
    const eye = [
      cam.target[0] + cam.distance * Math.cos(el) * Math.sin(az),
      cam.target[1] + cam.distance * Math.sin(el),
      cam.target[2] + cam.distance * Math.cos(el) * Math.cos(az),
    ];
    const view = lookAt(eye, cam.target, [0, 1, 0]);
    const projection = perspective(
      cam.fov,
      Math.max(1, this.canvas.width) / Math.max(1, this.canvas.height),
      Math.max(1, cam.distance * 0.003),
      cam.distance * 8,
    );
    this.eye = eye;
    this.view = view;
    this.projection = projection;
    return { eye, view, projection };
  }

  render() {
    if (!this.scene || !this.buffers) return;
    const gl = this.gl;
    this.resize();
    const { eye, view, projection } = this.cameraMatrices();
    gl.viewport(0, 0, this.canvas.width, this.canvas.height);
    gl.clearColor(0, 0, 0, 0);
    gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
    gl.enable(gl.DEPTH_TEST);
    gl.depthFunc(gl.LEQUAL);
    gl.enable(gl.CULL_FACE);
    gl.cullFace(gl.BACK);
    gl.frontFace(gl.CCW);

    gl.useProgram(this.meshProgram);
    gl.uniformMatrix4fv(this.meshUniforms.projection, false, projection);
    gl.uniformMatrix4fv(this.meshUniforms.view, false, view);
    gl.uniform3fv(this.meshUniforms.camera, eye);
    gl.uniform2f(this.meshUniforms.resolution, this.canvas.width, this.canvas.height);
    gl.uniform1f(this.meshUniforms.mode, this.mode);
    gl.uniform1f(this.meshUniforms.overlay, this.overlay);
    gl.uniform1f(this.meshUniforms.errorScale, Math.max(0.024, this.scene.summary.maxError || 0.05));
    gl.uniform1f(this.meshUniforms.contours, this.contours ? 1 : 0);
    this.bindMeshAttributes();
    gl.drawArrays(gl.TRIANGLES, 0, this.buffers.count);

    if (this.showBoundaries && this.lineVertexCount > 0) {
      gl.disable(gl.CULL_FACE);
      gl.enable(gl.BLEND);
      gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
      gl.depthMask(false);
      gl.useProgram(this.lineProgram);
      gl.uniformMatrix4fv(this.lineUniforms.projection, false, projection);
      gl.uniformMatrix4fv(this.lineUniforms.view, false, view);
      gl.uniform2f(this.lineUniforms.resolution, this.canvas.width, this.canvas.height);
      gl.bindBuffer(gl.ARRAY_BUFFER, this.lineBuffer);
      const attrs = [
        ["aPosition", 3, 0],
        ["aOther", 3, 3],
        ["aNormal", 3, 6],
        ["aSide", 1, 9],
        ["aConfidence", 1, 10],
        ["aStyle", 1, 11],
      ];
      for (const [name, size, offset] of attrs) {
        const loc = attribute(gl, this.lineProgram, name);
        if (loc < 0) continue;
        gl.enableVertexAttribArray(loc);
        gl.vertexAttribPointer(loc, size, gl.FLOAT, false, 12 * 4, offset * 4);
      }
      gl.drawArrays(gl.TRIANGLES, 0, this.lineVertexCount);
      gl.depthMask(true);
      gl.disable(gl.BLEND);
    }

    this.updateCalloutPositions();
    this.updateReadout();
  }

  bindMeshAttributes() {
    const gl = this.gl;
    const p = this.meshProgram;
    const specs = [
      ["aPosition", this.buffers.posBuf, 3, 0],
      ["aNormal", this.buffers.normBuf, 3, 0],
      ["aColor", this.buffers.colorBuf, 3, 0],
      ["aConfidence", this.buffers.confBuf, 1, 0],
      ["aDeviation", this.buffers.devBuf, 1, 0],
      ["aFeature", this.buffers.featureBuf, 1, 0],
    ];
    for (const [name, buffer, size, offset] of specs) {
      const loc = attribute(gl, p, name);
      if (loc < 0) continue;
      gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
      gl.enableVertexAttribArray(loc);
      gl.vertexAttribPointer(loc, size, gl.FLOAT, false, 0, offset);
    }
  }

  projectPoint(point) {
    if (!this.projection || !this.view) return null;
    const matrix = multiply(this.projection, this.view);
    const clip = transform(matrix, point, 1);
    if (clip[3] <= 0.001) return null;
    const ndcX = clip[0] / clip[3];
    const ndcY = clip[1] / clip[3];
    if (ndcX < -1.25 || ndcX > 1.25 || ndcY < -1.25 || ndcY > 1.25) return null;
    return {
      x: (ndcX * 0.5 + 0.5) * this.canvas.clientWidth,
      y: (-ndcY * 0.5 + 0.5) * this.canvas.clientHeight,
      ndcX,
      ndcY,
    };
  }

  buildCallouts() {
    this.calloutLayer.replaceChildren();
    this.callouts = [];
    if (!this.scene) return;
    const boundaryItems = this.scene.callouts.slice(0, 4);
    for (const c of boundaryItems) {
      const el = document.createElement("div");
      el.className = "callout boundary";
      el.style.setProperty("--callout-color", confidenceColor(c.confidence));
      const card = document.createElement("div");
      card.className = "callout-card";
      const title = document.createElement("span");
      title.className = "callout-title";
      title.textContent = (c.label || "BOUNDARY").replace(" / ", " ↔ ");
      const value = document.createElement("span");
      value.className = "callout-value";
      value.textContent = Number(c.confidence).toFixed(2);
      card.append(title, value);
      el.append(card);
      this.calloutLayer.append(el);
      this.callouts.push({
        el, position: c.position, normal: c.normal, type: "boundary", confidence: c.confidence,
      });
    }
    const probes = this.scene.probes.slice(0, 3);
    for (const p of probes) {
      const el = document.createElement("div");
      el.className = "callout probe";
      const card = document.createElement("div");
      card.className = "callout-card";
      const title = document.createElement("span");
      title.className = "callout-title";
      title.textContent = "Δ FORM";
      const value = document.createElement("span");
      value.className = "callout-value";
      const magnitude = Math.abs(Number(p.value));
      const shown = magnitude < 0.0005 ? "0.000" : `${p.value >= 0 ? "+" : "−"}${magnitude.toFixed(3)}`;
      value.textContent = `${shown} mm`;
      card.append(title, value);
      el.append(card);
      this.calloutLayer.append(el);
      this.callouts.push({
        el, position: p.position, normal: p.normal, type: "probe", confidence: 1,
      });
    }
  }

  updateCalloutPositions() {
    if (!this.scene) return;
    const showBoundaryLabels = this.mode !== 1 && this.showBoundaries;
    const placed = [];
    let probeIndex = 0;
    for (const item of this.callouts) {
      if (item.type === "boundary" && !showBoundaryLabels) {
        item.el.style.display = "none";
        continue;
      }
      if (item.type === "probe") {
        probeIndex++;
        if (this.mode === 0 && probeIndex > 1) {
          item.el.style.display = "none";
          continue;
        }
      }
      const projected = this.projectPoint(item.position);
      if (!projected) {
        item.el.style.display = "none";
        continue;
      }
      if (item.normal && this.eye) {
        const toEye = normalize(sub(this.eye, item.position));
        if (dot(normalize(item.normal), toEye) < -0.04) {
          item.el.style.display = "none";
          continue;
        }
      }
      const width = item.el.offsetWidth || 128;
      const height = item.el.offsetHeight || 20;
      const x = projected.x + 7;
      const y = projected.y;
      const rect = { x, y: y - height / 2, w: width + 20, h: height + 4 };
      const collides = placed.some((p) =>
        Math.abs((rect.x + rect.w / 2) - (p.x + p.w / 2)) < (rect.w + p.w) / 2 + 5 &&
        Math.abs(rect.y - p.y) < rect.h + 4
      );
      if (collides) {
        item.el.style.display = "none";
        continue;
      }
      item.el.style.display = "flex";
      item.el.style.left = `${x}px`;
      item.el.style.top = `${y}px`;
      placed.push(rect);
    }
  }

  updateReadout() {
    if (!this.camera) return;
    const az = ((this.camera.azimuth % 360) + 360) % 360;
    const azEl = document.getElementById("azReadout");
    const elEl = document.getElementById("elReadout");
    if (azEl) azEl.textContent = `${String(Math.round(az)).padStart(3, "0")}°`;
    if (elEl) elEl.textContent = `${String(Math.round(this.camera.elevation)).padStart(3, "0")}°`;
  }

  bindInteractions() {
    const canvas = this.canvas;
    canvas.addEventListener("pointerdown", (event) => {
      if (event.button !== 0 || !this.camera) return;
      this.dragging = true;
      this.lastPointer = [event.clientX, event.clientY];
      canvas.setPointerCapture(event.pointerId);
    });
    canvas.addEventListener("pointermove", (event) => {
      if (!this.dragging || !this.lastPointer) return;
      const dx = event.clientX - this.lastPointer[0];
      const dy = event.clientY - this.lastPointer[1];
      this.lastPointer = [event.clientX, event.clientY];
      this.camera.azimuth -= dx * 0.32;
      this.camera.elevation = clamp(this.camera.elevation + dy * 0.24, 4, 82);
      this.render();
    });
    const endDrag = (event) => {
      if (!this.dragging) return;
      this.dragging = false;
      this.lastPointer = null;
      try {
        canvas.releasePointerCapture(event.pointerId);
      } catch (_) {
        // Pointer capture may already have been released.
      }
    };
    canvas.addEventListener("pointerup", endDrag);
    canvas.addEventListener("pointercancel", endDrag);
    canvas.addEventListener("wheel", (event) => {
      if (!this.camera) return;
      event.preventDefault();
      const initial = this.initialCamera?.distance || this.camera.distance;
      this.camera.distance = clamp(
        this.camera.distance * Math.exp(event.deltaY * 0.0012),
        initial * 0.38,
        initial * 2.8,
      );
      this.render();
    }, { passive: false });
    canvas.addEventListener("dblclick", () => this.resetView());
  }

  setMode(mode) {
    this.mode = mode === "audit" ? 1 : mode === "split" ? 2 : 0;
    this.render();
  }

  setOverlay(value) {
    this.overlay = clamp(value, 0, 1);
    this.render();
  }

  setBoundaries(enabled) {
    this.showBoundaries = enabled;
    this.render();
  }

  setContours(enabled) {
    this.contours = enabled;
    this.render();
  }

  resetView() {
    if (!this.initialCamera) return;
    this.camera = {
      ...this.initialCamera,
      target: this.initialCamera.target.slice(),
    };
    this.render();
  }

  toggleAutoRotate() {
    this.autoRotate = !this.autoRotate;
    if (this.autoRotate) {
      this.lastRotationTime = performance.now();
      requestAnimationFrame((time) => this.rotationLoop(time));
    }
    return this.autoRotate;
  }

  rotationLoop(time) {
    if (!this.autoRotate) return;
    const dt = Math.min(48, time - this.lastRotationTime);
    this.lastRotationTime = time;
    this.camera.azimuth += dt * 0.008;
    this.render();
    requestAnimationFrame((next) => this.rotationLoop(next));
  }
}
