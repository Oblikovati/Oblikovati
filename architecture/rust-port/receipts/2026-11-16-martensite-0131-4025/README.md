<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Martensite v0.13.1 host checks

On 2026-11-16, an isolated Rust worktree based on `7de7c6988` used released
Martensite v0.13.1 commit `bef25f47548431c8f128f08f66a1f719bcee4982`. The
host's exact `rustix` dependencies were raised to 1.1.6 to satisfy the release.

- The adapter library passed 91 tests, the release probe passed four, and the
  sketch-prism suite passed 79, including parameter and named-face replay.
- The native frame-loop cube journey passed under Xvfb and Vulkan Lavapipe. It
  creates and edits the parameter-driven cube, uses Undo and Redo, saves and
  reopens it, and resolves its saved face keys.
- The profile-region suite passed eight tests and failed the curved per-face
  OCCT comparison. The same values were reported for the prior release in
  [Martensite #407](https://github.com/Oblikovati/Martensite/issues/407).
- The dependency test group passed 12 tests and failed the existing API pin
  check (#3965). Focused kernel pin and renderer pin tests passed separately.
- Strict Clippy for the adapter, viewport, head and build tool, and workspace
  formatting passed. Full gate and OS-level input were not checked here.

The logs retain the commands' results. The native run is supporting evidence,
not verified BDD acceptance: native build provenance remains open (#3978).
