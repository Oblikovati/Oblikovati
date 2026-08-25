// SPDX-License-Identifier: GPL-2.0-only

package sketch

// ConstrainOffset is Inventor's "Constrain Offset" (on by default): it binds freshly-created offset
// geometry to the source it was offset from so the copy stays associative — each offset line parallel
// to its source line, each offset circular curve concentric to its source, and the offset's own
// corners joined coincident so it stays a connected loop. The offset distance itself is left free, as
// Inventor does, so the user can drive it with a dimension.
//
// The type switches on sketch entities live HERE, inside the sketch package, per the sketch-entity
// seam (#1624); callers hand ordered source/offset entities (a projected-curve source is skipped —
// reference geometry has no sketch DOF to bind to).

// ConstrainOffsetLoop constrains a whole offset loop/chain: offsets[i] was produced from
// path.Entities()[i], so they pair element-for-element (as OffsetConnectedLoop returns them). It also
// joins the offset's corners coincident (wrapping the last to the first for a closed path). Returns
// the number of constraints added.
func (s *Sketch) ConstrainOffsetLoop(path *Path, offsets []Entity) int {
	ents := path.Entities()
	n := 0
	for i, off := range offsets {
		if i < len(ents) {
			n += s.constrainOffsetPair(ents[i].Entity, off)
		}
	}
	return n + s.joinOffsetCorners(offsets, path.IsClosed())
}

// ConstrainOffsetSingle constrains a single offset curve to its source (no corners to join).
func (s *Sketch) ConstrainOffsetSingle(source, offset Entity) int {
	return s.constrainOffsetPair(source, offset)
}

// constrainOffsetPair adds the one geometric constraint that keeps an offset associative to its
// source: parallel for a line, concentric for a circle/arc. It returns 0 (no constraint) when the
// source is neither — a projected curve, a spline — since those have no matching native binding.
func (s *Sketch) constrainOffsetPair(src, off Entity) int {
	if sl, ok := src.(*Line); ok {
		if ol, ok := off.(*Line); ok {
			s.geomCons.AddParallel(sl, ol)
			return 1
		}
		return 0
	}
	sc, sok := src.(CircularCurve)
	oc, ook := off.(CircularCurve)
	if sok && ook {
		s.geomCons.AddConcentric(sc, oc)
		return 1
	}
	return 0
}

// joinOffsetCorners makes each offset element's end point coincident with the next element's start
// point, so the offset stays a connected loop when the source moves. A closed path also joins the
// last element back to the first.
func (s *Sketch) joinOffsetCorners(offsets []Entity, closed bool) int {
	n := 0
	for i := 0; i+1 < len(offsets); i++ {
		n += s.coincidentCorner(offsets[i], offsets[i+1])
	}
	if closed && len(offsets) >= 2 {
		n += s.coincidentCorner(offsets[len(offsets)-1], offsets[0])
	}
	return n
}

// coincidentCorner joins the end of a to the start of b. A full circle has no endpoints, so a loop
// that is a single circle joins nothing.
func (s *Sketch) coincidentCorner(a, b Entity) int {
	_, aEnd, aok := offsetEndpoints(a)
	bStart, _, bok := offsetEndpoints(b)
	if !aok || !bok {
		return 0
	}
	s.geomCons.AddCoincident(aEnd, bStart)
	return 1
}

// offsetEndpoints returns an offset element's start and end points, in traversal order (the order
// buildOffsetEntities created them). ok is false for a curve with no two endpoints (a full circle).
func offsetEndpoints(e Entity) (start, end *Point, ok bool) {
	switch v := e.(type) {
	case *Line:
		return v.A, v.B, true
	case *Arc:
		return v.Start, v.End, true
	}
	return nil, nil, false
}
