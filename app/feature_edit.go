// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/model/feature"
)

// Editing a placed feature — Inventor's double-click / browser "Edit". It re-opens the feature
// as an interactive Tool so viewport picks route to it (the same path the creation tools use),
// letting the user change both scalar parameters (distance, radius, angle …) AND geometric
// references (re-pick a fillet's edges, a shell's open faces, a hole's placement face). Each
// change recomputes live; OK keeps it, Cancel restores the values captured when editing opened.
// A feature exposes what is editable via feature.Editable (params) and feature.ReferenceEditable
// (reference slots); a feature with neither is not editable (its Edit entry is greyed).

// FeatureEditTool edits one placed feature in place. armed is the index of the reference slot
// currently collecting picks (-1 when none is armed and clicks are ignored).
type FeatureEditTool struct {
	feature *feature.PartFeature
	params  []feature.EditableParam
	origP   []float64
	refs    []feature.EditableRefSlot
	restore []func() // ref-slot restorers captured at open, for Cancel
	armed   int
}

func newFeatureEditTool(f *feature.PartFeature) *FeatureEditTool {
	t := &FeatureEditTool{feature: f, armed: -1}
	if ed, ok := f.Definition().(feature.Editable); ok {
		t.params = ed.EditableParams()
	}
	if re, ok := f.Definition().(feature.ReferenceEditable); ok {
		t.refs = re.EditableRefs()
	}
	t.origP = make([]float64, len(t.params))
	for i, p := range t.params {
		t.origP[i] = p.Get()
	}
	t.restore = make([]func(), len(t.refs))
	for i, r := range t.refs {
		if r.Snapshot != nil {
			t.restore[i] = r.Snapshot()
		}
	}
	return t
}

func (t *FeatureEditTool) editable() bool { return len(t.params) > 0 || len(t.refs) > 0 }

// Name implements [Tool].
func (t *FeatureEditTool) Name() string { return "Edit " + t.feature.Name() }

// Start clears any armed reference slot (clicks do nothing until the user presses a slot's
// Select button, which arms it and sets the pick filter).
func (t *FeatureEditTool) Start(*Session) { t.armed = -1 }

// Pick routes a viewport pick to the armed reference slot: it appends the picked edge/face key
// (multi slots) or replaces the single reference, then recomputes so the change is immediate.
func (t *FeatureEditTool) Pick(s *Session, sel Selectable) {
	if t.armed < 0 || t.armed >= len(t.refs) {
		return
	}
	slot := t.refs[t.armed]
	pr, ok := pickedRefOf(sel, slot.Kind)
	if !ok {
		return
	}
	slot.Add(pr)
	if !slot.Multi {
		t.disarm(s) // a single-reference slot is satisfied by one pick
	}
	t.recompute(s)
}

// CanCommit reports the edit is always committable (an invalid edit goes Sick and keeps the
// dialog open via Commit's error).
func (t *FeatureEditTool) CanCommit() bool { return true }

// Commit recomputes with the edited parameters/references and records the edit. A sick result
// returns an error so the dialog stays open for correction.
func (t *FeatureEditTool) Commit(s *Session) error {
	return commitFeatureEdit(s, t.feature)
}

// Cancel restores the snapshotted parameters and references, recomputes, and clears the filter.
func (t *FeatureEditTool) Cancel(s *Session) {
	cancelFeatureEdit(s, t.feature, func() {
		for i, p := range t.params {
			p.Set(t.origP[i])
		}
		for _, restore := range t.restore {
			if restore != nil {
				restore()
			}
		}
	})
}

// Arm puts the i-th reference slot into pick mode; the engine then installs that slot's kind
// filter from AcceptedKinds. ClearSlot empties it.
func (t *FeatureEditTool) Arm(s *Session, i int) {
	if i < 0 || i >= len(t.refs) {
		return
	}
	t.armed = i
	s.installToolFilter()
}

// AcceptedKinds declares the kinds the armed reference slot accepts (nil when no slot is armed, so
// clicks fall through to the ambient filter and the tool's Pick ignores them).
func (t *FeatureEditTool) AcceptedKinds() []SelectionKind {
	if t.armed < 0 || t.armed >= len(t.refs) {
		return nil
	}
	return filterKinds(t.refs[t.armed].Kind)
}

// ClearSlot removes every reference from the i-th slot and recomputes (no-op for a slot that
// is not clearable, e.g. a mirror plane).
func (t *FeatureEditTool) ClearSlot(s *Session, i int) {
	if i < 0 || i >= len(t.refs) || t.refs[i].Clear == nil {
		return
	}
	t.refs[i].Clear()
	t.recompute(s)
}

// SlotClearable reports whether the i-th reference slot can be emptied (its Clear is set).
func (t *FeatureEditTool) SlotClearable(i int) bool {
	return i >= 0 && i < len(t.refs) && t.refs[i].Clear != nil
}

