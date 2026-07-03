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
//
// Each definition declares its own slots/scalars/snapshot (the redefineSlots/editableScalars/
// snapshotState half of [planeDefinition]), so create and redefine share ONE dispatch — the
// definition type. The former hand switches here trailed the create side: a kind added to the
// constructors but not to a switch was creatable-but-not-editable (#1634, audit I11; the #1521
// tool-without-edit-path shape).

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
// the definition has none (the origin frame, fixed-geometry planes). The definition itself
// declares its slots, so create and redefine dispatch through the same type (#1634).
func (w *WorkPlane) RedefineSlots() []WorkRefSlot { return w.def.redefineSlots(w) }

// EditableScalars returns the plane's editable scalar inputs in display order, or nil when it
// has none. Reuses the feature [EditableParam] shape (get/set in database units), so the head
// renders and the session converts them exactly like a feature's scalar fields. Set mutates
// the pointer-held definition, so a scalar edit and a slot re-pick compose in either order.
func (w *WorkPlane) EditableScalars() []EditableParam { return w.def.editableScalars() }

// IsRedefinable reports whether the plane exposes any editable scalar or reference, i.e. whether
// browser double-click should open it for editing.
func (w *WorkPlane) IsRedefinable() bool {
	return len(w.EditableScalars()) > 0 || len(w.RedefineSlots()) > 0
}

// SnapshotDefinition captures the plane's current definition and returns a closure that
// restores it — an edit's Cancel calls it to undo any re-picked references and scalar changes
// in one step. Every user definition is held behind a pointer (slots and scalar edits mutate
// one shared instance), so the snapshot copies the pointee and the restore copies it back.
func (w *WorkPlane) SnapshotDefinition() func() { return w.def.snapshotState() }

// snapshotPointee copies *d and returns a closure that copies it back — the restore half of
// [WorkPlane.SnapshotDefinition] for the pointer-held definitions.
func snapshotPointee[T any](d *T) func() {
	cp := *d
	return func() { *d = cp }
}

// The grounded kinds (origin planes, the absolute fixed-frame plane) expose nothing to
// redefine — declared explicitly, so their non-editability is a decision, not a default case.
func (fixedPlaneDef) redefineSlots(*WorkPlane) []WorkRefSlot { return nil }
func (fixedPlaneDef) editableScalars() []EditableParam       { return nil }
func (fixedPlaneDef) snapshotState() func()                  { return func() {} }

func (*fixedFramePlaneDef) redefineSlots(*WorkPlane) []WorkRefSlot { return nil }
func (*fixedFramePlaneDef) editableScalars() []EditableParam       { return nil }
func (*fixedFramePlaneDef) snapshotState() func()                  { return func() {} }

// pointCloudFitPlaneDef has no re-pickable inputs either: it is fit associatively to its
// source cloud (#645), so the fit — not a user pick — defines it. Re-fitting is a recompute,
// not a redefine. Before I11 this kind silently fell out of the redefine switches' default
// cases; now the decision is stated here.
func (*pointCloudFitPlaneDef) redefineSlots(*WorkPlane) []WorkRefSlot { return nil }
func (*pointCloudFitPlaneDef) editableScalars() []EditableParam       { return nil }
func (*pointCloudFitPlaneDef) snapshotState() func()                  { return func() {} }

// offsetPlaneDef: base-plane slot + offset scalar.
func (d *offsetPlaneDef) redefineSlots(w *WorkPlane) []WorkRefSlot {
	return []WorkRefSlot{w.slot("Base plane", WorkRefPlane, func(r WorkRef) { d.base = r })}
}
func (d *offsetPlaneDef) editableScalars() []EditableParam {
	return []EditableParam{{
		Label: "Offset", Unit: param.Length,
		Get: func() float64 { return d.distance() },
		Set: func(v float64) { d.setDistance(v) },
	}}
}
func (d *offsetPlaneDef) snapshotState() func() { return snapshotPointee(d) }

// threePointPlaneDef: three point slots.
func (d *threePointPlaneDef) redefineSlots(w *WorkPlane) []WorkRefSlot {
	return []WorkRefSlot{
		w.slot("Point 1", WorkRefPoint, func(r WorkRef) { d.a = r }),
		w.slot("Point 2", WorkRefPoint, func(r WorkRef) { d.b = r }),
		w.slot("Point 3", WorkRefPoint, func(r WorkRef) { d.c = r }),
	}
}
func (d *threePointPlaneDef) editableScalars() []EditableParam { return nil }
func (d *threePointPlaneDef) snapshotState() func()            { return snapshotPointee(d) }

// planeAndPointPlaneDef: base-plane + through-point slots.
func (d *planeAndPointPlaneDef) redefineSlots(w *WorkPlane) []WorkRefSlot {
	return []WorkRefSlot{
		w.slot("Base plane", WorkRefPlane, func(r WorkRef) { d.base = r }),
		w.slot("Through point", WorkRefPoint, func(r WorkRef) { d.point = r }),
	}
}
func (d *planeAndPointPlaneDef) editableScalars() []EditableParam { return nil }
func (d *planeAndPointPlaneDef) snapshotState() func()            { return snapshotPointee(d) }

// twoPlanesPlaneDef: the two bisected plane slots.
func (d *twoPlanesPlaneDef) redefineSlots(w *WorkPlane) []WorkRefSlot {
	return []WorkRefSlot{
		w.slot("Plane 1", WorkRefPlane, func(r WorkRef) { d.p1 = r }),
		w.slot("Plane 2", WorkRefPlane, func(r WorkRef) { d.p2 = r }),
	}
}
func (d *twoPlanesPlaneDef) editableScalars() []EditableParam { return nil }
func (d *twoPlanesPlaneDef) snapshotState() func()            { return snapshotPointee(d) }

