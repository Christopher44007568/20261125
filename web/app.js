const scenes = {
  gray: {
    name: "灰阶",
    images: {
      none: "images/gray_none.png",
      ordered: "images/gray_ordered.png",
      diffusion: "images/gray_diffusion.png",
    },
  },
  skin: {
    name: "肤色",
    images: {
      none: "images/skin_none.png",
      ordered: "images/skin_ordered.png",
      diffusion: "images/skin_diffusion.png",
    },
  },
  chroma: {
    name: "低亮度彩色",
    images: {
      none: "images/chroma_none.png",
      ordered: "images/chroma_ordered.png",
      diffusion: "images/chroma_diffusion.png",
    },
  },
};

const modes = ["none", "ordered", "diffusion"];
const imageWidth = 720;
const imageHeight = 196;
const flatHeight = 40;
const loaded = {};

const state = {
  scene: "gray",
  x: 0.68,
  y: 112,
  zoom: 10,
  enhance: true,
};

const tabs = [...document.querySelectorAll(".scene-tab")];
const rows = [...document.querySelectorAll(".matrix-row")];
const cells = [...document.querySelectorAll(".cell")];
const regionButtons = [...document.querySelectorAll("[data-region]")];
const zoomButtons = [...document.querySelectorAll("[data-zoom]")];
const viewButtons = [...document.querySelectorAll("[data-view]")];
const slider = document.querySelector("#position-slider");
const sliderValue = document.querySelector("#position-value");
const loupeScene = document.querySelector("#loupe-scene");
const cropInfo = document.querySelector("#crop-info");
const flatState = document.querySelector("#flat-state");
const viewLabels = [...document.querySelectorAll(".view-label")];
const cropCanvas = document.createElement("canvas");
const enhancedCanvas = document.createElement("canvas");

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

function preload() {
  for (const [sceneID, scene] of Object.entries(scenes)) {
    loaded[sceneID] = {};
    for (const mode of modes) {
      const image = new Image();
      image.decoding = "async";
      image.src = window.DITHER_ASSETS?.[sceneID]?.[mode] || scene.images[mode];
      image.addEventListener("load", () => drawLoupe());
      loaded[sceneID][mode] = image;
    }
  }
}

function selectScene(sceneID) {
  state.scene = sceneID;
  tabs.forEach((tab) => {
    const active = tab.dataset.scene === sceneID;
    tab.classList.toggle("is-active", active);
    tab.setAttribute("aria-selected", String(active));
  });
  rows.forEach((row) => {
    row.classList.toggle("is-active", row.dataset.sceneRow === sceneID);
  });
  loupeScene.textContent = scenes[sceneID].name;
  drawLoupe();
}

function setSample(x, y) {
  state.x = clamp(x, 0, 1);
  state.y = clamp(y, 0, imageHeight - 1);
  const percent = Math.round(state.x * 100);
  slider.value = String(percent);
  sliderValue.textContent = `${percent}%`;
  regionButtons.forEach((button) => {
    const flat = button.dataset.region === "flat";
    button.classList.toggle("is-active", flat === (state.y < flatHeight));
  });
  positionProbes();
  drawLoupe();
}

function positionProbes() {
  const left = `${state.x * 100}%`;
  const top = `${(state.y / imageHeight) * 100}%`;
  cells.forEach((cell) => {
    const probe = cell.querySelector(".probe");
    probe.style.left = left;
    probe.style.top = top;
  });
}

function drawLoupe() {
  const cropWidth = Math.round(480 / state.zoom);
  const cropHeight = Math.round(200 / state.zoom);
  let sx = Math.round(state.x * (imageWidth - 1) - cropWidth / 2);
  let sy = Math.round(state.y - cropHeight / 2);
  sx = clamp(sx, 0, imageWidth - cropWidth);
  sy = clamp(sy, 0, imageHeight - cropHeight);

  for (const mode of modes) {
    const canvas = document.querySelector(`#canvas-${mode}`);
    const context = canvas.getContext("2d", { willReadFrequently: true });
    const image = loaded[state.scene]?.[mode];
    context.imageSmoothingEnabled = false;
    context.clearRect(0, 0, canvas.width, canvas.height);
    let rawData = null;
    if (image?.complete && image.naturalWidth) {
      rawData = readCrop(image, sx, sy, cropWidth, cropHeight);
      if (state.enhance) {
        drawEnhanced(context, canvas, rawData, cropWidth, cropHeight);
      } else {
        context.drawImage(
          image,
          sx, sy, cropWidth, cropHeight,
          0, 0, canvas.width, canvas.height,
        );
      }
    }
    updateReadout(mode, rawData, cropWidth, cropHeight, sx, sy);
  }

  cropInfo.textContent = `x ${sx}—${sx + cropWidth} / y ${sy}—${sy + cropHeight}`;
  const isFlat = sy + cropHeight <= flatHeight;
  flatState.classList.toggle("is-flat", isFlat);
  flatState.textContent = isFlat ? "FLAT / QUIET" : "GRADIENT / ACTIVE";
  positionProbes();
}