// SlotRefCount returns how many references the i-th slot currently holds.
func (t *FeatureEditTool) SlotRefCount(i int) int {
	if i < 0 || i >= len(t.refs) {
		return 0
	}
	return t.refs[i].Count()
}

// ArmedSlot returns the index of the reference slot currently collecting picks, or -1.
func (t *FeatureEditTool) ArmedSlot() int { return t.armed }

func (t *FeatureEditTool) disarm(s *Session) {
	t.armed = -1
	s.installToolFilter() // armed = -1 ⇒ no restriction (back to the ambient filter)
}

func (t *FeatureEditTool) recompute(s *Session) {
	if part, err := activePart(s); err == nil {
		part.Features().MarkDirty(t.feature)
		part.Recompute()
	}
}

// BeginEditFeature opens a feature for editing (browser double-click / Edit menu). A
// feature whose creation tool can round-trip its definition re-opens that tool — the
// SAME property panel serves creation and editing, with every creation property
// available (editToolFor). Other editable features open the generic parameter/
// reference editor; a feature with neither is a no-op.
func (s *Session) BeginEditFeature(h FeatureHandle) {
	tool, ok := editToolFor(s, h.Feature)
	if !ok {
		t := newFeatureEditTool(h.Feature)
		if !t.editable() {
			return
		}
		tool = t
	}
	// StartTool first: it cancels any previous edit tool, whose Cancel restores that edit's
	// scope — beginning our scope before that would capture the rolled-back marker and then
	// have it cleared out from under us (the part would stay stuck mid-history after commit).
	s.StartTool(tool)
	s.beginEditScope(h.Feature.Seq()) // roll back to this feature: hide everything after it
}

// FeatureIsEditable reports whether a feature exposes editable parameters or references (so the
// browser shows/enables an Edit entry).
func FeatureIsEditable(f *feature.PartFeature) bool {
	if ed, ok := f.Definition().(feature.Editable); ok && len(ed.EditableParams()) > 0 {
		return true
	}
	if re, ok := f.Definition().(feature.ReferenceEditable); ok && len(re.EditableRefs()) > 0 {
		return true
	}
	return false
}

// ActiveFeatureEdit returns the running feature-edit tool, or nil when the active tool (if any)
// is not a feature edit.
func (s *Session) ActiveFeatureEdit() *FeatureEditTool {
	if s.tool == nil {
		return nil
	}
	t, _ := s.tool.tool.(*FeatureEditTool)
	return t
}

// IsEditingFeature reports whether a feature edit dialog should be open.
func (s *Session) IsEditingFeature() bool { return s.ActiveFeatureEdit() != nil }

// EditingFeatureName returns the name of the feature being edited (the dialog title), or "".
func (s *Session) EditingFeatureName() string {
	if t := s.ActiveFeatureEdit(); t != nil {
		return t.feature.Name()
	}
	return ""
}

// --- scalar parameter accessors (the dialog reads/writes in the document's units) ---

// EditFeatureParamCount returns how many editable scalar fields the open edit has.
func (s *Session) EditFeatureParamCount() int {
	if t := s.ActiveFeatureEdit(); t != nil {
		return len(t.params)
	}
	return 0
}

// EditFeatureParamLabel returns the i-th field's label.
func (s *Session) EditFeatureParamLabel(i int) string {
	if p, ok := s.editScalarParam(i); ok {
		return p.Label
	}
	return ""
}

// EditFeatureParamUnitName returns the i-th field's unit name (e.g. "mm", "deg"), or "".
func (s *Session) EditFeatureParamUnitName(i int) string {
	p, ok := s.editScalarParam(i)
	if !ok {
		return ""
	}
	return s.paramUnitName(p)
}

// EditFeatureParamIsInteger reports whether the i-th field is a whole number (a pattern count),
// so the dialog uses an integer input.
func (s *Session) EditFeatureParamIsInteger(i int) bool {
	p, ok := s.editScalarParam(i)
	return ok && p.Integer
}

// EditFeatureParamValue returns the i-th field's value in the document's preferred unit.
func (s *Session) EditFeatureParamValue(i int) float64 {
	p, ok := s.editScalarParam(i)
	if !ok {
		return 0
	}
	return s.paramDisplayValue(p)
}

// SetEditFeatureParamValue sets the i-th field from a value in the document's preferred unit.
func (s *Session) SetEditFeatureParamValue(i int, value float64) {
	if p, ok := s.editScalarParam(i); ok {
		s.setParamDisplayValue(p, value)
	}
}

