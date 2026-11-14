# Shared line-chain recipes

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go LineRecipe, LineChainRecipe and AddPolyline at
aa8297d1b0ac33c024f19e54821536d76e8a447a.
Kernel/API pins remain v0.2.5/v0.158.0 with no local source override.

The placement graph can now describe an entire open or closed line chain.
Consecutive lines share the same endpoint identity. Explicit closure uses the
first point's ID; it does not add another coincident point. Equal coordinates
at nonadjacent open-chain vertices retain their separate identities. A plain
chain has two degrees of freedom per vertex and no implicit constraints.

Length and Angle fields describe the final segment. Length binds its actual
endpoint indices, while Angle steers placement and adds no dimension. For a
closed chain, both fields describe the closing segment. A locked Length creates
one owned parameter and dimension through the existing recipe commit path.

Typed polyline creation and placement recipes now share vertex validation and
segment indexing. Rectangle and polygon placement use that same member builder.
The separate closed-boundary builder and polyline indexing loop are removed.
Typed polyline roles, identity allocation, errors and reply shape are unchanged.
Recipe validation rejects too few vertices, nonfinite coordinates and segments
whose existing field-length calculation is zero or nonfinite. This includes
coincident adjacent vertices and arithmetic underflow/overflow; no wider numeric
range or new tolerance is claimed by this step.

Twenty new independent Go fixtures cover two-, three- and five-vertex chains
in four quadrants, with open and closed forms. Open cases use LineRecipe or
LineChainRecipe. Closed topology is read from AddPolyline's actual point
identities, and its final fields come from LineRecipe on the actual closing
line. The fixture set now has 103 cases; the prior 83 are unchanged. Tests
compare members, fields, rank, shared IDs, typed-polyline parity, Undo/Redo and
serialization. Fully constrained open and closed chains retain zero DOF without
redundant equations through formula edits, and keep every line and endpoint ID.

The native adapter samples every chain segment in authored order. A foreground
owner test checks preview without document edits, commit with a locked final
Length, reference-key resolution, Undo/Redo, save, close and reopen. Captures
are compared with mapped X11 pixels. The open and closed results were inspected:
`/tmp/line-chain-false.png` and `/tmp/line-chain-true.png` (1440x900). The closed
case has the final diagonal edge; the open case does not. The prior rectangle
and blue background belong to the fixture. Temporary capture code was removed.

This is the model recipe needed by continuous Line placement. Mouse chaining,
the transaction held for the tool's lifetime, inference against existing
geometry, Enter/Escape completion, Close and local Undo remain open. The Go
requirements in issues #2024, #2032 and #1750 still apply. No reduced two-click
Line tool is exposed by this step. Menu controls, snapping, certified sketch
regions and general extrusion also remain open. This does not complete R3/R4
or the full port. Windows/macOS runtime acceptance remains open. Work is local.

Cross-platform and comment review checked finite coordinates, explicit vertex
and member order, bounded indices and closure identity. No map iteration picks
an endpoint. The Go dynamic-field rationale is retained. No new tolerance,
public wire contract or kernel algorithm was added. This work needs no kernel
priority change.

Go LineTool.Commit also adds its vertices as standalone point members through
Points.Add. LineChainRecipe and AddPolyline use defining points. The tool layer
must retain the Go tool's visible-point membership when it consumes this shared
chain graph. Tool transaction teardown must target its original document, as
specified by Go tool_transaction.go, including document and tool switches.

The full local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests used Xvfb with software
Vulkan. Every crate exceeded 80% line coverage. Model reached 98.84%; chain
layout, the chain recipe, the shared member builder and typed polyline adoption
reached 100%. Duplication was 0.79% (587/74,307 scanned lines, 68 clone pairs),
with no added duplicated lines or pairs. Evidence is in
`/tmp/line-chain-recipes-gate.log`. This result and the tool-lifecycle notes were
added after the gate and checked with `make docs-lint`; no code changed afterward.
