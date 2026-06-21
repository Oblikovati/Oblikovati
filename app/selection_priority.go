// SPDX-License-Identifier: GPL-2.0-only

package app

// Selection priority — Inventor's S11: with no tool active, the priority biases what an ambiguous
// viewport pick (a click or a box-select) prefers. It maps to the filter the no-tool pick honours,
// so the same setting drives both click selection and box-select granularity (#922): Part priority
// selects whole bodies/components, Face faces, Edge edges, and General accepts anything (the
// context default). A tool, when active, sets its own filter, which always wins.

// SelectionPriority biases no-tool picking toward a kind of geometry.
type SelectionPriority uint8

const (
	// PriorityGeneral accepts any geometry — the default, context-driven pick.
	PriorityGeneral SelectionPriority = iota
	// PriorityPart selects whole bodies (and components in an assembly).
	PriorityPart
	// PriorityFace selects faces.
	PriorityFace
	// PriorityEdge selects edges.
	PriorityEdge
)

// SelectionPriority returns the active no-tool selection priority.
func (s *Session) SelectionPriority() SelectionPriority { return s.selectionPriority }

// SetSelectionPriority sets the no-tool selection priority. The four priorities are presets of
// the richer SelectionFilterState (#1222): they enable just the kinds the priority accepts, so
// the combo on the View tab and the Selection Filter window stay a single source of truth.
func (s *Session) SetSelectionPriority(p SelectionPriority) {
	s.selectionPriority = p
	s.selectionFilterState.applyPriorityPreset(p)
}

// applyPriorityPreset enables exactly the kinds the priority accepts (General enables all). It
// reuses priorityFilter so the preset→kinds mapping is defined once.
func (st *SelectionFilterState) applyPriorityPreset(p SelectionPriority) {
	if p == PriorityGeneral {
		st.EnableAll()
		return
	}
	st.DisableAll()
	for _, k := range st.order {
		if priorityFilter(p).Accepts(k) {
			st.SetEnabled(k, true)
		}
	}
}

// pickFilter is the filter a viewport pick honours. An active tool's filter, or any explicitly
// restricted filter, always wins; only when nothing has restricted it (no tool, default
// accept-all filter) does the user's ambient SelectionFilterState apply. The no-tool click
// (Session.Pointer) and box-select consult it so the filter/priority biases both.
func (s *Session) pickFilter() *SelectionFilter {
	if s.tool == nil && !s.selection.Filter().IsRestricted() {
		return s.selectionFilterState.Filter()
	}
	return s.selection.Filter()
}

// priorityFilter maps a selection priority to the kinds a no-tool pick accepts.
func priorityFilter(p SelectionPriority) *SelectionFilter {
	switch p {
	case PriorityPart:
		return NewSelectionFilter(SelectBody, SelectOccurrence)
	case PriorityFace:
		return NewSelectionFilter(SelectFace)
	case PriorityEdge:
		return NewSelectionFilter(SelectEdge)
	default:
		return NewSelectionFilter() // General: accept everything
	}
}
