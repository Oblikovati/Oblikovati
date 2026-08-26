// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	stdmath "math"
	"slices"

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
	chamferType     types.ChamferType
	distance        float64
	distance2       float64 // two-distance mode: the setback on the second face
	angle           float64 // distance-and-angle mode: the chamfer-face angle, radians
	flatCorners     bool
	concaveStrategy types.ChamferConcaveStrategy // concave edges: outward fill (default) or inward relief
	tangentChain    bool                         // a plain pick selects the whole tangent chain (#1947)
	added           *feature.PartFeature
}

// NewChamferTool returns a chamfer tool with a default 1-unit setback, flat three-edge corners,
// and outward concave fill (all overridden from the session preferences in Start). It starts in
// the equal-distance mode; the two asymmetric modes seed a matching second setback / 45° so
// switching to one is immediately committable.
func NewChamferTool() *ChamferTool {
	return &ChamferTool{
		chamferType: types.ChamferDistance, distance: 1, distance2: 1, angle: stdmath.Pi / 4,
		flatCorners: true, concaveStrategy: types.ChamferConcaveOutward,
	}
}

// chamferTypeOrder maps the UI option index to the setback mode.
var chamferTypeOrder = []types.ChamferType{
	types.ChamferDistance, types.ChamferTwoDistances, types.ChamferDistanceAndAngle,
}

// ChamferTypeNames labels the setback-mode combo, in index order.
func ChamferTypeNames() []string {
	return []string{"Distance", "Two distances", "Distance and angle"}
}

// ChamferTypeIndex / SetChamferTypeIndex expose the setback mode as a [ChamferTypeNames] index.
// Until #2045 the tool could only ever author the equal-distance mode, though the definition and
// the wire API carried all three.
func (t *ChamferTool) ChamferTypeIndex() int {
	for i, ct := range chamferTypeOrder {
		if ct == t.chamferType {
			return i
		}
	}
	return 0
}

// SetChamferTypeIndex selects the setback mode from a combo index; out of range is ignored.
func (t *ChamferTool) SetChamferTypeIndex(i int) {
	if i >= 0 && i < len(chamferTypeOrder) {
		t.chamferType = chamferTypeOrder[i]
	}
}

// SetDistance2/Distance2 set the second face's setback (two-distance mode).
func (t *ChamferTool) SetDistance2(d float64) { t.distance2 = d }
func (t *ChamferTool) Distance2() float64     { return t.distance2 }

// SetAngleDegrees/AngleDegrees set the chamfer-face angle (distance-and-angle mode) in degrees,
// which is how the panel edits it; the definition stores radians.
func (t *ChamferTool) SetAngleDegrees(deg float64) { t.angle = deg * stdmath.Pi / 180 }
func (t *ChamferTool) AngleDegrees() float64       { return t.angle * 180 / stdmath.Pi }

// Name implements [Tool].
func (t *ChamferTool) Name() string { return "Chamfer" }

// Start sets the selection filter to edges and seeds the corner treatment and concave-edge
// strategy from the session's chamfer preferences.
func (t *ChamferTool) Start(s *Session) {
	t.flatCorners = s.ChamferFlatCorners()
	t.concaveStrategy = s.ChamferConcaveStrategy()
	t.tangentChain = s.TangentChainSelect()
}

// SetTangentChain/TangentChain choose whether a plain pick selects the whole tangent chain through
// the clicked edge (Inventor's tangent propagation) or just that edge. Shift+click always expands.
func (t *ChamferTool) SetTangentChain(on bool) { t.tangentChain = on }
func (t *ChamferTool) TangentChain() bool      { return t.tangentChain }

// AcceptedKinds declares chamfer picks edges (the edges to bevel).
func (t *ChamferTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectEdge} }

// Picks reports the picked edges for the unified highlight.
func (t *ChamferTool) Picks() []Selectable { return selectables(t.Edges()) }

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
	if e, ok := sel.(EdgeHandle); ok {
		t.addEdge(e)
	}
}

// PickWithMods adds the whole tangent chain through the clicked edge when tangent-chain mode is on
// (the default, Inventor's tangent propagation) or on Shift+click — the "select tangent chain /
// loop" selection (#1798/#1947); otherwise a plain click adds the single edge. Shift always expands.
func (t *ChamferTool) PickWithMods(s *Session, sel Selectable, mods Modifier) {
	e, ok := sel.(EdgeHandle)
	if ok && (t.tangentChain || mods.Has(ShiftMod)) {
		for _, h := range s.tangentChainHandles(e) {
			t.addEdge(h)
		}
		return
	}
	t.Pick(s, sel)
}

