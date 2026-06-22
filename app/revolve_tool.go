// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// preselectCenterline chooses the centerline a revolve should auto-select once a profile is
// picked, following Inventor's rules over the visible sketches:
//   - exactly one centerline in the profile's own sketch → that one;
//   - several in the profile's sketch → none (the user must pick);
//   - none in the profile's sketch but exactly one visible overall → that one;
//   - otherwise → none.
func preselectCenterline(profileSketch *sketch.Sketch, sketches []*sketch.Sketch) (*sketch.Sketch, *sketch.Line, bool) {
	var allSk []*sketch.Sketch
	var all, same []*sketch.Line
	for _, sk := range sketches {
		for _, l := range sk.Centerlines() {
			allSk = append(allSk, sk)
			all = append(all, l)
			if sk == profileSketch {
				same = append(same, l)
			}
		}
	}
	switch {
	case len(same) == 1:
		return profileSketch, same[0], true
	case len(same) > 1:
		return nil, nil, false
	case len(all) == 1:
		return allSk[0], all[0], true
	default:
		return nil, nil, false
	}
}

// RevolveTool is the interactive Revolve command: activate it, click a sketch region,
// choose the axis of revolution and the swept angle in the property window, and OK to
// add a revolve feature to the active part. It mirrors [ExtrudeTool] and is driven
// entirely by session input so a test exercises the full flow with synthetic clicks.
type RevolveTool struct {
	featureEditMode // set ⇒ this panel re-edits a committed revolve (see editRevolveTool)
	profile         *ProfileHandle
	axis            feature.WorkRef // origin axis (X/Y/Z) or a user work axis (fallback)
	useCenterln     bool            // revolve about the sketch's own (single) centerline
	centerline      *sketch.Line    // a specific centerline picked/pre-selected as the axis
	centerlineSk    *sketch.Sketch  // the centerline's sketch
	angle           float64         // swept angle in radians; 0 ⇒ full revolution
	operation       ops.PartFeatureOperation
	added           *feature.PartFeature
}

// NewRevolveTool returns a revolve tool defaulting to a full revolution about the Y
// origin axis that creates a new body.
func NewRevolveTool() *RevolveTool {
	return &RevolveTool{axis: feature.OriginYAxis, operation: ops.NewBody}
}

// Name implements [Tool].
func (t *RevolveTool) Name() string { return "Revolve" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *RevolveTool) Start(*Session) {}

// AcceptedKinds declares revolve's two steps: pick the region (profile), then — once a profile is
// set — pick the centerline axis (a sketch entity). The engine re-derives this after each pick.
func (t *RevolveTool) AcceptedKinds() []SelectionKind {
	if t.profile == nil {
		return []SelectionKind{SelectProfile}
	}
	return []SelectionKind{SelectSketchEntity}
}

// Picks reports the picked region and centerline for the unified highlight.
func (t *RevolveTool) Picks() []Selectable {
	var picks []Selectable
	if t.profile != nil {
		picks = append(picks, *t.profile)
	}
	if t.centerline != nil {
		picks = append(picks, SketchEntityHandle{Entity: t.centerline})
	}
	return picks
}

// Pick captures the region (then auto-advances to the centerline axis, pre-selecting one per
// Inventor's rules) or, once a profile is set, a centerline the user clicks to override it.
func (t *RevolveTool) Pick(s *Session, sel Selectable) {
	switch h := sel.(type) {
	case ProfileHandle:
		pc := h
		t.profile = &pc
		t.advanceToCenterline(s)
	case SketchEntityHandle:
		if l, ok := h.Entity.(*sketch.Line); ok && l.IsCenterline() {
			t.centerline, t.centerlineSk = l, sketchOfLine(s, l)
		}
	}
}

// advanceToCenterline moves the selection from the profile to the axis: it pre-selects a
// centerline (if the rules resolve one) and filters selection to sketch entities so the user can
// pick a different centerline.
func (t *RevolveTool) advanceToCenterline(s *Session) {
	if sk, line, ok := preselectCenterline(t.profile.Sketch, visiblePartSketches(s)); ok {
		t.centerline, t.centerlineSk = line, sk
	}
	// The engine re-derives the filter (now SketchEntity) from AcceptedKinds after this pick.
}

// visiblePartSketches returns the active part's visible 2D sketches (the centerline candidates).
func visiblePartSketches(s *Session) []*sketch.Sketch {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	var out []*sketch.Sketch
	for i := 0; i < part.Sketches().Count(); i++ {
		if sk := part.Sketches().Item(i); sk.Visible() {
			out = append(out, sk)
		}
	}
	return out
}

