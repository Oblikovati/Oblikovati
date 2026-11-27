// SPDX-License-Identifier: GPL-2.0-only
// Orchestrator Worker: owns the Martensite WASM thread pool and WebGPU OffscreenCanvas.

let runtime = null;
let redrawQueued = false;

function scheduleFrame() {
  if (redrawQueued || !runtime) return;
  redrawQueued = true;
  const tick = () => {
    redrawQueued = false;
    if (!runtime) return;
    try {
      runtime.render_frame();
    } catch (error) {
      self.postMessage({ type: "error", message: String(error) });
    }
  };
  if (typeof self.requestAnimationFrame === "function") {
    self.requestAnimationFrame(tick);
  } else {
    setTimeout(tick, 0);
  }
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
      const state = JSON.parse(runtime.state_json());
      reply({ ok: true, state });
      return;
    }

    if (!runtime) {
      reply({ ok: false, error: "WebCadRuntime is not initialized" });
      return;
    }

    switch (msg.type) {
      case "command": {
        const state = JSON.parse(runtime.dispatch_command_json(JSON.stringify(msg.command)));
        scheduleFrame();
        reply({ ok: true, state });
        break;
      }
      case "loadDocument": {
        const bytes = new Uint8Array(msg.bytes);
        const state = JSON.parse(runtime.load_document_bytes(msg.name, bytes));
        scheduleFrame();
        reply({ ok: true, state });
        break;
      }
      case "exportDocument": {
        const bytes = runtime.export_document_bytes();
        const copy = bytes.slice().buffer;
        reply({ ok: true, bytes: copy }, [copy]);
        break;
      }
      case "pointer": {
        const changed = runtime.pointer_event(
          msg.kind,
          msg.cssX,
          msg.cssY,
          msg.button,
          msg.shift,
          msg.control,
          msg.alt,
          msg.meta,
        );
        if (changed) scheduleFrame();
        break;
      }
      case "wheel": {
        const changed = runtime.wheel_event(msg.cssX, msg.cssY, msg.deltaY, msg.deltaMode);
        if (changed) scheduleFrame();
        break;
      }
      case "key": {
        const changed = runtime.key_event(
          msg.code,
          msg.key,
          msg.pressed,
          msg.shift,
          msg.control,
          msg.alt,
          msg.meta,
        );
        if (changed) {
          scheduleFrame();
          self.postMessage({ type: "state", state: JSON.parse(runtime.state_json()) });
        }
        break;
      }
      case "resize": {
        const extent = JSON.parse(runtime.resize(msg.cssWidth, msg.cssHeight, msg.dpr));
        scheduleFrame();
        if (msg.id !== undefined) reply({ ok: true, extent });
        break;
      }
      case "capture": {
        const rgba = await runtime.capture_rgba_async();
        const state = JSON.parse(runtime.state_json());
        const validationErrors = JSON.parse(runtime.validation_errors_json());
        const copy = rgba.slice().buffer;
        reply({ ok: true, rgba: copy, state, validationErrors }, [copy]);
        break;
      }
      case "state": {
        const state = JSON.parse(runtime.state_json());
        const validationErrors = JSON.parse(runtime.validation_errors_json());
        reply({ ok: true, state, validationErrors });
        break;
      }
      case "close": {
        runtime.close();
        runtime = null;
        reply({ ok: true });
        break;
      }
      default:
        reply({ ok: false, error: `Unknown worker message type: ${msg.type}` });
    }
  } catch (error) {
    if (msg.id !== undefined) {
      reply({ ok: false, error: String(error) });
    } else {
      self.postMessage({ type: "error", message: String(error) });
    }
  }
};
