# Native circular sketch flyouts

Issue: #3843. The released dependency remains Martensite 0.10.0 at
`59bc4bda93e9b17100017608296bf053e10e5cbf`.

Circle and Center Arc retain their primary ribbon buttons. Their arrow buttons
open menus with the primary and three-point forms. Both new choices use the
existing typed shell action and owner placement pipeline. They do not introduce
another kernel operation, recipe format or preview renderer.

The owner resolves the versioned `SketchConstructionPolicy` in document working
units. This reuses its physical accuracy bound; no new tolerance is introduced.
The accuracy is a metric result check, not a topology classification rule.
Scripts retain their explicit construction options.

## Verification plan

Pointer tests cover both choices, primary buttons, both framebuffer scales,
transaction refusal and context changes. Owner tests cover selection without
history edits and physical policy conversion. Existing circular engine tests
cover invalid input, retry, click/drag equivalence and history.

Native X11 acceptance must exercise both menus, private preview, cancel, invalid
collinear input, creation, undo/redo, save and fresh-process reopen. MCPBridge
provides read-only model checks and screenshots. Inspect the actual PNGs and
close each owned desktop. Run the full local gate, coverage, duplication and
Windows/macOS compile checks before the first push.

This does not complete the general Extrude or cube/named-face journey.

## Native acceptance evidence

The X11 driver uses OS pointer and keyboard events to select both flyouts, cancel
an incomplete circle, refuse three collinear inputs and retry the third point.
The circle and half-circle arc use the same radius. The resulting sketch is
healthy with DOF 8; this test intentionally leaves both curves free. Each curve
has one undo record. Undo/redo preserves the full geometry and reference keys.
Save and a fresh desktop process retain the same authored entities.

MCPBridge performs read-only inspection and PNG capture. Circle preview, refused
input, menus and the committed curves were visually inspected. The generic
placement failure message now identifies sketch placement rather than Slot,
which the refused-circle screenshot exposed. Its cause retains the input values
and expected shape. A regression test fixes that message contract.

The live driver is `/tmp/oblikovati-x11-native-circular-flyouts.py`; its artifacts
are in `/tmp/oblikovati-x11-native-circular-flyouts/`. Both owned desktops closed
normally. Windows x64 GNU and macOS ARM64 host/GUI Clippy checks passed. These are
compile checks, not native runtime evidence for those platforms.

## Final local qualification

The final `make gate` passed all 13 crates, including all 373 head tests on Xvfb.
Per-crate line coverage is 91.96% to 99.60%; GUI coverage is 98.22%. Duplication is
1.01% (912 of 90,134 lines). Markdown, SPDX, formatting and Clippy checks passed.
The final gate log is `/tmp/oblikovati-circular-flyouts-final-gate.log`.

The new positions use logical ribbon pixels and framebuffer scale. No physical
accuracy constant, ABI vector call, wire method or kernel algorithm was added.
The Windows/macOS checks and comment/coherence review were completed before
push. No new worktree or build target directory was created; pending branches
remain intact and incremental build storage remains disabled.
