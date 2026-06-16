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

// SetSelectionPriority sets the no-tool selection priority.
func (s *Session) SetSelectionPriority(p SelectionPriority) { s.selectionPriority = p }

// pickFilter is the filter a viewport pick honours. An active tool's filter, or any explicitly
// restricted filter, always wins; only when nothing has restricted it (no tool, default
// accept-all filter) does the selection priority apply. The no-tool click (Session.Pointer) and
// box-select consult it so a priority biases both.
func (s *Session) pickFilter() *SelectionFilter {
	if s.tool == nil && !s.selection.Filter().IsRestricted() {
		return priorityFilter(s.selectionPriority)
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
