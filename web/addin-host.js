// SPDX-License-Identifier: GPL-2.0-only
// Browser add-in manager owned by the Orchestrator Worker.
// Loads Go `js-wasm` add-in workers, enforces the Oblikovati API version handshake,
// fulfills synchronous SharedArrayBuffer host API calls, and forwards channel events.

const HEADER_BYTES = 16;
const STATE_OK = 1;
const STATE_ERR = 2;
const STATE_GROW = 3;
const textEncoder = new TextEncoder();

async function extractWasmFromZip(zipBuffer) {
  const bytes = new Uint8Array(zipBuffer);
  const view = new DataView(zipBuffer);
  const eocdOffset = findZipEocd(view, bytes.byteLength);
  if (eocdOffset < 0) {
    throw new Error("bundle is not a valid zip archive");
  }
  const entryCount = view.getUint16(eocdOffset + 10, true);
  let cursor = view.getUint32(eocdOffset + 16, true);
  for (let i = 0; i < entryCount; i++) {
    if (view.getUint32(cursor, true) !== 0x02014b50) {
      throw new Error("invalid zip central directory header");
    }
    const method = view.getUint16(cursor + 10, true);
    const compSize = view.getUint32(cursor + 20, true);
    const nameLen = view.getUint16(cursor + 28, true);
    const extraLen = view.getUint16(cursor + 30, true);
    const commentLen = view.getUint16(cursor + 32, true);
    const localOffset = view.getUint32(cursor + 42, true);
    const name = new TextDecoder().decode(bytes.subarray(cursor + 46, cursor + 46 + nameLen));
    cursor += 46 + nameLen + extraLen + commentLen;
    if (!name.toLowerCase().endsWith(".wasm")) {
      continue;
    }
    return readZipEntry(bytes, view, localOffset, method, compSize);
  }
  throw new Error("bundle zip does not contain a .wasm module");
}

function findZipEocd(view, byteLength) {
  const minOffset = Math.max(0, byteLength - 65557);
  for (let i = byteLength - 22; i >= minOffset; i--) {
    if (view.getUint32(i, true) === 0x06054b50) {
      return i;
    }
  }
  return -1;
}

async function readZipEntry(bytes, view, localOffset, method, compSize) {
  if (view.getUint32(localOffset, true) !== 0x04034b50) {
    throw new Error("invalid zip local file header");
  }
  const nameLen = view.getUint16(localOffset + 26, true);
  const extraLen = view.getUint16(localOffset + 28, true);
  const dataStart = localOffset + 30 + nameLen + extraLen;
  const compressed = bytes.subarray(dataStart, dataStart + compSize);
  if (method === 0) {
    return compressed.slice().buffer;
  }
  if (method !== 8) {
    throw new Error(`unsupported zip compression method ${method}`);
  }
  const stream = new Blob([compressed]).stream().pipeThrough(new DecompressionStream("deflate-raw"));
  return new Response(stream).arrayBuffer();
}

export class WebAddInHost {
  constructor({ callApi, checkCompatibility, onChannel, onLog }) {
    this.callApi = callApi;
    this.checkCompatibility = checkCompatibility;
    this.onChannel = onChannel;
    this.onLog = onLog;
    this.addins = new Map();
    this.nextMsgId = 1;
  }

  async load(source) {
    let loadMsg;
    if (source.bundleUrl) {
      const response = await fetch(source.bundleUrl);
      if (!response.ok) {
        throw new Error(`fetch bundle ${source.bundleUrl}: HTTP ${response.status}`);
      }
      const wasmBytes = await extractWasmFromZip(await response.arrayBuffer());
      loadMsg = { type: "load", wasmBytes };
    } else if (source.wasmBytes) {
      loadMsg = { type: "load", wasmBytes: source.wasmBytes };
    } else if (source.url) {
      loadMsg = { type: "load", url: source.url };
    } else {
      throw new Error("loadAddIn requires url, bundleUrl, or wasmBytes");
    }

    const workerUrl = new URL("./addin-worker.js", self.location.href).href;
    const worker = new Worker(workerUrl);
    const entry = {
      id: "",
      probe: null,
      worker,
      pending: new Map(),
      pendingLargeReply: null,
      active: false,
    };
    this.wireWorker(entry);

    try {
      const transfer = loadMsg.wasmBytes ? [loadMsg.wasmBytes] : [];
      const loaded = await this.sendWorker(entry, loadMsg, transfer);
      const probe = loaded.probe;
      this.checkCompatibility(probe.major, probe.minor);
      if (this.addins.has(probe.id)) {
        await this.unload(probe.id);
      }
      entry.id = probe.id;
      entry.probe = probe;
      this.addins.set(probe.id, entry);
      await this.sendWorker(entry, { type: "activate" });
      entry.active = true;
      return probe;
    } catch (error) {
      if (entry.id) {
        this.addins.delete(entry.id);
      }
      worker.terminate();
      throw error;
    }
  }

