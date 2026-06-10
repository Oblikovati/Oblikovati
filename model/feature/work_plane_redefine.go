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

// String renders the slot kind as the stable wire token ("plane"|"axis"|"point"|"face").
func (k WorkRefKind) String() string {
	switch k {
	case WorkRefPlane:
		return "plane"
	case WorkRefAxis:
		return "axis"
	case WorkRefPoint:
		return "point"
	case WorkRefFace:
		return "face"
	default:
		return "unknown"
	}
}

// WorkRefSlot is one re-pickable reference of a placed work plane. Set rebinds the slot to a
// new reference, so the next recompute re-derives the plane from the chosen geometry. It
// errors (leaving the definition untouched) when the reference is rejected — it would create
// a reference cycle, or it names a work feature that does not exist; see
// [WorkGeometry.validateRedefineRef]. The editor arms a slot, resolves a pick to a [WorkRef]
// of the slot's Kind, and calls Set.
type WorkRefSlot struct {
	Label string
	Kind  WorkRefKind
	Set   func(WorkRef) error
}

// slot wraps assign into a [WorkRefSlot] whose Set validates the reference (no cycles, no
// dangling user refs) before rebinding. The definitions are held behind pointers, so assign
// mutates the live definition in place and composes with scalar edits applied through
// [EditableScalars] (a value copy here once silently dropped a concurrent angle edit).
func (w *WorkPlane) slot(label string, kind WorkRefKind, assign func(WorkRef)) WorkRefSlot {
	return WorkRefSlot{Label: label, Kind: kind, Set: func(r WorkRef) error {
		if err := w.g.validateRedefineRef(r, w.key); err != nil {
			return err
		}
		assign(r)
		return nil
	}}
}

// RedefineSlots returns the plane's re-pickable reference inputs in display order, or nil when
// the definition has none (the origin frame, fixed-geometry planes).
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
		return []WorkRefSlot{w.slot("Base plane", WorkRefPlane, func(r WorkRef) { d.base = r })}
	case *threePointPlaneDef:
		return []WorkRefSlot{
			w.slot("Point 1", WorkRefPoint, func(r WorkRef) { d.a = r }),
			w.slot("Point 2", WorkRefPoint, func(r WorkRef) { d.b = r }),
			w.slot("Point 3", WorkRefPoint, func(r WorkRef) { d.c = r }),
		}
	case *planeAndPointPlaneDef:
		return []WorkRefSlot{
			w.slot("Base plane", WorkRefPlane, func(r WorkRef) { d.base = r }),
			w.slot("Through point", WorkRefPoint, func(r WorkRef) { d.point = r }),
		}
	case *twoPlanesPlaneDef:
		return []WorkRefSlot{
			w.slot("Plane 1", WorkRefPlane, func(r WorkRef) { d.p1 = r }),
			w.slot("Plane 2", WorkRefPlane, func(r WorkRef) { d.p2 = r }),
		}
	default:
		return nil
	}
}

// linePlaneSlots returns the slots for the planes defined by a line/axis (line-plane-angle,
// two-lines, normal-to-curve); nil for the other kinds.
func (w *WorkPlane) linePlaneSlots() []WorkRefSlot {
	switch d := w.def.(type) {
	case *linePlaneAnglePlaneDef:
		return []WorkRefSlot{
			w.slot("Line", WorkRefAxis, func(r WorkRef) { d.line = r }),
			w.slot("Plane", WorkRefPlane, func(r WorkRef) { d.base = r }),
		}
	case *twoLinesPlaneDef:
		return []WorkRefSlot{
			w.slot("Line 1", WorkRefAxis, func(r WorkRef) { d.l1 = r }),
			w.slot("Line 2", WorkRefAxis, func(r WorkRef) { d.l2 = r }),
		}
	case *normalToCurvePlaneDef:
		return []WorkRefSlot{
			w.slot("Curve", WorkRefAxis, func(r WorkRef) { d.curve = r }),
			w.slot("Point", WorkRefPoint, func(r WorkRef) { d.point = r }),
		}
	default:
		return nil
	}
}

// tangentSlots returns the slots for the surface-tangent planes (built on a B-rep face), each
// of which exposes that face as a re-pickable WorkRefFace; nil for every other kind.
func (w *WorkPlane) tangentSlots() []WorkRefSlot {
	switch d := w.def.(type) {
	case *torusMidPlaneDef:
		return []WorkRefSlot{w.slot("Torus face", WorkRefFace, func(r WorkRef) { d.face = r })}
	case *pointAndTangentPlaneDef:
		return []WorkRefSlot{
			w.slot("Point", WorkRefPoint, func(r WorkRef) { d.point = r }),
			w.slot("Tangent face", WorkRefFace, func(r WorkRef) { d.face = r }),
		}
	case *planeAndTangentPlaneDef:
		return []WorkRefSlot{
			w.slot("Parallel plane", WorkRefPlane, func(r WorkRef) { d.base = r }),
			w.slot("Tangent face", WorkRefFace, func(r WorkRef) { d.face = r }),
		}
	case *lineAndTangentPlaneDef:
		return []WorkRefSlot{
			w.slot("Line", WorkRefAxis, func(r WorkRef) { d.line = r }),
			w.slot("Tangent face", WorkRefFace, func(r WorkRef) { d.face = r }),
		}
	default:
		return nil
	}
}

// EditableScalars returns the plane's editable scalar inputs in display order, or nil when it
// has none. Reuses the feature [EditableParam] shape (get/set in database units), so the head
// renders and the session converts them exactly like a feature's scalar fields. Set mutates
// the pointer-held definition, so a scalar edit and a slot re-pick compose in either order.
func (w *WorkPlane) EditableScalars() []EditableParam {
	switch d := w.def.(type) {
	case *offsetPlaneDef:
		return []EditableParam{{
			Label: "Offset", Unit: param.Length,
			Get: func() float64 { return d.distance() },
			Set: func(v float64) { d.setDistance(v) },
		}}
	case *linePlaneAnglePlaneDef:
		return []EditableParam{{
			Label: "Angle", Unit: param.Angle,
			Get: func() float64 { return d.angle() },
			Set: func(v float64) { d.angle = func() float64 { return v } },
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
// in one step. Every user definition is held behind a pointer (slots and scalar edits mutate
// one shared instance), so the snapshot copies the pointee and the restore copies it back.
func (w *WorkPlane) SnapshotDefinition() func() {
	switch d := w.def.(type) {
	case *offsetPlaneDef:
		return snapshotPointee(d)
	case *threePointPlaneDef:
		return snapshotPointee(d)
	case *planeAndPointPlaneDef:
		return snapshotPointee(d)
	case *twoPlanesPlaneDef:
		return snapshotPointee(d)
	case *linePlaneAnglePlaneDef:
		return snapshotPointee(d)
	case *twoLinesPlaneDef:
		return snapshotPointee(d)
	case *normalToCurvePlaneDef:
		return snapshotPointee(d)
	case *torusMidPlaneDef:
		return snapshotPointee(d)
	case *pointAndTangentPlaneDef:
		return snapshotPointee(d)
	case *planeAndTangentPlaneDef:
		return snapshotPointee(d)
	case *lineAndTangentPlaneDef:
		return snapshotPointee(d)
	default:
		return func() {} // grounded kinds (fixed, fixed-frame) expose nothing to redefine
	}
}

// snapshotPointee copies *d and returns a closure that copies it back — the restore half of
// [WorkPlane.SnapshotDefinition] for the pointer-held definitions.
func snapshotPointee[T any](d *T) func() {
	cp := *d
	return func() { *d = cp }
}