// sketchOfLine finds which visible part sketch holds the line.
func sketchOfLine(s *Session, line *sketch.Line) *sketch.Sketch {
	for _, sk := range visiblePartSketches(s) {
		for i := 0; i < sk.Lines().Count(); i++ {
			if sk.Lines().Item(i) == line {
				return sk
			}
		}
	}
	return nil
}

// The options the property window drives: the revolution axis, the swept angle, and the
// boolean operation.
func (t *RevolveTool) SetAxis(ref feature.WorkRef) { t.axis = ref }
func (t *RevolveTool) Axis() feature.WorkRef       { return t.axis }
func (t *RevolveTool) SetAngle(radians float64)    { t.angle = radians }
func (t *RevolveTool) Angle() float64              { return t.angle }

// SubmitToken accepts a typed swept angle in DEGREES from the command line (M26 F02
// follow-up); the region (and any non-default axis) are picked in the viewport. With no
// angle typed the tool keeps its default full revolution.
func (t *RevolveTool) SubmitToken(_ *Session, tok CommandToken) error {
	if tok.Kind != ValueToken {
		return errors.New("revolve: expected an angle in degrees (pick the region in the viewport)")
	}
	t.angle = tok.Value * stdmath.Pi / 180
	return nil
}
func (t *RevolveTool) SetOperation(op ops.PartFeatureOperation) { t.operation = op }
func (t *RevolveTool) Operation() ops.PartFeatureOperation      { return t.operation }

// SetUseCenterline/UseCenterline choose to revolve about the sketch's own centerline (ignoring
// the axis selection) — the common Inventor flow when the profile sketch carries a centerline.
func (t *RevolveTool) SetUseCenterline(v bool) { t.useCenterln = v }
func (t *RevolveTool) UseCenterline() bool     { return t.useCenterln }

// Centerline returns the picked/pre-selected centerline axis (and true), or false when the axis
// is not a centerline yet — the head highlights it on hover and shows it as the chosen axis.
func (t *RevolveTool) Centerline() (*sketch.Line, bool) { return t.centerline, t.centerline != nil }

// CenterlineOutline returns the chosen centerline's 2D endpoints and its sketch plane, for the
// head to draw the selected axis (like a profile outline). False when no centerline is chosen.
func (t *RevolveTool) CenterlineOutline() ([]math.Point2, sketch.Plane, bool) {
	if t.centerline == nil || t.centerlineSk == nil {
		return nil, sketch.Plane{}, false
	}
	return lineOutline2D(t.centerline), t.centerlineSk.Plane(), true
}

// lineOutline2D returns a line's two endpoints as a 2D polyline.
func lineOutline2D(l *sketch.Line) []math.Point2 {
	return []math.Point2{l.StartPoint().Position(), l.EndPoint().Position()}
}

// HoveredCenterlineOutline returns the 2D outline + plane of a centerline under the cursor while
// a revolve tool is choosing its axis (a profile is picked), for the head to highlight the
// candidate axis. ok=false otherwise.
func (s *Session) HoveredCenterlineOutline(px, py float64) ([]math.Point2, sketch.Plane, bool) {
	rv := s.ActiveRevolve()
	if rv == nil {
		return nil, sketch.Plane{}, false
	}
	if _, picked := rv.PickedProfile(); !picked {
		return nil, sketch.Plane{}, false
	}
	sel, found := s.PickAt(px, py, NewSelectionFilter(SelectSketchEntity))
	if !found {
		return nil, sketch.Plane{}, false
	}
	h, isEnt := sel.(SketchEntityHandle)
	if !isEnt {
		return nil, sketch.Plane{}, false
	}
	l, isLine := h.Entity.(*sketch.Line)
	if !isLine || !l.IsCenterline() {
		return nil, sketch.Plane{}, false
	}
	sk := sketchOfLine(s, l)
	if sk == nil {
		return nil, sketch.Plane{}, false
	}
	return lineOutline2D(l), sk.Plane(), true
}

// SetFullRevolution sets the angle to a full turn (0, the model's "full" sentinel).
func (t *RevolveTool) SetFullRevolution() { t.angle = 0 }

// IsFullRevolution reports whether the tool will sweep a full turn.
func (t *RevolveTool) IsFullRevolution() bool { return t.angle <= 0 }

// PickedProfile returns the picked region (and true), or false when none picked yet.
func (t *RevolveTool) PickedProfile() (ProfileHandle, bool) {
	if t.profile == nil {
		return ProfileHandle{}, false
	}
	return *t.profile, true
}

// CanCommit reports whether a region has been picked (the axis has a default).
func (t *RevolveTool) CanCommit() bool { return t.profile != nil }

