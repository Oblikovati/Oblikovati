# Mixed datum program — partial R3

Tracking: [#3739](https://github.com/Oblikovati/Oblikovati/issues/3739).
This extends the single feature program in accepted ADR-0004. It does not add
another architecture record for the same boundary.

`DocumentSnapshot.work_geometry` owns one ordered `FeatureSequence` of fixed
planes, offset planes and checked line axes. All kinds share permanent feature
IDs, dependency validation, conditions, suppression, end-of-part cutoff and the
existing replay controller. The plane-only evaluator was removed. Plane query
interfaces and old recipe names delegate to this same mixed program.

Current evaluation exposes checked plane and axis views by stable ID. A wrong
kind is a named decline. Failed or suppressed geometry is unavailable. Sketch
orientation resolves the current owned axis direction and base plane on each
query. The explicit injected axis interface remains a dependency seam; it does
not provide a retry after owned geometry fails.

The document keeps its versioned `model.workPlanes` storage key. A mixed recipe
requires program version 3 and an axis definition uses wrapper version 2 around
its checked line recipe. Plane-only programs keep versions 1 or 2. An old reader
therefore rejects a mixed version instead of ignoring an unknown section and
silently losing axes. Restore validates kind-specific versions, exact source
bits, attributed dependencies and permanent ID high-water marks before adoption.

Sketch deletion uses the same exact incidence graph for both datum kinds.
Unused construction datums are removed in reverse validated feature order.
The complete edit remains one atomic history step. Undo restores both kinds;
deletion does not release their permanent IDs.

Linux tests cover mixed replay order, wrong-kind dependencies, suppression and
cutoff, current axis edits, recipe version refusal, exact round trips, shared
construction cleanup and retention while another sketch consumes the datums.
File-session tests cover undo/redo, save, reopen, independent copies and permanent
IDs after deletion. Existing plane, sketch and persistence tests remain required.

Two-point and plane-intersection axis definitions, points, edge references and
public axis/sketch creation routing remain open. They must extend this program.
No native rendering or B-rep kernel acceptance is claimed by these model tests.
