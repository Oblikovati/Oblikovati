// SPDX-License-Identifier: GPL-2.0-only
// Main-thread browser controller: owns DOM events and transfers OffscreenCanvas to the worker.

const canvas = document.getElementById("oblikovati-canvas");
const banner = document.getElementById("overlay-banner");
const docList = document.getElementById("doc-list");
const featureList = document.getElementById("feature-list");
const paramList = document.getElementById("param-list");
const statusText = document.getElementById("status-text");
const statusGpu = document.getElementById("status-gpu");
const statusWorkers = document.getElementById("status-workers");
const statusMesh = document.getElementById("status-mesh");

let worker = null;
let nextRequestId = 1;
const pending = new Map();
let currentState = null;

function setBanner(text, isError = false) {
  banner.textContent = text;
  banner.classList.toggle("hidden", !text);
  banner.classList.toggle("error", isError);
}

function sendRpc(message, transfer = []) {
  return new Promise((resolve, reject) => {
    const id = nextRequestId++;
    pending.set(id, { resolve, reject });
    worker.postMessage({ ...message, id }, transfer);
  });
}

function renderState(state) {
  currentState = state;
  document.body.dataset.theme = state.theme;
  statusText.textContent = state.statusMessage;
  statusGpu.textContent = `GPU: ${state.deviceName}`;
  statusWorkers.textContent = `Workers: ${state.activeWorkers}`;
  statusMesh.textContent = `Bodies: ${state.visibleBodies} · Tris: ${state.triangleCount}`;

  document.getElementById("btn-undo").disabled = !state.canUndo;
  document.getElementById("btn-redo").disabled = !state.canRedo;
  const projBtn = document.getElementById("btn-proj");
  projBtn.textContent = state.perspective ? "Persp" : "Ortho";
  projBtn.classList.toggle("active", state.perspective);
  document.getElementById("btn-normals").classList.toggle("active", state.normalDebug);

  docList.innerHTML = "";
  for (const doc of state.documents) {
    const li = document.createElement("li");
    li.className = `tree-item${doc.active ? " active" : ""}`;
    li.innerHTML = `<span>${doc.name}${doc.dirty ? " *" : ""}</span><span class="tree-meta">#${doc.id}</span>`;
    li.addEventListener("click", () => dispatchCommand({ type: "activateDocument", id: doc.id }));
    docList.appendChild(li);
  }

  featureList.innerHTML = "";
  for (const feat of state.features) {
    const li = document.createElement("li");
    li.className = "tree-item";
    li.innerHTML = `<span>${feat.name}</span><span class="tree-meta">${feat.status}</span>`;
    featureList.appendChild(li);
  }

  paramList.innerHTML = "";
  for (const param of state.parameters) {
    const row = document.createElement("div");
    row.className = "param-row";
    const label = document.createElement("label");
    label.textContent = param.name;
    const input = document.createElement("input");
    input.value = param.expression;
    input.addEventListener("change", () => {
      dispatchCommand({ type: "setParameter", name: param.name, expression: input.value });
    });
    row.append(label, input);
    paramList.appendChild(row);
  }
}

async function dispatchCommand(command) {
  try {
    const response = await sendRpc({ type: "command", command });
    renderState(response.state);
    return response.state;
  } catch (error) {
    statusText.textContent = String(error);
    throw error;
  }
}

