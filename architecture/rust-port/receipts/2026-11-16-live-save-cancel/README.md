<!-- SPDX-License-Identifier: GPL-2.0-only -->

# First-save cancellation OS input check

On 2026-11-16, a clean worktree at application commit
`53295a41a87b78da4f70741a88ccd78c2cf7953d` was built with
`CARGO_TARGET_DIR=/tmp/obk-live-save-target cargo build --offline --locked -p oblikovati-head --bin oblikovati`.
The GUI SHA-256 was `160c4e96ca4efdb07d0356cc0014d84a8468a61ab27fb1c218574eff6daea4ad`.
MCPBridge was built from clean commit `597a3d74477b96418c57b2e55f946ddc7129b7de`;
its library SHA-256 was `2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The display was Linux Xvfb, 1900 × 1100, with Vulkan Lavapipe.

XTest clicked New Part at `(103, 120)` and pressed Ctrl+S. The bridge capture
`save1.png` shows Save As with `Part1.opd` selected, a location, file type,
and preview controls. XTest clicked Cancel at `(373, 738)`, pressed Ctrl+S
again, then clicked Cancel again. `save2.png` is byte-identical to `save1.png`;
it shows the same selected name. The two cancellation captures are also
byte-identical. `cancelled2.png` shows the active unsaved `Part1` tab and its
model browser. The bridge made captures only; it did not send model commands.
The input driver, probe, and launch script are retained for inspection. Their
paths and coordinates are specific to this run. The GUI and Xvfb were closed.

This is an OS-input diagnostic for “Open and cancel the first save”, not a
pinned Cucumber receipt or a verified scenario. The run did not independently
prove build inputs (#3978), and it did not execute the scenario runner.
