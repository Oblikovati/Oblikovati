# Measure candidate diagnostic

This is diagnostic evidence, not verified acceptance. The live scenario source
is unversioned, absent from the pinned snapshot, and marked `needs_fixture`.
Its SHA-256 was `a4f61db6185add10d22ac01f80259578cc85a68abb9301454ddf8b7c9f6a23af`
before and after the run. Native build provenance remains open in #3978.

One filtered scenario passed all seven steps under Xvfb with Vulkan Lavapipe.
The fixture creates an arc through the host and opens Measure with a key.
Pointer movement and clicks use X11 XTEST. The test checks midpoint and curve
feedback, selection, clearing, and unchanged geometry. The captures were inspected.

On AMD Radeon 8060S / RADV with a 1440 × 900 drawable, release runs measured
120 bounded frame-loop calls after 20 warmups with a 200-segment sketch:

| Code | p50 | p95 | p99 | Maximum | Over 16.67 ms |
| --- | --- | --- | --- | --- | --- |
| Clean base `a02473e1e` | 42.442 ms | 49.809 ms | 51.897 ms | 52.228 ms | 120/120 |
| Candidate change | 47.844 ms | 50.096 ms | 51.124 ms | 52.961 ms | 120/120 |

The candidate-only Measure query measured p50 4.451 ms and p95 7.603 ms.
These are single diagnostic runs, not a controlled causal estimate. Each frame
uses a bounded loop with injected motion and excludes OS delivery and display
scanout. Raw samples were not retained. The test checks that Measure is open.
The base harness patch and source record the test-only additions. An earlier
invalid run is retained: Escape left sketch mode, so Measure was not active.
These results do not meet the 60 fps budget; see #4032 and #4035; also see Martensite #501.

The unrelated pointer-slot test failed on clean base due to a stale fixture
expectation (#4036). No kernel code was changed in this work.
