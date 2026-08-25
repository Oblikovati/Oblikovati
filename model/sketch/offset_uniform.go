// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Uniform offset constraining (Inventor parity). A committed offset loop gets ONE driving offset
// dimension, and every OTHER offset line is bound to the SAME distance by a driven offset constraint,
// so editing that one dimension moves the whole loop uniformly. It replaces the plain
// ConstrainOffsetLoop + AddOffsetLoopDimension pair: each element gets exactly one distance-binding
// constraint (no doubled parallels), so a rectangle offset is fully constrained (DOF 0) and rigidly
// uniform. Circular offsets stay concentric — their radius is NOT driven by the line dimension yet, a
// known limit for rounded loops (the arc keeps its offset radius). A driven constraint is a closure,
// so a serialized reload freezes it at its last distance (see OffsetConstraint).

// ConstrainOffsetLoopUniform constrains a whole offset loop associatively AND uniformly. offsets pair
// element-for-element with path.Entities(). Returns the driving dimension, or nil when the loop has
// nothing to dimension (e.g. an all-projected-curve offset).
func (s *Sketch) ConstrainOffsetLoopUniform(path *Path, offsets []Entity) *DimensionConstraint {
	ents := path.Entities()
	s.joinOffsetCorners(offsets, path.IsClosed())
	dim, dimIdx := s.dimensionFirstOffsetLine(ents, offsets)
	for i, off := range offsets {
		if i >= len(ents) || i == dimIdx {
			continue // the dimensioned segment already carries parallel + the driving dimension
		}
		s.constrainUniformOffsetPair(ents[i].Entity, off, dim)
	}
	return dim
}

// dimensionFirstOffsetLine adds the parallel + a driving offset dimension to the FIRST line
// source/offset pair, returning the dimension and that pair's index. For a pure-circle loop (no line)
// it dimensions the offset radius instead and returns index -1.
func (s *Sketch) dimensionFirstOffsetLine(ents []ProfileEntity, offsets []Entity) (*DimensionConstraint, int) {
	for i, off := range offsets {
		if i >= len(ents) {
			break
		}
		sl, okSrc := ents[i].Entity.(*Line)
		ol, okOff := off.(*Line)
		if !okSrc || !okOff {
			continue
		}
		s.geomCons.AddParallel(sl, ol)
		dist := stdmath.Abs(perpDistanceToLine(sl.A.Position(), sl.B.Position(), ol.A.Position()))
		if d, err := s.dimCons.AddOffsetDim(ol.A, sl, false, lengthExpr(math.Scalar(dist))); err == nil {
			return d, i
		}
		return nil, -1
	}
	if d, ok := s.offsetRadiusDimension(offsets); ok {
		return d, -1
	}
	return nil, -1
}

// constrainUniformOffsetPair binds one non-dimensioned offset element to its source at the SAME
// distance as the driving dimension: a line by a driven offset constraint (parallel + distance =
// ±dim), anything circular by concentric (radius kept at its offset value — not driven).
func (s *Sketch) constrainUniformOffsetPair(src, off Entity, dim *DimensionConstraint) {
	sl, okSrc := src.(*Line)
	ol, okOff := off.(*Line)
	if !okSrc || !okOff || dim == nil {
		s.constrainOffsetPair(src, off) // circular → concentric; a stray line without a dim → parallel
		return
	}
	sign := 1.0
	if perpDistanceToLine(sl.A.Position(), sl.B.Position(), ol.A.Position()) < 0 {
		sign = -1.0 // this segment's offset sits on the opposite side of its source's left normal
	}
	p := dim.Parameter()
	s.geomCons.AddOffsetDriven(sl, ol, func() float64 { return sign * p.ModelValue() })
}
