# Interactive selection engine — partial R4/R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This applies [ADR-0009](../decisions/ADR-0009-ui-and-selection.md).
The session crate now supplies the ordered selection set and shared filter policy
from Go `app/selection.go`, `selection_filter_state.go`, and
`selection_priority.go` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

`SelectionSet<T>` takes typed handles implementing `Selectable` and equality.
The caller must include owner identity in its handles. Equal repeated picks are
ignored, insertion order is retained, removal preserves survivor order, and
Shift/Ctrl toggle removes existing picks even if their category is now blocked.
Changing the filter affects additions, not existing membership. Clear retains
the declared filter. Borrowed slices cannot modify membership; cloned sets have
independent membership and filter state.

`SelectionFilter` distinguishes accept-all, blocked-all, and restricted kinds.
Its empty-input constructor preserves Go's accept-all behavior. These desktop
categories are application-owned classifications, not new wire DTOs or API
entity-kind values. All 23 Go categories are present, including sketch dimension
labels and geometric-constraint markers.

`AmbientSelectionFilter` preserves the 12-category snap-before-depth ordering.
Unlisted categories rank last. Invalid reorder positions do nothing. General,
Part, Face, and Edge presets update the same enabled state without resetting the
user's priority order. Disabling every managed category yields blocked-all;
enabling every managed category restores the unrestricted default. Unlisted
category enablement does not change the window's managed-category policy,
matching the Go implementation.

A running tool's declared filter wins, including its unrestricted filter. An
explicit restriction also wins without a tool. Only the default filter with no
active tool reads the ambient state. This keeps tool picks available when the
user has disabled ambient selection (#1222).

Ten independent Rust tests cover order, owner/category identity, duplicate
picks, removal, toggle, clear, clone isolation, all 23 filter categories, the
fixed default rank order, invalid moves, all four presets, unlisted kinds, and
tool/context precedence. The pinned Go selection filter, priority preset, and
pick-filter tests pass. The full local acceptance command is `make gate`.

Native pick-to-reference mapping, stale presentation rejection, candidate depth
resolution, preselection, highlighting, input modifiers, shell ownership, and
selection-change events remain required. The selection engine has no GPU or
model mutation dependency. It is not yet bound to the viewport and does not claim foreground UI acceptance.
The editor deletion integration below consumes this same selection engine.

## Checked sketch picks and editor deletion

`DocumentSession::sketch_pick` validates a native entity, dimension, or geometric
constraint against its placed sketch before minting a `SketchPick`. Its private
stamp records the session document ID, sketch ID, and current model generation.
The three target variants map to their existing desktop selection categories.
These are transient handles, not a new reference-key format or persisted recipe.

`delete_selected_sketch_picks` accepts the editor's active sketch, tool state,
and typed selection set. No active sketch, an active tool, or empty picks returns
an empty report without a history frame, generation change, or observer event.
For an edit, every pick must match the current document, sketch and generation.
An unrelated edit or undo invalidates old picks even if the same entities exist
again. Foreign document IDs cannot exploit equal sketch/entity IDs. These checks
precede the common document mixed-selection mutation and scalar cleanup.

The action uses the normal session transaction and automatic recompute path.
It preserves Go's four undo labels: Delete Sketch Entities, Dimensions,
Constraints, or Geometry according to the selection categories. A successful
adoption clears membership while retaining the declared filter. Refusal or an
observer veto retains all picks and the model recipe. Selection is transient:
undo, redo, or abort restores model state without recreating cleared picks.
The caller must refresh presentation handles for the new generation.

Borrowed workspace commands mint and delete through the same owner. Fourteen
additional tests cover all target categories, missing IDs, wrong relation
categories, each undo label, no-op contexts, duplicate picks, category filtering,
shared-source cleanup, undo/redo, group abort, observer veto, and stale/foreign
refusal. Both part and assembly file owners delete through the borrowed command,
save/reopen surviving geometry and underconstraint health, and refuse stale
picks after undo without dirtying the restored saved recipe. The pinned Go
Delete-key and selected-sketch deletion tests pass; full validation is `make gate`.

This implements the headless editor action. Native hit testing, modifier input,
selection-change notification, shell lifetime, and the foreground Delete-key
binding remain required. Generation checks on these handles do not establish
stale-renderer mapping or visible highlight acceptance.
