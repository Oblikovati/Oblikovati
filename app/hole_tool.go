// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	stdmath "math"

	"oblikovati.org/model/feature"
)

// HoleTool is the interactive Hole command: activate it, click a planar face, set the
// diameter and depth in the property window, and OK to drill a hole (a cylinder cut at
// the face centroid along the inward normal) into the active part. Diameter and depth are
// in database units; the property window converts to the document's display unit.
type HoleTool struct {
	featureEditMode             // set ⇒ this panel re-edits a committed hole (see editHoleTool)
	face            *FaceHandle // placement face picked this session
	seededFaceKey   []byte      // edit mode: the feature's existing placement-face key
	diameter        float64
	depth           float64
	through         bool // Through All: drill through the part with a true cylinder wall
	counterbore     bool // counterbore: a flat recess above the bore
	countersink     bool // countersink: a conical recess above the bore
	cDiameter       float64
	cDepth          float64
	sinkAngle       float64 // countersink included angle (radians)
	pointAngle      float64 // drilled blind-hole drill point: included angle (radians; 0 = flat)
	added           *feature.PartFeature
}

// NewHoleTool returns a hole tool with a default Ø1 × 2 drilled hole and a 118° drill point
// (the standard twist-drill angle, matching Inventor's default).
func NewHoleTool() *HoleTool {
	return &HoleTool{diameter: 1, depth: 2, cDiameter: 2, cDepth: 0.5, sinkAngle: stdmath.Pi / 2, pointAngle: 118 * stdmath.Pi / 180}
}

// Name implements [Tool].
func (t *HoleTool) Name() string { return "Hole" }

// Start sets the selection filter to faces so clicks pick a placement face.
func (t *HoleTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectFace)) }

// Pick captures the planar face the user clicked.
func (t *HoleTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok {
		fc := f
		t.face = &fc
	}
}

// The options the property window drives: diameter and depth (database units).
func (t *HoleTool) SetDiameter(d float64) { t.diameter = d }
func (t *HoleTool) Diameter() float64     { return t.diameter }
func (t *HoleTool) SetDepth(d float64)    { t.depth = d }
func (t *HoleTool) Depth() float64        { return t.depth }

// SetThroughAll/ThroughAll drive the "Through All" extent: when on, the hole drills through
// the whole part (an exact cylinder wall) and the depth is ignored.
func (t *HoleTool) SetThroughAll(v bool) { t.through = v }
func (t *HoleTool) ThroughAll() bool     { return t.through }

// SetCounterbore/Counterbore toggle the flat counterbore profile (a recess above the bore);
// setting it clears the countersink (the profiles are exclusive).
func (t *HoleTool) SetCounterbore(v bool) {
	t.counterbore = v
	if v {
		t.countersink = false
	}
}
func (t *HoleTool) Counterbore() bool            { return t.counterbore }
func (t *HoleTool) SetCounterDiameter(d float64) { t.cDiameter = d }
func (t *HoleTool) CounterDiameter() float64     { return t.cDiameter }
func (t *HoleTool) SetCounterDepth(d float64)    { t.cDepth = d }
func (t *HoleTool) CounterDepth() float64        { return t.cDepth }

// SetCountersink/Countersink toggle the conical countersink profile; setting it clears the
// counterbore. The recess uses the counterbore diameter and the sink (included) angle.
func (t *HoleTool) SetCountersink(v bool) {
	t.countersink = v
	if v {
		t.counterbore = false
	}
}
func (t *HoleTool) Countersink() bool      { return t.countersink }
func (t *HoleTool) SetSinkAngle(a float64) { t.sinkAngle = a }
func (t *HoleTool) SinkAngle() float64     { return t.sinkAngle }

// SetPointAngle/PointAngle drive a drilled blind hole's drill-point included angle (radians);
// 0 gives a flat bottom.
func (t *HoleTool) SetPointAngle(a float64) { t.pointAngle = a }
func (t *HoleTool) PointAngle() float64     { return t.pointAngle }

// ClearFace empties the placement face — the picked one and, in edit mode, the
// feature's retained key — returning the tool to its pick-a-face step.
func (t *HoleTool) ClearFace() {
	t.face = nil
	t.seededFaceKey = nil
}

// HasPlacement reports whether a placement face is set: picked this session or, in
// edit mode, retained from the feature's definition. The property panel's chip state.
func (t *HoleTool) HasPlacement() bool { return t.face != nil || len(t.seededFaceKey) > 0 }

// placementKey is the reference key the commit writes: a fresh pick wins over the
// retained one.
func (t *HoleTool) placementKey() []byte {
	if t.face != nil {
		return t.face.Face.ReferenceKey()
	}
	return t.seededFaceKey
}

