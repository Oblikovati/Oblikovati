# Ready hover frame phases

Base `6940d4e8531bb09e11f7fcd751653bcae91ccc21`, Martensite
`bef25f47548431c8f128f08f66a1f719bcee4982`, Rust 1.99 release,
AMD Radeon 8060S RADV, Xwayland, 1440 × 900. The test uses a 200-segment
sketch and scripted pointer motion; it excludes OS input delivery and display
scanout. The temporary instrumentation patch and raw log are included.

In the last 120 samples, median input was 2.282 ms, other owner work 0.629 ms,
and presentation 37.274 ms. The frame median was 40.821 ms and p95 was
45.136 ms. All 120 frames exceeded 16.67 ms. Phase timers and logging affect
execution; these values do not isolate GPU wait, redraw, or capture. This run
is not a proof of 60 fps or a controlled comparison with earlier runs.

An independent X11 probe of 10,000 `XStoreName` and `XFlush` pairs had a
median of 1.352 µs and p95 of 1.503 µs. This does not measure window-manager
work or prove that all title-update effects are negligible.
