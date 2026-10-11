# Associative plane-and-point construction

Tracking: [#3750](https://github.com/Oblikovati/Oblikovati/issues/3750).
The datum owner and permanent reference boundaries remain ADR-0004 and ADR-0005.

## Current geometry and dependencies

A plane-point source stores one base-plane reference and one through-point
reference. One replay path resolves both current results, retains the base axes,
and places the frame origin at the point. It preserves frame options. No copied
coordinates, second dependency graph, or display geometry determines the frame.

Stable feature reads name both drivers; document origins add no feature reads.
Parameter changes in the point and normal flips in the base remain live. A local
flip applies after construction. Failed drivers retain the source and have no
current plane result; recovery reconstructs the plane from current inputs.

## API, source, and history

The released typed API already defines the constructor. Canonical creation
requires exactly two references of the correct kinds. The list reports base-plane
slot 0 and through-point slot 1, with no scalar slots. Redefine permits one or
both unique reference slots. Scalar edits, duplicate slots, wrong kinds,
self/forward dependencies, and unknown references produce named declines.

Reference and display edits share one candidate, command, and undo record. Edits
retain the plane key and authored orientation/display metadata. A malformed
second reference cannot publish an earlier valid repick. Unchanged references
create no document generation. Undo/redo, abort, veto, source copy, and reopen use
the existing file-owned history and persistence boundaries.

The new definition requires source version 5 and datum program version 10.
Older program versions cannot contain it. Unknown source fields are refused.
Existing definitions retain their previous minimum versions.

## Linux evidence and remaining scope

Model tests check all origin-plane orientations, live scalar inputs, driver
orientation, exact reads, failure/recovery, source comparison, version checks,
and kind/dependency declines. Typed API tests check creation, slot metadata,
reference edits, atomic refusal, undo/redo, mixed display edits, and copy/reopen.
The Go `TestPlaneAndPointWorkPlane` regression passes at the fixed source baseline.

Acceptance requires the full Linux `make gate`. Other plane constructors,
assembly contexts, native viewport application, and kernel geometry acceptance
remain open. No kernel checkout or Windows/macOS pipeline is changed.
