<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Sketch view preference OS-input journey

On 2026-11-16, the standalone GUI was built from the clean application
worktree at `e6030c8a1` using an isolated target and
`cargo build --offline --locked -p oblikovati-head --bin oblikovati`.
The GUI SHA-256 was
`1e3c8bac5d131bfbdb16cb1a6c723fc763ffadc85a7557b34073b5572ddac06a`.
MCPBridge's library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The run used Linux Xvfb at 1900 × 1100, Vulkan Lavapipe, and an isolated
configuration directory. XTest sent the mouse events; MCPBridge captured the
window. An attempted `get_camera` query failed, so camera claims below are
visual observations, not numeric API measurements.

1. Click New Part `(70, 85)` and the ViewCube Home control `(1780, 216)`.
   `home.png` shows the oblique Home orientation.
2. Open Tools `(72, 10)` → Application Options `(122, 34)`. Clear Look at
   sketch plane `(756, 615)`, apply `(783, 637)`, and close the panel `(1149,
   390)`. `preference.png` shows the original checked state.
3. Click Start 2D Sketch `(70, 80)`, select YZ in the browser `(82, 315)`, and
   click its highlighted plane `(1100, 340)`. `yz-selected.png` shows the
   selected plane. `sketch-oblique.png` shows Sketch1 being edited with the
   ViewCube still oblique.
4. Finish Sketch `(1510, 82)`, reopen Application Options, check and apply the
   preference, and close the panel. Right-click Sketch1 `(75, 470)` and choose
   Edit Sketch `(125, 485)`. `sketch-menu.png` shows the menu and
   `sketch-right.png` shows the same Sketch1 in the Sketch environment with
   the ViewCube reading Right. The isolated options file reported
   `lookAtSketchPlane: true` after restoring the setting.

The GUI and Xvfb ended after the run. The retained input driver, probe, and
launch script use paths and coordinates specific to this run. This manual
journey used Part1 and Sketch1, not the Part6 and Sketch2 fixture of
`new-sketch-10`; it is supporting OS-input evidence, not a pinned BDD receipt.
The full scenario passed separately with frame-loop input. Build provenance is
not bounded (#3978); the typed API and Go compatibility remain open (#4010).
