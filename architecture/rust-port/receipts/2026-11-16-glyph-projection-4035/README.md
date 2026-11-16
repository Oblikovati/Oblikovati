# Constraint glyph projection diagnostic

The test used Rust 1.99.0 in the debug profile, Linux, Xvfb, and Vulkan
Lavapipe (`llvmpipe`, LLVM 20.1.2) at a checked 1440 × 900 drawable. It
compared 201 unchanged points projected individually and in a cached batch.
After 20 warmups, 120 paired samples gave individual projection a median of
3.701 ms and p95 of 4.433 ms. Cached projection had a median of 0.024 ms and
p95 of 0.032 ms. Each pair checked identical pixel coordinates. The compressed
log records the samples.

This probe measures projection only, in a debug build, without a full GUI frame
or input-to-display latency. It does not establish 60 fps or physical-GPU
performance. Changing the camera, surface, points, or sketch plane invalidates
the cached result; invalid batches fall back to individual glyph projection.

The live `constraint-glyph-hover` feature passed one scenario and six steps
under Xvfb with X11 pointer input. Its feature hash was
`5d30dc6745bfc4bf6415c21558d7d6ff3cae1df3b50862583a87ebd4e38f2926`
before and after the run. The screenshot was inspected. The source is not in
the pinned snapshot and requires a fixture, so this is a diagnostic rather
than verified acceptance. Build provenance remains open (#3978).
