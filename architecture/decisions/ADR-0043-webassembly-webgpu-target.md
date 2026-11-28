# ADR-0043: WebAssembly and WebGPU platform target

Date: 2026-11-27. Status: Accepted for the Rust web target.

Boundary: browser execution, canvas presentation, in-memory document storage, and
deployment for `app.oblikovati.org`.
Owner: `oblikovati-viewport`, `oblikovati-web`, `web/`, and `deploy/web/`.

## Context

Martensite `v0.16.0` splits backend-agnostic rendering into `martensite-render-core`
and provides a zero-`unsafe` WebGPU renderer (`martensite-webgpu`), browser canvas
windows (`martensite-tkservice::wasm::WasmWindow`), and a pre-warmed Web Worker
pool (`martensite-tkernel::osd::WasmWorkerPool`) on `wasm32-unknown-unknown` with
WebAssembly Threads (`+atomics`, `+bulk-memory`, `+mutable-globals`).

Oblikovati's desktop UI (`oblikovati-gui`) links Dear ImGui C++ through `imgui-sys`
and `cc`, and its desktop host (`oblikovati-head`) targets native X11, Wayland,
and Win32 windows. The browser target needs the same model, solver, session,
persistence, automation, and Martensite BRep kernel without C++ or OS windowing.

## Decision

Add `wasm32-unknown-unknown` (+atomics) + WebGPU as a first-class platform target
split into three browser realms:

1. **Main thread (`web/index.html`, `web/main.js`)** owns the DOM toolbar, Model
   Browser, parameter editor, file picker, and downloads. It transfers the viewport
   `<canvas>` via `transferControlToOffscreen()` and never blocks on `Atomics.wait`.
2. **Orchestrator Worker (`web/orchestrator.js`, `crates/web`)** runs `WebSession`
   on `oblikovati-session`, `oblikovati-automation`, `oblikovati-martensite`, and
   `oblikovati-viewport`. Documents persist through `MemoryDocumentFiles` in
   `oblikovati-persistence`.
3. **Compute Web Workers** run Martensite's pre-warmed shared-memory worker pool
   for parallel meshing, BVH construction, and BRep modeling.

`oblikovati-viewport` consumes shared renderer types from `martensite-render-core`
and adds `martensite-webgpu` alongside `martensite-vulkan` (preserving the
`cargo_tree.rs` boundary that keeps all `martensite-*` crates inside
`oblikovati-martensite` and `oblikovati-viewport`). `oblikovati-gui`,
`oblikovati-head`, `oblikovati-cli`, and `oblikovati-build` remain native-only.

## Interfaces and invariants

- `oblikovati-web` stays zero-`unsafe` (`unsafe_code = "forbid"`) and imports zero
  `martensite-*` crates directly.
- Every browser document, worker script, and `.wasm` asset is served with
  `Cross-Origin-Opener-Policy: same-origin`,
  `Cross-Origin-Embedder-Policy: require-corp`, and
  `Cross-Origin-Resource-Policy: same-origin` so `SharedArrayBuffer` stays active.
- Native builds remain pinned to stable Rust in `rust-toolchain.toml`. The nightly
  channel required for `-Z build-std=panic_abort,std` is isolated in
  `tools/build/wasm-toolchain.toml` and driven by `cargo xtask wasm-check`,
  `cargo xtask wasm-build`, and `cargo xtask wasm-package`.
- `corliss` (Inventor OLE `.ipt` import) stays gated off `wasm32` until upstream
  aligns `cfb` with `web-time`; native `.opd`/`.obk` YAML packages work fully in
  the browser.
- Production deployment to `app.oblikovati.org` uses an independent GitHub Actions
  workflow (`.github/workflows/web-deploy.yml`) triggered on merges to `rust`. It
  builds the static WebGPU bundle, pushes an immutable Nginx container image to
  GHCR, and rolls out `deploy/web/kubernetes/production.yaml` on the Hetzner K3s
  cluster without running test suites.

## Consequences

- Desktop Linux, Windows, and macOS code paths and Vulkan validation stay intact.
- Native Linux tests exercise `WebSession`, `MemoryDocumentFiles`, DOM input
  translation, and WebGPU offscreen rendering directly, keeping workspace coverage
  above the 80% gate.
- Headless Chromium E2E verification (`web/e2e_chromium.py`) exercises real browser
  WebGPU, cross-origin isolation, multi-threaded compute workers, parametric edits,
  and `.opd` round-trip export/import.