// linePlaneAnglePlaneDef: line + plane slots, plus the swing-angle scalar.
func (d *linePlaneAnglePlaneDef) redefineSlots(w *WorkPlane) []WorkRefSlot {
	return []WorkRefSlot{
		w.slot("Line", WorkRefAxis, func(r WorkRef) { d.line = r }),
		w.slot("Plane", WorkRefPlane, func(r WorkRef) { d.base = r }),
	}
}
func (d *linePlaneAnglePlaneDef) editableScalars() []EditableParam {
	return []EditableParam{{
		Label: "Angle", Unit: param.Angle,
		Get: func() float64 { return d.angle() },
		Set: func(v float64) { d.angle = func() float64 { return v } },
	}}
}
func (d *linePlaneAnglePlaneDef) snapshotState() func() { return snapshotPointee(d) }

// twoLinesPlaneDef: the two line slots.
func (d *twoLinesPlaneDef) redefineSlots(w *WorkPlane) []WorkRefSlot {
	return []WorkRefSlot{
		w.slot("Line 1", WorkRefAxis, func(r WorkRef) { d.l1 = r }),
		w.slot("Line 2", WorkRefAxis, func(r WorkRef) { d.l2 = r }),
	}
}
func (d *twoLinesPlaneDef) editableScalars() []EditableParam { return nil }
func (d *twoLinesPlaneDef) snapshotState() func()            { return snapshotPointee(d) }

// normalToCurvePlaneDef: curve + point slots.
func (d *normalToCurvePlaneDef) redefineSlots(w *WorkPlane) []WorkRefSlot {
	return []WorkRefSlot{
		w.slot("Curve", WorkRefAxis, func(r WorkRef) { d.curve = r }),
		w.slot("Point", WorkRefPoint, func(r WorkRef) { d.point = r }),
	}
}
func (d *normalToCurvePlaneDef) editableScalars() []EditableParam { return nil }
func (d *normalToCurvePlaneDef) snapshotState() func()            { return snapshotPointee(d) }

// torusMidPlaneDef: the torus face slot.
func (d *torusMidPlaneDef) redefineSlots(w *WorkPlane) []WorkRefSlot {
	return []WorkRefSlot{w.slot("Torus face", WorkRefFace, func(r WorkRef) { d.face = r })}
}
func (d *torusMidPlaneDef) editableScalars() []EditableParam { return nil }
func (d *torusMidPlaneDef) snapshotState() func()            { return snapshotPointee(d) }

// pointAndTangentPlaneDef: point + tangent-face slots.
func (d *pointAndTangentPlaneDef) redefineSlots(w *WorkPlane) []WorkRefSlot {
	return []WorkRefSlot{
		w.slot("Point", WorkRefPoint, func(r WorkRef) { d.point = r }),
		w.slot(labelTangentFace, WorkRefFace, func(r WorkRef) { d.face = r }),
	}
}
func (d *pointAndTangentPlaneDef) editableScalars() []EditableParam { return nil }
func (d *pointAndTangentPlaneDef) snapshotState() func()            { return snapshotPointee(d) }

// planeAndTangentPlaneDef: parallel-plane + tangent-face slots.
func (d *planeAndTangentPlaneDef) redefineSlots(w *WorkPlane) []WorkRefSlot {
	return []WorkRefSlot{
		w.slot("Parallel plane", WorkRefPlane, func(r WorkRef) { d.base = r }),
		w.slot(labelTangentFace, WorkRefFace, func(r WorkRef) { d.face = r }),
	}
}
func (d *planeAndTangentPlaneDef) editableScalars() []EditableParam { return nil }
func (d *planeAndTangentPlaneDef) snapshotState() func()            { return snapshotPointee(d) }

// lineAndTangentPlaneDef: line + tangent-face slots.
func (d *lineAndTangentPlaneDef) redefineSlots(w *WorkPlane) []WorkRefSlot {
	return []WorkRefSlot{
		w.slot("Line", WorkRefAxis, func(r WorkRef) { d.line = r }),
		w.slot(labelTangentFace, WorkRefFace, func(r WorkRef) { d.face = r }),
	}
}
func (d *lineAndTangentPlaneDef) editableScalars() []EditableParam { return nil }
func (d *lineAndTangentPlaneDef) snapshotState() func()            { return snapshotPointee(d) }

// Every definition kind must carry its full redefine story — a new kind missing one of the
// redefine methods is a build error here, not a plane that can be created but not re-edited
// (#1634, audit I11).
var (
	_ planeDefinition = fixedPlaneDef{}
	_ planeDefinition = (*fixedFramePlaneDef)(nil)
	_ planeDefinition = (*offsetPlaneDef)(nil)
	_ planeDefinition = (*threePointPlaneDef)(nil)
	_ planeDefinition = (*planeAndPointPlaneDef)(nil)
	_ planeDefinition = (*twoPlanesPlaneDef)(nil)
	_ planeDefinition = (*linePlaneAnglePlaneDef)(nil)
	_ planeDefinition = (*twoLinesPlaneDef)(nil)
	_ planeDefinition = (*normalToCurvePlaneDef)(nil)
	_ planeDefinition = (*torusMidPlaneDef)(nil)
	_ planeDefinition = (*pointAndTangentPlaneDef)(nil)
	_ planeDefinition = (*planeAndTangentPlaneDef)(nil)
	_ planeDefinition = (*lineAndTangentPlaneDef)(nil)
	_ planeDefinition = (*pointCloudFitPlaneDef)(nil)
)
