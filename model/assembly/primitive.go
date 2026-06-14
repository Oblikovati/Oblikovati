// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"

	"oblikovati.org/math"
)

// primKind classifies a constraint geometry input.
type primKind uint8

const (
	// pointKind is a single position (a vertex or work point).
	pointKind primKind = iota
	// lineKind is an axis: a point on the line plus its direction (an edge or a
	// cylinder/cone axis).
	lineKind
	// planeKind is a plane: a point on it plus its normal (a planar face or work plane).
	planeKind
)

// Primitive is a constraint geometry input expressed in a component's definition space:
// a point, an axis line, or a plane, plus an optional radius for a curved input (a
// cylinder's, used by tangent). The solver transforms it into assembly space through an
// occurrence's live placement on every residual evaluation, so a finite-difference step
// on the placement variables flows through cleanly.
type Primitive struct {
	kind   primKind
	point  math.Point3      // a point on the entity (plane origin / axis point / vertex)
	dir    math.UnitVector3 // plane normal or axis direction; ignored for a point
	radius math.Scalar      // cylinder radius for tangent; 0 otherwise
}

// unitZ is the placeholder direction stored on a point primitive (its direction is never
// read); +Z is an arbitrary valid unit vector.
func unitZ() math.UnitVector3 {
	z, _ := math.NewUnitVector3(0, 0, 1)
	return z
}

// PointPrimitive builds a point input (a vertex or work point).
func PointPrimitive(p math.Point3) Primitive {
	return Primitive{kind: pointKind, point: p, dir: unitZ()}
}

// LinePrimitive builds an axis input through p along dir (an edge or axis).
func LinePrimitive(p math.Point3, dir math.UnitVector3) Primitive {
	return Primitive{kind: lineKind, point: p, dir: dir}
}

// PlanePrimitive builds a plane input through p with the given normal (a planar face).
func PlanePrimitive(p math.Point3, normal math.UnitVector3) Primitive {
	return Primitive{kind: planeKind, point: p, dir: normal}
}

// CylinderPrimitive builds an axis input carrying a radius (a cylindrical face), for
// tangent constraints.
func CylinderPrimitive(axisPoint math.Point3, axisDir math.UnitVector3, radius math.Scalar) Primitive {
	return Primitive{kind: lineKind, point: axisPoint, dir: axisDir, radius: radius}
}

// TransformedBy maps the primitive's reference point and direction through m, returning the
// transformed primitive (radius unchanged). The head uses it with an occurrence's inverse
// placement to convert a face picked in assembly space back into the component's definition
// space, which is where a constraint input lives (the solver re-applies the placement).
func (p Primitive) TransformedBy(m math.Matrix4) Primitive {
	out := p
	out.point = m.TransformPoint(p.point)
	if d, err := math.UnitVector3FromVector(m.TransformVector(p.dir.AsVector())); err == nil {
		out.dir = d
	}
	return out
}

// worldPoint maps the primitive's reference point into assembly space through m.
func worldPoint(m math.Matrix4, prim Primitive) math.Point3 {
	return m.TransformPoint(prim.point)
}

// worldDir maps the primitive's direction into assembly space through m (rotation only —
// a direction has no position). The result is unit length when m is rigid.
func worldDir(m math.Matrix4, prim Primitive) math.Vector3 {
	return m.TransformVector(prim.dir.AsVector())
}

// tangentFrame returns two orthonormal vectors perpendicular to d. It is the basis used
// to express a "direction a is parallel to d" relation as exactly two residuals (the
// components of a along the frame), so an alignment removes its true two rotational DOF
// without a spurious third, redundant row.
func tangentFrame(d math.Vector3) (math.Vector3, math.Vector3) {
	seed := math.V3(1, 0, 0)
	if stdmath.Abs(d.X) > 0.9 {
		seed = math.V3(0, 1, 0)
	}
	u := d.Cross(seed)
	u = u.Scale(1 / u.Length())
	v := d.Cross(u)
	return u, v.Scale(1 / v.Length())
}

// alignResiduals returns the two residuals that vanish when a is parallel to target
// (either sense; the sense is fixed by the warm-started branch). It is the rotational
// core of mate/flush/insert.
func alignResiduals(a, target math.Vector3) []float64 {
	u, v := tangentFrame(target)
	return []float64{a.Dot(u), a.Dot(v)}
}

// perpDistanceResiduals returns the two residuals that vanish when point pA lies on the
// line through pB along axis — the perpendicular offset of pA from the axis, expressed in
// the axis's tangent frame. It makes two axes coincident (together with alignResiduals).
func perpDistanceResiduals(pA, pB math.Point3, axis math.Vector3) []float64 {
	u, v := tangentFrame(axis)
	d := pB.VectorTo(pA)
	return []float64{d.Dot(u), d.Dot(v)}
}

// gapResidual returns the signed offset of pB from pA along normal, minus the target
// offset — zero when the two planes are the wanted distance apart.
func gapResidual(pA, pB math.Point3, normal math.Vector3, offset float64) float64 {
	return pA.VectorTo(pB).Dot(normal) - offset
}
