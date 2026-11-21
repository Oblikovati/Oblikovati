# Docking restart: linked build and execution diagnostic

On 20 November 2026, the diagnostic built the native test runner and desktop
from application commit `f63007edfc97c527d1bb2812fa36ba127766463b`. It pinned
both built executables for one Bubblewrap execution against local scenario
snapshot `028492efd6fad2a4fb740048e4810c7c6e29234a`. The selected docking
restart scenario passed all four steps. The report checker passed in the same
process after the run. The runner and desktop hashes in the build and execution
observations match.

The run used an offline vendor tree, Rust 1.99.0, Xvfb at 1440 × 900, Openbox,
Vulkan Lavapipe, and the explicit private `/proc` diagnostic option. It checked
351 registry packages against cached archives. It did not verify 22 Git packages.
The staged system and runtime were assembled from local files; the runtime
preparation used an earlier local runtime inventory. The compressed observation
records the measured inputs and the report check.

This result is not reusable acceptance evidence. The local scenario snapshot
is not an upstream revision. System and toolchain origin, the kernel, devices,
concurrent host writes, and complete Git dependency provenance remain unbounded.
The private `/proc` mount exposes kernel inputs. No trusted builder attestation
was produced. Issue #3978 remains open, and this diagnostic does not change the
verified scenario count.
