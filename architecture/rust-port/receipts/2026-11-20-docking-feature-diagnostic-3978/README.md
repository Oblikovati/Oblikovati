# Complete docking feature diagnostic

On 20 November 2026, the linked diagnostic built the runner and desktop
from application commit `877eaf1342f13a2dcd65c57c6f48a4e0ca9c7a83`. It pinned
both executables for one isolated run against local scenario snapshot
`028492efd6fad2a4fb740048e4810c7c6e29234a`. All 10 docking scenarios and all
96 steps passed. The feature report checker validated every scenario, final
screenshot, and JUnit case. The runner and desktop build and execution hashes
matched. The live feature and native fixture still matched the snapshot after
the run.

The run used an offline vendor tree, Rust 1.99.0, Xvfb at 1440 × 900, Openbox,
Vulkan Lavapipe, and the explicit private `/proc` diagnostic option. It checked
351 registry packages against cached archives; 22 Git packages were not
verified. The system and runtime were staged from local files, using an earlier
local runtime inventory. The compressed observation records the measured inputs
and report check.

This result is not reusable acceptance evidence. The snapshot is local, not an
upstream revision. System and toolchain origin, kernel, devices, concurrent host
writes, and complete Git dependency provenance remain unbounded. No trusted
builder attestation was produced. Issue #3978 remains open; this run does not
change the verified scenario count.
