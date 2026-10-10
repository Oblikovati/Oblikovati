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
caller. In particular, an unavailable exported parameter currently causes an
explicit property-projection failure; no successful refresh is claimed in that
case. Automatic propagation must surface this failure in document health.

Tests cover health-only changes, value changes, no-ops, property updates and
projection rollback, open groups, missing tables/owners, unavailable sources,
self-sources, unchanged identity reservations, and preserved redo/save markers.
Navigating target history restores its recorded snapshot; the scheduler must run
refresh afterward to update its derived values from current sources. A test
exercises that explicit navigation-then-refresh sequence.

## Remaining work

Connect table creation and link editing to source resolution, stable ID
reservation, source parameter export, and the document reference graph. Propagate
source edits, close/reopen, and undo/redo through that graph with cycle handling.
Use the checked source-value boundary for those operations. No live
cross-document recompute or geometry integration is claimed by these read paths.
