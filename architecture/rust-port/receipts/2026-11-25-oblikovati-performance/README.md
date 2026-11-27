# Oblikovati desktop performance changes

The changes are on the application side of the Martensite v0.14.1 boundary.

- GUI capture remaps ImGui indices in linear time with a reusable dense
  table. It builds the output index list directly and preserves first-use
  vertex order, per-draw offsets, clipping, and framebuffer scaling. The
  viewport compiler reuses a lookup table for 8-bit sRGB conversion.
- Idle model hover batches consecutive unseen positions inside the canvas.
  Camera drags still apply every delta in order while building the GUI
  from the last position in the batch. Button, focus, and modifier changes
  remain input boundaries. Owner turns without input skip the separate
  pre-input GUI pass.
- ViewCube retains the 350 ms transition and waits for a presentation before
  advancing again. Its timer now allows up to 240 camera samples per second.
- Linux release mode uses opt-in Vulkan validation. Linux debug builds
  enable it by default, and `OBLIKOVATI_VULKAN_VALIDATION=1` requests it in
  either profile. `OBLIKOVATI_VSYNC=0` requests unsynchronized X11 Vulkan
  presentation through the existing Martensite driver option.
- X11 presentation defers reading pixels when no explicit screenshot or tab
  preview needs them. A later capture still targets the successfully
  presented frame, and resize invalidation follows the usual viewport path.

## Workload and verification

The optimized 1440×900 viewport opened
`docking-visible-solid-current.opd` on Radeon 8060S with RADV Mesa 26.1.6.
The baseline used the optimized executable tested in the
[post-DPI receipt](../2026-11-25-x11-post-dpi-performance/README.md), including
its test-only filesystem module visibility change. The paired build uses the
same pinned Martensite revision and the same local test-only change.

Both executables ran in a private rootful XWayland session. The Vulkan probe
forced the baseline swapchain to immediate mode. The optimized executable
requested immediate mode through `OBLIKOVATI_VSYNC=0` and the probe only
observed the requested and actual modes. The monitor refresh was 59.984 Hz.
Throughput measures the application's successful present calls.

| Workload | Baseline FPS / p95 | Optimized with validation FPS / p95 | Optimized release FPS / p95 |
| --- | ---: | ---: | ---: |
| Continuous redraw | 58.2 / 30.81 ms | 63.9 / 20.39 ms | 93.2 / 15.66 ms |
| Orbit drag | 30.1 / 44.85 ms | 55.0 / 24.51 ms | 64.5 / 23.27 ms |
| Face hover and selection | 18.0 / 109.83 ms | 47.5 / 30.30 ms | 56.7 / 26.82 ms |
| Animated ViewCube turns | 45.8 / 35.45 ms | 43.7 / 42.08 ms | 84.7 / 17.01 ms |

The 150% trial group ran while other Rust builds were active and shows
run-to-run variability. A lower-load run of the first optimized build
reached 127.0 FPS (9.23 ms p95) in continuous redraw. The final optimized
build measured 94.9 FPS (14.44 ms p95) at 100% during the paired series.
The 120 FPS target remains unverified for orbit, selection, and camera
animation. [Frame traces](runs/) · [Selected face](selected-face.png) ·
[Animation setting](animation-enabled.png)

Run the [shared probe](../2026-11-25-martensite-v0141-performance/probe/)
with `--frames 600` for continuous redraw, explicit `Xft.dpi` of 96 or 144,
and a fresh output directory per trial. For the optimized binary add
`--observe-present-mode` and launch with `OBLIKOVATI_VSYNC=0`. Set
`OBLIKOVATI_VULKAN_VALIDATION=1` for the paired validation runs. The
summarizer trims 200 successful presents in continuous runs and reports
intervals within the input window or individual animation turn.

## Martensite work

The renderer report specifies the requested capture API and the remaining
instrumentation at the native frame boundary. [Martensite performance follow-up](martensite.md)
