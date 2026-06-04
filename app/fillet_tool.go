// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/model/feature"
)

// FilletTool is the interactive Fillet command: activate it, click one or more convex edges,
// set the radius in the property window, and OK to round them. Each picked edge becomes a
// rolling-ball cylinder blend on the active part.
type FilletTool struct {
	edges  []EdgeHandle
	radius float64
	added  *feature.PartFeature
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

// CanCommit reports whether at least one edge is picked and the radius is positive.
func (t *FilletTool) CanCommit() bool { return len(t.edges) > 0 && t.radius > 0 }

// Commit rounds the picked edges on the active part and recomputes; a sick feature (a
// non-convex edge or a radius that overruns the geometry) keeps the tool open via an error.
func (t *FilletTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	keys := make([][]byte, len(t.edges))
	for i, e := range t.edges {
		keys[i] = e.Edge.ReferenceKey()
	}
	r := t.radius
	t.added = feature.NewDressUpFeatures(part.Features()).AddFillet(keys, func() float64 { return r })
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
func (t *FilletTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
