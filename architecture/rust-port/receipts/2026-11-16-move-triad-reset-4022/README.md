<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Move triad reset diagnostic

The pinned `move-14` scenario passed in diagnostic mode using frame-loop
input on snapshot `e76cb749ed43c73a91c24854057da7c890041f51`. The test
created an ellipse, selected it for Move, set Distance and Angle mode, and
repositioned the triad away from the origin through the UI. It checked the
triad's position and rendered line, then reset it and checked the origin,
blank fields, ellipse geometry, committed history and visible ellipse. Escape
closed Move without a placement. The snapshot inventory was verified before
and after the run. The report is `gap-report.json`.

A separate standalone run used XTest, Xvfb and MCPBridge with an isolated
configuration and YZ startup sketch. It created an ellipse with three mouse
clicks, opened Move and enabled Precise Input. An initial click missed the
reposition control after the toolbar resized; the following canvas clicks
moved the ellipse. The test closed Move, used Undo, and selected the original
ellipse again before the recorded reset sequence.

`nonzero.png` shows the triad at `(1120, 650)` in Relative mode. The Reset to
Origin control returned it to the sketch axes in `reset.png`; the Distance
and Angle fields were blank. `cancelled.png` shows the retained ellipse and
Ready after Escape. MCPBridge listed the same point and ellipse IDs immediately
before reset and after Escape. The GUI and Xvfb stopped after the run.

The standalone build came from worktree base `847f80eac` with uncommitted test
bindings. Its GUI SHA-256 was
`86e0f1cd3deec0642c47d050c80324f3de7711ce10d18c234dcc1ba4655d8794`;
MCPBridge SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The scripts and coordinates are specific to this run. This is diagnostic and
supporting OS-input evidence, not verified acceptance: native build provenance
remains open in #3978.

The full pinned Move diagnostic passed 12 of 27 scenarios. The other 15
stopped at unbound steps; subsequent failures after those steps are tainted.
The complete diagnostic is `full-gap-report.json`. No full gate was run.
