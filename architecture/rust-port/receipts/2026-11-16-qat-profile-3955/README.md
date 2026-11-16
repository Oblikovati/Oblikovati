<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Quick access profile diagnostic

On 2026-11-16, the corrected “Inspect and dismisses quick access command
choices” scenario was run against snapshot
`e76cb749ed43c73a91c24854057da7c890041f51`. The snapshot verified before and
after the run. The scenario is bound but fails at the first missing command,
`Refine Appearance`. Later failures in the report are tainted by that failure.

The application was based on `a70b3b3e1` with this binding change uncommitted
and the unrelated `README.md` modification present. The runner used frame-loop
input and Vulkan Lavapipe under Xvfb. This is a diagnostic, not verified BDD
acceptance; native build provenance remains open (#3978).
