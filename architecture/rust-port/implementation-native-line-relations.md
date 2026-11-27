# Native two-line constraint tools

Tracking: [#3834](https://github.com/Oblikovati/Oblikovati/issues/3834).

The Sketch ribbon exposes Parallel and Perpendicular. Both require two distinct
line picks and call the existing canonical typed constraint client. The session
owns picks and highlights; there is no second operand store or solve path. The
first pick does not change geometry or history. Duplicate picks remain inert.
Escape and replacement clear transient selection. Successful creation clears
picks; a solver refusal keeps the tool and its operands available for correction.

The shared analytic picking path applies the active entity filter before
returning a priority point. This removes a bypass that let line-only tools retain
isolated point picks. Existing point-capable tools keep their point priority.

The Go reference is `app/sketch_constraint_tools.go` and
`app/sketch_constraints.go`: Parallel and Perpendicular consume the first two
selected lines. The model remains responsible for convergence, dimensional
consistency, history and persistence. No public API extension is required.

Owner regressions cover line-only selection, duplicate picks, cancellation,
replacement, one-step undo/redo, stored constraint kind, normalized solved line
directions and refusal against fixed geometry without adoption. Real ImGui
click tests exercise both ribbon commands at normal and double display scale.
The final full local gate passed all 13 crates, including 358 head tests. Line
coverage ranges from 91.96% to 99.60%; duplicated lines are 1.00%. Windows x64
GNU and macOS ARM64 host/GUI Clippy checks passed. These are compile checks,
not native runtime claims for those platforms.

Native X11 mouse/keyboard acceptance created two connected lines and applied
each tool. Both changed DOF from six to five. Canonical constraint reads checked
the kind and source IDs; normalized line-direction residuals checked the solved
geometry. First-pick privacy, duplicate picks, Escape, undo and redo passed.
Undo restored the original coordinates and entity IDs. The Perpendicular result
was saved and reopened in a fresh process with the same geometry, IDs, constraint
and DOF five. MCPBridge supplied read-only inspections and captures. Parallel,
Perpendicular and reopened captures were inspected. Both final acceptance
desktops closed normally, and no application or Xvfb process remained.

This adds two supported sketch commands. It does not complete general Extrude
or the cube and named-face journey, which still requires the tracked kernel
analytic-region boundary.
