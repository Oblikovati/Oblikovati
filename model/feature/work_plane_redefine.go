// SPDX-License-Identifier: GPL-2.0-only

package feature

import "oblikovati.org/model/param"

// Redefining a placed work plane — Inventor's "redefine feature" for datum planes. A plane
// built from references (three points, two planes, a tangent face, …) is edited by re-picking
// those references; a plane with a scalar (offset distance, line-plane angle) also exposes that
// value. RedefineSlots and EditableScalars are the two halves the editor drives; together they
// make every user work plane editable on browser double-click (issue #132 follow-up). A plane
// whose definition has neither (the origin frame, an absolute fixed-frame plane) reports both
// empty and is not opened for editing.

// WorkRefKind is the kind of geometry a redefine slot accepts, so the editor sets the right
// pick filter and the caller maps the pick to the matching [WorkRef].
type WorkRefKind int

const (
	// WorkRefPlane accepts a plane reference: a work plane or a planar B-rep face.
	WorkRefPlane WorkRefKind = iota
	// WorkRefAxis accepts a line/axis reference: a work axis (or a linear edge, later).
	WorkRefAxis
	// WorkRefPoint accepts a point reference: a work point.
	WorkRefPoint
	// WorkRefFace accepts a B-rep face reference (the surface a tangent plane is built on).
	WorkRefFace
)

// WorkRefSlot is one re-pickable reference of a placed work plane. Set rebinds the slot to a
// new reference: it mutates the definition and swaps it back into the plane, so the next
// recompute re-derives the plane from the chosen geometry. The editor arms a slot, resolves a
// pick to a [WorkRef] of the slot's Kind, and calls Set.
type WorkRefSlot struct {
	Label string
	Kind  WorkRefKind
	Set   func(WorkRef)
}

// RedefineSlots returns the plane's re-pickable reference inputs in display order, or nil when
// the definition has none (the origin frame, fixed-geometry planes). Each slot's Set captures a
// copy of the current definition, replaces the one reference, and reassigns w.def — so the
// definition value structs need no pointer plumbing (offsetPlaneDef, already a pointer, is
// handled by mutating it directly).
func (w *WorkPlane) RedefineSlots() []WorkRefSlot {
	if slots := w.pointPlaneSlots(); slots != nil {
		return slots
	}
	if slots := w.linePlaneSlots(); slots != nil {
		return slots
	}
	return w.tangentSlots()
}

// pointPlaneSlots returns the slots for the planes defined by plane/point references (offset,
// three-point, plane-and-point, two-planes); nil for the other kinds.
func (w *WorkPlane) pointPlaneSlots() []WorkRefSlot {
	switch d := w.def.(type) {
	case *offsetPlaneDef:
		return []WorkRefSlot{{"Base plane", WorkRefPlane, func(r WorkRef) { d.base = r }}}
	case threePointPlaneDef:
		return []WorkRefSlot{
			{"Point 1", WorkRefPoint, func(r WorkRef) { d.a = r; w.def = d }},
			{"Point 2", WorkRefPoint, func(r WorkRef) { d.b = r; w.def = d }},
			{"Point 3", WorkRefPoint, func(r WorkRef) { d.c = r; w.def = d }},
		}
	case planeAndPointPlaneDef:
		return []WorkRefSlot{
			{"Base plane", WorkRefPlane, func(r WorkRef) { d.base = r; w.def = d }},
			{"Through point", WorkRefPoint, func(r WorkRef) { d.point = r; w.def = d }},
		}
	case twoPlanesPlaneDef:
		return []WorkRefSlot{
			{"Plane 1", WorkRefPlane, func(r WorkRef) { d.p1 = r; w.def = d }},
			{"Plane 2", WorkRefPlane, func(r WorkRef) { d.p2 = r; w.def = d }},
		}
	default:
		return nil
	}
}

