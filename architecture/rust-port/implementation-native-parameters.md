# Native parameter authoring

The native GUI sends canonical `ParameterSetArgs` requests through the typed API
client and the existing document owner. The Parameters panel holds a local draft;
Add and Apply commit through the owner. Cancel closes the panel. Selecting another
document clears the draft. Owner snapshots supply the parameter rows after edits
and history changes.

The native test
`native_new_part_button_creates_and_selects_an_unsaved_document` uses pointer and
keyboard events through the native frame loop. It creates a Part, adds
`size = 10 mm`, starts an XY sketch, and draws an anchored rectangle. Both driving
dimensions use `size`, and the test checks zero DOF. It exits the sketch, edits
`size` to `20 mm`, then verifies recomputation, undo, and redo. The application
window is closed at the end. These are application input events; the test does
not use external operating-system mouse automation.

A separate Linux acceptance run drove the standalone desktop through XTest mouse
and keyboard events. It used no API setup. The run created the Part and parameter,
drew the rectangle, changed `size`, and exercised Undo and Redo. Native-window
captures showed `size`, `d0`, and `d1` at `20 mm` and the enlarged rectangle after
Redo settled. The driver read the client window's screen position before sending
input. The isolated Xvfb process was closed after the run. This supplements the
model assertions in the frame-loop test; it does not cover Extrude or file dialogs.

Placement fields preview and commit through the same validated sketch command.
Point-coincidence inference uses the existing sketch preferences and point order.
An inferred anchor to the existing origin removes the rectangle's translation
degrees of freedom. Slot tests verify their inferred corner constraint and retain
the tool's stage prompt alongside the dimensional fields.

The full local `make gate` passed at `40e4d4a8` for all 13 application/build crates.
It included native Xvfb tests with software Vulkan and validation layers 1.4.313,
formatting, Clippy, build, tests, coverage, doc tests, Markdown/link/SPDX checks,
workflow lint, and duplication. Every crate exceeded 80% line coverage; GUI reached
98.71%, head 95.40%, and duplication was 0.79%. A separate native-window capture
run passed, and the Parameters panel and rectangle were visually checked.

The renderer pins Martensite main commit `a65734bd`. The model adapter migration
remains separate. The Extrude API/UI route, general profile selection, and the
complete cube edit/save/reopen/named-face UI acceptance remain open.
