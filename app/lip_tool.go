// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// LipTool is the interactive Lip command (#1076, plastic features): click one or more edges,
// set the bead width and height in the property window, choose lip (raised) or groove (cut),
// then OK to run the bead along them. Width sizes the bead along the first adjacent face,
// height along the second.
type LipTool struct {
	edges  []EdgeHandle
	width  float64
	height float64
	groove bool
	added  *feature.PartFeature
}

// NewLipTool returns a lip tool defaulting to a raised 1×1 bead.
func NewLipTool() *LipTool { return &LipTool{width: 1, height: 1} }

// Name implements [Tool].
func (t *LipTool) Name() string { return "Lip" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *LipTool) Start(*Session) {}

// AcceptedKinds declares lip picks edges (the edges to build the lip/groove along).
func (t *LipTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectEdge} }

// Picks reports the picked edges for the unified highlight.
func (t *LipTool) Picks() []Selectable { return edgeSelectables(t.Edges()) }

// Pick appends the clicked edge (ignoring one already chosen, so a double-click does not
// duplicate it).
func (t *LipTool) Pick(_ *Session, sel Selectable) {
	e, ok := sel.(EdgeHandle)
	if !ok || t.hasEdge(e) {
		return
	}
	t.edges = append(t.edges, e)
}

func (t *LipTool) hasEdge(e EdgeHandle) bool {
	for _, h := range t.edges {
		if h == e {
			return true
		}
	}
	return false
}

// The options the property window drives: width, height (database units) and groove (cut vs raise).
func (t *LipTool) SetWidth(v float64)  { t.width = posOrKeep(v, t.width) }
func (t *LipTool) Width() float64      { return t.width }
func (t *LipTool) SetHeight(v float64) { t.height = posOrKeep(v, t.height) }
func (t *LipTool) Height() float64     { return t.height }
func (t *LipTool) SetGroove(v bool)    { t.groove = v }
func (t *LipTool) Groove() bool        { return t.groove }

// Params exposes the bead width, height and groove flag for the generic property dialog.
func (t *LipTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{
			{Label: "Width", Get: t.Width, Set: t.SetWidth},
			{Label: "Height", Get: t.Height, Set: t.SetHeight},
		},
		Bools: []BoolParam{{Label: "Groove (cut)", Get: t.Groove, Set: t.SetGroove}},
	}
}

// Edges returns the picked edges (for the UI to list/highlight).
func (t *LipTool) Edges() []EdgeHandle { return append([]EdgeHandle(nil), t.edges...) }

// CanCommit reports whether at least one edge is selected and both dimensions are positive.
func (t *LipTool) CanCommit() bool { return len(t.edges) > 0 && t.width > 0 && t.height > 0 }

// Commit runs the bead along the picked edges on the active part and recomputes; a sick
// feature keeps the tool open by returning an error.
func (t *LipTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addLip(feature.NewDressUpFeatures(part.Features()))
	part.Recompute()
	s.recordEdit(part, "Lip")
	if !t.added.Health().OK() {
		return errors.New("lip: " + t.added.Health().Reason)
	}
	return nil
}

// addLip builds the lip feature into dress — shared by Commit and the preview.
func (t *LipTool) addLip(dress *feature.DressUpFeatures) *feature.PartFeature {
	w, h, gr := t.width, t.height, t.groove
	return dress.AddLip(t.selectedEdgeKeys(), func() float64 { return w }, func() float64 { return h }, gr)
}

// selectedEdgeKeys is the reference-key set a commit writes from this session's picks.
func (t *LipTool) selectedEdgeKeys() [][]byte {
	keys := make([][]byte, 0, len(t.edges))
	for _, e := range t.edges {
		keys = append(keys, e.Edge.ReferenceKey())
	}
	return keys
}

// DraftFeature returns the unattached lip feature the viewport previews before commit.
func (t *LipTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addLip(feature.NewDressUpFeatures(fs)), nil
	})
}

// Cancel restores the default selection filter.
func (t *LipTool) Cancel(*Session) {}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *LipTool) AddedFeature() *feature.PartFeature { return t.added }