// ClearProfile empties the picked profile — the property panel's selector clear (⊗) —
// dropping any centerline that was auto-selected from it, so the tool returns to its
// select-a-region step.
func (t *RevolveTool) ClearProfile() {
	t.profile = nil
	t.centerline = nil
	t.centerlineSk = nil
}

// SourceSketchName returns the sketch the picked profile comes from, for the property
// panel's breadcrumb; "" until a profile is picked.
func (t *RevolveTool) SourceSketchName() string {
	if t.profile == nil {
		return ""
	}
	return t.profile.Sketch.Name()
}

// Commit adds the revolve feature to the active part and recomputes; a sick feature
// (open profile, missing axis) keeps the tool open by returning an error.
func (t *RevolveTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if t.added, err = t.addRevolve(part, part.Features()); err != nil {
		return err
	}
	part.Recompute()
	s.recordEdit(part, "Revolve")
	if !t.added.Health().OK() {
		return errors.New("revolve: " + t.added.Health().Reason)
	}
	return nil
}

// commitEdit writes the panel state back into the committed revolve's definition,
// mirroring the create path's axis precedence (explicit axis / picked centerline / the
// sketch's own centerline).
func (t *RevolveTool) commitEdit(s *Session) error {
	def := t.target.Definition().(*feature.RevolveFeature).Definition()
	def.Sketch, def.ProfileIndex = t.profile.Sketch, t.profile.ProfileIndex
	angle := t.angle
	def.Angle = func() float64 { return angle }
	def.Operation = t.operation
	if err := t.writeEditAxis(s, def); err != nil {
		return err
	}
	return commitFeatureEdit(s, t.target)
}

// writeEditAxis maps the tool's axis choice back onto the definition's precedence trio.
func (t *RevolveTool) writeEditAxis(s *Session, def *feature.RevolveDefinition) error {
	def.Axis, def.AxisCenterline, def.AxisCenterlineSketch = nil, nil, nil
	switch {
	case t.centerline != nil:
		def.AxisCenterline, def.AxisCenterlineSketch = t.centerline, t.centerlineSk
	case t.useCenterln: // the definition's auto case: all three axis fields nil
	default:
		part, err := activePart(s)
		if err != nil {
			return err
		}
		axis, ok := part.WorkGeometry().AxisByRef(t.axis)
		if !ok {
			return errors.New("revolve edit: axis " + string(t.axis) + " not found")
		}
		def.Axis = axis
	}
	return nil
}

// addRevolve adds the revolve feature about the tool's chosen axis: a specific picked centerline,
// the sketch's own centerline, or a named work axis.
func (t *RevolveTool) addRevolve(part *compdef.PartComponentDefinition, fs *feature.PartFeatures) (*feature.PartFeature, error) {
	angle := func() float64 { return t.angle }
	revolves := feature.NewRevolveFeatures(fs)
	switch {
	case t.centerline != nil: // a specific picked/pre-selected centerline
		return revolves.AddAboutCenterlineLine(t.profile.Sketch, t.profile.ProfileIndex, t.centerlineSk, t.centerline, angle, t.operation), nil
	case t.useCenterln: // "about the sketch's own centerline" (auto, single)
		return revolves.AddAboutCenterline(t.profile.Sketch, t.profile.ProfileIndex, angle, t.operation), nil
	default:
		axis, ok := part.WorkGeometry().AxisByRef(t.axis)
		if !ok {
			return nil, errors.New("revolve: axis " + string(t.axis) + " not found")
		}
		return revolves.Add(t.profile.Sketch, t.profile.ProfileIndex, axis, angle, t.operation), nil
	}
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *RevolveTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the revolve steps (Inventor's status-bar prompts).
func (t *RevolveTool) Prompt(*Session) string {
	if t.profile == nil {
		return "Select a region to revolve"
	}
	return "Set the axis and angle, then click OK"
}

// DraftFeature returns the unattached revolve feature the viewport previews before commit
// (satisfying DraftPreviewable), built by the same addRevolve the commit uses — so the
// translucent solid preview is exactly what OK creates. Empty until a region is picked.
func (t *RevolveTool) DraftFeature(s *Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	part, err := activePart(s)
	if err != nil {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addRevolve(part, fs)
	})
}

// Cancel restores the default selection filter.
func (t *RevolveTool) Cancel(s *Session) {
	if t.IsEditing() {
		cancelFeatureEdit(s, t.target, t.restoreDef)
		return
	}
}

// FullTurn is the swept angle of a complete revolution, for the property window's
// "Full" button to set an explicit angle when the user switches to a partial angle.
const FullTurn = 2 * stdmath.Pi
