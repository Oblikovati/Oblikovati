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

## Ordered health and quarantine controller — partial R3

Tracking: [#3725](https://github.com/Oblikovati/Oblikovati/issues/3725).
Implementation: `crates/model/src/feature/evaluation` and `evaluation.rs`.
Go sources: `model/feature/engine_recompute.go` and `engine_graph.go` at the
same baseline above. The controller implements the health protocol around
execution; it does not implement kernel operations or store geometry.

The caller supplies the immutable sequence, exact replay range, and ordered
health records for its proven unchanged prefix. Prefix length and identities
must match the recipe. Prefix health must agree with explicit suppression and
known unavailable drivers; blocked causes must name the exact immediate
drivers in stable ID order. A healthy prefix cannot hide a failed dependency.
These checks establish structural consistency, not freshness: the session
owner must prove that the prefix belongs to the current recipe and generation.

Each active tail entry is classified once as evaluate, suppressed, or blocked.
An evaluation must receive one result for its exact feature ID before replay
can advance. Failed and blocked entries quarantine their dependency readers.
Independent entries continue. Typed backend failures stay on their feature
records; blocked entries retain their immediate causes, so callers can walk
the ordered report to the original failure. Skipped entries pass the running
body state through without calling the kernel. A new replay does not retain
old quarantine when the failed feature succeeds.

Blocked dependencies take precedence over explicit suppression. The Go loop
checks suppression first, which can hide an upstream failure through a
suppressed intermediary. Rust retains the authored suppression flag and
reports the intermediary as blocked, preserving the full quarantine required
by ADR-0004. A regression test covers this difference. Suppression on a healthy
dependency path retains the Go pass-through behavior.

Out-of-order results, duplicate results, advancing with a pending evaluation,
and incomplete completion are named protocol errors. Rejection leaves health
and replay position unchanged. Zero-length replay tails still return complete
prefix health. Cancellation can discard the controller without editing the
recipe; it must not publish an incomplete report.

Tests cover failure diamonds, independent branches, prefix failures and blocked
causes, suppression, recovery, recipe isolation, cutoff exclusion, empty tails,
and protocol rejection. An independent reachability calculation checks every
failure root on all 64 forward dependency graphs with four features. The Go
baseline's local failure, dependent quarantine, clean-prefix reuse, and explicit
suppression tests also pass with `CGO_ENABLED=0`.

The eventual executor must validate kernel output before reporting success,
retain bodies from the last valid prefix after failure, and discard invalid
partial bodies. The controller does not prove those geometry postconditions.
Kernel body isolation, generation checks, coherent session adoption, concrete
feature definitions, history and persistence remain open. No R3 exit or full
application feature is declared complete by these state tests.
