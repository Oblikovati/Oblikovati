// SPDX-License-Identifier: GPL-2.0-only
// Dedicated Web Worker that runs a single Go `js-wasm` add-in (`wasm_exec.js`).
// Bridges synchronous Go `client.Caller.Call` calls back to the Orchestrator Worker
// over SharedArrayBuffer + Atomics.wait (COOP/COEP cross-origin isolated).

importScripts("./wasm_exec.js");

const HEADER_BYTES = 16;
const DEFAULT_BUFFER_BYTES = 4 * 1024 * 1024;
const STATE_PENDING = 0;
const STATE_OK = 1;
const STATE_ERR = 2;
const STATE_GROW = 3;

let callSab = new SharedArrayBuffer(HEADER_BYTES + DEFAULT_BUFFER_BYTES);
let registered = null;
let registerResolver = null;
let addInId = "";
const channelListeners = new Map();

function ensureSharedBuffer(minPayloadBytes) {
  const needed = HEADER_BYTES + Math.max(DEFAULT_BUFFER_BYTES, minPayloadBytes);
  if (callSab.byteLength < needed) {
    callSab = new SharedArrayBuffer(needed);
  }
  return callSab;
}

function synchronousHostCall(method, reqBytes) {
  const payload = reqBytes instanceof Uint8Array ? reqBytes.slice() : new Uint8Array(0);
  let sab = ensureSharedBuffer(DEFAULT_BUFFER_BYTES);
  let ctrl = new Int32Array(sab, 0, 4);
  Atomics.store(ctrl, 0, STATE_PENDING);
  Atomics.store(ctrl, 1, 0);
  self.postMessage(
    { type: "hostCall", addInId, method: String(method), req: payload.buffer, sab },
    [payload.buffer],
  );
  Atomics.wait(ctrl, 0, STATE_PENDING);

  let state = Atomics.load(ctrl, 0);
  if (state === STATE_GROW) {
    const required = Atomics.load(ctrl, 1);
    sab = ensureSharedBuffer(required);
    ctrl = new Int32Array(sab, 0, 4);
    Atomics.store(ctrl, 0, STATE_PENDING);
    Atomics.store(ctrl, 1, 0);
    self.postMessage({ type: "hostCallRead", addInId, sab });
    Atomics.wait(ctrl, 0, STATE_PENDING);
    state = Atomics.load(ctrl, 0);
  }

  const len = Math.max(0, Atomics.load(ctrl, 1));
  const out = new Uint8Array(len);
  if (len > 0) {
    out.set(new Uint8Array(sab, HEADER_BYTES, len));
  }
  return { ok: state === STATE_OK, data: out };
}

function emitChannel(channel, payload) {
  self.postMessage({
    type: "addinChannel",
    addInId,
    channel: String(channel),
    payload,
  });
}

function onChannel(channel, callback) {
  const key = String(channel);
  if (!channelListeners.has(key)) {
    channelListeners.set(key, []);
  }
  channelListeners.get(key).push(callback);
}

globalThis.__obkHost = {
  register(exportsObj) {
    registered = exportsObj;
    addInId = String(exportsObj.id());
    if (registerResolver) {
      const resolve = registerResolver;
      registerResolver = null;
      resolve(exportsObj);
    }
  },
  call: synchronousHostCall,
  emit: emitChannel,
  onChannel,
  log(message) {
    self.postMessage({ type: "addinLog", addInId, message: String(message) });
  },
};

async function loadWasmModule(msg) {
  const go = new Go();
  let bytes;
  if (msg.wasmBytes) {
    bytes = msg.wasmBytes;
  } else if (msg.url) {
    const response = await fetch(msg.url);
    if (!response.ok) {
      throw new Error(`fetch ${msg.url}: HTTP ${response.status}`);
    }
    bytes = await response.arrayBuffer();
  } else {
    throw new Error("loadAddIn requires url or wasmBytes");
  }

  const registeredPromise = new Promise((resolve) => {
    registerResolver = resolve;
  });
  const instantiated = await WebAssembly.instantiate(bytes, go.importObject);
  const runPromise = go.run(instantiated.instance);
  runPromise.catch((error) => {
    self.postMessage({ type: "addinError", addInId, message: String(error) });
  });

  const exportsObj = await Promise.race([
    registeredPromise,
    new Promise((_, reject) =>
      setTimeout(() => reject(new Error("add-in did not call __obkHost.register within 15s")), 15000),
    ),
  ]);

  return {
    id: String(exportsObj.id()),
    manifest: String(exportsObj.manifest()),
    major: Number(exportsObj.apiMajor()),
    minor: Number(exportsObj.apiMinor()),
    hasAutomation: Boolean(exportsObj.hasAutomation()),
  };
}

self.onmessage = async (event) => {
  const msg = event.data;
  const reply = (payload, transfer = []) => {
    if (msg.id !== undefined) {
      self.postMessage({ id: msg.id, ...payload }, transfer);
    }
  };

  try {
    switch (msg.type) {
      case "load": {
        const probe = await loadWasmModule(msg);
        reply({ ok: true, probe });
        break;
      }
      case "activate": {
        if (!registered) throw new Error("add-in is not loaded");
        const rc = Number(await registered.activate());
        if (rc !== 0) throw new Error(`ObkAddInActivate returned ${rc}`);
        reply({ ok: true });
        break;
      }
      case "deactivate": {
        if (!registered) {
          reply({ ok: true });
          return;
        }
        const rc = Number(await registered.deactivate());
        if (rc !== 0) throw new Error(`ObkAddInDeactivate returned ${rc}`);
        reply({ ok: true });
        break;
      }
      case "notify": {
        if (registered) {
          const bytes = msg.eventBytes ? new Uint8Array(msg.eventBytes) : new Uint8Array(0);
          await registered.notify(bytes);
        }
        reply({ ok: true });
        break;
      }
      case "automation": {
        if (!registered) throw new Error("add-in is not loaded");
        const req = msg.reqBytes ? new Uint8Array(msg.reqBytes) : new Uint8Array(0);
        const res = await registered.automation(String(msg.method || ""), req);
        const rawBytes = res.data instanceof Uint8Array ? res.data : new Uint8Array(0);
        if (!res.ok) {
          const msgText = new TextDecoder().decode(rawBytes) || "add-in automation call failed";
          reply({ ok: false, error: msgText });
          return;
        }
        const out = rawBytes.slice().buffer;
        reply({ ok: true, data: out }, [out]);
        break;
      }
      case "channel": {
        const listeners = channelListeners.get(String(msg.channel)) || [];
        for (const fn of listeners) {
          fn(msg.payload);
        }
        reply({ ok: true });
        break;
      }
      default:
        reply({ ok: false, error: `Unknown add-in worker command: ${msg.type}` });
    }
  } catch (error) {
    reply({ ok: false, error: String(error) });
  }
};
