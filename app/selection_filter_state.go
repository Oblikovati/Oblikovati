// SPDX-License-Identifier: GPL-2.0-only

package app

// SelectionFilterState is the user-controlled, no-tool ambient selection setting that the
// Selection Filter & Priority window edits (#1222). It holds, over the viewport-pickable
// SelectionKinds, (a) which kinds are enabled (pickable) and (b) their priority order — the
// kind nearer the top wins an ambiguous pick (see RayPicker, #1222 PBI 2). It is the single
// source of truth for the ambient filter: SelectionPriority presets write into it, and
// Session.pickFilter reads its Filter() when no tool has restricted selection.
//
// Example: disable SelectFace so a click can no longer grab a face, leaving the edge behind
// it pickable.
type SelectionFilterState struct {
	enabled map[SelectionKind]bool
	order   []SelectionKind
}

// defaultFilterableKinds is the ordered set of kinds a viewport pick (RayPicker) can produce,
// in the priority order that reproduces the historical pick behaviour: the exact "snap"
// targets (datum point/axis, cloud point, vertex, edge) outrank the depth-sorted face/plane/
// profile/sketch hits, matching snapPick + the append order in RayPicker.Pick. Reordering in
// the window deviates from this default; until then nothing changes.
func defaultFilterableKinds() []SelectionKind {
	return []SelectionKind{
		SelectWorkPoint,
		SelectWorkAxis,
		SelectPointCloudPoint,
		SelectVertex,
		SelectEdge,
		SelectOccurrence,
		SelectBody,
		SelectFace,
		SelectMeshFace,
		SelectWorkPlane,
		SelectProfile,
		SelectSketchEntity,
	}
}

// NewSelectionFilterState returns the default state: every kind enabled in the default
// priority order, which is equivalent to the context-default accept-all pick.
func NewSelectionFilterState() *SelectionFilterState {
	order := defaultFilterableKinds()
	enabled := make(map[SelectionKind]bool, len(order))
	for _, k := range order {
		enabled[k] = true
	}
	return &SelectionFilterState{enabled: enabled, order: order}
}

// Enabled reports whether picks of kind k are currently allowed.
func (st *SelectionFilterState) Enabled(k SelectionKind) bool { return st.enabled[k] }

// SetEnabled activates or deactivates picking of kind k.
func (st *SelectionFilterState) SetEnabled(k SelectionKind, on bool) { st.enabled[k] = on }

// Order returns a copy of the priority order, top (highest priority) first.
func (st *SelectionFilterState) Order() []SelectionKind {
	out := make([]SelectionKind, len(st.order))
	copy(out, st.order)
	return out
}

// Move relocates the kind at index from to index to, shifting the rest — the drag-and-drop
// reorder of the priority list. Out-of-range or no-op indices leave the order unchanged.
func (st *SelectionFilterState) Move(from, to int) {
	n := len(st.order)
	if from < 0 || from >= n || to < 0 || to >= n || from == to {
		return
	}
	k := st.order[from]
	st.order = append(st.order[:from], st.order[from+1:]...)
	st.order = append(st.order[:to], append([]SelectionKind{k}, st.order[to:]...)...)
}

// EnableAll activates every kind — the window's "Select All" button.
func (st *SelectionFilterState) EnableAll() {
	for _, k := range st.order {
		st.enabled[k] = true
	}
}

// DisableAll deactivates every kind — the window's "Deselect All" button; a click then
// selects nothing (Filter reports a blocking filter).
func (st *SelectionFilterState) DisableAll() {
	for _, k := range st.order {
		st.enabled[k] = false
	}
}

// Rank returns the priority index of kind k (0 = highest); kinds not managed here sort last
// so an unlisted candidate never outranks a listed one during ambiguous-pick resolution.
func (st *SelectionFilterState) Rank(k SelectionKind) int {
	for i, kk := range st.order {
		if kk == k {
			return i
		}
	}
	return len(st.order)
}

// Filter builds the SelectionFilter a no-tool pick honours: accept-all when every kind is
// enabled (the default, equal to the context default), a kind-restricted filter when some are
// disabled, and a blocking filter when none are enabled.
func (st *SelectionFilterState) Filter() *SelectionFilter {
	kinds := make([]SelectionKind, 0, len(st.order))
	for _, k := range st.order {
		if st.enabled[k] {
			kinds = append(kinds, k)
		}
	}
	switch len(kinds) {
	case len(st.order):
		return NewSelectionFilter() // all enabled — accept everything (context default)
	case 0:
		return newBlockingFilter() // Deselect All — accept nothing
	default:
		return NewSelectionFilter(kinds...)
	}
}

// selectionKindLabels are the human labels shown per kind in the Selection Filter window.
var selectionKindLabels = map[SelectionKind]string{
	SelectWorkPoint:       "Work Points",
	SelectWorkAxis:        "Work Axes",
	SelectPointCloudPoint: "Point-Cloud Points",
	SelectVertex:          "Vertices",
	SelectEdge:            "Edges",
	SelectOccurrence:      "Components",
	SelectBody:            "Solid Bodies",
	SelectFace:            "Faces",
	SelectMeshFace:        "Mesh Facets",
	SelectWorkPlane:       "Work Planes",
	SelectProfile:         "Sketch Profiles/Areas",
	SelectSketchEntity:    "Sketch Elements",
}

// SelectionKindLabel is the human label shown for a kind in the Selection Filter window.
func SelectionKindLabel(k SelectionKind) string {
	if label, ok := selectionKindLabels[k]; ok {
		return label
	}
	return "Unknown"
}
