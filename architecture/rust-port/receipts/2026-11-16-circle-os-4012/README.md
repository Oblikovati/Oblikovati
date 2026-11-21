<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Zero-radius circle OS-input check

On 2026-11-16, the standalone GUI was built from the clean worktree at
`dfdcf2505ecd0392dc9d39150cc7ecad12051813` with
`cargo build --offline --locked -p oblikovati-head --bin oblikovati`, an
isolated target, `CARGO_PROFILE_DEV_DEBUG=0`, and `CARGO_INCREMENTAL=0`.
The GUI SHA-256 was
`86e0f1cd3deec0642c47d050c80324f3de7711ce10d18c234dcc1ba4655d8794`;
the MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.

The run used Xvfb at 1900 × 1100, Vulkan Lavapipe, and an isolated
configuration with `startupSketchPlane: xy`. XTest sent mouse and keyboard
events; MCPBridge captured the window and queried the sketch. After clicking
New Part `(70, 85)`, `ready.png` showed the Sketch ribbon. The sketch entity
query reported `1 entity(s): #0 point id=1`, the projected origin.

Clicking Circle `(177, 68)` and then `(1100, 500)` twice produced `zero.png`:
the status reads `Select point on circle`, and the input field reads
`0.000 mm`. No error window or circle was visible. The entity query still
reported only the origin point. Escape produced `escaped.png`, with `Ready`
in the status bar; the entity query again reported only the origin point.

The GUI and Xvfb stopped after the run. The retained scripts use paths and
coordinates specific to this run. This is supporting OS-input evidence for
`circle-center-point-09`, not a pinned BDD acceptance receipt. The native
runner build provenance and receipt policy remain open in #3978.