// PickedFace returns the placement face (and true), or false when none picked yet.
func (t *HoleTool) PickedFace() (FaceHandle, bool) {
	if t.face == nil {
		return FaceHandle{}, false
	}
	return *t.face, true
}

// CanCommit reports whether a face is picked, the diameter is positive, the depth is positive
// (unless Through All), and — for a counterbore — the recess is larger than the bore and shallow
// enough to leave bore below it.
func (t *HoleTool) CanCommit() bool {
	if !t.HasPlacement() || t.diameter <= 0 || (!t.through && t.depth <= 0) {
		return false
	}
	if t.counterbore && !t.counterboreValid() {
		return false
	}
	if t.countersink && !t.countersinkValid() {
		return false
	}
	return true
}

// counterboreValid reports whether the counterbore options are consistent: a wider recess than
// the bore, a positive recess depth, and (for a blind hole) a bore deeper than the recess.
func (t *HoleTool) counterboreValid() bool {
	return t.cDiameter > t.diameter && t.cDepth > 0 && (t.through || t.depth > t.cDepth)
}

// countersinkValid reports whether the countersink options are consistent: a wider sink than the
// bore and an included angle in (0, π).
func (t *HoleTool) countersinkValid() bool {
	return t.cDiameter > t.diameter && t.sinkAngle > 0 && t.sinkAngle < stdmath.Pi
}

// Commit drills the hole into the active part and recomputes; a sick feature (lost face,
// boolean failure) keeps the tool open by returning an error.
func (t *HoleTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addFeature(feature.NewHoleFeatures(part.Features()))
	part.Recompute()
	s.recordEdit(part, "Hole")
	if !t.added.Health().OK() {
		return errors.New("hole: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// addFeature creates the hole feature matching the tool's options (counterbore vs drilled,
// through vs depth), capturing the current values into the recipe's closures.
func (t *HoleTool) addFeature(holes *feature.HoleFeatures) *feature.PartFeature {
	key := t.face.Face.ReferenceKey()
	d, depth, cd, cdep, ang := t.diameter, t.depth, t.cDiameter, t.cDepth, t.sinkAngle
	switch {
	case t.countersink:
		pf := holes.AddCountersink(key, konst(d), konst(depth), konst(cd), konst(ang))
		pf.Definition().(*feature.HoleFeature).Definition().ThroughAll = t.through
		return pf
	case t.counterbore:
		pf := holes.AddCounterbore(key, konst(d), konst(depth), konst(cd), konst(cdep))
		pf.Definition().(*feature.HoleFeature).Definition().ThroughAll = t.through
		return pf
	case t.through:
		return holes.AddDrilledThrough(key, konst(d))
	default:
		pf := holes.AddDrilled(key, konst(d), konst(depth))
		pf.Definition().(*feature.HoleFeature).Definition().PointAngle = konst(t.pointAngle)
		return pf
	}
}

// commitEdit writes the panel state back into the committed hole's definition — the
// same properties the create path captures, including a seat-type change.
func (t *HoleTool) commitEdit(s *Session) error {
	def := t.target.Definition().(*feature.HoleFeature).Definition()
	def.PlacementFaceKey = t.placementKey()
	def.Diameter, def.Depth = konst(t.diameter), konst(t.depth)
	def.ThroughAll = t.through
	def.PointAngle = konst(t.pointAngle)
	def.CounterDiameter, def.CounterDepth, def.CounterAngle = konst(t.cDiameter), konst(t.cDepth), konst(t.sinkAngle)
	def.Type = t.holeType()
	return commitFeatureEdit(s, t.target)
}

// holeType maps the tool's mutually-exclusive seat flags onto the definition's type.
func (t *HoleTool) holeType() feature.HoleType {
	switch {
	case t.counterbore:
		return feature.CounterboreHole
	case t.countersink:
		return feature.CountersinkHole
	default:
		return feature.DrilledHole
	}
}

// konst wraps a captured value as a parameter closure.
func konst(v float64) func() float64 { return func() float64 { return v } }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *HoleTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the hole steps.
func (t *HoleTool) Prompt(*Session) string {
	if t.face == nil {
		return "Select a planar face to place the hole on"
	}
	return "Set the diameter and depth, then click OK"
}

// Cancel restores the default selection filter.
func (t *HoleTool) Cancel(s *Session) {
	if t.IsEditing() {
		cancelFeatureEdit(s, t.target, t.restoreDef)
		return
	}
	s.Selection().SetFilter(NewSelectionFilter())
}
