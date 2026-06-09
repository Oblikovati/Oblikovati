// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// AutoDimension fully constrains the sketch (Inventor's Auto Dimension): it greedily adds
// constraints — real length/radius dimensions first, then point grounds to anchor the
// rest — until 0 DOF remains. Each candidate is accepted only if it strictly reduces the
// DOF *without* introducing redundancy, so the result is **well-constrained, never
// over-constrained** (unlike grounding whole entities, which double-fixes shared points).
// It returns the number of constraints added; an already-constrained sketch adds nothing.
func (s *Sketch) AutoDimension() int {
	added := 0
	for s.DegreesOfFreedom() > 0 {
		if !s.applyOneAutoCandidate() {
			break // no candidate can reduce DOF further without redundancy
		}
		added++
	}
	return added
}

// applyOneAutoCandidate adds the first candidate that strictly lowers DOF without adding
// redundancy, returning whether one was applied. Candidates are trial-added and undone if
// they don't help (the rank analysis is non-mutating, so trials don't move geometry).
func (s *Sketch) applyOneAutoCandidate() bool {
	before := s.AnalyzeConstraints()
	for _, add := range s.autoCandidates() {
		undo := add()
		after := s.AnalyzeConstraints()
		if after.DOF < before.DOF && after.Redundant <= before.Redundant {
			return true
		}
		undo()
	}
	return false
}

// autoCandidates lists the constraints AutoDimension may add, each as an add→undo pair:
// length dimensions on lines, radius dimensions on circles/arcs, then a ground per unique
// point to anchor any remaining (translational/positional) freedom.
func (s *Sketch) autoCandidates() []func() func() {
	dc := s.DimensionConstraints()
	var cs []func() func()
	for _, e := range s.Entities() {
		switch g := e.(type) {
		case *Line:
			line := g
			cs = append(cs, dimCandidate(func() (*DimensionConstraint, error) {
				return dc.AddDistance(line.A, line.B, lengthExpr(line.A.Position().DistanceTo(line.B.Position())))
			}, dc))
		case *Circle:
			circ := g
			cs = append(cs, dimCandidate(func() (*DimensionConstraint, error) {
				return dc.AddRadius(circ, lengthExpr(circ.Radius))
			}, dc))
		case *Arc:
			arc := g
			cs = append(cs, dimCandidate(func() (*DimensionConstraint, error) {
				return dc.AddDistance(arc.Center, arc.Start, lengthExpr(arc.Radius()))
			}, dc))
		}
	}
	for _, p := range s.uniqueEntityPoints() {
		cs = append(cs, func() func() {
			g := s.GeometricConstraints().AddGroundPoints(p)
			return func() { s.GeometricConstraints().Delete(g) }
		})
	}
	return cs
}

// dimCandidate wraps a dimension factory as an add→undo pair (a failed add is a no-op
// candidate that the caller rejects via the unchanged DOF).
func dimCandidate(add func() (*DimensionConstraint, error), dc *DimensionConstraints) func() func() {
	return func() func() {
		d, err := add()
		if err != nil {
			return func() {}
		}
		return func() { dc.Delete(d) }
	}
}

// uniqueEntityPoints returns each distinct constrainable point referenced by the sketch's
// entities, in first-seen order.
func (s *Sketch) uniqueEntityPoints() []*Point {
	seen := map[*Point]bool{}
	var out []*Point
	for _, e := range s.Entities() {
		for _, p := range entityPoints(e) {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

// lengthExpr formats a database-unit (cm) length as a dimension expression at full
// precision, so a dimension placed by AutoDimension pins the current geometry exactly.
func lengthExpr(cm math.Scalar) string {
	return fmt.Sprintf("%.12g cm", float64(cm))
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
