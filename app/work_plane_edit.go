// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// Editing / redefining a placed work plane — Inventor's double-click / browser "Edit" on a
// datum plane. An offset plane edits its distance; a reference-built plane (three points, two
// planes, a tangent face, …) re-picks those references; a line-plane-angle plane edits its
// angle. This mirrors the feature-edit flow (app/feature_edit.go): scalar fields plus armed
// reference slots that route viewport/browser picks. A re-pick recomputes immediately; scalar
// edits land on OK, which keeps the result — Cancel restores the definition captured when
// editing opened. See issue #132.

// WorkPlaneEditTool edits one placed work plane: its scalar inputs (offset/angle) and its
// re-pickable reference slots. armed is the index of the slot currently collecting picks (-1
// when none is armed and clicks are ignored).
type WorkPlaneEditTool struct {
	plane      *feature.WorkPlane
	scalars    []feature.EditableParam
	origScalar []float64
	slots      []feature.WorkRefSlot
	restoreDef func() // restores the whole definition (refs + scalars) on Cancel
	armed      int
}

func newWorkPlaneEditTool(wp *feature.WorkPlane) *WorkPlaneEditTool {
	t := &WorkPlaneEditTool{
		plane:      wp,
		scalars:    wp.EditableScalars(),
		slots:      wp.RedefineSlots(),
		restoreDef: wp.SnapshotDefinition(),
		armed:      -1,
	}
	t.origScalar = make([]float64, len(t.scalars))
	for i, p := range t.scalars {
		t.origScalar[i] = p.Get()
	}
	return t
}

// Name implements [Tool].
func (t *WorkPlaneEditTool) Name() string { return "Edit " + t.plane.Name() }

// Start clears any armed slot (clicks do nothing until the user presses a slot's Select).
func (t *WorkPlaneEditTool) Start(*Session) { t.armed = -1 }

// Pick routes a viewport/browser pick to the armed reference slot: it maps the pick to a
// WorkRef of the slot's kind, re-points the plane, and recomputes so the change is immediate.
func (t *WorkPlaneEditTool) Pick(s *Session, sel Selectable) {
	if t.armed < 0 || t.armed >= len(t.slots) {
		return
	}
	slot := t.slots[t.armed]
	ref, ok := workRefOf(sel, slot.Kind)
	if !ok {
		return
	}
	if err := slot.Set(ref); err != nil {
		s.notice = err.Error() // a refused reference (it would create a cycle); the slot stays armed
		return
	}
	t.disarm(s) // a single-reference slot is satisfied by one pick
	t.recompute(s)
}

// CanCommit reports the edit is always committable (an invalid redefinition goes Sick and
// keeps the dialog open via Commit's error).
func (t *WorkPlaneEditTool) CanCommit() bool { return true }

// Commit recomputes with the edited scalars/references, ends the edit scope, and records the
// edit. A sick plane returns an error so the dialog stays open for correction.
func (t *WorkPlaneEditTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	s.endEditScope()
	part.Recompute()
	s.recordEdit(part, "Edit "+t.plane.Name())
	if !t.plane.Health().OK() {
		return errors.New("work plane edit: " + t.plane.Health().Reason)
	}
	return nil
}

// Cancel restores the snapshotted scalars and definition, recomputes, and clears the filter.
func (t *WorkPlaneEditTool) Cancel(s *Session) {
	for i, p := range t.scalars {
		p.Set(t.origScalar[i])
	}
	t.restoreDef()
	s.endEditScope()
	if part, err := activePart(s); err == nil {
		part.Recompute()
	}
}

// Arm puts the i-th reference slot into pick mode; the engine installs that slot's kind filter
// from AcceptedKinds.
func (t *WorkPlaneEditTool) Arm(s *Session, i int) {
	if i < 0 || i >= len(t.slots) {
		return
	}
	t.armed = i
	s.installToolFilter()
}

// AcceptedKinds declares the kinds the armed redefine slot accepts (nil when no slot is armed).
func (t *WorkPlaneEditTool) AcceptedKinds() []SelectionKind {
	if t.armed < 0 || t.armed >= len(t.slots) {
		return nil
	}
	return workRefFilterKinds(t.slots[t.armed].Kind)
}

// ArmedSlot returns the index of the reference slot currently collecting picks, or -1.
func (t *WorkPlaneEditTool) ArmedSlot() int { return t.armed }

func (t *WorkPlaneEditTool) disarm(s *Session) {
	t.armed = -1
	s.installToolFilter() // armed = -1 ⇒ no restriction (back to the ambient filter)
}

func (t *WorkPlaneEditTool) recompute(s *Session) {
	if part, err := activePart(s); err == nil {
		part.Recompute()
	}
}

// workRefOf maps a viewport/browser pick to a feature.WorkRef of the slot's kind: a work plane
// or planar face for a plane slot, a work axis for an axis slot, a work point for a point slot,
// a B-rep face for a face slot. ok=false for a mismatched pick.
func workRefOf(sel Selectable, kind feature.WorkRefKind) (feature.WorkRef, bool) {
	// A B-rep face satisfies both a face slot and a plane slot (a planar face is a valid
	// plane reference — offset-from-face and friends).
	if h, ok := sel.(FaceHandle); ok && (kind == feature.WorkRefFace || kind == feature.WorkRefPlane) {
		return feature.FaceRef(h.Face.ReferenceKey()), true
	}
	switch kind {
	case feature.WorkRefPlane:
		if h, ok := sel.(WorkPlaneHandle); ok {
			return h.Plane.Key(), true
		}
	case feature.WorkRefAxis:
		if h, ok := sel.(WorkAxisHandle); ok {
			return h.Axis.Key(), true
		}
	case feature.WorkRefPoint:
		if h, ok := sel.(WorkPointHandle); ok {
			return h.Point.Key(), true
		}
	}
	return "", false
}