// linePlaneSlots returns the slots for the planes defined by a line/axis (line-plane-angle,
// two-lines, normal-to-curve); nil for the other kinds.
func (w *WorkPlane) linePlaneSlots() []WorkRefSlot {
	switch d := w.def.(type) {
	case linePlaneAnglePlaneDef:
		return []WorkRefSlot{
			{"Line", WorkRefAxis, func(r WorkRef) { d.line = r; w.def = d }},
			{"Plane", WorkRefPlane, func(r WorkRef) { d.base = r; w.def = d }},
		}
	case twoLinesPlaneDef:
		return []WorkRefSlot{
			{"Line 1", WorkRefAxis, func(r WorkRef) { d.l1 = r; w.def = d }},
			{"Line 2", WorkRefAxis, func(r WorkRef) { d.l2 = r; w.def = d }},
		}
	case normalToCurvePlaneDef:
		return []WorkRefSlot{
			{"Curve", WorkRefAxis, func(r WorkRef) { d.curve = r; w.def = d }},
			{"Point", WorkRefPoint, func(r WorkRef) { d.point = r; w.def = d }},
		}
	default:
		return nil
	}
}

// tangentSlots returns the slots for the surface-tangent planes (built on a B-rep face), each
// of which exposes that face as a re-pickable WorkRefFace; nil for every other kind.
func (w *WorkPlane) tangentSlots() []WorkRefSlot {
	switch d := w.def.(type) {
	case torusMidPlaneDef:
		return []WorkRefSlot{{"Torus face", WorkRefFace, func(r WorkRef) { d.face = r; w.def = d }}}
	case pointAndTangentPlaneDef:
		return []WorkRefSlot{
			{"Point", WorkRefPoint, func(r WorkRef) { d.point = r; w.def = d }},
			{"Tangent face", WorkRefFace, func(r WorkRef) { d.face = r; w.def = d }},
		}
	case planeAndTangentPlaneDef:
		return []WorkRefSlot{
			{"Parallel plane", WorkRefPlane, func(r WorkRef) { d.base = r; w.def = d }},
			{"Tangent face", WorkRefFace, func(r WorkRef) { d.face = r; w.def = d }},
		}
	case lineAndTangentPlaneDef:
		return []WorkRefSlot{
			{"Line", WorkRefAxis, func(r WorkRef) { d.line = r; w.def = d }},
			{"Tangent face", WorkRefFace, func(r WorkRef) { d.face = r; w.def = d }},
		}
	default:
		return nil
	}
}

// EditableScalars returns the plane's editable scalar inputs in display order, or nil when it
// has none. Reuses the feature [EditableParam] shape (get/set in database units), so the head
// renders and the session converts them exactly like a feature's scalar fields.
func (w *WorkPlane) EditableScalars() []EditableParam {
	switch d := w.def.(type) {
	case *offsetPlaneDef:
		return []EditableParam{{
			Label: "Offset", Unit: param.Length,
			Get: func() float64 { return d.distance() },
			Set: func(v float64) { d.setDistance(v) },
		}}
	case linePlaneAnglePlaneDef:
		// d is a copy of the value-typed def, so Set must reassign w.def for the edit to stick
		// (unlike *offsetPlaneDef above, which is mutated through its pointer).
		return []EditableParam{{
			Label: "Angle", Unit: param.Angle,
			Get: func() float64 { return d.angle() },
			Set: func(v float64) { d.angle = func() float64 { return v }; w.def = d },
		}}
	default:
		return nil
	}
}

// IsRedefinable reports whether the plane exposes any editable scalar or reference, i.e. whether
// browser double-click should open it for editing.
func (w *WorkPlane) IsRedefinable() bool {
	return len(w.EditableScalars()) > 0 || len(w.RedefineSlots()) > 0
}

// SnapshotDefinition captures the plane's current definition and returns a closure that
// restores it — an edit's Cancel calls it to undo any re-picked references and scalar changes
// in one step. The offset plane is a pointer (mutated in place), so its value is snapshotted;
// every other kind is restored by swapping the captured definition value back.
func (w *WorkPlane) SnapshotDefinition() func() {
	saved := w.def
	if op, ok := saved.(*offsetPlaneDef); ok {
		cp := *op
		return func() { *op = cp; w.def = op }
	}
	return func() { w.def = saved }
}