// addEdge appends an edge unless it is already selected.
func (t *ChamferTool) addEdge(e EdgeHandle) {
	if !t.hasEdge(e) {
		t.edges = append(t.edges, e)
	}
}

func (t *ChamferTool) hasEdge(e EdgeHandle) bool {
	return slices.Contains(t.edges, e)
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

// CanCommit reports whether at least one edge is selected, the setback is positive, and the
// mode's second input is usable — a zero second distance or a degenerate angle would build a
// chamfer with no face.
func (t *ChamferTool) CanCommit() bool {
	if t.EdgeCount() == 0 || t.distance <= 0 {
		return false
	}
	switch t.chamferType {
	case types.ChamferTwoDistances:
		return t.distance2 > 0
	case types.ChamferDistanceAndAngle:
		return t.angle > 0 && t.angle < stdmath.Pi/2
	default:
		return true
	}
}

// Commit finishes the tool: an in-place edit writes back through the session, a fresh
// chamfer goes through the host-driven create path (CommitFeature).
func (t *ChamferTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	return t.CommitFeature(s) // create path drives the slim host (I12, #1635)
}

// CommitFeature bevels the picked edges on the active part through the ToolHost seam, so
// the create-commit no longer depends on the whole *Session (satisfies hostedTool, #1635).
// A sick feature (a distance that overruns the geometry) keeps the tool open via an error.
func (t *ChamferTool) CommitFeature(h ToolHost) error {
	part, err := h.ActivePart()
	if err != nil {
		return err
	}
	t.added = t.addChamfer(feature.NewDressUpFeatures(part.Features()))
	part.Recompute()
	h.recordEdit(part, "Chamfer")
	if !t.added.Health().OK() {
		return errors.New("chamfer: " + t.added.Health().Reason)
	}
	return nil
}

// addChamfer builds the chamfer feature into collection dress — the shared constructor used by
// both Commit (the part's engine) and DraftFeature (a scratch engine).
func (t *ChamferTool) addChamfer(dress *feature.DressUpFeatures) *feature.PartFeature {
	// Mint-time anchors are captured by AddChamferDef against the running body (ADR-0043 P6b),
	// uniformly with every other authoring path — no per-tool capture needed here.
	return dress.AddChamferDef(t.chamferDefinition())
}

// chamferDefinition is the definition the tool's current panel state describes. Only the
// equal-distance mode takes the flat-corner treatment: an asymmetric chamfer leaves the corner
// planes to meet at a point, so the toggle has nothing to blend there.
func (t *ChamferTool) chamferDefinition() *feature.ChamferDefinition {
	d, d2, a := t.distance, t.distance2, t.angle
	def := &feature.ChamferDefinition{
		EdgeKeys:        t.selectedEdgeKeys(),
		Distance:        func() float64 { return d },
		Type:            t.chamferType,
		FlatCorners:     t.flatCorners,
		ConcaveStrategy: t.concaveStrategy,
	}
	switch t.chamferType {
	case types.ChamferTwoDistances:
		def.Distance2, def.FlatCorners = func() float64 { return d2 }, true
	case types.ChamferDistanceAndAngle:
		def.Angle, def.FlatCorners = func() float64 { return a }, true
	}
	return def
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ChamferTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature returns the unattached chamfer feature the viewport previews before commit
// (satisfying DraftPreviewable), built by the same addChamfer the commit uses. Empty until an
// edge is selected.
func (t *ChamferTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addChamfer(feature.NewDressUpFeatures(fs)), nil
	})
}

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
}

// commitEdit writes the panel state back into the committed chamfer's definition.
func (t *ChamferTool) commitEdit(s *Session) error {
	def := t.target.Definition().(*feature.ChamferFeature).Definition()
	edited := t.chamferDefinition()
	edited.EdgeAnchors = edgeHandleAnchors(t.edges)
	edited.GeomEdges = def.GeomEdges // an externally-authored edge set is not the panel's to rewrite
	*def = *edited
	return commitFeatureEdit(s, t.target)
}

// ClearEdges empties the edge selection — the picks and, in edit mode, the feature's
// retained keys — returning the tool to its pick-edges step.
func (t *ChamferTool) ClearEdges() {
	t.edges = nil
	t.seededEdgeKeys = nil
}
