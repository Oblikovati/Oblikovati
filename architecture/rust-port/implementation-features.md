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

## Conditional suppression with real parameter inputs — partial R3

Tracking: [#3726](https://github.com/Oblikovati/Oblikovati/issues/3726).
Implementation: `crates/model/src/feature/suppression` and the replay controller.
Go source: `model/feature/suppression.go` at the same baseline above.

Each authored condition stores a stable parameter ID, one of six comparisons,
and a finite numeric threshold in document working units. Both input and
threshold must have the same dimension. The six comparisons preserve the Go
exact scalar boundaries; they are numeric authoring rules, not topological
predicates. Parameter rename does not change the reference. Tolerance model
selection affects the compared value; display units do not convert it.

The condition read joins the definition footprint, including a missing input
that may later recover. Replacing or clearing a condition removes only its own
read, preserving a shared definition read. Definition replacement retains the
condition. Condition equality preserves exact threshold bits, including signed
zero, for authored edit decisions. Identity reservations and serialization
still belong to the future concrete feature recipe and session owners.

Replay receives an immutable parameter collection through
`new_with_parameters`. Blocked dependencies are checked first, then explicit
suppression, then the conditional read. A true condition skips execution; a
false condition permits it. A missing, failed, text, Boolean, or dimensionally
incompatible input becomes `ConditionFailed` with its typed cause. Readers
are quarantined while independent entries continue. The Go missing-input
behavior silently enables the feature; Rust refuses that degradation under
ADR-0004. Failed reads never use the last-good inspection value.

An active conditional entry without parameter inputs returns a protocol error
without advancing replay. A reused prefix must match the current condition
outcome and the current typed failure. The session must still prove prefix
freshness for all other definition inputs and invalidate old generations.

Tests cover all six comparison boundaries, dimension errors, invalid thresholds,
selected tolerance values, rename, nonnumeric and failed inputs, stable missing
input recovery, shared footprints, condition clearing, exact threshold edits,
real replay toggles, local failure quarantine, and rejected stale prefix health.
Kernel bodies, feature package persistence, and session adoption remain open.

## Fixed and offset work planes — partial R3

Tracking: [#3727](https://github.com/Oblikovati/Oblikovati/issues/3727).
Implementation: `crates/model/src/feature/work_plane` and `work_plane.rs`.
Go sources: fixed-frame definitions in `work_plane_defs.go`, offset definitions
in `work_plane.go`, and origin frames in `work_geometry.go`, at the baseline above.

Concrete work-plane definitions now evaluate real validated construction frames.
A fixed definition owns a `SketchPlane`. An offset definition names an origin
plane or a stable feature ID and reads a constant working length or a parameter's
selected model value. Its origin is `base.origin + base.normal * distance`;
its authored axes and validation policy remain those of the base. Negative
offsets retain the same normal. No tessellation, proximity search, hidden unit
factor, or coordinate repair is involved.

`AuthoredFeature::work_plane` derives the definition footprint. Replay rejects
an entry that omits a declared input before it evaluates any frames. Origin
frames have no driver feature; referenced planes and parameter offsets enter
the existing typed footprint. Source features must precede readers in the
existing sequence graph. The resolver is a project-owned interface.

`evaluate_work_planes` evaluates an immutable recipe into a fresh map of current
frames plus ordered replay health. This is concrete construction-geometry
execution, using the same suppression and quarantine controller. Only valid
successful frames enter the map. A suppressed source produces no frame, so a
reader reports an unavailable reference. Failed reads and invalid frames block
dependents while independent frames continue. Rolled-back planes have no
current result. A later valid parameter input can restore the frame chain.

The evaluator currently starts from an empty construction result. It does not
reuse geometric prefix caches or serve as the full part-body engine. The next
owner integration must connect permanent feature reservations, exact recipe
comparison, parameter invalidation, clean-prefix reuse, session generations,
undo, and document package persistence. Body replay still requires Martensite
isolation, validation, and provenance.

Definition recipes have their own version 1, retain stable references and
parameter IDs, and contain no evaluated caches or scalar closures. Fixed
frames are validated again on restore; unsupported versions, unknown fields
and variants, invalid axes, and nonfinite constant offsets are refused. The
external Rust enum representation preserves floating-point values through the
workspace's JSON configuration; tests retain signed-zero frame and offset bits.
These records are not a legacy Go work-feature importer or a complete document
feature-list codec.

Tests cover all origin offset directions, oblique fixed frames, axis and policy
preservation, parameter edits and tolerance selection, missing references,
failed input recovery, suppression, conditional failure, cutoff exclusion,
overflow rejection, derived footprints, and strict recipe restore. The Go
offset and parameter-footprint tests pass with `CGO_ENABLED=0`. Other work-plane
constructors, points, axes, coordinate systems, live sketch attachment, and
native presentation remain open. No R3 milestone is closed by this evidence.

## Ordered work-plane program recipes — partial R2

Tracking: [#3728](https://github.com/Oblikovati/Oblikovati/issues/3728).
Implementation: `feature/work_plane/collection` and `feature/suppression/recipe`.
Go reference: feature metadata in `serialize.go`, work-plane records in
`serialize_work_plane.go`, and the ordered replay sources at the baseline above.
This is a Rust collection schema, not an implicit importer for those Go records.

Program schema 1 preserves stable IDs, display names, order, definitions,
explicit suppression, conditional suppression, declared definition reads,
unattributed-read status, and the authored end-of-part marker. It stores no
evaluated frames or health. A condition stores its stable parameter input,
comparison, exact working-unit threshold, and the canonical quantity category
from the existing unit service. Named display units such as `mm` are refused
as categories. Known nonnumeric categories and nonfinite thresholds are also
refused by the shared condition constructor.

Restore builds a fresh collection through the existing dependency graph and
definition validators. Condition reads join their declared definition reads.
Additional conservative reads remain intact. Export and restore reject a
missing mandatory definition read; export also validates constant offsets
before JSON can turn a nonfinite value into a null. Duplicate or forward
feature identities, unsupported versions, unknown fields and variants, invalid
definitions, and invalid conditions are named failures. A rejected candidate
cannot replace the caller's collection.

The authored cutoff is retained even when deletion left it beyond the current
tail. Effective evaluation clamps it, as the existing Go and Rust sequence
rules do. Appending a later entry therefore has the same meaning before and
after restore. Protected-removal diagnostics sort reader IDs so rebuilding a
graph after reorder does not change the refusal's reader order.

`same_work_plane_recipe` compares complete authored state with exact floating
bits, including frame axes, validation policy, constant offsets, and condition
thresholds. It also compares identity, labels, order, suppression, declared
reads, and cutoff. It does not compare derived health or construction frames.
This supplies the equality rule required by the future history and save owner.

Tests compare restored programs against original real frame results, ordered
health, read invalidation, and full authored equality. They cover empty and
clamped programs, independent ownership, exact signed-zero values, invalid
versions and dependencies, invalid categories and thresholds, strict schema
fields, retained extra reads, condition clearing, and stable refusal diagnostics.
Document snapshot ownership, permanent feature reservations, undo/redo,
package fields and live sketch attachment remain open. Collection JSON
round-trip does not establish application save/reopen or close R2.

## Work-plane document history and storage: partial R2

Issue #3729 connects the ordered work-plane program to the document owner:

- `DocumentSnapshot.work_planes` owns authored state. Undo, redo, abort, nested
  transactions, unchanged-frame detection, and refresh use the same owner and
  exact recipe comparison. Derived frames and health are not saved or cloned
  into history.
- A separate feature ID allocator lives outside snapshots. Failed commands,
  deletion, abort, and undo never rewind its mark. A candidate cannot introduce
  an ID that this edit did not reserve. The namespace remains separate from
  parameter, sketch, sketch entity, and derived table namespaces.
- Schema-3 `model.workPlanes` stores the strict version-1 program recipe and
  `featureHighWater`. Restore validates recipes, read dependencies, program
  order, and live IDs before document registration. Save and copy reject a
  lower retained mark, preserve unrelated package sections, and retain marks
  when every plane is deleted.
- Workspace dirty checks include both exact authored plane state and permanent
  reservations. Save preparation passes the mark to the storage owner. Open,
  create from package, template creation, and copy restore that same mark.
  Failed storage does not adopt a clean baseline or change file identity.
- Nonempty legacy `model.features` records require an explicit importer. The
  loader refuses them before claiming a document identity. Raw package decode
  still retains these records for inspection and future import.

Model, package, and workspace tests cover failed edits and saves, exhausted and
unreserved IDs, undo/redo, deleted reservations after reopen, independent copy
allocation, package-based creation, legacy refusal, and signed-zero changes.
Reopened parameter-driven planes evaluate with the restored parameter owner.
Tests use the named in-memory file boundary; no native window is claimed.
The empty-program cutoff regression fails against `65103ab2`: clearing `Some(0)`
must save `None`. Unchanged storage now compares the complete authored program
and reservation mark, so an empty collection cannot hide a cutoff change.

This step supplies actual recipe save/reopen support for fixed and offset
planes. It does not complete R2 or R3: live sketch attachment to feature planes,
clean-prefix geometry reuse, kernel-backed feature definitions, native host
commands, and viewport acceptance remain open. The stored program contains no
Martensite handles or B-rep body results.

## Work-plane hosted sketches: partial R3

Issue #3730 replaces copied placement as the only sketch representation with an
owned fixed frame or a stable work-plane feature reference. A linked recipe
contains no last-good frame. `PlacedSketch::plane` needs a current resolver for
linked placement and refuses an absent frame. Fixed sketches need no resolver.
`DocumentSnapshot::resolve_sketch_plane` evaluates the current immutable plane
program with that document's parameters. `require_resolved_sketch` checks both
available placement and current, fully constrained 2D verification before
returning a borrowed view for feature consumption.

The attributed sketch footprint includes local dimension parameters and its
host feature key. Placement changes do not reauthor local 2D coordinates or
invalidate an otherwise current 2D solve. Failed, suppressed, excluded, and
missing host features have no usable current frame. Fixed placement stays
independent of unrelated work-plane failures. Replacing a linked placement with
a fixed frame removes the host reference.

The placed-sketch recipe keeps version 1 for fixed-only collections. Version 2
adds `plane_feature`; a linked record omits `plane` and `plane_options`. Restore
refuses zero IDs, ambiguous or incomplete placement records, and feature
references in version 1. The existing package owner preserves references
through history, file save/reopen, and independent copies. Solve evidence is
not stored. The initial tests used the existing recompute command after reopen;
issue #3731 below supplies automatic verification during workspace registration.

Tests cover parameter-driven model-space projection, failure and recovery,
fixed detachment, exact reference dirty checks, numerical readiness, malformed
recipes and read declarations, direct linked creation, undo/redo, save/reopen,
and copy. The Go oracle tests
`TestWorkPlaneOffsetParameterEditTargetsHostedSketch` and
`TestWorkPlaneOffsetTargetedMatchesFullRebuild` pass at `aa8297d1`.
Those Go tests also cover solids and targeted body replay, which this Rust step
does not yet implement or claim. Native host commands, associative face
placement, kernel-backed sketch consumers, and viewport acceptance remain open.

## Initial workspace verification: partial R3

Issue #3731 verifies pending 2D sketch readers before a loaded or created
history is registered. The workspace and file workspace accept an injected
`SolveOptions` policy, which also reaches later automatic reader edits. Default
constructors retain the existing bounded numerical defaults.

Initial verification uses the existing owned refresh path. It changes no undo
cursor, reserves no recipe identities, emits no transaction events, and records
local solver failures instead of refusing independent document contents. A
history with an open transaction is refused before identity claim or session-ID
reservation. After a claim is attempted, its session ID remains reserved even
when identity collision resolution fails, as before this change.

The clean baseline is the original authored input, captured before verification.
A health-only solve leaves a loaded file clean. If convergence repairs its
coordinates to match current parameter targets, those coordinates remain dirty
until save. Failed and iteration-limited results retain coordinates and expose
their decline to checked consumers. Convergence with free variables still does
not qualify a sketch for feature consumption.

Tests exercise fixed and linked reopen without explicit recompute, saved stale
coordinates, local parameter failure, zero-iteration and invalid numerical
policies, free variables, package and template creation, unchanged file bytes,
empty undo/redo and transaction observations, and preparation failure with an
existing active document. The saved-file baseline survives initial verification,
and a subsequent save/reopen retains both corrected coordinates and clean state.

This completes the initial 2D verification path for the current workspace. It
does not complete R3: associative face placement, native host commands,
kernel-backed sketch consumers, incremental body replay, and viewport acceptance
remain open. Work-plane frames still resolve from immutable current state and
are not copied into history or stored in the file.

## Selected sketch solve: partial R3

Issue #3732 adds the owner command needed by the shared `sketch.solve` API.
`DocumentSketches::solve_one` validates a stable sketch identity before polling
or numerical work. Selection is a scope of the existing solver; it does not add
a second solve engine. Explicit selection retries a current failed sketch and
leaves all unrelated geometry and verification stamps unchanged.

Document and workspace commands use the document's injected `SolveOptions`
policy. The command joins an open transaction or creates one bounded history
step. Cancellation retains the complete input, and local numerical failures
retain coordinates while recording diagnostics. The explicit command does not
start an automatic solve of unrelated pending readers.

Tests cover missing identities before solver work, selected geometry and health,
cancellation, failure and retry, injected iteration limits, an unrelated pending
reader, unchanged saved state, group abort, undo/redo, and file save/reopen.
Issue #3733 below connects the typed API dispatch and read-only constraint-status
query to this owner. This does not complete R3 or certify kernel/native behavior.

## Sketch diagnostic API: partial R3

Issue #3733 connects the shared `sketch.solve` and `sketch.constraintStatus`
methods and DTOs to the file owner. The active document must have the retained
package kind `DOCUMENT_PART`, matching the Go router. Ordered API indices resolve
to stable sketch identities on each call. Negative, absent, and out-of-range
indices are refused before commands start. Bare recipe sessions do not infer a
part-document kind or advertise these calls.

Solves use the document's injected numerical policy and the existing selected
owner command. Response projection happens before candidate adoption. A failed
response projection discards the candidate; a local solver error records failed
health and reaches the client as an error. Health-only solves do not add an undo
step, but evaluation generations remain monotonic. Geometry changes join the
current transaction, including abort and commit-veto behavior. The solve changes
no parameter outputs and starts no unrelated linked-document refresh.

The constraint query uses the same owner's rank policy without iteration,
geometry changes, history movement, or a generation update. Counts are checked
before conversion to the API's signed integer fields. Go's `well`, `under`, and
`over` wire status values remain unchanged. Convergence with free variables is
locally healthy; redundant constraints are a warning and report `healthy=false`.
This local health does not approve a sketch for feature consumption, which still
requires resolved placement and zero free variables.

Typed-client tests cover statuses and health, ordered removal, malformed inputs,
empty parts, absent focus, wrong kinds, read-only ownership, numerical declines,
local failure health, unchanged queries, grouping, veto, undo/redo, and save/reopen.
The Go solver and sketch oracle tests also pass. Native transport, the remaining
sketch commands, associative placement, and kernel-backed consumers remain open.

## Construction-plane lifecycle state: partial R3

Issue #3734 ports the authored construction flag required by Go sketch deletion
and construction-orphan cleanup. The flag is attached to a stable work-plane
feature identity. It is a consumer-tied lifecycle property, separate from its
frame, suppression, and future visibility state. Ordinary planes default to
false. Definition replacement, rename, reorder, and recipe restore retain it.

Work-plane program version 2 carries construction state. Ordinary programs keep
version 1 and omit the new false field. The loader accepts both versions, treats
a missing flag as false, and refuses `construction=true` in a version-1 program
with the offending feature identity. Individual definition recipes remain at
version 1 because their geometric meaning has not changed.

Exact authored comparisons include the flag. Owner commands therefore preserve
dirty state, undo/redo, group abort, and the saved-file baseline. Save, reopen,
and copy retain the flag and permanent feature identity. Tests also check unknown
identity refusal, unchanged flags, and byte-identical evaluated frames. The Go
construction-default and lifecycle tests pass as reference evidence; the Rust
cleanup operation is covered by the follow-up below.

The construction-state change itself does not delete planes. The follow-up below
adds consumed-sketch refusal, orphan pruning, and the shared deletion command.
Native/kernel acceptance remains open.

## Sketch deletion and construction cleanup — partial R3

Tracking: [#3735](https://github.com/Oblikovati/Oblikovati/issues/3735).
The file host now routes the shared typed `sketch.delete` command through the
document owner. It selects the stable sketch identity from the active part's
ordered index. Invalid indices, document kinds, and malformed requests fail
before mutation. Reply preparation, deletion, and plane cleanup share one
transaction and one undo step; a rejected reply or observer veto restores the
candidate. Deletion does not numerically solve unrelated sketches.

Deletion derives an incidence index from authored read footprints. Known sketch
consumers and unknown read attribution cause a named decline before mutation.
Only construction planes that had consumers before deletion can be pruned.
Reverse validated feature order removes the orphan chain in one pass. Ordinary
planes, initially unused construction planes, other sketches, and suppressed or
cutoff consumers retain their references. Entity namespaces remain distinct.

Tests cover exact consumers, chain cleanup, stable IDs, local frame preservation,
undo/redo, group abort, response rejection, veto, saved baselines, and save/reopen
with permanent reservation marks. The Go construction tests pass at migration
baseline `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
Reference: `model/compdef/work_construction_cleanup.go` and
`addin/router/sketch_dependents.go`.

This covers the current work-plane and sketch catalogue. Future body-feature
recipes must supply their exact consumed-sketch keys to this same incidence
source. Legacy body feature import is still refused. Native selection,
kernel-backed consumers, and viewport acceptance remain open.

## Sketch frame orientation — partial R3

The numerical frame operation and Go reference tests are recorded in
[orientation evidence](implementation-sketch-orientation.md).
