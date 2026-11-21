<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Part sketch-view preference OS-input check

On 2026-11-16, a desktop built from a worktree based on `8103ddf22` ran on
Linux Xvfb at 1900 × 1100 with Vulkan Lavapipe. The worktree had the uncommitted
implementation now in `21cf862ac`; its tracked Git tree was
`e343194ff6be1d6734aed7034abd8a59348bfdfd`. The GUI SHA-256 was
`1e3c8bac5d131bfbdb16cb1a6c723fc763ffadc85a7557b34073b5572ddac06a`.
MCPBridge's library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The configuration directory was temporary and isolated from user settings.

XTest clicked Tools at `(72, 10)`, Application Options at `(122, 34)`, and
Look at sketch plane at `(756, 615)`. `options.png` shows the initial checked
setting and `staged.png` shows the unchecked draft. Clicking Apply Part at
`(783, 637)` wrote `part.lookAtSketchPlane: false` in the isolated options
file. The bridge captured `applied.png`. Clicking the checkbox again and Revert
Part at `(876, 637)` left that stored value false; `reverted.png` captures the
result. The bridge was used only for window captures. The input driver, probe,
and launch script are retained; their paths and coordinates are specific to
this run. The GUI and Xvfb processes were closed after the check.

The GUI unit test and the native frame-loop test also passed. The latter checks
staging, revert, apply, and reload. Strict GUI and head Clippy, formatting,
and documentation checks passed. This is an OS-input diagnostic, not a BDD
scenario receipt: it does not exercise sketch entry or establish bounded build
provenance (#3978). The typed API and Go compatibility remain open (#4010).
