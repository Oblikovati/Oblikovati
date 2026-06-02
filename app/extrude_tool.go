// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/renderer"
)

// ExtrudeTool is the interactive Extrude command: activate it, click a sketch
// profile, set a distance, and OK to add an extrude feature to the active part —
// the full Inventor extrude flow, driven entirely by session input so a test
// exercises it with synthetic clicks. It is the worked example proving geometry
// flows end to end from the UI.
type ExtrudeTool struct {
	profile   *ProfileHandle
	distance  float64
	operation ops.PartFeatureOperation
	added     *feature.PartFeature
}

// NewExtrudeTool returns an extrude tool defaulting to a new-body extrusion.
func NewExtrudeTool() *ExtrudeTool {
	return &ExtrudeTool{operation: ops.NewBody}
}

// Name implements [Tool].
func (t *ExtrudeTool) Name() string { return "Extrude" }

// Start sets the selection filter to profiles (so clicks pick a profile).
func (t *ExtrudeTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectProfile)) }

// Pick captures the profile the user clicked.
func (t *ExtrudeTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		t.profile = &p
	}
}

// SetDistance sets the extrusion distance (the value the in-canvas field / spinner
// would set).
func (t *ExtrudeTool) SetDistance(d float64) { t.distance = d }

// SetOperation chooses join/cut/intersect/new-body.
func (t *ExtrudeTool) SetOperation(op ops.PartFeatureOperation) { t.operation = op }

// CanCommit reports whether a profile and a non-zero distance have been gathered.
func (t *ExtrudeTool) CanCommit() bool { return t.profile != nil && t.distance != 0 }

// Commit adds the extrude feature to the active part and recomputes; a sick feature
// (e.g. an open profile) keeps the tool open by returning an error.
func (t *ExtrudeTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	dist := t.distance
	t.added = feature.NewExtrudeFeatures(part.Features()).
		AddByDistanceExtent(t.profile.Sketch, t.profile.ProfileIndex, t.operation, func() float64 { return dist })
	part.Recompute()
	if !t.added.Health().OK() {
		return errors.New("extrude: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ExtrudeTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the extrude steps (Inventor's status-bar prompts).
func (t *ExtrudeTool) Prompt(*Session) string {
	switch {
	case t.profile == nil:
		return "Select a profile to extrude"
	case t.distance == 0:
		return "Specify the extrude distance"
	default:
		return "Click OK to create the extrusion"
	}
}

// Preview returns a transient wireframe of the prism the tool will create — the
// bottom and top profile loops plus vertical connectors — so the viewport shows a
// live preview before OK (Inventor's in-canvas preview). Empty until a profile and
// distance are set.
func (t *ExtrudeTool) Preview(*Session) []renderer.DrawItem {
	if !t.CanCommit() {
		return nil
	}
	plane := t.profile.Sketch.Plane()
	poly := t.profile.Sketch.Profiles().Item(t.profile.ProfileIndex).OuterLoop().Polygon()
	up := plane.Normal().AsVector().Scale(t.distance)
	n := len(poly)
	var pts []math.Point3
	for _, p := range poly { // bottom ring [0,n)
		pts = append(pts, plane.ToModel(p))
	}
	for i := 0; i < n; i++ { // top ring [n,2n)
		pts = append(pts, pts[i].TranslateBy(up))
	}
	var idx []int
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		idx = append(idx, i, j)     // bottom loop
		idx = append(idx, n+i, n+j) // top loop
		idx = append(idx, i, n+i)   // vertical
	}
	return []renderer.DrawItem{{Primitive: renderer.Lines, Positions: pts, Indices: idx, Color: [4]float32{1, 0.6, 0, 1}}}
}

// Cancel restores the default selection filter.
func (t *ExtrudeTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// activePart returns the active document's part component definition, or an error.
func activePart(s *Session) (*compdef.PartComponentDefinition, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, errors.New("app: no active document")
	}
	part, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return nil, errors.New("app: active document is not a part")
	}
	return part, nil
}
