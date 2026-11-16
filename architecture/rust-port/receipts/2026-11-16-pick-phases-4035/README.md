# Sketch pick phase diagnostics

Base: `rust-work` at `9bca248ad`, Martensite `bef25f47548431c8f128f08f66a1f719bcee4982`, Rust 1.99.0 release build. The machine was an AMD Ryzen AI MAX+ 395. Other builds were active and load varied. Each test used 20 warmups and 120 measured samples. The compressed logs retain individual samples. These tests do not measure input delivery, a complete GUI frame, or display scanout. Projection timing includes result allocation and excludes its later drop.

For 200 lines, analytic catalogue construction was 2.285 µs median and curve candidate search was 368.071 µs median. For 1,000 lines the corresponding values were 10.930 µs and 1.853 ms. The query uses a model-only line chain and excludes screen projection.

The separate checked projection test used Lavapipe and Xvfb. Both the native window and offscreen target reported 1440 × 900. For 201 points, native projection was 1.548 ms median and 2.159 ms p95; offscreen was 0.091 ms median and 0.101 ms p95. For 1,001 points, native was 6.375 ms median and 8.068 ms p95; offscreen was 0.558 ms median and 0.628 ms p95. Native conversion obtains the X11 window size for each point. These measurements are consistent with overhead in that path; they do not isolate its cause.

Martensite #505 tracks a checked batch projection API. Martensite #504 tracks efficient checked color previews. Host tracking is #4035 and #4032. Neither this diagnostic nor those issues establish the 60 fps target.

Run the analytic phase test with:

```sh
cargo test -p oblikovati-martensite --test planar_pick --release --locked --offline measure_analytic_pick_phases -- --ignored --nocapture
```

Run the projection test with:

```sh
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json xvfb-run -a -s '-screen 0 1440x900x24' cargo test -p oblikovati-viewport --test native --release --locked --offline measure_sketch_projection_batches -- --ignored --nocapture
```

`scenario-sync.json` records a separate live feature-file comparison. It is not an acceptance run or a new scenario pin.
