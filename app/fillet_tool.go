// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/api/types"
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
	variable        bool // variable mode: each edge blends startRadius → endRadius (#323)
	startRadius     float64
	endRadius       float64
	midPoints       []FilletMidPoint            // optional intermediate radius stops along each edge (#695)
	cornerType      feature.FilletCornerType    // shared-corner treatment (miter default)
	concaveStrategy types.FilletConcaveStrategy // concave edges: outward fill (default) or inward recess
	crossSection    feature.FilletCrossSection  // blend cross-section: arc (default), G2, or conic (#1284)
	rho             float64                     // conic fullness (0<ρ<1, 0.5 = parabola)
	added           *feature.PartFeature
}

// NewFilletTool returns a fillet tool with a default 1-unit radius, mitered corners, and outward
// concave fill.
func NewFilletTool() *FilletTool {
	return &FilletTool{radius: 1, startRadius: 1, endRadius: 1, concaveStrategy: types.FilletConcaveOutward, rho: 0.5}
}

// filletCrossOrder maps the UI option index to the cross-section enum.
var filletCrossOrder = []feature.FilletCrossSection{feature.FilletArc, feature.FilletG2, feature.FilletConic}

// FilletCrossSectionOptions are the cross-section labels for the property panel, in index order.
func FilletCrossSectionOptions() []string {
	return []string{"Circular (G1)", "Curvature (G2)", "Conic"}
}

// CrossSectionIndex returns the selected cross-section as a [FilletCrossSectionOptions] index.
func (t *FilletTool) CrossSectionIndex() int {
	for i, c := range filletCrossOrder {
		if c == t.crossSection {
			return i
		}
	}
	return 0
}

// SetCrossSectionIndex selects the cross-section from a [FilletCrossSectionOptions] index.
func (t *FilletTool) SetCrossSectionIndex(i int) {
	if i >= 0 && i < len(filletCrossOrder) {
		t.crossSection = filletCrossOrder[i]
	}
}

// Rho/SetRho get and set the conic cross-section's fullness (0<ρ<1; only used when Conic is selected).
func (t *FilletTool) Rho() float64     { return t.rho }
func (t *FilletTool) SetRho(r float64) { t.rho = r }

// filletConcaveOrder maps the UI option index to the concave-strategy enum (0 outward, 1 inward).
var filletConcaveOrder = []types.FilletConcaveStrategy{types.FilletConcaveOutward, types.FilletConcaveInward}

// FilletConcaveOptions are the concave-edge strategy labels for the property panel, in index order.
func FilletConcaveOptions() []string { return []string{"Outward (fill)", "Inward (recess)"} }

// ConcaveStrategyIndex returns the selected concave strategy as a [FilletConcaveOptions] index.
func (t *FilletTool) ConcaveStrategyIndex() int {
	if t.concaveStrategy == types.FilletConcaveInward {
		return 1
	}
	return 0
}

// SetConcaveStrategyIndex selects the concave strategy from a [FilletConcaveOptions] index.
func (t *FilletTool) SetConcaveStrategyIndex(i int) {
	if i >= 0 && i < len(filletConcaveOrder) {
		t.concaveStrategy = filletConcaveOrder[i]
	}
}

// filletCornerOrder maps the UI option index to the corner-type enum.
var filletCornerOrder = []feature.FilletCornerType{
	types.FilletCornerMiter, types.FilletCornerSetback, types.FilletCornerRound,
}

// FilletCornerOptions are the corner-treatment labels for the property panel, in index order.
func FilletCornerOptions() []string { return []string{"Miter (crease)", "Setback", "Round (sphere)"} }

// CornerTypeIndex returns the selected corner treatment as a [FilletCornerOptions] index.
func (t *FilletTool) CornerTypeIndex() int {
	for i, c := range filletCornerOrder {
		if c == t.cornerType {
			return i
		}
	}
	return 0
}

// SetCornerTypeIndex selects the corner treatment from a [FilletCornerOptions] index.
func (t *FilletTool) SetCornerTypeIndex(i int) {
	if i >= 0 && i < len(filletCornerOrder) {
		t.cornerType = filletCornerOrder[i]
	}
}

// Name implements [Tool].
func (t *FilletTool) Name() string { return "Fillet" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *FilletTool) Start(*Session) {}

// AcceptedKinds declares fillet picks edges (the edges to round).
func (t *FilletTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectEdge} }

// Picks reports the picked edges for the unified highlight.
func (t *FilletTool) Picks() []Selectable { return edgeSelectables(t.Edges()) }

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

