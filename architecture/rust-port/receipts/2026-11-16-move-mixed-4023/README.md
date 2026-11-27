<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Mixed coordinate and angle Move diagnostic

Pinned `move-19` and `move-20` run with native frame-loop input against
snapshot `e76cb749ed43c73a91c24854057da7c890041f51`. The fixture creates a
rotated ellipse on YZ. It enters the requested values through the toolbar,
checks the sketch-space displacement and visible screen direction, and checks
Undo and the redo label. The Y/angle setup also closes and reopens Move after
selecting X/angle and checks that mode was retained. The reports in this
folder are diagnostic; they do not establish reusable BDD acceptance (#3978).

A separate standalone run used XTest, Xvfb, Vulkan Lavapipe, and an isolated
configuration with a YZ startup sketch. `original.png` shows the ellipse drawn
by three mouse clicks. In Relative mode, `x_fields.png` shows X `10` and Angle
`45`; `x_moved.png` shows the accepted up-left move. Done and Ctrl+Z restored
the original view. `retained.png` shows X and Angle after reopening Move.
`y_fields.png` shows Y `5` and Angle `45`; `y_moved.png` shows the second
accepted move, and `y_undo.png` shows the result after Done and Ctrl+Z.
MCPBridge then listed the original point and one ellipse. The standalone GUI
and Xvfb stopped after the run.

The standalone build used worktree base `568f2e9fe` with uncommitted test
bindings. Its GUI SHA-256 was
`86e0f1cd3deec0642c47d050c80324f3de7711ce10d18c234dcc1ba4655d8794`;
MCPBridge SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The scripts and screen coordinates are specific to this run. The source
snapshot was verified before and after the focused runs. The OS-input run is
supporting evidence, not a pinned acceptance receipt.

The full Move diagnostic passed 14 of 27 scenarios; 13 stopped at unbound
steps. It ran before the final rendered-curve assertion was added to the two
focused tests. `full-gap-report.json` retains that run. The full gate was not
run. The Go reference at `aa8297d1b` has sketch HUD input but no equivalent
mixed-mode toolbar behavior was found.
