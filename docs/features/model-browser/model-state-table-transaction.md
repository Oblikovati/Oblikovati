# Model-state table transaction design

Issue: #4119. Status: draft; no public API release.

The Properties, Suppression and Parameters tabs edit one table. A user can stage
changes on more than one tab before selecting OK. The host must commit that
whole draft as one document transaction. A per-tab commit can leave partial
changes or more than one undo entry.

## Public contract

Add one typed `modelStates.getTable` query and one typed `modelStates.setTable`
command to the API repository. Replace the unreleased property-only draft with
these methods. Keep the Go contract, wire DTOs, Go client and generated Rust
bindings in the API repository. The application must not declare its own DTOs.

The query returns the document generation; ordered member rows; ordered visible
property, feature-suppression and parameter columns; available items; and
retained hidden columns. Each column has one cell for each row, in row order.
Use stable member, feature and parameter identities, not their display names.
A cell carries its authored value or an explicit inherited state; an inherited
empty property differs from an authored empty string. Parameter cells retain
the authored expression and expose its evaluation health separately from the
expression. Suppression cells distinguish Compute and Suppress.

The set request contains the complete staged table and the generation sampled
by the query. Existing member rows use persistent identities. Inserted rows use
unique client draft keys, not document identities; the response maps those keys
to the identities reserved by the host. Reordered and renamed rows retain their
persistent identities. The Primary row stays first and cannot be deleted or renamed. Hidden
columns retain their authored member values unless an explicit delete operation
says otherwise.

One request covers all categories, member insertions, deletions, renames and
ordering. The host must reject duplicate identities, missing or duplicate cells,
unknown columns, invalid values, invalid member names, stale generation and
references to deleted rows before adoption. The response is prepared against
the validated candidate before the transaction is committed. Cancellation sends
no set request.

### Proposed request and response shapes

A row reference contains exactly one of a persistent `memberId` or a nonempty
client `draftKey`. The latter is scoped to one request and is never persisted.
Each requested row has a reference and a name. A new row also has a source row
reference. Sources must resolve to an existing row or to an earlier new row;
cycles and duplicate draft keys are errors. Each accepted new row is returned
with both its draft key and reserved persistent identity.

The table has one ordered sequence of column references. Each reference has a
kind and exactly one matching identity: a property set/name, a feature ID, or
a parameter ID. The kind is a public API enum defined once in `types`. The
kind and identity form a unique key. A separate complete set of typed property,
suppression and parameter column records carries each column's `visible` state,
inherited base and per-row cells. The ordered sequence contains each visible
column exactly once; hidden retained columns are absent from that sequence but
remain in their typed collection. This preserves a mixed grid order without
using an untyped union or losing hidden values.

Property cells contain an optional typed property value. Suppression cells
contain an optional Boolean override; no override inherits the authored feature
state. Parameter cells contain an optional expression override; no override
inherits the base expression. A query also returns evaluated parameter values
and health diagnostics for display. Evaluated values and diagnostics are
read-only and are never accepted as authored inputs. A write therefore contains
only row edits, column order, visibility, base references and overrides. It does
not trust a client-supplied solved value, feature result or health status.

The query advertises available property, feature and parameter identities and
the current active row. The set request echoes the generation and complete
staged rows, order and typed columns. The active row is not changed by a table
write except when its row is deleted. An omitted collection is invalid; an
explicit empty collection means no visible or retained columns of that kind.
The response contains the newly projected table plus the draft-key-to-member-ID
mapping. The host checks every echoed inherited base against its source and
rejects changed bases; otherwise a stale client could silently rebase overrides.

## Host and document model

Store member-specific properties, feature suppression and parameter expressions
in the document-owned model-state recipe. Keep the visible column selection and
the retained values independently: removing a column does not remove the model
value. Copy or insertion must explicitly identify the source member and copy its
values. Deletion removes only the deleted member's values. Names, identities,
ordering, inherited values and overrides survive native YAML save and reopen.

