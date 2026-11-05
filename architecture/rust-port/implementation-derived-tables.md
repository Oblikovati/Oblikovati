# Derived parameter tables — partial R2

## Source resolution and enumeration

[#3656](https://github.com/Oblikovati/Oblikovati/issues/3656) adds
`FileWorkspace::resolve_parameter_source` and the file-owner query
`parameters.derivedTables.list`. Sources resolve against owned file paths or a
unique unbound document title. Bound display titles are not path aliases.
Resolution performs no file I/O and does not activate the source. Duplicate
unbound titles produce an ambiguity error; missing sources return no owner.
Only part and assembly documents supply derived parameter sources.

The canonical API response retains table order, linked names, produced-parameter
references, and feature ownership. Candidate names come from the current source
recipe in parameter order and include numeric user parameters. Model parameters
and text/boolean parameters are excluded. Missing sources remain in the result
with a health diagnostic. A retained table that has not been synchronized reports
that state even if its source is now open; enumeration does not change the recipe.
An empty collection uses the canonical null table list, as in Go.

Typed-client tests cover ordered candidates, live source edits, provenance,
resolved/missing/unavailable link health, empty workspaces and collections,
drawing refusal, ambiguous names, and focus/history preservation. The existing
model synchronization operation supplies health updates in the tests; this is
not evidence of automatic workspace propagation.

## Current source values and working scales

[#3657](https://github.com/Oblikovati/Oblikovati/issues/3657) adds
`FileWorkspace::derived_source_values`. It reads healthy numeric user parameters
and transfers each quantity into the target document's working units. Missing
sources return no snapshot. Failed parameters are omitted even when they retain
a last-good value; later synchronization can diagnose those absent drivers.
Text, boolean, and non-user parameters are not source values.

`UnitsOfMeasure::transfer_quantity` applies the working-scale ratio to length,
its square to area, and its cube to volume. Other arithmetic dimensions retain
their values. Equal scales and signed zero retain their exact input bits. The
conversion orders arithmetic by representability so an extreme intermediate
ratio or product does not destroy a representable result. Nonfinite inputs,
nonarithmetic dimensions, final overflow, and final underflow to zero are refused.
A source conversion error names the source document and parameter.

Tests cover all arithmetic dimensions, reverse transfers, exact equal-scale bits,
extreme binary scale ratios and subnormal values, source failure exclusion,
missing sources, target refusal, and no target mutation or I/O. This is a read
boundary for linking and refresh; it does not itself update any table.

## Owned refresh without a target undo step

[#3658](https://github.com/Oblikovati/Oblikovati/issues/3658) adds
`FileWorkspace::refresh_derived_table`. It resolves the source, transfers current
values, and refreshes the target through its session owner. Missing sources feed
the model's unavailable-link state. Direct self-sources are refused.

The history owner adopts a validated candidate without changing undo/redo entries,
saved checkpoints, or identity reservations. Changed values or evaluation health
advance the evaluation generation and notify the observer once; no synthetic
transaction event is emitted. Identical refreshes do neither. Open transactions
refuse refresh. This follows Go's rule that a derived refresh is a computed
consequence of the source edit rather than a new target edit.

Exposed property refresh is part of the same candidate. Any model or property
projection error leaves recipe, generation, and history unchanged and reaches the
caller. [#3659](https://github.com/Oblikovati/Oblikovati/issues/3659) makes an
unavailable exported parameter remove its stale property projection while retaining
its failed health. This allows a missing-source refresh to publish the failure.
Direct property reads report the unhealthy parameter; recovery rebuilds its value.

Tests cover health-only changes, value changes, no-ops, property updates and
unhealthy projection removal, open groups, missing tables/owners, unavailable sources,
self-sources, unchanged identity reservations, and preserved redo/save markers.
Navigating target history restores its recorded snapshot; the scheduler must run
refresh afterward to update its derived values from current sources. A test
exercises that explicit navigation-then-refresh sequence.

## Ordered workspace refresh

[#3660](https://github.com/Oblikovati/Oblikovati/issues/3660) adds
`FileWorkspace::refresh_derived_from`. It builds a dependency graph from retained
links and uses the existing model graph to order the changed document and its
readers, with sources before readers and document insertion order for ties.
The changed document is included so refresh can follow target history navigation.
Missing sources have no graph edge but still produce unavailable table health.

The whole workspace graph is checked for cycles and ambiguous source names before
any table changes. Each table refresh is atomic. A failed table is reported with
its document, table ID, and reason; other tables in that document can still refresh.
Readers of a failed document are skipped and report their blocking sources.
Independent branches continue. Thus a document can occur in both the changed and
failed results. Open source transactions cannot supply preview values, including
sources outside the affected graph. Refresh retains target undo/redo and focus.

Tests cover chains, independent branches, source undo, repeated no-op refresh,
cycles, unknown sources, open source and target transactions, failure isolation,
and recovery. The scheduler is also used by the API command boundary below.

## Automatic API command refresh

[#3661](https://github.com/Oblikovati/Oblikovati/issues/3661) connects file-owned
API edits, transaction end/abort, undo/redo, and history jumps to the scheduler.
Refresh waits while a nested transaction remains open. The source command finishes
before propagation; a refresh failure explicitly reports that the command completed
and names failed tables or blocked readers. It does not imply source rollback.
Graph preflight failures use the same partial-success context. A rejected source
command does not trigger refresh.

Target history navigation also refreshes its restored recipe from current source
values, without adding target undo entries or discarding redo. The bare session
host has no file ownership or source resolution and retains its local behavior.
Typed-client tests cover edits, nested end, abort, undo/redo, jumps, target history,
busy-target recovery, ambiguous-source failures, and focus preservation.

## File lifecycle refresh

[#3662](https://github.com/Oblikovati/Oblikovati/issues/3662) connects file-owned
API commands to a full ordered refresh when the set of open document IDs or bound
paths changes. This includes open/create, close/close-all, Save As, and batch file
operations. Unchanged bindings do not trigger lifecycle refresh. Source links
remain path/title based: moving a source makes its old path unavailable; binding
it back restores values. Closing a source publishes unavailable parameter health;
reopening that path refreshes readers from the newly owned document.

A partial close-all or batch operation still refreshes the remaining workspace.
Its original command error is retained. If refresh also fails, the response
includes both errors and states that ownership already changed. A successful
command followed by refresh failure is also reported as partial completion.
No target undo entry is created. All documents are ordered through the same model
graph traversal; missing sources do not require keeping a closed owner alive.

Tests cover close/reopen, source rebinding, unchanged history, failed refresh after
successful creation, and partial close-all with successful or failed refresh.

## Derived-table identity reservations

[#3663](https://github.com/Oblikovati/Oblikovati/issues/3663) adds
`RecipeReservations::allocate_derived_table` to the document history owner.
Table IDs use their own positive signed namespace, matching the API and imported
Go table records. The counter starts at the largest retained table ID. Parameter
IDs can have the same numeric value because they belong to a different namespace.

Reservations live outside undo snapshots and survive failed edits, abort, and
undo. New table insertion must use an ID reserved by that edit; evaluation refresh
cannot introduce table identities. Redo restores the original IDs. Exhaustion at
`i64::MAX` returns a named error without changing the counter or recipe.

Tests cover imported IDs, namespace overlap, rejected insertion, refresh refusal,
undo/redo, abort, failed edits, and signed exhaustion. The live-owner guarantee
is extended across storage by the reservation capture below. Legacy packages without a saved mark infer it from their retained tables.

## Table creation and link editing

[#3664](https://github.com/Oblikovati/Oblikovati/issues/3664) implements the canonical
`parameters.derivedTables.add` and `parameters.derivedTables.setLinked` commands in
`FileHost`, listed separately in `DERIVED_COMMAND_METHODS`. The bare session host
cannot resolve other documents. File-owner operations resolve an open part or
assembly, transfer source quantities, and validate the proposed graph before edits.
Self-links and cycles are refused even when the linked subset is empty.

Creation reserves a table ID and parameter IDs through the target history owner.
Relinking retains kept parameter identities, local names, and annotations; removed
links lose their produced parameters. Duplicate, missing, nonnumeric, or unhealthy
source names are refused. Both commands return the shared table-info projection.
Successful commands feed the automatic refresh scheduler.

Linking also marks selected source parameters as exposed properties, following Go
`app/derived_tables.go` at `aa8297d1`. Source projection is prepared first through
an exclusively borrowed history candidate. Target failure or observer veto drops
that candidate; target success publishes it with no remaining validation failure
point. Source export changes advance its generation and notify its observer but
create no source undo entry. As in Go, later target undo or bounded-group abort does
not unexport the source. An open source transaction refuses linking.

Tests cover typed creation/relinking, kept IDs, source property projection, value
propagation, empty subsets, duplicate and missing names, source transactions,
cycles, target veto, target undo, reservation reuse refusal, and bounded-group
abort. The Go router lifecycle, bad-argument, source-value, and provenance tests
also pass locally as reference evidence. Native UI acceptance remains pending.

## Renamed-link package round trips

[#3665](https://github.com/Oblikovati/Oblikovati/issues/3665) implements
[ADR-0014](../decisions/ADR-0014-documents-and-storage.md). A renamed local derived
parameter stores its source-to-local mapping in `localNames`, under schema 3.
Schema-2 recipes still use implicit same-name links and remain schema 2 when no
extension is needed. A promoted recipe remains schema 3 after later edits.

Restore rejects mappings outside the selected source names, absent explicit local
names, non-derived targets, duplicate ownership, and schema-2 extension use.
Mappings are built before table adoption. Tests cover promotion, repeated encoded
round trips, source refresh after reopen, dependent expressions, invalid mappings,
and unchanged schema-2 behavior. The baseline Go reader rejects future schemas in
`persistence/migration.go`; extended recipes cannot be opened there.

## Reservation marks across file lifetimes

[#3666](https://github.com/Oblikovati/Oblikovati/issues/3666) stores the history-owned
`derivedTableHighWater` in schema-3 model metadata. Save and Save Copy preparation
receive that mark separately from the undo snapshot. Failed storage does not adopt
the candidate mark or clear the dirty baseline. Successful save records both the
recipe and reservation baseline; a changed mark alone requires saving.

Open and creation from a package restore the counter before edits. A missing mark
uses the largest live table ID for legacy files. Marks must be signed integers,
nonnegative, and at least the largest live or already retained reservation. A
schema-2 recipe cannot declare this extension. A never-used table namespace keeps
schema 2; saving a used namespace promotes to schema 3 even after all tables were
removed. The schema-3 recipe remains an unreleased Rust format; this adds durable
allocation state under the identity rules of ADR-0005 without changing Go import.

Tests cover add/undo/close-saving/reopen, deletion/Save Copy/reopen, package-based
creation, dirty-state handling, failed atomic replacement, mark regression,
invalid types and ranges, and signed exhaustion. Parameter identity persistence is covered by the schema-3 bindings below.

## Parameter identities across file lifetimes

[#3667](https://github.com/Oblikovati/Oblikovati/issues/3667) adds `parameterIds`
(name-to-ID bindings) and `parameterHighWater` to the unreleased schema-3 recipe.
Both fields must occur together, name exactly the live parameters, and use unique
nonzero IDs within the saved unsigned reservation mark. Legacy records without
either field retain record-order allocation when imported. Derived tables, groups,
expressions, and other name-based sections reconnect to those restored identities.

Saving a used parameter namespace promotes the file to schema 3. Save preparation
receives both counters from the history owner; neither is an undo-snapshot field.
Parameter reservations also contribute to dirty state, including IDs consumed by
rejected edits. Save Copy retains the counters without clearing the original dirty
baseline. Schema-3 parameter capture updates renamed bindings and retains the mark.

Tests cover nonsequential IDs, bindings in a different order from records, renamed
parameters, empty live sets with deleted reservations, missing/extra/duplicate
bindings, zero/out-of-range IDs, signed-invalid marks, unsigned exhaustion, and
add/undo/save-or-copy/reopen through the typed API. Legacy schema-2 imports still
work; the full Rust port and native application acceptance are not complete.

## Remaining work

Complete the general document reference graph beyond parameter-table links.
Direct low-level file owner calls still require explicit refresh; automatic hooks
are at the API command boundary. Geometry integration and native live application
acceptance remain pending.