function wireToolbar() {
  document.getElementById("btn-new-part").addEventListener("click", () => {
    dispatchCommand({ type: "newPart", name: null });
  });
  document.getElementById("btn-sample-part").addEventListener("click", () => {
    dispatchCommand({ type: "loadSamplePart" });
  });
  document.getElementById("btn-undo").addEventListener("click", () => {
    dispatchCommand({ type: "undo" });
  });
  document.getElementById("btn-redo").addEventListener("click", () => {
    dispatchCommand({ type: "redo" });
  });
  document.getElementById("btn-fit").addEventListener("click", () => {
    dispatchCommand({ type: "fit" });
  });
  for (const [id, view] of [
    ["btn-iso", "iso"],
    ["btn-front", "front"],
    ["btn-top", "top"],
    ["btn-right", "right"],
  ]) {
    document.getElementById(id).addEventListener("click", () => {
      dispatchCommand({ type: "orient", view });
    });
  }
  document.getElementById("btn-proj").addEventListener("click", () => {
    dispatchCommand({ type: "setProjection", perspective: !currentState?.perspective });
  });
  document.getElementById("btn-normals").addEventListener("click", () => {
    dispatchCommand({ type: "setNormalDebug", enabled: !currentState?.normalDebug });
  });
  document.getElementById("btn-theme").addEventListener("click", () => {
    const next = currentState?.theme === "dark" ? "light" : "dark";
    dispatchCommand({ type: "setTheme", theme: next });
  });

  const fileInput = document.getElementById("file-input");
  document.getElementById("btn-open-file").addEventListener("click", () => fileInput.click());
  fileInput.addEventListener("change", async () => {
    const file = fileInput.files?.[0];
    if (!file) return;
    const buffer = await file.arrayBuffer();
    const response = await sendRpc({ type: "loadDocument", name: file.name, bytes: buffer }, [buffer]);
    renderState(response.state);
    fileInput.value = "";
  });

  document.getElementById("btn-export-file").addEventListener("click", async () => {
    const response = await sendRpc({ type: "exportDocument" });
    const blob = new Blob([response.bytes], { type: "application/x-yaml" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "document.opd";
    link.click();
    URL.revokeObjectURL(url);
  });
}

function wireCanvasInput() {
  const sendPointer = (kind, e) => {
    const rect = canvas.getBoundingClientRect();
    worker.postMessage({
      type: "pointer",
      kind,
      cssX: e.clientX - rect.left,
      cssY: e.clientY - rect.top,
      button: e.button,
      shift: e.shiftKey,
      control: e.ctrlKey,
      alt: e.altKey,
      meta: e.metaKey,
    });
  };
  canvas.addEventListener("contextmenu", (e) => e.preventDefault());
  canvas.addEventListener("pointerdown", (e) => {
    canvas.setPointerCapture(e.pointerId);
    canvas.focus();
    sendPointer(0, e);
  });
  canvas.addEventListener("pointerup", (e) => sendPointer(1, e));
  canvas.addEventListener("pointermove", (e) => sendPointer(2, e));
  canvas.addEventListener(
    "wheel",
    (e) => {
      e.preventDefault();
      const rect = canvas.getBoundingClientRect();
      worker.postMessage({
        type: "wheel",
        cssX: e.clientX - rect.left,
        cssY: e.clientY - rect.top,
        deltaY: e.deltaY,
        deltaMode: e.deltaMode,
      });
    },
    { passive: false },
  );
  canvas.addEventListener("keydown", (e) => {
    worker.postMessage({
      type: "key",
      code: e.code,
      key: e.key,
      pressed: true,
      shift: e.shiftKey,
      control: e.ctrlKey,
      alt: e.altKey,
      meta: e.metaKey,
    });
  });
  const observer = new ResizeObserver((entries) => {
    for (const entry of entries) {
      const { width, height } = entry.contentRect;
      if (width > 0 && height > 0) {
        worker.postMessage({
          type: "resize",
          cssWidth: width,
          cssHeight: height,
          dpr: window.devicePixelRatio || 1,
        });
      }
    }
  });
  observer.observe(canvas);
}

async function boot() {
  if (!window.crossOriginIsolated) {
    setBanner(
      "Cross-Origin Isolation (COOP/COEP) is required for multi-threaded WebAssembly.",
      true,
    );
    window.__oblikovatiWebError = "crossOriginIsolated is false";
    return;
  }
  if (!("gpu" in navigator)) {
    setBanner("WebGPU is unavailable in this browser.", true);
    window.__oblikovatiWebError = "navigator.gpu is unavailable";
    return;
  }

  let mainThreadTicks = 0;
  const ticker = setInterval(() => {
    mainThreadTicks += 1;
  }, 16);

  const rect = canvas.getBoundingClientRect();
  const dpr = window.devicePixelRatio || 1;
  canvas.width = Math.max(1, Math.round((rect.width || 960) * dpr));
  canvas.height = Math.max(1, Math.round((rect.height || 640) * dpr));
  const offscreen = canvas.transferControlToOffscreen();

  worker = new Worker("./orchestrator.js", { type: "module" });
  worker.onmessage = (event) => {
    const msg = event.data;
    if (msg.type === "state") {
      renderState(msg.state);
      return;
    }
    if (msg.type === "error") {
      setBanner(msg.message, true);
      return;
    }
    const waiter = pending.get(msg.id);
    if (!waiter) return;
    pending.delete(msg.id);
    if (msg.ok) waiter.resolve(msg);
    else waiter.reject(new Error(msg.error || "Worker request failed"));
  };
  worker.onerror = (error) => {
    setBanner(`Worker error: ${error.message}`, true);
    window.__oblikovatiWebError = error.message;
  };

  const maxWorkers = Math.max(1, Math.min(navigator.hardwareConcurrency || 4, 8));
  try {
    const response = await sendRpc(
      { type: "init", canvas: offscreen, dpr, maxWorkers, validation: true },
      [offscreen],
    );
    clearInterval(ticker);
    renderState(response.state);
    setBanner("");
    wireToolbar();
    wireCanvasInput();
    window.__oblikovatiWeb = {
      crossOriginIsolated: window.crossOriginIsolated,
      mainThreadBlocked: mainThreadTicks === 0,
      dispatchCommand,
      capture: () => sendRpc({ type: "capture" }),
      state: () => sendRpc({ type: "state" }),
      loadDocument: (name, bytes) => {
        const buf = new Uint8Array(bytes).buffer;
        return sendRpc({ type: "loadDocument", name, bytes: buf }, [buf]);
      },
      exportDocument: () => sendRpc({ type: "exportDocument" }),
      close: () => sendRpc({ type: "close" }),
    };
    window.__oblikovatiWebReady = true;
  } catch (error) {
    clearInterval(ticker);
    setBanner(String(error), true);
    window.__oblikovatiWebError = String(error);
  }
}

await boot();
