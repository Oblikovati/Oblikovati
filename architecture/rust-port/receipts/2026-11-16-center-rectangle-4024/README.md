<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Centered rectangle diagnostic

The initial pinned diagnostic passed `rectangle-center-08` and stopped five
other early scenarios at unbound wording. `red-gap-report.json` retains that
run; later failures in those cases are tainted. After binding the current
wording and checking shared corners and both construction diagonal midpoints,
the group diagnostic passed `rectangle-center-01`, `-02`, `-06`, `-07`,
`-08`, and `-09`. The group report is `group-gap-report.json`. A later focused
`-01` run passed after the topology helper was split; see
`final-gap-report.json`. The scenario snapshot was
`e76cb749ed43c73a91c24854057da7c890041f51`, verified before and after.
These frame-loop runs are diagnostics, not verified acceptance (#3978).
The focused `-09` run also checked that the four actual side lengths are
4, 4, 8 and 8 mm; see `numeric-gap-report.json`.

The Go reference at `go-legacy` uses a center and corner and creates four
sides and two construction diagonals. Its tool deactivates after placement;
the pinned scenario instead requires a prompt for another center.

A separate standalone run used XTest, Xvfb, Vulkan Lavapipe and an isolated
YZ startup sketch. A mouse-created centered rectangle appears in `created.png`.
MCPBridge reported four ordinary lines and two construction lines. Escape and
Ctrl+Z left only the projected origin (`undo.png`); Ctrl+Y restored all six
line IDs (`redo.png`). A second run created a mouse rectangle, then entered
`8`, Tab, `4`, Enter for another centered rectangle. `typed.png`, `numeric.png`
and `numeric_ready.png` show the input and result. MCPBridge reported eight
ordinary lines and four construction lines. Both standalone windows and Xvfb
instances stopped after the runs.

The standalone build used worktree base `5ef342a86` with uncommitted test
bindings. Its GUI SHA-256 was
`86e0f1cd3deec0642c47d050c80324f3de7711ce10d18c234dcc1ba4655d8794`;
MCPBridge SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The scripts and screen coordinates are specific to this run. The OS-input
observations support the scenarios but are not pinned acceptance receipts.

The full feature diagnostic passed eight of 15 scenarios; seven stopped at
unbound steps. It ran before the final side-length assertion was added to the
focused `-09` test. The report is `full-gap-report.json`. The full gate was
not run.
