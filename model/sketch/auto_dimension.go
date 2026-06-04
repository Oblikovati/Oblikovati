// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "github.com/Oblikovati/oblikovati/math"

// AutoDimension fully constrains the sketch by grounding its under-constrained geometry in
// insertion order until 0 DOF remains (Inventor's Auto Dimension reaching a fully-defined
// sketch). It returns the number of ground constraints it added. A sketch that is already
// fully constrained adds nothing.
func (s *Sketch) AutoDimension() int {
	count := 0
	for _, e := range s.Entities() {
		if s.DegreesOfFreedom() == 0 {
			break
		}
		if isAnnotation(e) {
			continue
		}
		s.GeometricConstraints().AddGround(e)
		count++
	}
	return count
}

// isAnnotation reports whether an entity carries no constrainable points (image/fill/text/
// derived curves) and so cannot be grounded.
func isAnnotation(e Entity) bool {
	return len(entityPoints(e)) == 0
}

// OffsetChain offsets a connected chain of lines (given in order, end-to-start) by signed
// distance d, mitering consecutive offsets at their intersections so the result stays a
// connected chain. Returns the offset lines. A single line offsets like OffsetEntity.
func (s *Sketch) OffsetChain(lines []*Line, d math.Scalar) []*Line {
	out := make([]*Line, len(lines))
	for i, l := range lines {
		out[i] = s.offsetLine(l, d)
	}
	for i := 0; i+1 < len(out); i++ {
		miterJoin(s, out[i], out[i+1])
	}
	return out
}

// miterJoin moves the shared corner of two consecutive offset lines to the intersection of
// their supports (so the offset chain stays connected). Parallel neighbours are left as is.
func miterJoin(s *Sketch, a, b *Line) {
	t, _, ok := lineLineParams(a, b)
	if !ok {
		return
	}
	join := s.newPoint(lerpLine(a, t))
	a.B = join
	b.A = join
}
