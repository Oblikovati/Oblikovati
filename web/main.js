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
const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();
let mcpState = {
  addInId: "com.oblikovati.mcp-bridge",
  tools: [],
  registeredWebMCP: false,
};

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
    if (msg.type === "addinChannel") {
      handleAddInChannel(msg.addInId, msg.channel, msg.payload);
      return;
    }
    if (msg.type === "addinLog") {
      console.info(`[addin:${msg.addInId}] ${msg.message}`);
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
    const addinsApi = {
      loadUrl: (url) => rpc({ type: "loadAddIn", url }),
      loadBundleZip: (bundleUrl) => rpc({ type: "loadAddIn", bundleUrl }),
      loadBytes: (wasmBytes) => rpc({ type: "loadAddIn", wasmBytes }, [wasmBytes]),
      loadFromCatalogue: (catalogueBaseUrl, addInId) =>
        loadAddInFromCatalogue(catalogueBaseUrl, addInId, res.apiVersion),
      unload: (addInId) => rpc({ type: "unloadAddIn", addInId }),
      list: async () => (await rpc({ type: "listAddIns" })).addins,
      callAutomation: async (addInId, method, req) => {
        const bytes =
          req instanceof Uint8Array
            ? req.slice().buffer
            : textEncoder.encode(typeof req === "string" ? req : JSON.stringify(req ?? {})).buffer;
        const reply = await rpc({ type: "addinAutomation", addInId, method, reqBytes: bytes }, [
          bytes,
        ]);
        return new Uint8Array(reply.data);
      },
      sendChannel: (addInId, channel, payload) =>
        rpc({ type: "addinChannel", addInId, channel, payload }),
    };

    window.__OBLIKOVATI_WEB__ = {
      status: () => rpc({ type: "status" }),
      capture: () => rpc({ type: "capture" }),
      openDocumentBytes: (name, buffer) =>
        rpc({ type: "openDocument", name, bytes: buffer }, [buffer]),
      apiVersion: res.apiVersion,
      addins: addinsApi,
      requestSpaceMouse: () => connectSpaceMouseWebHID(addinsApi),
      initialStatus: res.status,
    };

    await autoLoadQueryAddIns(addinsApi);
  } catch (error) {
    showBootMessage(formatInitError(error), true);
  }
}

async function loadAddInFromCatalogue(catalogueBaseUrl, addInId, apiVersion) {
  const base = String(catalogueBaseUrl || "https://addins.oblikovati.org").replace(/\/+$/, "");
  const query = apiVersion ? `?api=${apiVersion.major}.${apiVersion.minor}` : "";
  const resp = await fetch(`${base}/catalogue${query}`);
  if (!resp.ok) {
    throw new Error(`catalogue query failed: HTTP ${resp.status}`);
  }
  const data = await resp.json();
  const entry = (data.addins || []).find((item) => item.name === addInId);
  if (!entry || !entry.versions || entry.versions.length === 0) {
    throw new Error(`add-in ${addInId} not found in catalogue`);
  }
  const bundle = entry.versions[0].bundles?.["js-wasm"];
  if (!bundle || !bundle.url) {
    throw new Error(`add-in ${addInId} has no js-wasm bundle for this API version`);
  }
  return rpc({ type: "loadAddIn", bundleUrl: bundle.url });
}

async function autoLoadQueryAddIns(addinsApi) {
  const params = new URLSearchParams(window.location.search);
  for (const url of params.getAll("addin")) {
    if (url) {
      await addinsApi.loadUrl(url);
    }
  }
  for (const bundleUrl of params.getAll("addin_bundle")) {
    if (bundleUrl) {
      await addinsApi.loadBundleZip(bundleUrl);
    }
  }
  const relayUrl = params.get("mcp_relay");
  if (relayUrl && window.__OBLIKOVATI_MCP__) {
    await window.__OBLIKOVATI_MCP__.connectRelay(relayUrl);
  }
}

function handleAddInChannel(addInId, channel, payload) {
  if (channel === "webmcp:register") {
    registerWebMCPBridge(addInId, payload);
  }
}

function registerWebMCPBridge(addInId, payload) {
  const tools = Array.isArray(payload?.tools) ? payload.tools : [];
  mcpState = { addInId, tools, registeredWebMCP: false };

  const invoke = async (method, body) => {
    const raw = await window.__OBLIKOVATI_WEB__.addins.callAutomation(addInId, method, body);
    const text = textDecoder.decode(raw);
    return text ? JSON.parse(text) : {};
  };

  window.__OBLIKOVATI_MCP__ = {
    listTools: () => mcpState.tools.slice(),
    callTool: (name, args = {}) => invoke("mcp.callTool", { name, arguments: args }),
    request: (jsonrpc) => invoke("mcp.jsonrpc", jsonrpc),
    connectRelay: (url) =>
      window.__OBLIKOVATI_WEB__.addins.sendChannel(addInId, "mcp:relay:connect", { url }),
    disconnectRelay: () =>
      window.__OBLIKOVATI_WEB__.addins.sendChannel(addInId, "mcp:relay:disconnect", {}),
  };

  if (navigator.modelContext && typeof navigator.modelContext.registerTool === "function") {
    for (const tool of tools) {
      navigator.modelContext.registerTool({
        name: tool.name,
        description: tool.description || "",
        inputSchema: tool.inputSchema || { type: "object", properties: {} },
        execute: async (input) => window.__OBLIKOVATI_MCP__.callTool(tool.name, input || {}),
      });
    }
    mcpState.registeredWebMCP = true;
  }
}

async function connectSpaceMouseWebHID(addinsApi) {
  if (!navigator.hid || typeof navigator.hid.requestDevice !== "function") {
    throw new Error("WebHID is not supported in this browser");
  }
  const devices = await navigator.hid.requestDevice({
    filters: [{ vendorId: 0x046d }, { vendorId: 0x256f }],
  });
  if (!devices || devices.length === 0) {
    return { connected: false };
  }
  const device = devices[0];
  if (!device.opened) {
    await device.open();
  }
  device.addEventListener("inputreport", (event) => {
    const view = event.data;
    const report = new Uint8Array(view.byteLength + 1);
    report[0] = event.reportId & 0xff;
    report.set(new Uint8Array(view.buffer, view.byteOffset, view.byteLength), 1);
    addinsApi
      .sendChannel("com.oblikovati.3d-connexion", "webhid:report", {
        report: Array.from(report),
      })
      .catch(() => {});
  });
  return { connected: true, productName: device.productName || "SpaceMouse" };
}

boot();
