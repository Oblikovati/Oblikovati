<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Live Move scenario diagnostics

On 2026-11-17, the existing debug head test binary
`oblikovati_head-0ac4c9be877773e3` ran two filtered scenarios against the live,
unversioned MCAD-Scenarios source. The copied feature has SHA-256
`42fe188a2145d98a47e57b310df3e4ba7cd80f071d54cd1df9ada567c2b94f1b`.
The runner reported no source change during either run. The binary was not
rebuilt for this diagnostic, so its exact build inputs and application revision
are not established. The window used the AMD Radeon 8060S desktop at 1440 × 900.

- `move-17` passed one scenario and nine steps with frame-loop input. The
  generated ellipse fixture is not the recorded source fixture. The final
  capture shows the ellipse and Ready state after Escape; it does not prove the
  intermediate menu or toolbar appearance against the reference.
- `move-18` stopped at its second step because the recorded two-line fixture
  step has no binding. Later steps were not run. This is not evidence that the
  coupled displacement works or fails.

These are diagnostics, not verified scenario acceptance. The source is not the
pinned snapshot, the recorded fixture is not established, and runner build
provenance remains open in #3978. The test processes exited after their runs.
