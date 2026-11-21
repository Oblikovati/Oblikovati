# Native sketch redraw and capture timing

Draft branch base `44725c062`; the worktree contains uncommitted changes. The
test source SHA-256 is `9a037a0be71f9202c3315ee35c70b5c00d73cbbd47eb6a15d87b61d18bc570a7`.
The lockfile SHA-256 is `b0695ff8c4df2279345908e21ebd90dbeadcaf4f2ab2dbda3d55c471daf245d9`.
The run used Rust 1.99.0 release mode, Linux 7.2.15, and AMD Radeon 8060S
RADV at a verified 1440 × 900 drawable. Vulkan validation was disabled.

The test shows a static 200-line sketch with no GUI packet or input events.
After 20 warmups it measured 120 paired redraw and host capture calls. The
redraw median was 25.509 ms and p95 was 27.331 ms. The capture median was
0.478 ms and p95 was 0.624 ms. Their paired total median was 25.963 ms, p95
27.920 ms, and maximum 30.620 ms. All 120 totals exceeded 16.67 ms.

Redraw includes the kernel's GPU completion wait and color/depth readback;
the later host capture clones the retained frame. These timings do not isolate
the GPU wait from readback, or measure the full application, input delivery,
display scanout, or user-perceived latency. They do not prove 60 fps. The raw
samples are in `samples.log`; the relevant requests remain Martensite #501
and #504 and Oblikovati #4032.

```sh
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/radeon_icd.json \
  cargo test -p oblikovati-viewport --release --lib \
  measure_native_sketch_redraw_and_capture --locked --offline -- \
  --ignored --nocapture --test-threads=1
```
