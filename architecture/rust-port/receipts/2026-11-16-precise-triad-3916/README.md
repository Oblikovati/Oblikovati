<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Precise Input triad diagnostic

The pinned `precise-input-08` diagnostic initially stopped at an unbound
Reposition Triad step. The red report is retained as `red-gap-report.json`;
its later failures are tainted. After binding the corrected wording, the
focused native frame-loop diagnostic passed all steps. The setup uses a YZ
sketch with an isolated line and selects Relative reference mode, which displays
the movable triad. The test checks its owner position and rendered line before
and after reset, the Point prompt, the options menu, toolbar dismissal, Ready,
and unchanged sketch geometry and history. The focused report is
`gap-report.json`.

The full pinned Precise Input diagnostic passed all three portable scenarios;
four capture-only scenarios were excluded. The complete report is
`full-gap-report.json`. The scenario snapshot was
`e76cb749ed43c73a91c24854057da7c890041f51` and was checked before and after.
These are diagnostics, not verified acceptance (#3978).

A separate standalone run used XTest, Xvfb, Vulkan Lavapipe, and an isolated
YZ startup sketch. It drew a line, started Point, opened Precise Input, and
selected Relative mode. `relocated_clear.png` shows the triad at the line
endpoint with the pointer away; `reset_clear.png` shows it at the origin.
`menu_clear.png` shows the three options, and `closed_clear.png` shows Point
still requesting a center after Close. `ready_clear.png` shows Ready after
Escape. MCPBridge reported the same three point IDs and one line ID before
and after. The GUI and Xvfb stopped after the run.

The standalone build used worktree base `c8939c687` with uncommitted test
bindings. Its GUI SHA-256 was
`86e0f1cd3deec0642c47d050c80324f3de7711ce10d18c234dcc1ba4655d8794`;
MCPBridge SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The scripts and coordinates are specific to this run. The OS-input evidence
is supporting observation, not a pinned acceptance receipt.
