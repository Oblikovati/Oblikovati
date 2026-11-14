# Analytic sketch picking implementation

Date: 2026-11-05. Tracking: [#3816](https://github.com/Oblikovati/Oblikovati/issues/3816).
Design: [ADR-0030](../decisions/ADR-0030-analytic-sketch-pick-witnesses.md).

## Implemented boundary

Martensite v0.8.0, commit `59aca38960bdfbaa75a07a80d9b2de522005ddac`, supplies
analytic point-to-curve extrema through the existing authored curve constructor.
The host queries each C2 interval, includes finite boundaries and internal knots,
and checks native position, minimum classification and normal stationarity.
Completed native extrema do not certify root completeness. No display samples
participate in picking. A failed query shows a notice and retains selection.

The read index resolves points, including hidden defining points, and finite
curves to authored entity IDs. Points have priority, as in Go `sketch_select.go`;
equal screen distances use the entity ID as a total tie-break. The hit radius is
Go's 12 pixels. It is separate from modelling precision. Camera gestures and
active placement tools retain their inputs.

The document session owns checked editor picks. API readers and the GUI snapshot
read that owner. Stale generation handles are not published; modified clicks
cannot merge them with fresh picks. Ordinary empty clicks clear selection.
Browser and editor picks share replacement behavior. Document switches retire
both, and selection creates no model history.

The native overlay accepts whole-sketch and individual-entity identities through
one selection packet. Selected hidden defining points receive markers. Selection
survives theme and display refresh; other entities retain their normal colour.
The existing canonical selection response retains Go's kinds and empty reference
entries for sketch entities. No public wire method or DTO is added.

## Evidence and limits

The adapter suite passes the analytic and construction checks, including the new analytic endpoint,
trimmed arc, C0 knot, rational quarter-circle, ellipse, periodicity, offset,
trimmed spline, interval budget and invalid witness checks. The existing Go and
OCCT construction assertions remain unchanged. Owner tests exercise mouse
selection, modifiers, point priority, failed projection and stale generations.
Native pixel checks cover individual curves, hidden endpoints and other entities.

Live X11 mouse and keyboard input through the Go MCPBridge selected one curve,
added and removed another with Ctrl, cleared empty space and selected an endpoint.
The canonical API reported the same counts. An MCP screenshot showed one blue
edge and three red edges. The same desktop changed size, used Undo/Redo, saved,
closed and reopened the healthy DOF=0 sketch. A second desktop reopened the file
with empty transient selection. Both desktops closed normally.

The full local gate passed with all 13 crates above 80% line coverage and 0.86%
duplication. It includes 312 head checks and the native viewport suite. A
successful pick publishes a selection-count notice, including after the window
has consumed an earlier failure; owner and native-window tests cover that path. Local logs and MCP captures are in
`/tmp/oblikovati-analytic-sketch-picking-*` and
`/tmp/oblikovati-x11-analytic-sketch-picking` for this session.

Constraint ribbon tools, dimension labels and constraint-marker picking are not
accepted by this change. They remain work under #3816. This does not accept the
full cube journey or change the solid adapter's kernel qualification limits.
