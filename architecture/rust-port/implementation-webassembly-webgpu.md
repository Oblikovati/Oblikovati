# WebAssembly and WebGPU platform target evidence

Status: Verified on native Linux WebGPU (Mesa lavapipe) and headless Chromium
`148.0.7778.96` with real WebGPU and cross-origin isolated WebAssembly Threads.
Design record: [ADR-0043](../decisions/ADR-0043-webassembly-webgpu-target.md).

## Scope delivered

- **Core crate `wasm32 +atomics` readiness:** Enabled `getrandom/wasm_js` in
  `oblikovati-model`, gated native-only `corliss` off `wasm32` in
  `oblikovati-persistence`, added `MemoryDocumentFiles` for atomic in-memory
  `.opd` storage, and fixed `wasm32-unknown-unknown` absolute-path validation in
  `oblikovati-session`.
- **Viewport WebGPU and browser target (`oblikovati-viewport`):** Switched shared
  rendering types to `martensite-render-core`, added `martensite-webgpu` alongside
  `martensite-vulkan`, added portable W3C DOM pointer/wheel/keyboard/resize
  translation (`web_input.rs`), `WebClipboard`, `WebWindow`, `ViewportWake` on
  `wasm32`, `ViewportRuntime::new_webgpu_async` / `open_offscreen_canvas` /
  `open_html_canvas`, and `Viewport::present_frame` / `present_async` /
  `capture_async`.
- **Full desktop UI on WebAssembly (`oblikovati-gui`, `oblikovati-head`, `crates/web`, `web/`):**
  Cross-compiled `imgui-sys` and `oblikovati-gui` C++ widget ABI with WASI SDK 34.0
  (`wasm32-wasip1-threads` + `-matomics`) plus a Rust-allocator / WASI-free C shim
  (`wasm_c_abi.rs`, `wasm_c_shim.c`), enabled `NativeHeadWindow` + `DesktopOwner`
  (`WebDesktopHost`) on `wasm32`, wired `WebCadRuntime` to `WebDesktopHost`, and
  replaced the custom HTML chrome with a pure full-viewport `<canvas>` host (`web/`).
- **Build tooling and CI (`tools/build/`, `.github/workflows/`):** Added
  `tools/build/wasm-toolchain.toml` (`nightly-2026-09-25`), `cargo xtask wasm-check`,
  `cargo xtask wasm-build`, `cargo xtask wasm-package`, `Makefile` targets, and a
  `wasm` check job in `.github/workflows/ci.yml`.
- **Independent deployment (`deploy/web/`, `.github/workflows/web-deploy.yml`):**
  Added a lightweight test-free build-and-deploy workflow triggered on merges into
  `rust` (and manual dispatch) that publishes `ghcr.io/oblikovati/app.oblikovati.org`
  with Nginx COOP/COEP/CORP headers and rolls out `deploy/web/kubernetes/production.yaml`
  for `https://app.oblikovati.org` on the Hetzner K3s cluster, plus least-privilege
  RBAC in `deploy/web/kubernetes/deployer-rbac.yaml`.

## Verification commands and results

1. `cargo xtask wasm-check`: checked all 14 WASM-eligible workspace crates
   (including `oblikovati-gui`, `oblikovati-cli`, `oblikovati-head`, and
   `oblikovati-web`) on `wasm32-unknown-unknown` with
   `+atomics,+bulk-memory,+mutable-globals`.
2. `cargo xtask wasm-build`: compiled `oblikovati-web` (bundling the full
   `oblikovati-gui` + `oblikovati-head` desktop UI) in release mode and
   generated `web/pkg/` with `wasm-bindgen 0.2.130` (`75,030,517` bytes raw,
   `22,985,226` bytes gzipped).
3. `python3 web/e2e_chromium.py` (Chromium `148.0.7778.96`, real WebGPU):
   - `crossOriginIsolated: true`, `overlayHidden: true`
   - `deviceName: "WebGPU Core Device"`, `activeWorkers: 8`
   - `homeNonBlackPixels: 921589`, `homeDistinctColorBuckets: 256`
   - Clicking `New Part` opened the live Part workspace (`partOpenDocuments: 1`,
     3D Model ribbon, Model Browser, 3D WebGPU viewport + ViewCube, Material/
     Appearance browsers)
   - `validationErrors: 0`, `consoleLogs: []`
4. Native WebGPU, coverage, duplication, and workspace checks:
   - `oblikovati-persistence`, `oblikovati-session`, `oblikovati-viewport`,
     `oblikovati-web`, and `oblikovati-build` (118 tests) all passed.
   - `cargo llvm-cov --locked -p oblikovati-web --all-targets --summary-only`:
     `89.59%` line coverage (exceeds `80%` gate).
   - `jscpd --config .jscpd.json` on all new Rust modules: `0.74%` duplication
     (below `2.99%` threshold).
   - Workspace clippy (`-D warnings`), `cargo fmt --check`, `cargo xtask docs`,
     `markdownlint-cli2`, and `actionlint` all passed.
