# Native dimension creation tools

Date: 2026-11-06. Tracking: [#3823](https://github.com/Oblikovati/Oblikovati/issues/3823)
and [#3824](https://github.com/Oblikovati/Oblikovati/issues/3824).

## Implemented boundary

The Sketch ribbon starts General Dimension. Analytic model picks supply two
points, one line, a circle or arc, or two lines through the session selection
owner. One line remains open for the second-line angle choice, as in Go #2022.
A separate viewport click places the label. Placement chooses aligned,
horizontal or vertical distance as in Go #2025. No model edit is made by picking,
preview or placement.

The value panel starts with the measured value in document display units. Enter
or Apply submits an expression through the existing typed dimension client.
Reference dimensions are created in one call. Escape, Cancel, tool replacement
and changed document history retire the draft. A monotonic transient draft key
refuses actions for an older editor without allocating model identities.

The shared scalar evaluator checks the expression before the API call. The model
measures preview inputs through the same constructors, branch choices and
measurement path as an owned dimension, on an isolated sketch copy. No permanent
reservation or owner state changes during measurement. The copy's probe identity
never escapes the scalar query. The original geometry, including existing maximum
IDs, remains unchanged.

[ADR-0032](../decisions/ADR-0032-converged-dimension-authoring.md) requires the
existing convergence check before adoption when target sources are available.
[ADR-0033](../decisions/ADR-0033-independent-dimension-authoring.md) also
requires zero redundant equations in that same solve report under the default
policy. The explicit document Apply Driving setting retains the Go contract.
A redundant driving angle keeps the editor open for an explicit Reference
correction.
The fixed-point conflict that previously returned success with unhealthy DOF zero
is now a refusal. Its parameters and geometry stay unchanged, and the editor
keeps its picks and expression. Placement lock redundancy is classified before
the solve, with the previous convert-and-resolve path removed. Existing explicitly
reported unavailable-formula API authoring and later repair keep their contract. Their labels name source
unavailability instead of presenting a healthy target.

Preview and authored labels are model-plane anchors projected by the current
native camera. Their text uses shared model measurements and document display
formatting. Semantic theme colours distinguish driving, driven and preview text.
GUI packets retain physical coordinates and apply display scaling once. Labels
draw behind chrome. Local annotation or projection failures name their cause and
retain other labels.

## Verification and limits

Targeted model, typed API, owner and real ImGui tests cover the constructor
measurements, numeric refusal, permanent identity retirement, parameter-driven
edits, undo/redo, stale dialogs, cancellation, label projection, theme colours and
both display scales. The final full local gate passed all 13 crates with line
coverage from 91.96% to 99.60%, 0.96% duplicated lines, and all 333 head tests.
The ten session dimension tests also cover save/reopen, copy, reply projection
rejection, atomic driving refusal and reference retry on fixed geometry.
The save fixture now asserts that its fixed-endpoint annotation is a reference
and that restored geometry has no redundant equations.

Live X11 mouse and keyboard input with the Go MCPBridge passed preview,
placement, cancellation and correction after an invalid expression. The desktop
created a 10 mm dimension, bound it to a new `size` parameter, changed size to
20 mm, used undo/redo, saved and reopened the file in a fresh process. The sketch
remained healthy at DOF 3. The canonical API retained the expression and text
position; MCP screenshots showed preview and saved labels. Both desktops closed
normally. A third desktop selected both lines of a parameter-driven rectangle,
refused the redundant driving angle without mutation, then accepted an explicit
Reference correction. The failed name `d2` stayed retired; the accepted angle was
`d3`. Undo/redo kept the sketch healthy at DOF zero. A fourth desktop created a
reference radius through the checkbox, retained DOF three, used undo/redo and
saved the result. All four desktops closed normally. MCP screenshots of the
preview, editor, refusal, accepted labels and reopened file were inspected.
This is dimension creation acceptance, not a fully constrained cube.

Labels use authored text positions. Automatic positions for older dimensions
without a text point, extension/leader graphics, label picking/dragging and
post-placement dimension editing remain separate port work. General Dimension
creation does not complete ribbon/browser parity or the full cube, Extrude and
named-face journey. The tracked Martensite solid-adapter limits still apply.

## Platform admission follow-up

The upstream platform repair for [#3795](https://github.com/Oblikovati/Oblikovati/issues/3795)
is integrated without dropping the feature's history. Private dimension state
and operand modules follow their Linux desktop-owner callers. Public GUI packets
and dimension annotation types remain portable. Windows GNU and macOS ARM64
all-target/all-feature host and GUI Clippy pass. This is compile evidence;
hosted native qualification remains required. The combined Linux gate passed
all 13 crates above 80% line coverage, all 333 head tests, and 0.96% duplication.
The repeated X11 distance workflow passed preview, cancellation, expression
correction, size edits, undo/redo, save and fresh-process reopen. Both desktops
closed normally, and MCP screenshots were inspected. The unchanged angle and
reference-radius paths retain their earlier live evidence and automated checks.