// workRefFilterKinds maps a redefine slot's kind to the selection-filter kinds that pick it.
func workRefFilterKinds(kind feature.WorkRefKind) []SelectionKind {
	switch kind {
	case feature.WorkRefPlane:
		return []SelectionKind{SelectWorkPlane, SelectFace} // a planar face is a valid plane ref
	case feature.WorkRefAxis:
		return []SelectionKind{SelectWorkAxis}
	case feature.WorkRefPoint:
		return []SelectionKind{SelectWorkPoint}
	default: // WorkRefFace
		return []SelectionKind{SelectFace}
	}
}

// BeginEditWorkPlane opens a work plane for editing (browser double-click / Edit menu). Origin
// coordinate-system planes and plane kinds with nothing to edit (fixed-frame) are a no-op.
func (s *Session) BeginEditWorkPlane(h WorkPlaneHandle) {
	if h.Plane == nil || h.Plane.IsCoordinateSystemElement() || !h.Plane.IsRedefinable() {
		return
	}
	t := newWorkPlaneEditTool(h.Plane)
	// StartTool first — its cancel of a previous edit tool restores that edit's scope (see
	// BeginEditFeature); only then is it safe to capture the marker for this scope.
	s.StartTool(t)
	s.beginEditScope(h.Plane.Seq())
}

// ActiveWorkPlaneEdit returns the running work-plane edit tool, or nil.
func (s *Session) ActiveWorkPlaneEdit() *WorkPlaneEditTool {
	if s.tool == nil {
		return nil
	}
	t, _ := s.tool.tool.(*WorkPlaneEditTool)
	return t
}

// IsEditingWorkPlane reports whether a work-plane edit dialog should be open.
func (s *Session) IsEditingWorkPlane() bool { return s.ActiveWorkPlaneEdit() != nil }

// EditPlaneName returns the name of the work plane being edited (the dialog title), or "".
func (s *Session) EditPlaneName() string {
	if t := s.ActiveWorkPlaneEdit(); t != nil {
		return t.plane.Name()
	}
	return ""
}

// --- scalar accessors (the dialog reads/writes in the document's units) ---

// EditPlaneScalarCount returns how many editable scalar fields the open edit has.
func (s *Session) EditPlaneScalarCount() int {
	if t := s.ActiveWorkPlaneEdit(); t != nil {
		return len(t.scalars)
	}
	return 0
}

// EditPlaneScalarLabel returns the i-th scalar field's label.
func (s *Session) EditPlaneScalarLabel(i int) string {
	if p, ok := s.editPlaneScalar(i); ok {
		return p.Label
	}
	return ""
}

// EditPlaneScalarUnitName returns the i-th scalar field's unit name ("mm", "deg", …), or "".
func (s *Session) EditPlaneScalarUnitName(i int) string {
	p, ok := s.editPlaneScalar(i)
	if !ok {
		return ""
	}
	return s.paramUnitName(p)
}

// EditPlaneScalarValue returns the i-th scalar in the document's preferred unit.
func (s *Session) EditPlaneScalarValue(i int) float64 {
	p, ok := s.editPlaneScalar(i)
	if !ok {
		return 0
	}
	return s.paramDisplayValue(p)
}

// SetEditPlaneScalarValue sets the i-th scalar from a value in the document's preferred unit.
func (s *Session) SetEditPlaneScalarValue(i int, value float64) {
	if p, ok := s.editPlaneScalar(i); ok {
		s.setParamDisplayValue(p, value)
	}
}

func (s *Session) editPlaneScalar(i int) (feature.EditableParam, bool) {
	t := s.ActiveWorkPlaneEdit()
	if t == nil || i < 0 || i >= len(t.scalars) {
		return feature.EditableParam{}, false
	}
	return t.scalars[i], true
}

// --- reference-slot accessors (re-pick geometry) ---

// EditPlaneRefSlotCount returns how many reference slots the open edit has.
func (s *Session) EditPlaneRefSlotCount() int {
	if t := s.ActiveWorkPlaneEdit(); t != nil {
		return len(t.slots)
	}
	return 0
}

// EditPlaneRefSlotLabel returns the i-th slot's label (e.g. "Tangent face").
func (s *Session) EditPlaneRefSlotLabel(i int) string {
	if t := s.ActiveWorkPlaneEdit(); t != nil && i >= 0 && i < len(t.slots) {
		return t.slots[i].Label
	}
	return ""
}

// EditPlaneRefSlotArmed reports whether the i-th slot is currently collecting picks.
func (s *Session) EditPlaneRefSlotArmed(i int) bool {
	t := s.ActiveWorkPlaneEdit()
	return t != nil && t.ArmedSlot() == i
}

// EditPlaneArmRefSlot arms the i-th reference slot for picking (sets the pick filter).
func (s *Session) EditPlaneArmRefSlot(i int) {
	if t := s.ActiveWorkPlaneEdit(); t != nil {
		t.Arm(s, i)
	}
}
