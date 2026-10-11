# Feature sequence implementation evidence

## Ordered recipe foundation — R2

Tracking: [#3724](https://github.com/Oblikovati/Oblikovati/issues/3724).
Design: [ADR-0004](../decisions/ADR-0004-session-and-feature-evaluation.md)
and [ADR-0005](../decisions/ADR-0005-identity-and-references.md).
Implementation: `crates/model/src/feature` and `feature.rs`.
Go sources: `model/feature/engine.go`, `reorder.go`, and `engine_graph.go`
at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The model owns an ordered collection of typed definitions. Each entry has a
stable caller-supplied feature ID, display name, explicit suppression state,
and read footprint. IDs are independent of labels and collection positions.
The collection does not allocate IDs; its future session owner must reserve
them permanently outside undo snapshots, as it does for sketch IDs.

Feature reads use the existing dependency graph. Duplicate identities,
missing drivers, cycles, forward dependencies, and removal with live readers
are refused before changing authored state. Suppression retains dependencies.
Reorder preserves each driver before its reader, including suppressed entries.
The Rust removal refusal strengthens the Go collection's permissive deletion;
the future delete command must explicitly remove or quarantine dependents.

The end-of-part marker is an exclusive positional cutoff, as in the Go feature
engine. `None` evaluates the whole program; `Some(0)` excludes every feature.
An explicit cutoff remains positional during reorder and append. It is clamped
to the available length after removal. An out-of-range new cutoff is refused.
The separate Go component-definition marker does not become a second Rust owner.

Edits report the replay range `[start,end)`. State before `start` is the clean
prefix; the evaluator must discard old output from `start` onward. An empty
range still has meaning: removing the last feature or rolling back to zero
must discard output even though no feature runs. Reorder starts at the lower
changed index; suppression and replacement start at the edited entry; moving
the cutoff requests replay from zero. Rename does not affect geometry.
Repeated reorder, suppression, and cutoff settings return no replay request.
Definition replacement always requests replay: generic `PartialEq` cannot
certify exact authored equality, especially signed zero in floating values.
The concrete command may compare its exact recipe before requesting replacement.

One typed footprint currently covers document-local parameter, sketch, and
feature reads. Changed inputs select the earliest active reader and its suffix.
An unattributed read conservatively responds to any nonempty change set.
Empty attributed reads are distinct from unknown reads. Trailing rolled-back
readers do not request replay; suppressed active readers retain invalidation.
The model has no file or workspace dependency. Work geometry represented by
features uses feature identities; qualified external and occurrence reads
remain to be added with their concrete owners under the same mechanism.

The collection produces no bodies and owns no result cache. Its caller must
combine pending edit ranges, preserve the earliest invalidated prefix, and
tag execution with the document generation before adopting results. Cloning
this recipe state does not prove Martensite shape isolation. Session history,
package recipes, permanent feature reservations, concrete feature definitions,
conditional suppression, failed-feature quarantine, and kernel-backed replay
remain open. This work does not close R2, R3, or the migration epic.

## Validation

Rust tests cover atomic rejection, identity, clone isolation, no-op state edits,
suppression, cutoff movement, append/removal at the cutoff, exact floating
recipe replacement, typed read namespaces, and conservative invalidation.
An independent edge-order check covers all 64 forward dependency graphs on
four features and all 16 single-feature moves in each graph.

The Go baseline tests for dependency-safe reorder, valid reorder, explicit
suppression, and the exclusive end-of-part marker pass with `CGO_ENABLED=0`.
They test real Go feature behavior; Rust state tests do not claim solid parity.
Run `make gate` for all present Rust workspace crates on local Linux.
Kernel and native UI acceptance are separate required checks.
