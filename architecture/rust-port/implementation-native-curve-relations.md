# Native curve relation tools

Tracking: [#3836](https://github.com/Oblikovati/Oblikovati/issues/3836) and
[#3837](https://github.com/Oblikovati/Oblikovati/issues/3837).

The Sketch ribbon exposes Collinear, Equal, Concentric and Tangent through the
existing canonical typed constraint client. Collinear accepts two lines. Equal
accepts two lines or two circular curves. Concentric accepts two circular curves.
Tangent accepts a line and circular curve, or two circular curves. Circles and
arcs use the same circular operand category. Point picks are filtered out.

The Go reference is `app/sketch_constraint_tools.go` and
`app/sketch_constraints.go`. Equal keeps Go's line-pair priority; Tangent keeps
Go's line/circular priority. An incomplete mixed selection remains transient
until a valid pair is available. Operand order is normalized before the typed
request. Session selection owns highlights and duplicate suppression. Escape,
replacement and history retain the shared tool lifecycle. Solver refusal keeps
the operands and original document recipe.

Qualification found incorrect Collinear rank for connected lines.
[ADR-0034](../decisions/ADR-0034-identity-collinear-equations.md) classifies
shared endpoint identity once and removes only incidence equations already
supplied by identity. Distinct coincident points retain both equations. The
disjoint formula and its AD derivatives retain their original bits. Zero-length
directions still fail. Authored relation identity and storage stay unchanged.

Owner regressions cover every circle/arc pair, both line/circular pick orders,
stored relation kinds, solved geometry, wrong categories, incomplete mixed Equal
picks, cancellation, conflict refusal and undo/redo. Real ImGui tests cover all
four commands at normal and double display scale. Windows x64 GNU and macOS ARM64 host/GUI Clippy checks passed. These are
compile checks, not native runtime claims for those platforms.

Native X11 mouse/keyboard acceptance passed Collinear, Equal Length, Concentric,
Equal Radius and Tangent. Canonical reads checked stored kinds and source IDs;
independent measurements checked directions, lengths, centres, radii and tangent
contact. Collinear correctly retained DOF five for three connected free points.
Undo restored original coordinates and IDs for every relation. Point filtering,
duplicate picks, cancellation and redo passed. A saved Tangent result reopened
in a fresh process with the same geometry, IDs, constraint and DOF five. MCPBridge
supplied read-only inspections and captures. All five result captures and the
reopened result were inspected. Both acceptance desktops closed normally.

The final full local gate passed all 13 crates, including 365 head tests.
Coverage ranges from 91.96% to 99.60%; duplicated lines are 1.00%. The model
regression matrix and original disjoint-line bit checks pass in that same gate.

This does not complete the general Extrude, cube or named-face journey. The
tracked kernel analytic-region boundary remains a prerequisite.

The combined SDK qualification candidate in #3833 remains separate. Its exact
hosted run 42162243927 passed Linux x64 native tests but failed Linux ARM64 at
Vulkan instance creation with `ERROR_EXTENSION_NOT_PRESENT`. The ARM64 runner
installed validation layers 1.3.275. This retains the runtime gap tracked by
Martensite#388; no platform test is disabled by this tool port.
