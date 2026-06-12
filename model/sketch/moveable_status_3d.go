// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// 3D moveable-status classification — the Sketch3D side of M06-F11 (#626),
// reusing the dimension-agnostic free-variable analysis of
// moveable_status.go.

// MoveableClassifier3D classifies 3D entities against one snapshot of the
// constraint system.
type MoveableClassifier3D struct {
	cons []Constraint
	free map[*math.Scalar]bool
}

// MoveableClassifier captures the current 3D constraint analysis.
func (s *Sketch3D) MoveableClassifier() *MoveableClassifier3D {
	cons := s.Constraints()
	return &MoveableClassifier3D{cons: cons, free: freeVariableSet(cons, s.variables())}
}

// MoveableStatus classifies one entity; use [Sketch3D.MoveableClassifier] for
// many.
func (s *Sketch3D) MoveableStatus(e Entity) types.GeometryMoveableStatus {
	return s.MoveableClassifier().Of(e)
}

// Of classifies one 3D entity.
func (mc *MoveableClassifier3D) Of(e Entity) types.GeometryMoveableStatus {
	vars, known := entityFreedomVars3D(e)
	if !known {
		return types.MoveableUnknown
	}
	if len(vars) == 0 {
		return types.MoveableFixed
	}
	for _, v := range vars {
		if mc.free[v] {
			return types.MoveableFree
		}
	}
	return types.MoveableByDimensionChange
}

// entityFreedomVars3D returns the scalar DOFs that move a 3D entity; known is
// false for kinds the classifier does not model.
func entityFreedomVars3D(e Entity) (vars []*math.Scalar, known bool) {
	switch t := e.(type) {
	case *Point3D:
		return point3DVars(t), true
	case *Line3D:
		return append(point3DVars(t.A), point3DVars(t.B)...), true
	case *Circle3D:
		return append(point3DVars(t.Center), &t.Radius), true
	case *Arc3D:
		return append(append(point3DVars(t.Center), point3DVars(t.Start)...), point3DVars(t.End)...), true
	case *Spline3D:
		return spline3DVars(t), true
	case *SplineHandle3D:
		return point3DVars(t.End), true
	case *FixedSpline3D, *EquationCurve3D:
		return nil, true // immutable/derived geometry owns no drag DOFs
	}
	return nil, false
}

// point3DVars are one 3D point's scalar DOFs.
func point3DVars(p *Point3D) []*math.Scalar { return []*math.Scalar{&p.X, &p.Y, &p.Z} }

// spline3DVars collects a 3D spline's defining-point DOFs.
func spline3DVars(sp *Spline3D) []*math.Scalar {
	out := make([]*math.Scalar, 0, len(sp.Points)*3)
	for _, p := range sp.Points {
		out = append(out, point3DVars(p)...)
	}
	return out
}
