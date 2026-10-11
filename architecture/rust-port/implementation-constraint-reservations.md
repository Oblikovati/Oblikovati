# Permanent sketch constraint reservations

[Issue #3776](https://github.com/Oblikovati/Oblikovati/issues/3776) adds the
identity prerequisite for inferred line creation in
[issue #3775](https://github.com/Oblikovati/Oblikovati/issues/3775).

Constraint IDs remain local to each sketch. `DocumentEditHistory` owns a
monotonic allocator for each reserved sketch namespace outside undo snapshots.
`RecipeReservations::allocate_sketch_constraint` is the authoring path. Allocation
can precede publication of a new sketch. Deleted sketches retain their marks.
An allocation failure names the sketch and retains the underlying identity error.

An edit can retain an existing constraint ID or add one from its own reservation
interval in the same sketch. A failed mutation or validation still consumes its
reservations. Undo, abort, commit veto, and deletion do not rewind them. Refresh
has no allocation capability and cannot resurrect a retired ID or invent a new
one. Reserving an ID in another sketch cannot authorize a relation here.

The native document remains schema 3. Its `model.planarSketches` record now
contains an ordered `constraintHighWater` mapping from sketch ID to the last
reserved constraint ID. A missing field in an older file derives initial marks
from live constraints once. An explicit field must be a mapping that covers all
live constraints; `null` and marks for unreserved sketch namespaces are errors.
Zero marks are omitted from the captured map. Save cannot reduce or drop a
previously retained mark, including one for a deleted sketch.

Open and template creation restore these marks before identity registration.
Save and copy pass the complete history reservation record through
`DocumentPackage::with_sketch_reservations`. Workspace dirty checks compare that
record with the saved baseline, so an undone or failed edit can require save even
when its geometry matches the saved recipe. The standalone `with_sketches` codec
can derive live initial marks and retains prior marks. It does not replace the
history owner's allocation path.

Model tests cover independent namespaces, legacy initialization, explicit mark
coverage, exhaustion diagnostics, invalid namespaces, mutation and validation
failure, undo/redo, abort, deletion, refresh, and restoration without rewind.
Persistence tests cover stable capture, malformed metadata, legacy initialization,
strict captures, deleted namespaces, and monotonic replacement. Named file-owner
fakes exercise failed and aborted reservations through save/reopen, copies of
deleted sketches, registration failure before GUID allocation, commit veto, and
the dirty baseline after undo. Authoring fixtures now reserve constraints through
the owner instead of assigning IDs directly.

Validation is local Linux. This work does not add inferred relations, line API
creation, a native viewport, or kernel package acceptance. Those application
features remain required work under #3775 and the port plan.
