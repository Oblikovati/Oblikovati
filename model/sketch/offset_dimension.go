// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Offset dimensioning. Inventor puts a driving offset dimension on the geometry when you commit an
// offset, so the offset distance is editable straight away. AddOffset*Dimension adds ONE such
// dimension at the CURRENT offset distance: the perpendicular distance from the offset to a source
// LINE (AddOffsetDim), or — for a purely circular offset — the offset curve's radius. sources/offsets
// pair element-for-element (as OffsetConnectedLoop returns them). It returns (nil,false) when nothing
// dimensionable exists (an all-projected-curve offset). The value is snapshotted so the dimension is
// satisfied at creation — no solve jump — and drives the offset from then on.

// AddOffsetLoopDimension dimensions a loop/chain offset from its source path and offset entities.
func (s *Sketch) AddOffsetLoopDimension(path *Path, offsets []Entity) (*DimensionConstraint, bool) {
	return s.addOffsetDimension(chainSourceEntities(path), offsets)
}

// AddOffsetSingleDimension dimensions a single-curve offset.
func (s *Sketch) AddOffsetSingleDimension(source, offset Entity) (*DimensionConstraint, bool) {
	return s.addOffsetDimension([]Entity{source}, []Entity{offset})
}

// chainSourceEntities returns a chain's source entities in traversal order.
func chainSourceEntities(path *Path) []Entity {
	ents := path.Entities()
	out := make([]Entity, len(ents))
	for i, pe := range ents {
		out[i] = pe.Entity
	}
	return out
}

// addOffsetDimension prefers a perpendicular line offset dimension, falling back to a radius on a
// purely circular offset.
func (s *Sketch) addOffsetDimension(sources, offsets []Entity) (*DimensionConstraint, bool) {
	if d, ok := s.offsetLineDimension(sources, offsets); ok {
		return d, true
	}
	return s.offsetRadiusDimension(offsets)
}

// offsetLineDimension puts a perpendicular offset dimension on the first line source/offset pair, at
// the current offset distance.
func (s *Sketch) offsetLineDimension(sources, offsets []Entity) (*DimensionConstraint, bool) {
	for i, off := range offsets {
		if i >= len(sources) {
			break
		}
		sl, lineSrc := sources[i].(*Line)
		ol, lineOff := off.(*Line)
		if !lineSrc || !lineOff {
			continue
		}
		dist := math.Scalar(stdmath.Abs(perpDistanceToLine(sl.A.Position(), sl.B.Position(), ol.A.Position())))
		if d, err := s.dimCons.AddOffsetDim(ol.A, sl, false, lengthExpr(dist)); err == nil {
			return d, true
		}
	}
	return nil, false
}

// offsetRadiusDimension puts a radius dimension on the first circular offset, at its current radius.
func (s *Sketch) offsetRadiusDimension(offsets []Entity) (*DimensionConstraint, bool) {
	for _, off := range offsets {
		if oc, ok := off.(CircularCurve); ok {
			if d, err := s.dimCons.AddRadius(oc, lengthExpr(oc.CurveRadius())); err == nil {
				return d, true
			}
		}
	}
	return nil, false
}