// SetVariable/Variable toggle the variable-radius mode: each picked edge blends from
// StartRadius to EndRadius instead of a constant Radius (#323).
func (t *FilletTool) SetVariable(v bool) { t.variable = v }
func (t *FilletTool) Variable() bool     { return t.variable }

// SetStartRadius/StartRadius and SetEndRadius/EndRadius set the variable blend's end radii.
func (t *FilletTool) SetStartRadius(r float64) { t.startRadius = r }
func (t *FilletTool) StartRadius() float64     { return t.startRadius }
func (t *FilletTool) SetEndRadius(r float64)   { t.endRadius = r }
func (t *FilletTool) EndRadius() float64       { return t.endRadius }

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

// CanCommit reports whether at least one edge is selected and the active mode's radii
// are positive.
func (t *FilletTool) CanCommit() bool {
	if t.variable {
		return t.EdgeCount() > 0 && t.startRadius > 0 && t.endRadius > 0 && t.midPointsValid()
	}
	return t.EdgeCount() > 0 && t.radius > 0
}

// variableSets builds one variable edge set per key, each blending start → end through any
// intermediate radius stops (#323, #695 — a variable set carries exactly one edge, so corners
// stay constant-radius blends). Every set shares the same stops (read-only closures).
func (t *FilletTool) variableSets(keys [][]byte) []feature.FilletEdgeSet {
	r0, r1 := t.startRadius, t.endRadius
	mids := t.radiusPoints()
	sets := make([]feature.FilletEdgeSet, len(keys))
	for i, k := range keys {
		sets[i] = feature.FilletEdgeSet{
			EdgeKeys:     [][]byte{k},
			StartRadius:  func() float64 { return r0 },
			EndRadius:    func() float64 { return r1 },
			RadiusPoints: mids,
		}
	}
	return sets
}

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
	t.added = t.addFillet(feature.NewDressUpFeatures(part.Features()))
	part.Recompute()
	s.recordEdit(part, "Fillet")
	if !t.added.Health().OK() {
		return errors.New("fillet: " + t.added.Health().Reason)
	}
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *FilletTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature returns the unattached fillet feature the viewport previews before commit
// (satisfying DraftPreviewable), built by the same addFillet the commit uses — so the
// translucent preview is exactly what OK creates. Empty until an edge is selected.
func (t *FilletTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addFillet(feature.NewDressUpFeatures(fs)), nil
	})
}

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
}

// addFillet appends the picked edges in the active mode: the legacy constant-radius
// form, or one variable set per edge.
func (t *FilletTool) addFillet(dress *feature.DressUpFeatures) *feature.PartFeature {
	var pf *feature.PartFeature
	if t.variable {
		pf = dress.AddFilletSetsCorner(t.variableSets(t.selectedEdgeKeys()), t.cornerType)
	} else {
		r := t.radius
		pf = dress.AddFilletCorner(t.selectedEdgeKeys(), func() float64 { return r }, t.cornerType)
	}
	def := pf.Definition().(*feature.FilletFeature).Definition()
	def.ConcaveStrategy = t.concaveStrategy
	def.CrossSection = t.crossSection
	def.Rho = t.rho
	// Mint-time anchors are captured by the AddFillet* builder against the running body (ADR-0043
	// P6b), uniformly with every other authoring path — no per-tool capture needed here.
	return pf
}

// commitEdit writes the panel state back into the committed fillet's definition: the
// constant form clears any edge sets (the legacy fields take over), the variable form
// rewrites the sets.
func (t *FilletTool) commitEdit(s *Session) error {
	def := t.target.Definition().(*feature.FilletFeature).Definition()
	def.CornerType = t.cornerType
	def.ConcaveStrategy = t.concaveStrategy
	if t.variable {
		def.EdgeKeys, def.Radius, def.EdgeSets = nil, nil, t.variableSets(t.selectedEdgeKeys())
		return commitFeatureEdit(s, t.target)
	}
	def.EdgeKeys, def.Radius, def.EdgeSets = t.selectedEdgeKeys(), konst(t.radius), nil
	def.EdgeAnchors = edgeHandleAnchors(t.edges)
	return commitFeatureEdit(s, t.target)
}

// ClearEdges empties the edge selection — the picks and, in edit mode, the feature's
// retained keys — returning the tool to its pick-edges step.
func (t *FilletTool) ClearEdges() {
	t.edges = nil
	t.seededEdgeKeys = nil
}
