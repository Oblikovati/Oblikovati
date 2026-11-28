// SPDX-License-Identifier: GPL-2.0-only
// Orchestrator Worker: owns the Martensite WASM thread pool, WebGPU OffscreenCanvas,
// the full Oblikovati desktop UI (oblikovati-gui + oblikovati-head), and js-wasm add-ins.

import { WebAddInHost } from "./addin-host.js";

let runtime = null;
let addInHost = null;
let timerId = null;
let rafScheduled = false;

function cancelScheduledTick() {
  if (timerId !== null) {
    clearTimeout(timerId);
    timerId = null;
  }
  rafScheduled = false;
}

function scheduleWake(wakeAfterMs) {
  if (!runtime || wakeAfterMs < 0) return;
  if (wakeAfterMs <= 16) {
    if (rafScheduled) return;
    rafScheduled = true;
    const run = () => {
      rafScheduled = false;
      if (!runtime) return;
      try {
        scheduleWake(runtime.tick());
      } catch (error) {
        self.postMessage({ type: "error", message: String(error) });
      }
    };
    if (typeof self.requestAnimationFrame === "function") {
      self.requestAnimationFrame(run);
    } else {
      setTimeout(run, 16);
    }
    return;
  }
  if (timerId !== null) clearTimeout(timerId);
  timerId = setTimeout(() => {
    timerId = null;
    if (!runtime) return;
    try {
      scheduleWake(runtime.tick());
    } catch (error) {
      self.postMessage({ type: "error", message: String(error) });
    }
  }, wakeAfterMs);
}

self.onmessage = async (event) => {
  const msg = event.data;
  const reply = (payload, transfer = []) => {
    self.postMessage({ id: msg.id, ...payload }, transfer);
  };

  try {
    if (msg.type === "init") {
      const glueUrl = new URL("./pkg/oblikovati_web.js", self.location.href).href;
      const wasmMod = await import(glueUrl);
      await wasmMod.default();
      runtime = await wasmMod.WebCadRuntime.create(
        msg.canvas,
        msg.dpr || 1,
        msg.maxWorkers || 0,
        glueUrl,
        Boolean(msg.validation),
      );
      addInHost = new WebAddInHost({
        callApi: (method, reqBytes) => {
          const out = runtime.call_api(method, reqBytes);
          scheduleWake(runtime.tick());
          return out;
        },
        checkCompatibility: (major, minor) => runtime.check_addin_compatibility(major, minor),
        onChannel: (addInId, channel, payload) => {
          self.postMessage({ type: "addinChannel", addInId, channel, payload });
        },
        onLog: (addInId, message) => {
          self.postMessage({ type: "addinLog", addInId, message });
        },
      });
      scheduleWake(runtime.tick());
      const status = JSON.parse(runtime.status_json());
      const apiVersion = JSON.parse(runtime.api_version_json());
      reply({ ok: true, status, apiVersion });
      return;
    }

    if (!runtime) {
      if (msg.id !== undefined) {
        reply({ ok: false, error: "WebCadRuntime is not initialized" });
      }
      return;
    }

    switch (msg.type) {
      case "pointer":
        scheduleWake(
          runtime.pointer_event(
            msg.kind,
            msg.cssX,
            msg.cssY,
            msg.button,
            msg.shift,
            msg.control,
            msg.alt,
            msg.meta,
          ),
        );
        break;
      case "wheel":
        scheduleWake(runtime.wheel_event(msg.cssX, msg.cssY, msg.deltaY, msg.deltaMode));
        break;
      case "key":
        scheduleWake(
          runtime.key_event(
            msg.code,
            msg.key,
            msg.pressed,
            msg.shift,
            msg.control,
            msg.alt,
            msg.meta,
          ),
        );
        break;
      case "focus":
        scheduleWake(runtime.focus_event(Boolean(msg.focused)));
        break;
      case "resize": {
        const res = JSON.parse(runtime.resize(msg.cssWidth, msg.cssHeight, msg.dpr));
        scheduleWake(res.wakeAfterMs);
        if (msg.id !== undefined) reply({ ok: true, extent: [res.width, res.height] });
        break;
      }
      case "openDocument": {
        const bytes = new Uint8Array(msg.bytes);
        scheduleWake(runtime.open_document_bytes(msg.name, bytes));
        const status = JSON.parse(runtime.status_json());
        reply({ ok: true, status });
        break;
      }
      case "capture": {
        const rgba = await runtime.capture_rgba_async();
        const status = JSON.parse(runtime.status_json());
        const validationErrors = JSON.parse(runtime.validation_errors_json());
        const copy = rgba.slice().buffer;
        reply({ ok: true, rgba: copy, status, validationErrors }, [copy]);
        break;
      }
      case "status": {
        const status = JSON.parse(runtime.status_json());
        const validationErrors = JSON.parse(runtime.validation_errors_json());
        reply({ ok: true, status, validationErrors });
        break;
      }
      case "loadAddIn": {
        const probe = await addInHost.load({
          url: msg.url,
          bundleUrl: msg.bundleUrl,
          wasmBytes: msg.wasmBytes,
        });
        reply({ ok: true, probe });
        break;
      }
      case "unloadAddIn": {
        const removed = await addInHost.unload(msg.addInId);
        reply({ ok: true, removed });
        break;
      }
      case "listAddIns": {
        reply({ ok: true, addins: addInHost.list() });
        break;
      }
      case "addinAutomation": {
        const req = msg.reqBytes ? new Uint8Array(msg.reqBytes) : new Uint8Array(0);
        const res = await addInHost.callAutomation(msg.addInId, msg.method, req);
        reply({ ok: true, data: res.data }, [res.data]);
        break;
      }
      case "addinChannel": {
        await addInHost.sendChannel(msg.addInId, msg.channel, msg.payload);
        reply({ ok: true });
        break;
      }
      case "close": {
        cancelScheduledTick();
        if (addInHost) {
          await addInHost.closeAll();
          addInHost = null;
        }
        runtime.close();
        runtime = null;
        reply({ ok: true });
        break;
      }
      default:
        if (msg.id !== undefined) {
          reply({ ok: false, error: `Unknown worker message type: ${msg.type}` });
        }
    }
  } catch (error) {
    if (msg.id !== undefined) {
      reply({ ok: false, error: String(error) });
    } else {
      self.postMessage({ type: "error", message: String(error) });
    }
  }
};
