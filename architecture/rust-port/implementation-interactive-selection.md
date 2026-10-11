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
model mutation dependency. It is not yet bound to the viewport or the mixed
sketch deletion command, and does not claim foreground UI acceptance.
