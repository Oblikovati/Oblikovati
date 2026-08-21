// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// near3D reports whether two model points are within a small tolerance.
func near3D(a, b math.Point3) bool { return float64(a.DistanceTo(b)) < 1e-9 }

// glyphAt returns the first glyph within tolerance of at, or false.
func glyphAt(glyphs []ConstraintGlyph3D, at math.Point3) (ConstraintGlyph3D, bool) {
	for _, g := range glyphs {
		if near3D(g.At, at) {
			return g, true
		}
	}
	return ConstraintGlyph3D{}, false
}

// TestConstraintGlyphs3DMarksEachLineOfAParallel is the #1998 (3D) core: a relation between two
// lines draws one marker per line, at the line's midpoint — not four (one per endpoint) and not one
// floating between them.
func TestConstraintGlyphs3DMarksEachLineOfAParallel(t *testing.T) {
	s := NewSketches3D().Add()
	l1 := s.AddLine3D(math.P3(0, 0, 0), math.P3(2, 0, 0)) // midpoint (1,0,0)
	l2 := s.AddLine3D(math.P3(0, 1, 0), math.P3(2, 1, 0)) // midpoint (1,1,0)
	s.GeometricConstraints3D().add(NewParallel3D(l1, l2))

	glyphs := s.ConstraintGlyphs()
	if len(glyphs) != 2 {
		t.Fatalf("parallel of two lines drew %d markers, want 2 (one per line)", len(glyphs))
	}
	for _, at := range []math.Point3{math.P3(1, 0, 0), math.P3(1, 1, 0)} {
		g, ok := glyphAt(glyphs, at)
		if !ok {
			t.Errorf("no marker at %v (a line midpoint); got %v", at, anchorsOf(glyphs))
			continue
		}
		if g.Kind != ParallelKind {
			t.Errorf("marker at %v kind = %v, want ParallelKind", at, g.Kind)
		}
	}
}

// TestConstraintGlyphs3DCollapsesCoincidentPoints: a coincidence between two points that no line
// owns as an operand draws ONE marker at their shared place, not two stacked on it.
func TestConstraintGlyphs3DCollapsesCoincidentPoints(t *testing.T) {
	s := NewSketches3D().Add()
	l1 := s.AddLine3D(math.P3(0, 0, 0), math.P3(2, 0, 0))
	l2 := s.AddLine3D(math.P3(2, 0, 0), math.P3(2, 2, 0))
	s.GeometricConstraints3D().add(NewCoincident3D(l1.B, l2.A)) // both at (2,0,0)

	glyphs := s.ConstraintGlyphs()
	if len(glyphs) != 1 {
		t.Fatalf("coincidence of two points drew %d markers, want 1 at the shared point; got %v",
			len(glyphs), anchorsOf(glyphs))
	}
	if !near3D(glyphs[0].At, math.P3(2, 0, 0)) {
		t.Errorf("coincidence marker at %v, want the shared point (2,0,0)", glyphs[0].At)
	}
	if glyphs[0].Kind != CoincidentKind {
		t.Errorf("coincidence marker kind = %v, want CoincidentKind", glyphs[0].Kind)
	}
}

// TestConstraintGlyphs3DCoversEveryEntityKind guards the entityDefiningPoints3D type switch: every
// concrete 3D entity that carries points must return them, or its constraints are silently unmarked.
func TestConstraintGlyphs3DCoversEveryEntityKind(t *testing.T) {
	p := &Point3D{X: 1, Y: 2, Z: 3}
	cases := map[string]Entity{
		"Line3D":    &Line3D{A: p, B: &Point3D{}},
		"Circle3D":  &Circle3D{Center: p},
		"Arc3D":     &Arc3D{Center: p, Start: &Point3D{}, End: &Point3D{}},
		"Ellipse3D": &Ellipse3D{Center: p},
		"Spline3D":  &Spline3D{Points: []*Point3D{p, {}}},
		"Point3D":   p,
	}
	for name, e := range cases {
		if len(entityDefiningPoints3D(e)) == 0 {
			t.Errorf("entityDefiningPoints3D(%s) returned no points — its constraints would be unmarked", name)
		}
	}
}

// anchorsOf lists glyph anchors for a failure message.
func anchorsOf(glyphs []ConstraintGlyph3D) []math.Point3 {
	out := make([]math.Point3, len(glyphs))
	for i, g := range glyphs {
		out[i] = g.At
	}
	return out
}

// TestCentroid3DAveragesPoints pins the midpoint math the anchor rests on.
func TestCentroid3DAveragesPoints(t *testing.T) {
	c := centroid3D([]*Point3D{{X: 0, Y: 0, Z: 0}, {X: 4, Y: 2, Z: 6}})
	if !near3D(c, math.P3(2, 1, 3)) {
		t.Errorf("centroid3D = %v, want (2,1,3)", c)
	}
}
