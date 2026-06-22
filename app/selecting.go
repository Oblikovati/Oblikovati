// SPDX-License-Identifier: GPL-2.0-only

package app

// The selection engine: tools DECLARE what they select (Selecting) and REPORT what they have
// picked (Picking), and the Session centrally derives the active selection filter from those
// declarations. This replaces the old pattern where every tool imperatively called
// Selection().SetFilter(...) in Start/Cancel/Commit and exposed bespoke pick accessors — a
// source of inconsistent highlighting and filter bugs (e.g. a tool that forgot to accept a
// kind, or named its accessor so the head could not find it). With these contracts the host
// owns filtering and highlighting uniformly for every tool. See ADR-0041.

// Selecting is implemented by every interactive tool that picks geometry. AcceptedKinds reports
// the selection kinds pickable AT THE TOOL'S CURRENT STEP — it is re-read after each pick, so a
// multi-step tool returns different kinds as it advances (e.g. Extrude: a profile, then, once a
// "to face" extent is chosen, a termination face). An empty result means "no restriction": the
// session falls back to the ambient SelectionFilterState (the user's Selection Filter, #1222).
type Selecting interface {
	AcceptedKinds() []SelectionKind
}

// Picking exposes what a tool has selected so far, as a uniform list — the single contract the
// head reads to highlight a tool's picks (replacing the previous duck-typed accessors). Order is
// the tool's pick order; the head highlights each by its SelectionKind.
type Picking interface {
	Picks() []Selectable
}

// faceSelectables / edgeSelectables / profileSelectables / vertexSelectables widen a tool's typed
// pick slice to []Selectable for its Picks() implementation, so the engine highlights them
// uniformly without each tool writing a loop.
func faceSelectables(fs []FaceHandle) []Selectable {
	out := make([]Selectable, len(fs))
	for i, f := range fs {
		out[i] = f
	}
	return out
}

func edgeSelectables(es []EdgeHandle) []Selectable {
	out := make([]Selectable, len(es))
	for i, e := range es {
		out[i] = e
	}
	return out
}

func profileSelectables(ps []ProfileHandle) []Selectable {
	out := make([]Selectable, len(ps))
	for i, p := range ps {
		out[i] = p
	}
	return out
}

// installToolFilter derives the active selection filter from the running tool's declaration. A
// tool implementing Selecting with a non-empty AcceptedKinds gets exactly those kinds; an empty
// declaration (or a tool that does not implement Selecting) leaves the filter unrestricted so the
// ambient SelectionFilterState applies via pickFilter. Called on tool start and after every pick
// so per-step kind changes take effect immediately.
func (s *Session) installToolFilter() {
	if s.tool == nil {
		return
	}
	sel, ok := s.tool.tool.(Selecting)
	if !ok {
		return // a tool that does not declare its selection manages its own filter (or none)
	}
	if kinds := sel.AcceptedKinds(); len(kinds) > 0 {
		s.selection.SetFilter(NewSelectionFilter(kinds...))
		return
	}
	s.restoreSelectionFilter()
}

// restoreSelectionFilter returns the stored filter to unrestricted (accept-all). It is NOT a
// reset to "select anything": when no tool is active, pickFilter layers the user's ambient
// SelectionFilterState on top of an unrestricted filter, so this hands selection back to the
// ambient setting. The Session calls it when a tool ends (commit/cancel).
func (s *Session) restoreSelectionFilter() { s.selection.SetFilter(NewSelectionFilter()) }

// ToolPicks returns the active tool's current picks — the uniform source the head highlights,
// from the Picking contract. A tool that does not implement Picking contributes nothing (its
// feedback, if any, comes from a dedicated overlay).
func (s *Session) ToolPicks() []Selectable {
	if s.tool == nil {
		return nil
	}
	if p, ok := s.tool.tool.(Picking); ok {
		return p.Picks()
	}
	return nil
}
