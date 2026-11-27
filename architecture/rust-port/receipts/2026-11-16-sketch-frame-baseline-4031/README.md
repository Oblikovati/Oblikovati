# Sketch presentation measurement evidence

Base: Oblikovati Rust `5b82105d67631c027e6e7ed607c881b58a3a66ef`; pinned Martensite `bef25f47548431c8f128f08f66a1f719bcee4982`. The build uses Rust 1.99.0 release mode. `summary.json` contains environment, baseline summaries, limits and file hashes.

The first two baseline runs emitted summary statistics only. Their individual frame samples were not recorded. `phase-profile.log.gz` has raw per-frame timings from a subsequent, instrumented run. The final patch is a later instrumentation revision, so it is not an exact reconstruction of the earlier runs.

`present-exploratory.log.gz` was run before drawable extent was checked. Its requested 3840 × 2560 data must not be used as 4K results: the subsequent check in `present-extent-check.log.gz` found 2559 × 1545. The latter log also contains heavily loaded 1440 × 900 data and a deliberate assertion failure at the clamped size.

`xvfb-software-loaded.log.gz` uses software Vulkan under substantial concurrent CPU load. It is exploratory and cannot establish physical-GPU performance or a controlled effect of resolution. `xvfb-radv-failed.log.gz` records the attempted hardware run on Xvfb, which lacks DRI3 support for that target. Still frames were forced by the test; an ordinary idle application waits rather than presenting continuously.

The timer measures the owner frame from event polling to the return from present, with injected pointer events. It excludes OS input delivery and display scanout. Phase timers add instrumentation overhead and GUI time overlaps the input and GUI phase measurements.

To run the final harness in a disposable clean worktree at the base commit:

```sh
gzip -dc architecture/rust-port/receipts/2026-11-16-sketch-frame-baseline-4031/harness.patch.gz | git apply
CARGO_TARGET_DIR=/tmp/obk-perf-4031-target cargo test -p oblikovati-head --release --lib measure_pointer_sketch_frame_latency -- --nocapture --test-threads=1 > /tmp/obk-perf-4031-present-verified.log 2>&1
```

The Xwayland command asserts the requested drawable size and fails if the window manager clamps it. The 4K software test was run with:

```sh
xvfb-run -a -s '-screen 0 4096x3000x24 -noreset' env VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json CARGO_TARGET_DIR=/tmp/obk-perf-4031-target cargo test -p oblikovati-head --release --lib measure_pointer_sketch_frame_latency -- --nocapture --test-threads=1 > /tmp/obk-perf-4031-xvfb-software.log 2>&1
```

The attempted RADV Xvfb command was identical except for `VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/radeon_icd.json` and output `/tmp/obk-perf-4031-xvfb-radv.log`.

The phase-profile command used the same `CARGO_TARGET_DIR` and `cargo test` arguments, before viewport split timing and extent/motion controls were added, with output redirected to `/tmp/obk-perf-4031-phases.log`.
