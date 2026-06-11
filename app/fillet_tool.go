// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// FilletTool is the interactive Fillet command: activate it, click one or more convex edges,
// set the radius in the property window, and OK to round them. Each picked edge becomes a
// rolling-ball cylinder blend on the active part.
type FilletTool struct {
	featureEditMode // set ⇒ this panel re-edits a committed fillet (see editFilletTool)
	edges           []EdgeHandle
	seededEdgeKeys  [][]byte // edit mode: the feature's existing edge keys (their edges are consumed, so no live handles exist)
	radius          float64
	added           *feature.PartFeature
}

// NewFilletTool returns a fillet tool with a default 1-unit radius.
func NewFilletTool() *FilletTool { return &FilletTool{radius: 1} }

// Name implements [Tool].
func (t *FilletTool) Name() string { return "Fillet" }

// Start sets the selection filter to edges so clicks pick edges to round.
func (t *FilletTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectEdge)) }

// Pick appends the clicked edge (ignoring one already chosen).
func (t *FilletTool) Pick(_ *Session, sel Selectable) {
	e, ok := sel.(EdgeHandle)
	if !ok || t.hasEdge(e) {
		return
	}
	t.edges = append(t.edges, e)
}

func (t *FilletTool) hasEdge(e EdgeHandle) bool {
	for _, h := range t.edges {
		if h == e {
			return true
		}
	}
	return false
}

// SetRadius/Radius set the fillet radius (database units).
func (t *FilletTool) SetRadius(r float64) { t.radius = r }
func (t *FilletTool) Radius() float64     { return t.radius }

// Edges returns the picked edges (for the UI to list/highlight).
func (t *FilletTool) Edges() []EdgeHandle { return append([]EdgeHandle(nil), t.edges...) }

// EdgeCount counts the selection the panel shows: edges picked this session plus, in
// edit mode, the feature's retained edges.
func (t *FilletTool) EdgeCount() int { return len(t.seededEdgeKeys) + len(t.edges) }

// selectedEdgeKeys is the reference-key set a commit writes: the retained keys plus
// this session's picks.
func (t *FilletTool) selectedEdgeKeys() [][]byte {
	keys := cloneKeys(t.seededEdgeKeys)
	for _, e := range t.edges {
		keys = append(keys, e.Edge.ReferenceKey())
	}
	return keys
}

// CanCommit reports whether at least one edge is selected and the radius is positive.
func (t *FilletTool) CanCommit() bool { return t.EdgeCount() > 0 && t.radius > 0 }

// Commit rounds the picked edges on the active part and recomputes; a sick feature (a
// non-convex edge or a radius that overruns the geometry) keeps the tool open via an error.
func (t *FilletTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	r := t.radius
	t.added = feature.NewDressUpFeatures(part.Features()).AddFillet(t.selectedEdgeKeys(), func() float64 { return r })
	part.Recompute()
	s.recordEdit(part, "Fillet")
	if !t.added.Health().OK() {
		return errors.New("fillet: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *FilletTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the fillet steps.
func (t *FilletTool) Prompt(*Session) string {
	if len(t.edges) == 0 {
		return "Click one or more edges to fillet"
	}
	return "Set the radius, then click OK"
}

// Cancel restores the default selection filter.
func (t *FilletTool) Cancel(s *Session) {
	if t.IsEditing() {
		cancelFeatureEdit(s, t.target, t.restoreDef)
		return
	}
	s.Selection().SetFilter(NewSelectionFilter())
}

// commitEdit writes the panel state back into the committed fillet's definition.
func (t *FilletTool) commitEdit(s *Session) error {
	def := t.target.Definition().(*feature.FilletFeature).Definition()
	def.EdgeKeys = t.selectedEdgeKeys()
	def.Radius = konst(t.radius)
	return commitFeatureEdit(s, t.target)
}

// ClearEdges empties the edge selection — the picks and, in edit mode, the feature's
// retained keys — returning the tool to its pick-edges step.
func (t *FilletTool) ClearEdges() {
	t.edges = nil
	t.seededEdgeKeys = nil
}
