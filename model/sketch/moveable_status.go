// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// GeometryMoveableStatus — "can this entity be dragged?" — for interactive
// tools (M06-F11, Oblikovati/Oblikovati#626). The classification is exact, not
// heuristic: a variable is draggable iff its coordinate direction is NOT in
// the row space of the constraint Jacobian (moving it violates no constraint
// to first order). Entities whose variables are all determined split into
// fixed (pinned by a fix constraint or owning no variables at all) versus
// moveable-by-dimension-change (a driving dimension would have to relax).

// MoveableStatus classifies entity e against the sketch's current constraint
// system. Classifying many entities? Use [Sketch.MoveableClassifier] — the
// Jacobian analysis is computed once per classifier, not per entity.
func (s *Sketch) MoveableStatus(e Entity) types.GeometryMoveableStatus {
	return s.MoveableClassifier().Of(e)
}

// MoveableClassifier classifies entities against one snapshot of the
// constraint system.
type MoveableClassifier struct {
	cons []Constraint
	free map[*math.Scalar]bool
}

// MoveableClassifier captures the current constraint analysis.
func (s *Sketch) MoveableClassifier() *MoveableClassifier {
	cons := s.Constraints()
	return &MoveableClassifier{cons: cons, free: freeVariableSet(cons, s.variables())}
}

// Of classifies one entity.
func (mc *MoveableClassifier) Of(e Entity) types.GeometryMoveableStatus {
	vars, known := entityFreedomVars(e)
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
	if entityHasFixedPoint(mc.cons, vars) {
		return types.MoveableFixed
	}
	return types.MoveableByDimensionChange
}

// HasDrivingDimensionOn reports whether a driving (non-reference) dimension acts on any of entity
// e's freedom variables. It distinguishes the two ways an entity can be MoveableByDimensionChange:
// determined by a driving dimension (a fully-dimensioned line — Relax Mode's job to move, #791) or
// determined purely by geometric constraints (a fillet arc tangent to still-free lines, which a
// direct drag should move within the sketch's freedom, #2160). The drag gate uses it to allow the
// latter while still refusing the former.
func (s *Sketch) HasDrivingDimensionOn(e Entity) bool {
	vars, ok := entityFreedomVars(e)
	if !ok {
		return false
	}
	owned := make(map[*math.Scalar]bool, len(vars))
	for _, v := range vars {
		owned[v] = true
	}
	for _, d := range s.dimCons.All() {
		if d.Driven() {
			continue // a reference dimension only reports; it holds nothing
		}
		for _, v := range d.Variables() {
			if owned[v] {
				return true
			}
		}
	}
	return false
}

// entityFreedomVars returns the scalar DOFs that move entity e; known is
// false for kinds the classifier does not model (annotations, images).
func entityFreedomVars(e Entity) (vars []*math.Scalar, known bool) {
	switch t := e.(type) {
	case *Point:
		return []*math.Scalar{&t.X, &t.Y}, true
	case *Line:
		return []*math.Scalar{&t.A.X, &t.A.Y, &t.B.X, &t.B.Y}, true
	case *Circle:
		return []*math.Scalar{&t.Center.X, &t.Center.Y, &t.Radius}, true
	case *Arc:
		return []*math.Scalar{&t.Center.X, &t.Center.Y, &t.Start.X, &t.Start.Y, &t.End.X, &t.End.Y}, true
	case *Ellipse:
		return []*math.Scalar{&t.Center.X, &t.Center.Y, &t.MajorRadius, &t.MinorRadius}, true
	case *EllipticalArc:
		return []*math.Scalar{&t.Center.X, &t.Center.Y, &t.MajorRadius, &t.MinorRadius}, true
	case *Spline:
		return splineFreedomVars(t), true
	case *SplineHandle:
		return []*math.Scalar{&t.End.X, &t.End.Y}, true
	case *FixedSpline, *OffsetSpline, *EquationCurve:
		return nil, true // derived/immutable geometry owns no drag DOFs
	}
	return nil, false
}

// splineFreedomVars collects a spline's defining-point DOFs.
func splineFreedomVars(sp *Spline) []*math.Scalar {
	out := make([]*math.Scalar, 0, len(sp.Points)*2)
	for _, p := range sp.Points {
		out = append(out, &p.X, &p.Y)
	}
	return out
}

// freeVariableSet returns which variables of the universe remain free under
// the constraints: variable j is determined iff its unit direction eⱼ lies in
// the Jacobian's row space, tested by projecting eⱼ onto an orthonormal basis
// of the rows.
func freeVariableSet(cons []Constraint, universe []*math.Scalar) map[*math.Scalar]bool {
	basis := orthonormalRows(jacobian(cons, universe))
	free := make(map[*math.Scalar]bool, len(universe))
	for j, v := range universe {
		projected := 0.0
		for _, b := range basis {
			projected += b[j] * b[j] // eⱼ·b = b[j]
		}
		free[v] = 1-projected > freeVarTol
	}
	return free
}

// freeVarTol is the residual-norm² threshold below which a coordinate direction
// counts as captured by the constraints. The Jacobian rows are now exact analytic
// derivatives (#1417), so the slack absorbs Gram–Schmidt round-off rather than
// finite-difference noise; it is left unchanged to keep classifications stable.
const freeVarTol = 1e-6

// orthonormalRows builds an orthonormal basis of the row space (modified
// Gram–Schmidt, dropping near-dependent rows — the same job matrixRank does,
// but keeping the basis).
func orthonormalRows(rows [][]float64) [][]float64 {
	var basis [][]float64
	for _, row := range rows {
		r := append([]float64(nil), row...)
		for _, b := range basis {
			dot := dotVec(r, b)
			for k := range r {
				r[k] -= dot * b[k]
			}
		}
		if n := stdmath.Sqrt(dotVec(r, r)); n > 1e-7 {
			for k := range r {
				r[k] /= n
			}
			basis = append(basis, r)
		}
	}
	return basis
}

// dotVec is the dot product of two equal-length vectors.
func dotVec(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// entityHasFixedPoint reports whether a fix constraint pins any of the
// entity's variables — the distinction between "fixed" and "would move if a
// dimension were relaxed".
func entityHasFixedPoint(cons []Constraint, vars []*math.Scalar) bool {
	owned := make(map[*math.Scalar]bool, len(vars))
	for _, v := range vars {
		owned[v] = true
	}
	for _, c := range cons {
		fix, isFix := c.(*FixConstraint)
		if !isFix {
			continue
		}
		if owned[&fix.P.X] || owned[&fix.P.Y] {
			return true
		}
	}
	return false
}
