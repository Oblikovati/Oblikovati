// SPDX-License-Identifier: GPL-2.0-only
// Thin browser host: hands a full-viewport OffscreenCanvas and DOM events to the
// Orchestrator Worker running the exact Oblikovati desktop UI (oblikovati-gui + oblikovati-head).

const canvas = document.getElementById("viewport");
const bootOverlay = document.getElementById("boot-overlay");
const bootMessage = document.getElementById("boot-message");

let worker = null;
let nextRequestId = 1;
const pending = new Map();
let ready = false;

function showBootMessage(text, isError = false) {
  bootOverlay.classList.remove("hidden");
  bootMessage.textContent = text;
  bootMessage.classList.toggle("error", isError);
}

function hideBootOverlay() {
  bootOverlay.classList.add("hidden");
}

function rpc(payload, transfer = []) {
  return new Promise((resolve, reject) => {
    const id = nextRequestId++;
    pending.set(id, { resolve, reject });
    worker.postMessage({ id, ...payload }, transfer);
  });
}

function formatInitError(raw) {
  const message = String(raw);
  if (message.includes("request_adapter failed") || message.includes("WebGPU Core adapter")) {
    return (
      `${message}\n\n` +
      "WebGPU is required. On Linux Chromium, enable chrome://flags/#enable-unsafe-webgpu " +
      "and chrome://flags/#enable-vulkan (or launch with --enable-unsafe-webgpu --enable-features=Vulkan)."
    );
  }
  return message;
}

function wireCanvasInput() {
  const sendPointer = (kind, event) => {
    if (!worker || !ready) return;
    const rect = canvas.getBoundingClientRect();
    worker.postMessage({
      type: "pointer",
      kind,
      cssX: event.clientX - rect.left,
      cssY: event.clientY - rect.top,
      button: event.button,
      shift: event.shiftKey,
      control: event.ctrlKey,
      alt: event.altKey,
      meta: event.metaKey,
    });
  };

  canvas.addEventListener("pointerdown", (event) => {
    canvas.focus({ preventScroll: true });
    canvas.setPointerCapture(event.pointerId);
    sendPointer(0, event);
    event.preventDefault();
  });

  canvas.addEventListener("pointerup", (event) => {
    if (canvas.hasPointerCapture(event.pointerId)) {
      canvas.releasePointerCapture(event.pointerId);
    }
    sendPointer(1, event);
    event.preventDefault();
  });

  canvas.addEventListener("pointermove", (event) => {
    sendPointer(2, event);
  });

  canvas.addEventListener("contextmenu", (event) => {
    event.preventDefault();
  });

  canvas.addEventListener(
    "wheel",
    (event) => {
      if (!worker || !ready) return;
      event.preventDefault();
      const rect = canvas.getBoundingClientRect();
      worker.postMessage({
        type: "wheel",
        cssX: event.clientX - rect.left,
        cssY: event.clientY - rect.top,
        deltaY: event.deltaY,
        deltaMode: event.deltaMode,
      });
    },
    { passive: false },
  );

  const forwardKey = (pressed, event) => {
    if (!worker || !ready) return;
    if (
      event.code === "Tab" ||
      event.code.startsWith("F") && event.code.length <= 3 ||
      ((event.ctrlKey || event.metaKey) &&
        ["KeyS", "KeyZ", "KeyY", "KeyO", "KeyN"].includes(event.code))
    ) {
      event.preventDefault();
    }
    worker.postMessage({
      type: "key",
      code: event.code,
      key: event.key,
      pressed,
      shift: event.shiftKey,
      control: event.ctrlKey,
      alt: event.altKey,
      meta: event.metaKey,
    });
  };

  window.addEventListener("keydown", (event) => forwardKey(true, event));
  window.addEventListener("keyup", (event) => forwardKey(false, event));
  window.addEventListener("focus", () => {
    if (worker && ready) worker.postMessage({ type: "focus", focused: true });
  });
  window.addEventListener("blur", () => {
    if (worker && ready) worker.postMessage({ type: "focus", focused: false });
  });

  const observer = new ResizeObserver(() => {
    if (!worker || !ready) return;
    const rect = canvas.getBoundingClientRect();
    if (rect.width <= 0 || rect.height <= 0) return;
    worker.postMessage({
      type: "resize",
      cssWidth: rect.width,
      cssHeight: rect.height,
      dpr: window.devicePixelRatio || 1,
    });
  });
  observer.observe(canvas);

  window.addEventListener("dragover", (event) => event.preventDefault());
  window.addEventListener("drop", async (event) => {
    event.preventDefault();
    if (!worker || !ready) return;
    const file = event.dataTransfer?.files?.[0];
    if (!file) return;
    const buffer = await file.arrayBuffer();
    await rpc({ type: "openDocument", name: file.name, bytes: buffer }, [buffer]);
  });
}

async function boot() {
  if (!window.crossOriginIsolated) {
    showBootMessage(
      "Cross-Origin Isolation (COOP/COEP) is required for WebAssembly Threads.",
      true,
    );
    return;
  }
  if (typeof canvas.transferControlToOffscreen !== "function") {
    showBootMessage("OffscreenCanvas is required for Oblikovati WebGPU.", true);
    return;
  }

  const dpr = window.devicePixelRatio || 1;
  const rect = canvas.getBoundingClientRect();
  canvas.width = Math.max(1, Math.round((rect.width || window.innerWidth) * dpr));
  canvas.height = Math.max(1, Math.round((rect.height || window.innerHeight) * dpr));
  const offscreen = canvas.transferControlToOffscreen();

  worker = new Worker("./orchestrator.js", { type: "module" });
  worker.onmessage = (event) => {
    const msg = event.data;
    if (msg.id !== undefined && pending.has(msg.id)) {
      const waiter = pending.get(msg.id);
      pending.delete(msg.id);
      if (msg.ok) waiter.resolve(msg);
      else waiter.reject(new Error(msg.error || "Worker request failed"));
      return;
    }
    if (msg.type === "error") {
      showBootMessage(formatInitError(msg.message), true);
    }
  };
  worker.onerror = (event) => {
    showBootMessage(formatInitError(event.message || "Orchestrator worker error"), true);
  };

  wireCanvasInput();
  const maxWorkers = Math.min(8, Math.max(2, navigator.hardwareConcurrency || 4));
  const validation = new URLSearchParams(window.location.search).get("validation") === "1";

  try {
    const res = await rpc(
      {
        type: "init",
        canvas: offscreen,
        dpr,
        maxWorkers,
        validation,
      },
      [offscreen],
    );
    ready = true;
    hideBootOverlay();
    canvas.focus({ preventScroll: true });
    window.__OBLIKOVATI_WEB__ = {
      status: () => rpc({ type: "status" }),
      capture: () => rpc({ type: "capture" }),
      openDocumentBytes: (name, buffer) =>
        rpc({ type: "openDocument", name, bytes: buffer }, [buffer]),
      initialStatus: res.status,
    };
  } catch (error) {
    showBootMessage(formatInitError(error), true);
  }
}

boot();
