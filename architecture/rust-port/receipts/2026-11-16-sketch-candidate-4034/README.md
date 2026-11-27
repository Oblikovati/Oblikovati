# Sketch tool candidate diagnostic

This is diagnostic evidence, not verified acceptance. The live feature source
was unversioned, absent from the pinned snapshot, and marked `needs_fixture`
and `needs_capture`. The feature SHA-256 was
`f5e24ccd8a393e2ce319b45405b40bc78a1304f3b74b7df99652af5493ef765e`
before and after the run. Native build provenance remains open in #3978.

One filtered scenario passed all five steps under Xvfb and Vulkan Lavapipe.
The fixture uses a host command; pointer movement uses X11 XTEST. It checks a
circle against the active theme candidate color and confirms no selection.
The screenshot was inspected. Separate tests cover panel and pointer exit,
cancellation, tool filters, and the earlier glyph and selection scenarios.

The release query test on 200 segments measured p50 6.440 ms and p95 7.295 ms
before index reuse and projection batching. A later run measured p50 2.303 ms
and p95 2.920 ms. These are single runs, not a controlled causal estimate;
they time candidate resolution only. The intermediate indexed run is retained.

On AMD Radeon 8060S / RADV, with an observed 1440 × 900 drawable, the same
release test measured full bounded frame-loop calls with injected mouse motion:

| Code | p50 | p95 | p99 | Maximum | Over 16.67 ms |
| --- | --- | --- | --- | --- | --- |
| Clean base `cd4aff6fb` | 37.925 ms | 43.202 ms | 45.745 ms | 47.532 ms | 120/120 |
| Candidate change | 42.074 ms | 45.680 ms | 46.677 ms | 48.604 ms | 120/120 |

Each test had 20 warmups and 120 measured samples. It creates one bounded frame
loop per input, and excludes OS input delivery and display scanout. Raw samples
were not retained. The base harness patch records its test-only additions. The
candidate timing was repeated after the window-method move and before the final
behavior-preserving witness-selection extraction. The results show that 60 fps is not established; they are not a continuous-loop or
input-to-photon measurement. Further work: #4032, #4035 and Martensite #501.

The full planar-pick suite has a rotated-ellipse failure. The same case fails
on clean base `cd4aff6fb` with the same result; it is Martensite #406. No kernel
code was changed in this work.
