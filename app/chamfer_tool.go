// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// ChamferTool is the interactive Chamfer command: activate it, click one or more convex
// edges, set the setback distance in the property window, and OK to bevel them. Each
// picked edge becomes a wedge cut on the active part.
type ChamferTool struct {
	edges       []EdgeHandle
	distance    float64
	flatCorners bool
	added       *feature.PartFeature
}

// NewChamferTool returns a chamfer tool with a default 1-unit setback and flat three-edge
// corners (overridden from the session preference in Start).
func NewChamferTool() *ChamferTool { return &ChamferTool{distance: 1, flatCorners: true} }

// Name implements [Tool].
func (t *ChamferTool) Name() string { return "Chamfer" }

// Start sets the selection filter to edges and seeds the corner treatment from the
// session's chamfer preference.
func (t *ChamferTool) Start(s *Session) {
	t.flatCorners = s.ChamferFlatCorners()
	s.Selection().SetFilter(NewSelectionFilter(SelectEdge))
}

// SetFlatCorners/FlatCorners choose whether a vertex where three picked edges meet is
// blended into a flat triangular face (true) or left pointy (false).
func (t *ChamferTool) SetFlatCorners(flat bool) { t.flatCorners = flat }
func (t *ChamferTool) FlatCorners() bool        { return t.flatCorners }

// Pick appends the clicked edge (ignoring one already chosen, so a double-click does not
// duplicate it).
func (t *ChamferTool) Pick(_ *Session, sel Selectable) {
	e, ok := sel.(EdgeHandle)
	if !ok || t.hasEdge(e) {
		return
	}
	t.edges = append(t.edges, e)
}

func (t *ChamferTool) hasEdge(e EdgeHandle) bool {
	for _, h := range t.edges {
		if h == e {
			return true
		}
	}
	return false
}

// SetDistance/Distance set the chamfer setback (database units).
func (t *ChamferTool) SetDistance(d float64) { t.distance = d }
func (t *ChamferTool) Distance() float64     { return t.distance }

// Edges returns the picked edges (for the UI to list/highlight).
func (t *ChamferTool) Edges() []EdgeHandle { return append([]EdgeHandle(nil), t.edges...) }

// CanCommit reports whether at least one edge is picked and the distance is positive.
func (t *ChamferTool) CanCommit() bool { return len(t.edges) > 0 && t.distance > 0 }

// Commit bevels the picked edges on the active part and recomputes; a sick feature keeps
// the tool open by returning an error.
func (t *ChamferTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	keys := make([][]byte, len(t.edges))
	for i, e := range t.edges {
		keys[i] = e.Edge.ReferenceKey()
	}
	d := t.distance
	t.added = feature.NewDressUpFeatures(part.Features()).AddChamferCorners(keys, func() float64 { return d }, t.flatCorners)
	part.Recompute()
	s.recordEdit(part, "Chamfer")
	if !t.added.Health().OK() {
		return errors.New("chamfer: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ChamferTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the chamfer steps.
func (t *ChamferTool) Prompt(*Session) string {
	if len(t.edges) == 0 {
		return "Click one or more edges to chamfer"
	}
	return "Set the distance, then click OK"
}

// Cancel restores the default selection filter.
func (t *ChamferTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
