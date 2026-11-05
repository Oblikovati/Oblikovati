# Inferred line-chain authoring

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go LineTool.Commit and ApplyLineInference at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.
The kernel and API pins remain v0.2.5 and v0.158.0.

The model now authors the continuous Line tool's chain in one atomic edit.
Vertices are visible sketch point members. Each segment shares the previous
vertex, and closure shares the first vertex. A closed two-point request remains
one open segment, as in Go. Point and line member order follows the Go tool.
Construction and centerline modes apply to lines, not standalone vertices.

The existing line inference selector and constraint builder serve both fresh
lines and chains. Each segment excludes both of its endpoints from snap targets.
Later vertices are not allocated until their segment is processed. Snap changes
to a shared point reach earlier reference lines before direction selection.
The new segment is appended afterward, so it cannot select itself as a reference.
This preserves Go's snap ties, family priorities, constraint order and proposal
records when automatic constraint application is disabled.

The operation first resolves current parameter-driven source geometry with the
caller's solve policy. All chain edits remain in a private candidate until they
succeed. Failures retain consumed identity reservations but publish no points,
lines, constraints or source-coordinate changes. The shared chain-index iterator
also serves plain polyline and recipe authoring; there is no separate topology
indexing loop or direction algorithm.

The independent Go generator in
`crates/model/tests/inferred_line_chains/go_oracle.go` captures 80 cases. Run it
from the pinned Go module to reproduce `go-chains.json`. It records point and
member order, endpoint identity, snap targets, direction targets and cumulative
constraint counts. Cases cover both closure modes and five inference settings,
including equal-distance targets, repeated coordinates and earlier chain lines.

Eighteen cases contain 33 Go direction records derived from a zero-length probe
or reference. These are explicit exceptions to direction parity: the existing
Rust selector refuses direction evidence from a zero vector. Tests preserve that
rule, compare the remaining records, and account for omitted Go constraints.
A valid alternative reference may still supply a direction. The model operation
can retain a zero-length authored line; this is not certification of a valid
profile or permission to create a solid from it.

Additional checks cover format modes, Undo/Redo, serialization, permanent ID
retirement, invalid input and policy, missing sketches, mid-chain allocation
failure, constraint allocation failure, and a parameter change in the same edit.
The native test loads open and closed authored chains, checks visible point and
line keys, compares mapped X11 pixels with capture, and checks save/close/reopen.

This completes model authoring only. Shared document-context preview, field
locks, mouse chaining, the tool-lifetime transaction, Enter/Escape completion,
Close and local Undo remain open. The native test consumes an authored package;
it does not claim interactive Line tool completion. R3/R4 remain open.

Cross-platform review: member, vertex and constraint order are explicit. Identity
does not depend on coordinates, map order or addresses. Snap distance retains
the existing working-unit conversion. No new tolerance, kernel algorithm,
wire method or kernel priority change was needed. The existing Go zero-vector
difference is documented rather than copied into Rust.

The full local `make gate` passed across all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, per-crate coverage,
Markdown/link checks, SPDX, workflow lint and duplication. Every crate exceeded
80% line coverage. Chain authoring reached 98.94%, the shared line-adoption path
100%, and the snap selector 100%. Duplication remained 0.79%
(587/74,487 scanned lines, 68 clone pairs); no duplicated lines or pairs were
added. Evidence: `/tmp/inferred-line-chains-gate.log`.

Native images `/tmp/inferred-chain-false.png` and
`/tmp/inferred-chain-true.png` were inspected. Both show visible vertex markers;
the closed chain adds the final diagonal. The original rectangle and blue
background belong to the fixture. Temporary capture code was removed before
the full gate. This receipt was added afterward and checked with
`make docs-lint`; no code changed after the gate.
