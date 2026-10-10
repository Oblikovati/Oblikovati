# Typed automation host — partial R2

Tracking: [#3627](https://github.com/Oblikovati/Oblikovati/issues/3627) and
[#3628](https://github.com/Oblikovati/Oblikovati/issues/3628), plus parameter
inspection in [#3629](https://github.com/Oblikovati/Oblikovati/issues/3629) and
collection queries in [#3631](https://github.com/Oblikovati/Oblikovati/issues/3631),
and member detail in [#3632](https://github.com/Oblikovati/Oblikovati/issues/3632).
The `oblikovati-automation` crate provides `SessionQueryHost`, an implementation
of the canonical API client `Caller` transport over an immutable workspace
borrow. All method names, argument types, and result DTOs are imported from the
API crate. `QUERY_METHODS` publishes the exact implemented inventory.

## Implemented inventory

| Method | Context and behavior |
| --- | --- |
| `parameters.getDetail` | Inspect member presentation, engineering value, tolerance, choices, and dependencies. |
| `parameters.getSettings` | Read all document parameter settings and the canonical display mode. |
| `parameters.groups.list` | Read groups in creation order and members in parameter order. |
| `parameters.list` | Inspect parameters in insertion order, including failed parameters. |
| `parameters.get` | Inspect a named parameter's expression, kind, value, and health. |
| `parameters.drivenBy` | Read direct driver names in parameter insertion order. |
| `parameters.dependents` | Read direct reader names in parameter insertion order. |
| `documents.getUnits` | Read active document preferences, display precision/format, and working scale. |
| `units.convert` | Convert compatible physical unit names; no document is required. |
| `units.getTypeFromString` | Resolve a registered unit name to its category without a document. |
| `units.getStringFromType` | Read the active document's preferred unit for a category. |
| `units.getStringFromValue` | Format a stored working-unit value using active document display rules. |
| `units.getPreciseStringFromValue` | Format a stored working-unit value at full precision. |
| `units.getValueFromExpression` | Evaluate nominal parameter values; convert bare numbers through document preferences. |
| `units.getDatabaseUnitsFromExpression` | Evaluate and return the resolved quantity category. |
| `units.isExpressionValid` | Return evaluation or dimension errors as a validity result. |
| `units.compatibleUnits` | Test dimensional compatibility; evaluation failures return false. |
| `units.getDrivingParameters` | Return distinct names in source order without resolving them. |
| `units.getLocaleCorrectedExpression` | Validate canonical syntax and preserve the source text. |

`SessionQueryHost` selects a supported read-only method before executing it. Other methods,
including edits, are refused explicitly. Request/response serialization uses the
canonical types, including their nullable field behavior. A root JSON null maps
to the argument type's default, matching Go decoding; malformed JSON is refused.
Physical conversion and formatting preserve the model's finite-value and
dimension checks. No DTO or numeric-unit implementation is duplicated here.

Forty-six integration tests call the host through the real Rust API `Client`. They
cover all nineteen query methods and thirteen edits, active-document changes, display formats, non-default
working scale, missing document context, dimensional failures, malformed calls,
and unsupported mutation methods. They assert unchanged dirty state, generation,
notification counts, and undo history. The API transport is exercised directly;
these are local Linux tests, not C ABI or native add-in tests.

The host has no session queue, C callback registry, remote transport, or add-in
lifecycle. Those remain R2/R4 work. The limited method
inventory is not evidence that all operations in the API version are implemented.

Expression evaluation uses the existing model evaluator and reads healthy nominal
parameter values, without applying model tolerance offsets. As in Go,
`getValueFromExpression` converts only unitless results through the requested
category's preferred unit; already dimensioned values retain their resolved
working-unit value. Callers use `isExpressionValid` or `compatibleUnits` to test
the requested category. Syntax-only calls need no active document and do not
resolve names or require a successful numeric evaluation. Empty dependency lists
serialize as an array, not null.

The expression query tests cover non-default working scale, preferred inches,
nominal parameter references, mismatched dimensions, unresolved names, parse
failures, nonfinite results, and division by zero. The existing Go router unit
service tests also pass on Linux. Native transport and mutation dispatch remain
unimplemented.

Parameter inspection returns precise nominal values, literal text, and Boolean
values. Failed parameters retain their last-good value and report the failure;
a numeric parameter that never evaluated has the Go-compatible unitless zero
value. Health messages identify unresolved names, unavailable external links,
cycles, evaluation failures, blocked dependencies, nonfinite values, or nonnumeric
dependencies. Parameter-kind spellings are shared with package export.
The canonical DTO omits empty neighbor lists; an empty parameter list is an array.
The query tests check order, values, errors, direct links, and unchanged session
state. Parameter value/member mutations remain to be wired.

Collection queries preserve all eight parameter settings fields, including empty
standard-tolerance strings, false flags, and zero angular precision. Display-mode
names share the model conversion used by package export. Group queries return
stable internal names, display names, add-in client IDs, and ordered member names;
the canonical API omits empty group/member lists. Tests cover all five display
modes, all settings fields, empty collections, missing document context, creation
and membership order, and unchanged session state.

Member detail includes annotations, action restrictions, built-in/renamed flags,
custom-property formatting, optional expression choices, and graph neighbors.
Numeric parameters include their selected engineering value and tolerance band;
text and Boolean parameters omit the numeric tolerance. The stored zero tolerance
has the canonical default type. Displayed values remain precise nominal values.
Inspection retains the last-good engineering value of a failed parameter with its
health diagnostic; the model's feature-value read continues to reject that state.
Precision and tolerance vocabularies are shared with persistence. Tests cover
numeric defaults, tolerance offsets, failed values, typed literals, presentation,
choices, graph links, missing context/names, and unchanged session state.

## Transactional settings edits

[#3633](https://github.com/Oblikovati/Oblikovati/issues/3633) adds `SessionHost`, a
synchronous canonical `Caller` over exclusive workspace command access. It routes
known queries to `SessionQueryHost` and publishes supported edits separately in
`EDIT_METHODS`. The first edit is `parameters.setSettings`. An unsupported method
is refused before document mutation. Overlapping synchronous calls return an
explicit busy/reentry error instead of a borrow panic.

Settings updates clone and validate the collection settings, then run through
`SessionCommands::run` with the label `Edit Parameter Settings`. The response is
encoded inside the candidate edit before commit. Session policy controls veto,
property synchronization, notifications, grouped edits, no-op detection, and
undo/redo. No separate host undo log exists. Omitted and null fields preserve
values; explicit false, zero, and empty strings replace them. Root null means an
empty update. Arrays and scalar request roots are refused by the shared decoder.

An owner-borrow regression test checks rejection of overlapping query/edit calls
and successful use after the borrow is released. Six additional integration tests exercise the real typed client and wire calls:
undo/redo, live query results, atomic validation errors, optional/reset fields,
no-op redo preservation, existing groups, missing context, unported edits, and
observer veto rollback. The display-mode parser is shared with package restore.
This is an owner-side transport, not a bounded queue or native callback registry;
those boundaries must be added before native add-in use is claimed.

## Transactional group edits

[#3634](https://github.com/Oblikovati/Oblikovati/issues/3634) adds the five group
edits: `parameters.groups.add`, `delete`, `setDisplayName`, `addMember`, and
`removeMember`. The host decodes each call into a prepared edit before opening its
transaction. Settings and group edits share the same session command and response
encoding path. Labels preserve the Go session's group-operation names.

Group creation preserves add-in provenance and defaults an empty display name to
the internal name. Display renaming keeps the internal key unchanged. Membership
requires both an existing group and parameter; it cannot use the UI's automatic
group-creation path. Removing one membership preserves other groups. Repeated
membership operations are no-ops and do not add undo records.

Deletion preserves parameters unless `deleteParameters` is true. A cascade uses
the model aggregate's dependency checks and refuses deletion that would sicken a
surviving reader. The candidate transaction prevents partial deletion; a successful
cascade is one undo step. The canonical void reply is `{}`. Tests cover the typed
lifecycle, order, independent membership, strict lookup, validation errors,
cascade refusal/success with undo/redo, grouped abort, and observer veto. Existing
settings and reentry tests also run against the shared edit path.

## Parameter identity and lifecycle edits

[#3635](https://github.com/Oblikovati/Oblikovati/issues/3635) adds
`parameters.rename`, `parameters.convert`, and `parameters.delete`. Rename uses
the model's stable-ID rewrite so dependent expressions and group projections
follow the new name. Conversion accepts trimmed, case-insensitive user/model/
reference names, preserves identity and graph links, and refuses built-in or
derived parameters through the model. Deletion uses the dependency-aware public
remove operation, never the owner-only forced removal path.

All three calls use the shared candidate transaction, including exposed-property
synchronization and response encoding. Undo restores parameter names, category,
expressions, group membership, and exposed properties together. No-op rename and
conversion add no undo entry. Tests cover typed responses, dependency rewrites,
stable IDs, conversion rules, invalid names, deletion refusal, undo/redo, veto,
and exposed-property rename/deletion. The selected Go rename/conversion tests
also pass on Linux.

Session transaction observations are active. The Go host's `parameter.changed`
push notifications are not yet delivered to add-ins; that requires the pending
outbound event and native add-in transport boundary. These edit implementations
do not claim full event-surface parity.

## Parameter creation and numeric expression edits

[#3636](https://github.com/Oblikovati/Oblikovati/issues/3636) adds `parameters.add`
and `parameters.set`. Creation reserves identities through the document's
`RecipeReservations`, outside undo snapshots. A failed or undone creation never
makes its reserved ID reusable in the live document. Numeric creation uses the
user category unless the trimmed kind is model, matching the Go router. Text
creation permits empty strings; Boolean creation preserves Go `ParseBool`
spellings. Literal creation always creates user parameters.

Numeric edits qualify bare values with the existing parameter's preferred unit,
including the last-good category of a failed parameter. They use the model graph
to rebind and recompute dependent parameters, and report health when evaluation
fails. Syntax, cyclic edits, read-only edits, invalid creation types, and name
collisions are refused through the candidate transaction. The current model
keeps text/Boolean values typed; numeric expression edits cannot replace those
literal kinds. Their separate typed setter surface remains to be connected.

Tests cover allocation after undo, typed creation, Boolean spellings, model
creation, dependency recomputation, failed-value recovery, invalid input, group
abort, and veto. Model parameters also report `inUse` without parameter readers,
matching the Go model contract. Numeric parameter recomputation is implemented;
feature/geometry recomputation after an edit still requires the pending feature
engine. This does not claim feature or outbound notification completion.

## Value lists and tolerance sweeps

[#3637](https://github.com/Oblikovati/Oblikovati/issues/3637) adds
`parameters.setExpressionList` and `parameters.setAllModelValueType`. List edits
return canonical member detail, retain authored order when requested, sort through
the model otherwise, and clear the list for null/empty expressions. They do not
change the current parameter value. Model editability checks apply to both list
replacement and clearing.

Sweeps accept the canonical nominal/lower/upper/median names and validate every
eligible toleranced parameter before changing any selection. The returned count
includes eligible parameters already at that selection, matching Go. Untoleranced
parameters and nominal expression values remain unchanged; engineering model
values reflect the chosen tolerance bound. Failed sweeps and observer veto leave
all selections unchanged. Identical updates add no undo record. The model sweep
and value-list engines remain the single implementations used by the host.

Tests cover list order/flags/clearing, every sweep selection, eligible counts,
nominal-value preservation, failed inputs, read-only lists, no-op history, undo,
and veto. Feature geometry recomputation remains pending with the feature engine.

## Parameter tolerance edits

[#3638](https://github.com/Oblikovati/Oblikovati/issues/3638) connects
`parameters.setTolerance` to the existing model tolerance and ISO fit operations.
All nine modes return canonical member detail. Expressions resolve against
nominal parameter values; bare literals use document display units. Absolute
limits become deviations from the nominal value. Bandless modes clear fit
annotations and deviations. The selected engineering value follows the new band
without changing nominal expressions.

The adapter explicitly checks operand dimensions after literal parsing. A suffix
can override the parser's default category, but an angle must not become a length
tolerance. This is a stricter validation than the Go resolver's literal fallback.
Malformed values, reversed bands, unsupported fits, and observer veto leave the
snapshot and undo history unchanged. Tests cover all modes, expression and unit
resolution, returned detail, undo/redo, rollback, and veto. Feature recomputation
and outbound parameter notifications remain pending.

## Parameter presentation and exposure updates

[#3639](https://github.com/Oblikovati/Oblikovati/issues/3639) adds
`parameters.update`. Optional fields preserve existing state when omitted or
null; explicit empty strings, false values, zero precision, and empty action
lists reset their fields. Display formats, model-value selections, action masks,
and custom-property enum names use the model's shared vocabulary. A supplied
custom-property format replaces the whole stored format, as in Go.

The session applies annotations, selected tolerance value, and exposure projection
in one transaction. A late invalid field, failed parameter exposure, or observer
veto rolls back the whole update. Undo and redo include the owned document
property. Identical updates do not add history. Custom-property formatting
preferences are stored and returned; the exposure projection currently uses typed
parameter values, not formatted text. This call does not complete downstream
export formatting, feature recomputation, or outbound parameter notifications.

Tests cover optional-field semantics, all display and selection modes, every
custom precision name, action ordering and clearing, property exposure,
undo/redo, invalid late fields, failed exposure, and veto.

## Transaction controls

[#3640](https://github.com/Oblikovati/Oblikovati/issues/3640) adds canonical
`transaction.begin`, `end`, `abort`, `undo`, `redo`, and `state`. Commands use the
active document's existing session history, without an additional automatic
transaction. Nested groups retain the outer label; only the outer end commits.
Abort cancels all nested levels. A failed edit leaves the open batch available
for explicit abort, and a commit veto restores the pre-batch snapshot.

State is also available through the immutable query host. It returns empty
history when no document is active and reports committed undo/redo labels in
Go order. During an open group, the state still describes committed history;
undo/redo commands refuse navigation until the group closes. Transaction commands
cannot run through the query-only host and do not change document focus.

Tests cover nested batching, returned state, undo/redo, no-op groups, malformed
begin requests, empty context/history, failed edits followed by abort, redo
preservation, veto, reentry refusal, and isolation between documents. History
browsing, saved checkpoints, absolute cursor jumps, and native transaction-event
delivery remain separate work.

## Save checkpoints for history browsing

[#3641](https://github.com/Oblikovati/Oblikovati/issues/3641) adds per-step save
markers to the owned history. The session exposes the full chronological stream,
including redo steps, through canonical history-entry values. A successful
workspace save marks the current committed step after file replacement and
identity adoption. A failed replacement creates no marker. Saving the initial
state creates no synthetic entry.

Markers follow their steps through undo/redo. A new branch discards the old redo
steps and their markers; merging new edits into a saved step clears its marker.
This avoids assigning a discarded branch's save evidence to a new step at the
same cursor depth. Marking is refused while a transaction is open. The markers
are session history evidence, separate from the dirty-state recipe comparison.
Tests cover navigation, branch replacement, merged edits, repeated marks, open
transactions, and successful/failed saves through the persistence target.
The wire history query still needs document display-name ownership before it can
return the complete canonical response.

## Named transaction history query

[#3642](https://github.com/Oblikovati/Oblikovati/issues/3642) adds
`transaction.history` to both hosts. Document zero selects the active document;
an explicit ID reads that owner's stream without activation or observer events.
The reply includes chronological committed entries, saved markers, and the
current cursor position. Preview edits are excluded. Empty streams encode an
empty array; missing documents are errors.

The workspace retains the file owner's display name. Open and template creation
use custom package titles when present, otherwise the file basename without its
extension. A persisted title equal to that derived name remains derived, as in
Go. Save As updates derived titles after successful persistence and preserves
custom titles. Derived overrides are cleared in the retained package so saving
and reopening cannot turn an old filename into a custom name. Bare workspace
registrations without file metadata have an empty title.

Tests cover multiple documents, missing context, previews, saved redo entries,
Unicode custom titles, template creation, repeated Save As, close/reopen, and
stale-ID refusal. Absolute cursor jumps and native event delivery remain pending.

## Absolute transaction cursor jumps

[#3643](https://github.com/Oblikovati/Oblikovati/issues/3643) adds
`transaction.jumpTo`. The session checks that no transaction group is open and
that the requested cursor is in the full stream's range before moving. Explicit
document IDs do not change focus; zero targets the active document. Each move
uses the normal undo/redo operation, including observer events, veto, generation
updates, and property snapshots. The reply is the resulting canonical history.

A veto stops at the last accepted step, matching Go; the operation does not hide
partial progress by silently restoring earlier steps. Callers can query history
after an error. A jump to the current cursor is a no-op with no new events or
generation. Even that no-op is refused while a group is open. Tests cover forward
and backward navigation, no-op behavior, explicit targets, invalid bounds and
payloads, preview refusal, and a veto after one accepted undo. Native transport
and event delivery remain pending.

## Document property calls

[#3644](https://github.com/Oblikovati/Oblikovati/issues/3644) adds
`documents.listProperties`, `getProperty`, and `setProperty`. These calls require
an explicit open document ID; zero does not select the active document. Queries
preserve set and property order, canonical typed values, and parameter provenance.
Empty lists retain Go's null-array representation. Writes create custom sets or
replace values without moving them, and record one `Set Property` undo step on
the addressed document without changing focus. Unchanged writes add no history.

The edit dispatcher now passes the whole document snapshot and selects the
request's document before running the normal session transaction. Parameter calls
still use the active document. Validation and observer veto restore the whole
snapshot. Direct writes to parameter-owned properties are refused with the source
parameter named, so the normal exposure refresh cannot silently overwrite a
successful property response. Edit that parameter instead. Empty set names are
also refused by the Rust model. Canonical property variants discard unit metadata,
as in Go; an omitted value follows Go's empty-string default.

Tests cover all supported value kinds, explicit document isolation, ordered
custom/standard sets, replacement, undo/redo, no-ops, invalid requests, owned
property refusal, and veto. Native transport and outbound events remain pending.

## File-owned document discovery

[#3645](https://github.com/Oblikovati/Oblikovati/issues/3645) adds `FileQueryHost`
and `documents.list`. The immutable host reads document information from the
file workspace and delegates existing queries to `SessionQueryHost`. It does not
expose mutable workspace ownership or permit commands through a query transport.
Document IDs retain open order; names and subtypes come from the owned package
and display-name policy; dirty and active flags come from current session state.
An empty workspace returns an empty document array.

Background open preserves focus independently of logical document visibility.
The open API added below supports visible and hidden loaded documents. Deferred
reference-document lifetimes remain pending; visibility is logical state, not
proof of a rendered window. Bare recipe-only sessions do not invent file metadata for
this query. The package's document-kind vocabulary is shared with CLI `info`.

Tests exercise all four document kinds, custom titles/subtypes, creation order,
empty discovery, dirty changes through save/edit/undo, close and background
reopen, delegated parameter/history queries, and command refusal. Native GUI,
mutating file API transport, and complete geometry reconstruction remain pending.

## Mutable file API host

[#3646](https://github.com/Oblikovati/Oblikovati/issues/3646) adds `FileHost`,
borrowing the file owner, GUID source, and application version. It serves existing
queries, parameter/property edits, and all transaction calls, plus
`documents.activate`, `save`, and `close`. A shared private command-owner interface
provides immutable inspection and borrowed session commands to both hosts; it
never exposes a replaceable mutable workspace or duplicates the history engine.

Save uses only the owned path and package. The canonical response is prepared
before I/O, including a UTF-8 path check, so an unrepresentable response cannot
follow a successful write. Close with force discards changes; otherwise the file
owner saves dirty state before closing. Storage failure leaves ownership,
identity, and dirty state intact. Activation changes subsequent active-document
commands without moving recipe state. Synchronous reentry is refused.

Tests exercise API edit/group/commit/save/jump/redo/close followed by file reopen,
save-on-close and discard, write failures, bad IDs, activation, path-response
validation, and reentry. The existing file owner still reconstructs only the
ported recipe sections. Open/create/Save As/copy/batch API calls, native transport,
and full feature reconstruction remain pending.

## Save As and Save Copy As

[#3647](https://github.com/Oblikovati/Oblikovati/issues/3647) connects
`documents.saveAs` and `documents.saveCopyAs` to `FileHost`. Save As resolves and
checks its destination once inside the file owner, validates the canonical UTF-8
response path, and uses the shared save operation before rebinding. Its response
names the resolved destination. A storage or response-path failure preserves the
old binding, identity, and dirty state.

Save Copy As uses the existing independent-copy operation. It applies optional
name/subtype metadata only to the new package, mints a separate file identity,
and preserves the source owner and dirty state. Its response retains the supplied
target filename, matching Go. Copies cannot overwrite an open document's owned
destination. Tests cover resolved aliases, retained old files, persisted edits,
copy metadata and identity, storage failures, occupied destinations, and a
non-UTF-8 resolved Save As path refused before writing. Open/create and batch-save
API routing remain pending.

## Batch-save API routing

[#3648](https://github.com/Oblikovati/Oblikovati/issues/3648) exposes
`documents.batchSave` through `FileHost` using the existing file-owner batch
operation. Save, Save As, and Save Copy As retain ordered per-item results and
success counts. An item failure does not prevent later items from running.
Invalid operations, missing source documents, and duplicate resolved destinations
refuse the whole batch before writing. Empty batches return an empty result array.

API tests cover an open-transaction failure followed by a successful save,
request-order preservation, all three operations, binding and identity behavior,
whole-batch refusal, and empty requests. Underlying persistence remains the same
atomic per-file writer; a batch is not an all-or-nothing multi-file transaction.
Native transport and full geometry reconstruction remain pending. Creation
routing is recorded in [document creation](implementation-document-creation.md).

## Document open API

[#3649](https://github.com/Oblikovati/Oblikovati/issues/3649) connects
`documents.open` to `FileHost`. A new owner obtains its observer from an injected
`DocumentObserverFactory`. The file owner invokes that factory lazily after
canonical cached-path checks. Cached opens preserve unsaved recipe state and do
not reread the package or create another observer. New opens use the existing
validation and identity-registration path.

The API activates the loaded document and applies the requested logical
visibility, matching Go's ordinary open path. Visibility changes do not edit the
recipe or add history. Empty paths and `deferContent=true` are refused before
observer creation; deferred reference stubs are not yet implemented. Missing or
failed factories and malformed packages leave no registered document/path owner.

Tests cover open/edit/cached-open/save-close/reopen, observer creation counts,
cached access without a factory or disk contents, hidden loaded documents,
unchanged dirty/history state for visibility, invalid options, factory failure,
malformed files, and retry. The file owner reconstructs the ported recipe
sections; full feature reconstruction, native transport, and reference stubs
remain pending.

## Close all documents

[#3652](https://github.com/Oblikovati/Oblikovati/issues/3652) routes
`documents.closeAll` through the same owned save/discard operation as individual
close. It snapshots document IDs in open order and stops on the first failure,
with the failed ID in the error. Earlier successful closes remain committed;
this is not an atomic multi-document transaction, matching the Go router.
An empty workspace returns a zero count.

Typed-client tests cover forced discard and recreation, save-before-close for
bound documents, and partial progress when an unbound document needs Save As.
The later documents remain open and unwritten after that failure. The Go
`TestCloseAllDocuments` and `TestCloseDocumentByID` reference tests pass locally.

## Derived parameter table deletion

[#3655](https://github.com/Oblikovati/Oblikovati/issues/3655) connects
`parameters.derivedTables.delete` to the existing model operation through the
shared edit owner. The `Edit Derived Parameters` transaction removes produced
parameters, updates dependent health, and can be undone or redone. Missing tables,
feature-owned tables, and observer vetoes refuse the edit without changing the
recipe or undo history. Typed-client tests verify these cases.

This is the deletion path for retained recipe tables. Source resolution and
enumeration are recorded in [derived tables](implementation-derived-tables.md).
API table creation/link edits and the workspace reference graph remain unported. The local recipe operation does not establish cross-document
recompute or reference-graph parity.

Connected client registration, discovery, and interest availability are recorded
in [client application implementation](implementation-clients.md).