// SetEditFeatureParamText sets the i-th field from a unit-bearing string that may be
// a parameter expression ("d0 + 5 mm") or a literal ("12 mm"), routing it through the
// active part's parameter evaluator so feature dialogs accept parameters like the
// sketch-dimension editor does. Returns an error the dialog can surface on bad input
// (Oblikovati.API#187, UI side).
func (s *Session) SetEditFeatureParamText(i int, text string) error {
	p, ok := s.editScalarParam(i)
	if !ok {
		return fmt.Errorf("SetEditFeatureParamText: no editable parameter at index %d", i)
	}
	return s.setParamText(p, text)
}

func (s *Session) editScalarParam(i int) (feature.EditableParam, bool) {
	t := s.ActiveFeatureEdit()
	if t == nil || i < 0 || i >= len(t.params) {
		return feature.EditableParam{}, false
	}
	return t.params[i], true
}

// --- reference-slot accessors (re-pick geometry) ---

// EditFeatureRefSlotCount returns how many reference slots the open edit has.
func (s *Session) EditFeatureRefSlotCount() int {
	if t := s.ActiveFeatureEdit(); t != nil {
		return len(t.refs)
	}
	return 0
}

// EditFeatureRefSlotLabel returns the i-th slot's label (e.g. "Edges", "Placement face").
func (s *Session) EditFeatureRefSlotLabel(i int) string {
	if t := s.ActiveFeatureEdit(); t != nil && i >= 0 && i < len(t.refs) {
		return t.refs[i].Label
	}
	return ""
}

// EditFeatureRefSlotRefCount returns how many references the i-th slot currently holds.
func (s *Session) EditFeatureRefSlotRefCount(i int) int {
	if t := s.ActiveFeatureEdit(); t != nil {
		return t.SlotRefCount(i)
	}
	return 0
}

// EditFeatureRefSlotClearable reports whether the i-th slot's Clear button should be shown.
func (s *Session) EditFeatureRefSlotClearable(i int) bool {
	t := s.ActiveFeatureEdit()
	return t != nil && t.SlotClearable(i)
}

// EditFeatureRefSlotArmed reports whether the i-th slot is currently collecting picks.
func (s *Session) EditFeatureRefSlotArmed(i int) bool {
	t := s.ActiveFeatureEdit()
	return t != nil && t.ArmedSlot() == i
}

// EditFeatureArmRefSlot arms the i-th reference slot for picking (sets the pick filter).
func (s *Session) EditFeatureArmRefSlot(i int) {
	if t := s.ActiveFeatureEdit(); t != nil {
		t.Arm(s, i)
	}
}

// EditFeatureClearRefSlot empties the i-th reference slot and recomputes.
func (s *Session) EditFeatureClearRefSlot(i int) {
	if t := s.ActiveFeatureEdit(); t != nil {
		t.ClearSlot(s, i)
	}
}

// pickedRefOf builds a feature.PickedRef from a viewport pick matching the slot kind: an edge
// for RefEdges, a face for RefFaces/RefFace, a sketch profile for RefProfile, a planar face or
// work plane for RefPlane. ok=false for a mismatched pick.
func pickedRefOf(sel Selectable, kind feature.RefKind) (feature.PickedRef, bool) {
	switch kind {
	case feature.RefEdges:
		if e, ok := sel.(EdgeHandle); ok {
			return feature.PickedRef{Key: e.Edge.ReferenceKey()}, true
		}
	case feature.RefFaces, feature.RefFace:
		if f, ok := sel.(FaceHandle); ok {
			return feature.PickedRef{Key: f.Face.ReferenceKey()}, true
		}
	case feature.RefProfile:
		if p, ok := sel.(ProfileHandle); ok {
			return feature.PickedRef{Sketch: p.Sketch, Profile: p.ProfileIndex}, true
		}
	case feature.RefPlane:
		return pickedPlaneRef(sel)
	}
	return feature.PickedRef{}, false
}

// pickedPlaneRef derives a plane reference from a planar face (its plane + key) or a work plane.
func pickedPlaneRef(sel Selectable) (feature.PickedRef, bool) {
	if f, ok := sel.(FaceHandle); ok {
		if pl, ok := f.Face.Geometry().(geom.Plane); ok {
			return feature.PickedRef{Origin: pl.Origin, Normal: pl.Normal(), PlaneKey: f.Face.ReferenceKey()}, true
		}
		return feature.PickedRef{}, false
	}
	if w, ok := sel.(WorkPlaneHandle); ok {
		pl := w.Plane.Plane()
		return feature.PickedRef{Origin: pl.Origin(), Normal: pl.Normal().AsVector()}, true
	}
	return feature.PickedRef{}, false
}

// filterKinds maps a reference kind to the selection-filter kinds that pick it.
func filterKinds(kind feature.RefKind) []SelectionKind {
	switch kind {
	case feature.RefEdges:
		return []SelectionKind{SelectEdge}
	case feature.RefProfile:
		return []SelectionKind{SelectProfile}
	case feature.RefPlane:
		return []SelectionKind{SelectFace, SelectWorkPlane}
	default:
		return []SelectionKind{SelectFace}
	}
}