function readCrop(image, sx, sy, width, height) {
  cropCanvas.width = width;
  cropCanvas.height = height;
  const context = cropCanvas.getContext("2d", { willReadFrequently: true });
  context.imageSmoothingEnabled = false;
  context.drawImage(image, sx, sy, width, height, 0, 0, width, height);
  return context.getImageData(0, 0, width, height);
}

function drawEnhanced(context, canvas, imageData, width, height) {
  enhancedCanvas.width = width;
  enhancedCanvas.height = height;
  const source = imageData.data;
  const output = new Uint8ClampedArray(source.length);
  const gain = 8;
  const radius = 2;
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      let count = 0;
      let red = 0;
      let green = 0;
      let blue = 0;
      for (let oy = -radius; oy <= radius; oy++) {
        const sy = clamp(y + oy, 0, height - 1);
        for (let ox = -radius; ox <= radius; ox++) {
          const sx = clamp(x + ox, 0, width - 1);
          const index = (sy * width + sx) * 4;
          red += source[index];
          green += source[index + 1];
          blue += source[index + 2];
          count += 1;
        }
      }
      const index = (y * width + x) * 4;
      output[index] = source[index] + (source[index] - red / count) * gain;
      output[index + 1] = source[index + 1] + (source[index + 1] - green / count) * gain;
      output[index + 2] = source[index + 2] + (source[index + 2] - blue / count) * gain;
      output[index + 3] = 255;
    }
  }
  const enhanced = new ImageData(output, width, height);
  enhancedCanvas.getContext("2d").putImageData(enhanced, 0, 0);
  context.imageSmoothingEnabled = false;
  context.drawImage(enhancedCanvas, 0, 0, width, height, 0, 0, canvas.width, canvas.height);
}

function updateReadout(mode, imageData, width, height, sx, sy) {
  if (!imageData) {
    return;
  }
  const data = imageData.data;
  let count = 0;
  let sum = 0;
  let squareSum = 0;
  let minimum = 255;
  let maximum = 0;
  for (let i = 0; i < data.length; i += 4) {
    const luma = 0.2126 * data[i] + 0.7152 * data[i + 1] + 0.0722 * data[i + 2];
    count += 1;
    sum += luma;
    squareSum += luma * luma;
    minimum = Math.min(minimum, luma);
    maximum = Math.max(maximum, luma);
  }
  const mean = count ? sum / count : 0;
  const variance = count ? Math.max(0, squareSum / count - mean * mean) : 0;
  const sigma = Math.sqrt(variance);
  const sigmaNode = document.querySelector(`#sigma-${mode}`);
  sigmaNode.textContent = `σ ${sigma.toFixed(2)}`;
  sigmaNode.classList.toggle("has-noise", sigma > 0.08);
  document.querySelector(`#range-${mode}`).textContent =
    `L ${minimum.toFixed(0)}—${maximum.toFixed(0)} · ${sx + Math.floor(width / 2)},${sy + Math.floor(height / 2)}`;
}

tabs.forEach((tab) => {
  tab.addEventListener("click", () => selectScene(tab.dataset.scene));
});

cells.forEach((cell) => {
  cell.addEventListener("click", (event) => {
    const rect = cell.getBoundingClientRect();
    const x = (event.clientX - rect.left) / rect.width;
    const y = (event.clientY - rect.top) / rect.height * imageHeight;
    selectScene(cell.dataset.scene);
    setSample(x, y);
  });
});

regionButtons.forEach((button) => {
  button.addEventListener("click", () => {
    const y = button.dataset.region === "flat"
      ? flatHeight / 2
      : flatHeight + (imageHeight - flatHeight) / 2;
    setSample(state.x, y);
  });
});

zoomButtons.forEach((button) => {
  button.addEventListener("click", () => {
    state.zoom = Number(button.dataset.zoom);
    zoomButtons.forEach((node) => node.classList.toggle("is-active", node === button));
    drawLoupe();
  });
});

viewButtons.forEach((button) => {
  button.addEventListener("click", () => {
    state.enhance = button.dataset.view === "enhanced";
    viewButtons.forEach((node) => node.classList.toggle("is-active", node === button));
    viewLabels.forEach((label) => {
      label.textContent = state.enhance ? "Δ ×8" : "RAW 8-BIT";
    });
    drawLoupe();
  });
});

slider.addEventListener("input", () => {
  const percent = Number(slider.value);
  sliderValue.textContent = `${percent}%`;
  state.x = percent / 100;
  positionProbes();
  drawLoupe();
});

preload();
selectScene("gray");
setSample(0.68, 112);
