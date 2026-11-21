<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Circle same-position OS input check

On 2026-11-16, a desktop built from application commit
`961d4be87205f4d382c6f1edbfe9605a49ee15ac` was tested on Linux Xvfb at
1900 × 1100 with Vulkan Lavapipe. The source worktree had an unrelated
modified README. The GUI binary SHA-256 was
`fe46585ca50593be631611e2416ca671eeb44be1c8e689cd8a83d403ad0c8616`.
MCPBridge was built from clean commit `597a3d74477b96418c57b2e55f946ddc7129b7de`;
its library SHA-256 was
`f0e14e9be2d0c7e3e98e7f3cb6ec9d34dc19148eec7476e1a632f64c15df65d8`.

XTest mouse input created a Part, started a sketch on XY, chose Circle, and
clicked the same clear viewport position for the center and circumference.
The MCPBridge window capture `same-point.png` shows `0.000 mm` and the status
`Select point on circle`, with no error dialog. The read-only entity query
returned the same single origin point before and during the attempt. XTest
then pressed Escape. `after-escape.png` shows `Ready`, and the final entity
query still returned the same single point. The retained text files record the
three query results. No circle was added.

The retained input driver used these coordinates after resizing the window:
`click 103 120` (New Part), `click 105 120` (Start 2D Sketch), `click 130 520`
(select XY in the browser), `click 1110 360` (accept the XY plane),
`click 280 100` (Circle), `click 1250 650` twice, then `escape`.
The bridge probe called `list_sketch_entities` with `sketchIndex` 0 at the
three checkpoints. The input driver and bridge probe are included for
inspection; their local paths and coordinates are specific to this run. The GUI and Xvfb ended after
the test. This is an OS-input diagnostic for the behavior in
`circle-center-point-09`, not a pinned Cucumber receipt or a verified scenario:
the build provenance and receipt chain remain open in #3978.
