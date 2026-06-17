// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/api/types"
	"oblikovati.org/model/feature"
)

// ChamferTool is the interactive Chamfer command: activate it, click one or more convex
// edges, set the setback distance in the property window, and OK to bevel them. Each
// picked edge becomes a wedge cut on the active part.
type ChamferTool struct {
	featureEditMode // set ⇒ this panel re-edits a committed chamfer (see editChamferTool)
	edges           []EdgeHandle
	seededEdgeKeys  [][]byte // edit mode: the feature's existing edge keys (their edges are consumed, so no live handles exist)
	distance        float64
	flatCorners     bool
	concaveStrategy types.ChamferConcaveStrategy // concave edges: outward fill (default) or inward relief
	added           *feature.PartFeature
}

// NewChamferTool returns a chamfer tool with a default 1-unit setback, flat three-edge corners,
// and outward concave fill (all overridden from the session preferences in Start).
func NewChamferTool() *ChamferTool {
	return &ChamferTool{distance: 1, flatCorners: true, concaveStrategy: types.ChamferConcaveOutward}
}

// Name implements [Tool].
func (t *ChamferTool) Name() string { return "Chamfer" }

// Start sets the selection filter to edges and seeds the corner treatment and concave-edge
// strategy from the session's chamfer preferences.
func (t *ChamferTool) Start(s *Session) {
	t.flatCorners = s.ChamferFlatCorners()
	t.concaveStrategy = s.ChamferConcaveStrategy()
	s.Selection().SetFilter(NewSelectionFilter(SelectEdge))
}

// SetFlatCorners/FlatCorners choose whether a vertex where three picked edges meet is
// blended into a flat triangular face (true) or left pointy (false).
func (t *ChamferTool) SetFlatCorners(flat bool) { t.flatCorners = flat }
func (t *ChamferTool) FlatCorners() bool        { return t.flatCorners }

// concaveStrategyNames orders the concave-strategy combo: index 0 outward, 1 inward.
var concaveStrategyNames = []types.ChamferConcaveStrategy{types.ChamferConcaveOutward, types.ChamferConcaveInward}

// ConcaveStrategyNames labels the concave-edge combo (the UI renders these in index order).
func ConcaveStrategyNames() []string { return []string{"Outward (fill)", "Inward (relief)"} }

// ConcaveStrategyIndex / SetConcaveStrategyIndex expose the concave-edge strategy as a combo
// index (0 outward, 1 inward) for the property panel, mirroring the relief-shape combos.
func (t *ChamferTool) ConcaveStrategyIndex() int {
	if t.concaveStrategy == types.ChamferConcaveInward {
		return 1
	}
	return 0
}

// SetConcaveStrategyIndex selects the concave-edge strategy from the combo index (out of range
// is ignored, keeping the current strategy).
func (t *ChamferTool) SetConcaveStrategyIndex(i int) {
	if i >= 0 && i < len(concaveStrategyNames) {
		t.concaveStrategy = concaveStrategyNames[i]
	}
}

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

// EdgeCount counts the selection the panel shows: edges picked this session plus, in
// edit mode, the feature's retained edges.
func (t *ChamferTool) EdgeCount() int { return len(t.seededEdgeKeys) + len(t.edges) }

// selectedEdgeKeys is the reference-key set a commit writes: the retained keys plus
// this session's picks.
func (t *ChamferTool) selectedEdgeKeys() [][]byte {
	keys := cloneKeys(t.seededEdgeKeys)
	for _, e := range t.edges {
		keys = append(keys, e.Edge.ReferenceKey())
	}
	return keys
}

// CanCommit reports whether at least one edge is selected and the distance is positive.
func (t *ChamferTool) CanCommit() bool { return t.EdgeCount() > 0 && t.distance > 0 }

// Commit bevels the picked edges on the active part and recomputes; a sick feature keeps
// the tool open by returning an error.
func (t *ChamferTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	d := t.distance
	t.added = feature.NewDressUpFeatures(part.Features()).AddChamferConcave(t.selectedEdgeKeys(), func() float64 { return d }, t.flatCorners, t.concaveStrategy)
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

// Cancel restores the default selection filter (or, in edit mode, the definition).
func (t *ChamferTool) Cancel(s *Session) {
	if t.IsEditing() {
		cancelFeatureEdit(s, t.target, t.restoreDef)
		return
	}
	s.Selection().SetFilter(NewSelectionFilter())
}

// commitEdit writes the panel state back into the committed chamfer's definition.
func (t *ChamferTool) commitEdit(s *Session) error {
	def := t.target.Definition().(*feature.ChamferFeature).Definition()
	def.EdgeKeys = t.selectedEdgeKeys()
	def.Distance = konst(t.distance)
	def.FlatCorners = t.flatCorners
	def.ConcaveStrategy = t.concaveStrategy
	return commitFeatureEdit(s, t.target)
}

// ClearEdges empties the edge selection — the picks and, in edit mode, the feature's
// retained keys — returning the tool to its pick-edges step.
func (t *ChamferTool) ClearEdges() {
	t.edges = nil
	t.seededEdgeKeys = nil
}
