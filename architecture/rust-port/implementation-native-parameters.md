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

The same OS-input journey also runs with the real MCPBridge add-in loaded. Its
`get_sketch` tool reads the Rust owner's canonical `sketch.get` query. The bridge
observed a healthy sketch with zero DOF at 10 mm, after editing to 20 mm, after
Undo, and after Redo. The first corner must land on the displayed origin; a click
one pixel below it retained two translation DOF under the current physical snap
distance. These are read-only observations, with no API setup or geometry edits.
The repeated run used the bridge's `capture_window` tool to obtain each completed
native frame as a PNG image. The returned image bytes matched the host's PNG file.
The 10 mm rectangle and the 20 mm parameter panel after Redo were visually checked.
The window and MCP listener were closed at the end of the run.

`sketch.list` and `sketch.get` inspect the current owner without recording history.
They report names, planes, authored entity counts, structural DOF, current solve
health, and feature consumption. Desktop edit state retains the existing focus
and sketch continuity guards: switching documents or deleting a sketch clears
the observation even when history restores the same identity. Tests cover both
immutable and command hosts, projected-source freshness, free coordinates, and
invalid owners or indices. Visibility remains on for all displayed sketches;
sharing and style edits are not exposed in this port.

Issue #3796 records an add-in wrapper delegation defect found by this live test.
The common forwarding path now keeps Part creation, sketch entry/exit, and
rectangle placement available in both script and add-in wrappers. The native
wrapper regression failed before the fix and passed after it.

`viewport.captureWindow` now queues bounded view work through the same file owner.
It reports the observed drawable size and publishes the next completed frame,
including GUI content, with an atomic PNG replacement. Capture requests do not
edit a document or add history. The default path uses an injected temporary
directory and a per-desktop namespace. A resize that changes the requested extent
or a file-write failure produces a visible capture error and leaves the window
live. Tests cover queue bounds, invalid requests, exact PNG row/alpha/sRGB values,
native API-to-frame transfer, and file failures. The separate GUI-free
`viewport.capture` contract is not yet ported.

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

The renderer pins Martensite main commit `66fe4eb0`, which includes the merged
renderer font changes. The model adapter migration
remains separate. The Extrude API/UI route, general profile selection, and the
complete cube edit/save/reopen/named-face UI acceptance remain open.