Create one private candidate from the complete request. Validate row identity,
column identity, expression syntax, parameter dependency graphs and feature
references against the same source generation. Evaluate each affected member
without changing the live active member. Recompute the active member's model,
feature health and properties from that candidate. Refuse the whole command if
required validation or active-member recomputation fails; do not publish a
partly updated document. Preserve the current active member unless the request
explicitly removes it, in which case select Primary as part of the same edit.

Publish the validated candidate in one history transaction. Undo and redo move
all row, column and value changes together. Reserve new member identities through
the normal history allocator so undo cannot cause an identity to be reused.
Switching model states projects the selected member's property and parameter
values and feature suppression into the active document before replay and scene
refresh. Ordinary edits outside the table still follow Single Member or All
Members scope.

The current suppression model resolves only the active member, and parameter
expressions are currently document-wide. The table work must add explicit
per-member resolution and persistence before enabling those editing tabs. It
must use the existing feature replay path, including naming and failure handling;
it must not maintain a second geometric model in the UI.

## Atomic replay and dependent documents (#4122)

The source edit, the evaluated part and every affected derived document form
one publication. A successful replay is not enough if its result is discarded
and the host runs a second, fallible refresh after the history commit.

Prepare the source candidate and capture its full named replay, including the
first feature. Check feature health before accepting it. Build dependent
candidates in dependency order from the prepared source and upstream candidates,
not from the old live snapshots. Validate all dependent transitions while their
owners are exclusively held. A failed replay or dependent refresh discards all
prepared results and leaves history and displayed geometry unchanged.

Ask the source transaction observer against the private source candidate before
changing live history. An aborted live edit advances the history generation and
can invalidate the old retained bodies, even when the old recipe is restored.
A veto must discard all candidates and leave the live generation unchanged.
After the observer accepts, publish only through
operations that cannot fail: commit the validated source transition, adopt the
exact captured bodies and checkpoints, then publish the validated dependent
snapshots. Notify observers only after the complete publication. The existing
post-commit partial derived refresh is not suitable for this transaction.

The transaction tests must inject failures in body adoption, dependent refresh
and the observer veto. Each test must compare source history and geometry and
every affected dependent snapshot with their values before the command. The
success test must check the exact validated replay, one undo/redo step, and
native save and reopen. Keep the table write unregistered until these tests pass.

## Desktop behavior

The dockable table owns an uncommitted draft. Switching tabs changes selection
panes only; all staged columns remain visible in the grid. The host supplies the
same typed query to GUI and API clients. OK sends one generation-checked command;
Cancel and the close button discard the draft. A stale or rejected draft remains
open with a clear error and cannot overwrite the newer document.

Keep invalid expression text in the draft and mark the affected cell. Verify
runs validation without committing. OK reports invalid expressions and leaves
the table open. A valid accepted draft closes only after host acknowledgment.
Do not expose tabs or editing controls whose host behavior is absent.

Cache read projections by document identity and generation. Do not perform
member replay, validation or serialization on each rendered frame or keystroke.
Run expensive verification on demand and allow the window to remain responsive.

## Evidence required

- Contract parity and typed-client tests cover full mixed requests, explicit
  inheritance, retained hidden columns, stable IDs and rejected stale drafts.
- Host tests cover a mixed property, suppression, parameter and row change as
  one undoable edit; failed validation must leave the whole document unchanged.
- Per-member feature and parameter evaluation tests cover activation, copying,
  insertion, deletion, naming and recompute failures.
- Native YAML tests and an actual save, close and reopen cover all categories,
  expression text, member identities, hidden columns and active-member state.
- Native mouse and keyboard tests exercise category switching, cancel, accept,
  invalid expressions and undo/redo, with real scene changes for suppression.
- The current hash of each scenario source and a successful complete feature run
  are required before the feature inventory records the feature as done.