  async unload(id) {
    const entry = this.addins.get(id);
    if (!entry) return false;
    this.addins.delete(id);
    try {
      if (entry.active) {
        await this.sendWorker(entry, { type: "deactivate" });
      }
    } finally {
      entry.active = false;
      entry.worker.terminate();
    }
    return true;
  }

  list() {
    return Array.from(this.addins.values()).map((entry) => ({
      ...entry.probe,
      active: entry.active,
    }));
  }

  async callAutomation(id, method, reqBytes) {
    const entry = this.addins.get(id);
    if (!entry || !entry.active) {
      throw new Error(`add-in ${id} is not active`);
    }
    const copy = reqBytes instanceof Uint8Array ? reqBytes.slice().buffer : new ArrayBuffer(0);
    return this.sendWorker(entry, { type: "automation", method, reqBytes: copy }, [copy]);
  }

  async sendChannel(id, channel, payload) {
    const entry = this.addins.get(id);
    if (!entry || !entry.active) {
      throw new Error(`add-in ${id} is not active`);
    }
    await this.sendWorker(entry, { type: "channel", channel, payload });
  }

  notifyAll(eventBytes) {
    for (const entry of this.addins.values()) {
      if (!entry.active) continue;
      const copy = eventBytes.slice().buffer;
      entry.worker.postMessage({ type: "notify", eventBytes: copy }, [copy]);
    }
  }

  async closeAll() {
    const ids = Array.from(this.addins.keys());
    for (const id of ids) {
      try {
        await this.unload(id);
      } catch (_) {
        // Worker termination already handled in unload.
      }
    }
  }

  wireWorker(entry) {
    entry.worker.onmessage = (event) => {
      const msg = event.data;
      if (msg.type === "hostCall") {
        this.handleHostCall(entry, msg);
        return;
      }
      if (msg.type === "hostCallRead") {
        this.handleHostCallRead(entry, msg);
        return;
      }
      if (msg.type === "addinChannel") {
        this.onChannel?.(msg.addInId || entry.id, msg.channel, msg.payload);
        return;
      }
      if (msg.type === "addinLog") {
        this.onLog?.(msg.addInId || entry.id, msg.message);
        return;
      }
      if (msg.type === "addinError") {
        this.onLog?.(msg.addInId || entry.id, `error: ${msg.message}`);
        return;
      }
      if (msg.id !== undefined && entry.pending.has(msg.id)) {
        const waiter = entry.pending.get(msg.id);
        entry.pending.delete(msg.id);
        if (msg.ok) waiter.resolve(msg);
        else waiter.reject(new Error(msg.error || "add-in worker call failed"));
      }
    };
    entry.worker.onerror = (event) => {
      for (const waiter of entry.pending.values()) {
        waiter.reject(new Error(event.message || "add-in worker crashed"));
      }
      entry.pending.clear();
    };
  }

  handleHostCall(entry, msg) {
    let ok = true;
    let payload;
    try {
      const req = new Uint8Array(msg.req || new ArrayBuffer(0));
      payload = this.callApi(msg.method, req);
    } catch (error) {
      ok = false;
      payload = textEncoder.encode(String(error));
    }
    this.writeHostReply(entry, msg.sab, ok, payload);
  }

  handleHostCallRead(entry, msg) {
    const pending = entry.pendingLargeReply;
    entry.pendingLargeReply = null;
    if (!pending) {
      this.writeHostReply(entry, msg.sab, false, textEncoder.encode("missing pending host reply"));
      return;
    }
    this.writeHostReply(entry, msg.sab, pending.ok, pending.payload);
  }

  writeHostReply(entry, sab, ok, payload) {
    const ctrl = new Int32Array(sab, 0, 4);
    const capacity = sab.byteLength - HEADER_BYTES;
    if (payload.byteLength > capacity) {
      entry.pendingLargeReply = { ok, payload };
      Atomics.store(ctrl, 1, payload.byteLength);
      Atomics.store(ctrl, 0, STATE_GROW);
      Atomics.notify(ctrl, 0);
      return;
    }
    if (payload.byteLength > 0) {
      new Uint8Array(sab, HEADER_BYTES, payload.byteLength).set(payload);
    }
    Atomics.store(ctrl, 1, payload.byteLength);
    Atomics.store(ctrl, 0, ok ? STATE_OK : STATE_ERR);
    Atomics.notify(ctrl, 0);
  }

  sendWorker(entry, payload, transfer = []) {
    return new Promise((resolve, reject) => {
      const id = this.nextMsgId++;
      entry.pending.set(id, { resolve, reject });
      entry.worker.postMessage({ id, ...payload }, transfer);
    });
  }
}
