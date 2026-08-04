// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// Show Constraints needs a marker per geometric constraint. The anchor is derived from the
// constraint's own Variables, so these tests pin that derivation for the shapes it has to cover:
// point relations, relations spanning two entities, and relations on a scalar DOF (a radius) that
// is not a coordinate at all.

// glyphOfKind returns the single glyph of the given kind, failing if there is not exactly one.
func glyphOfKind(t *testing.T, sk *Sketch, kind ConstraintKind) ConstraintGlyph {
	t.Helper()
	var found []ConstraintGlyph
	for _, g := range sk.ConstraintGlyphs() {
		if g.Kind == kind {
			found = append(found, g)
		}
	}
	if len(found) != 1 {
		t.Fatalf("got %d glyphs of kind %q, want exactly 1", len(found), kind)
	}
	return found[0]
}

// closeTo reports whether two sketch points coincide within a tight tolerance.
func closeTo(a, b math.Point2) bool {
	return stdmath.Abs(float64(a.X-b.X)) < 1e-9 && stdmath.Abs(float64(a.Y-b.Y)) < 1e-9
}

// TestCoincidenceGlyphSitsOnThePointsItJoins: a coincidence acts on two points, so its marker
// belongs at their midpoint — which is where they end up once the solver honours it.
func TestCoincidenceGlyphSitsOnThePointsItJoins(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(4, 2))
	sk.GeometricConstraints().AddCoincident(a, b)

	g := glyphOfKind(t, sk, CoincidentKind)
	if want := math.P2(2, 1); !closeTo(g.At, want) {
		t.Errorf("coincidence glyph at %v, want the midpoint %v", g.At, want)
	}
}

// TestLineRelationGlyphAveragesEachOperandOnce is why the anchor de-duplicates points: a relation
// between two lines exposes four coordinates per line, and counting a point once per coordinate
// would still average correctly here — but a shared endpoint would be counted twice and drag the
// marker onto the busier operand.
func TestLineRelationGlyphAveragesEachOperandOnce(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	// Two lines sharing the point (0,0): l1 runs to (4,0), l2 to (0,4). The three DISTINCT points
	// average to (4/3, 4/3); double-counting the shared corner would pull it to (1,1).
	shared := sk.Points().Add(math.P2(0, 0))
	l1 := sk.Lines().Add(shared, sk.Points().Add(math.P2(4, 0)))
	l2 := sk.Lines().Add(shared, sk.Points().Add(math.P2(0, 4)))
	sk.GeometricConstraints().AddPerpendicular(l1, l2)

	g := glyphOfKind(t, sk, PerpendicularKind)
	if want := math.P2(4.0/3.0, 4.0/3.0); !closeTo(g.At, want) {
		t.Errorf("perpendicular glyph at %v, want %v — the shared corner was counted twice", g.At, want)
	}
}

// TestRadiusRelationGlyphSitsAtTheCentres: an equal-radius constraint reads two radii, but its
// operands expose their centres alongside them, so the marker lands between the circles. This is
// the case that proved the anchor needs no special handling for scalar DOFs — mapping radii to
// their centres changed nothing.
func TestRadiusRelationGlyphSitsAtTheCentres(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	c1 := sk.Circles().Add(sk.Points().Add(math.P2(0, 0)), 1)
	c2 := sk.Circles().Add(sk.Points().Add(math.P2(6, 0)), 2)
	sk.GeometricConstraints().AddEqualRadius(c1, c2)

	g := glyphOfKind(t, sk, EqualRadiusKind)
	if want := math.P2(3, 0); !closeTo(g.At, want) {
		t.Errorf("equal-radius glyph at %v, want the centres' midpoint %v", g.At, want)
	}
}

// pinnedRelation is a named fake standing in for a system-owned constraint that DOES act on
// geometry. No shipped kind is both non-deletable and numeric — the two existing system-owned
// records constrain nothing, so they are already excluded for having no position — which would
// leave the "only draw what Delete can remove" rule untested and free to rot. This fake exercises
// it directly.
type pinnedRelation struct {
	constraintBase
	P *Point
}

func (c *pinnedRelation) Residuals() []float64           { return []float64{0} }
func (c *pinnedRelation) Variables() []*math.Scalar      { return []*math.Scalar{&c.P.X, &c.P.Y} }
func (c *pinnedRelation) ConstraintKind() ConstraintKind { return FixKind }
func (c *pinnedRelation) Deletable() bool                { return false }

// TestNonDeletableConstraintsGetNoGlyph: a marker the user can select but not delete is a dead
// end, so what is drawn is restricted to what Delete can actually remove.
func TestNonDeletableConstraintsGetNoGlyph(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	p := sk.Points().Add(math.P2(3, 3))
	sk.GeometricConstraints().Add(&pinnedRelation{constraintBase: newConstraint(), P: p})

	if got := sk.ConstraintGlyphs(); len(got) != 0 {
		t.Errorf("got %d markers %v, want none — a system-owned relation was offered for deletion", len(got), got)
	}
}

// TestRecordsWithNoGeometryGetNoGlyph: a text box auto-creates an anchor record that lives in the
// constraint collection but constrains nothing numerically, so there is nowhere on the sketch to
// draw it.
func TestRecordsWithNoGeometryGetNoGlyph(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	sk.TextBoxes().Add(math.P2(1, 1), "hello", 0.5, 0, TextLeft)
	if sk.GeometricConstraints().Count() == 0 {
		t.Fatal("a text box should carry its anchor constraint; this test proves nothing without it")
	}
	a := sk.Points().Add(math.P2(0, 0))
	sk.GeometricConstraints().AddCoincident(a, sk.Points().Add(math.P2(2, 0)))

	glyphs := sk.ConstraintGlyphs()
	if len(glyphs) != 1 || glyphs[0].Kind != CoincidentKind {
		t.Errorf("got %d markers %v, want only the user's coincidence", len(glyphs), glyphs)
	}
}

// TestEveryUserConstraintGetsExactlyOneGlyph: the count has to match, or Show Constraints would
// quietly hide relations the user placed.
func TestEveryUserConstraintGetsExactlyOneGlyph(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(4, 0))
	l1 := sk.Lines().Add(a, b)
	l2 := sk.Lines().Add(sk.Points().Add(math.P2(0, 3)), sk.Points().Add(math.P2(4, 3)))
	g := sk.GeometricConstraints()
	g.AddHorizontal(a, b)
	g.AddParallel(l1, l2)
	g.AddEqualLength(l1, l2)

	if got := len(sk.ConstraintGlyphs()); got != 3 {
		t.Errorf("got %d glyphs for 3 user constraints", got)
	}
}

// TestGlyphsForGeometryBuiltTheWayToolsBuildIt is the regression for the defect the live app
// exposed while every test above passed: the tests hand-built fixtures with Points().Add, but a
// line tool — and the wire — draw with Lines().AddByTwoPoints, which allocates endpoints WITHOUT
// registering them in the Points collection. Walking the collection therefore found the fixtures'
// points and none of a real sketch's, so no marker was drawn in the running app.
func TestGlyphsForGeometryBuiltTheWayToolsBuildIt(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	bottom := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(8, 0))
	right := sk.Lines().AddByTwoPoints(math.P2(8, 0), math.P2(8, 5))
	g := sk.GeometricConstraints()
	g.AddLineHorizontal(bottom)
	g.AddPerpendicular(bottom, right)

	glyphs := sk.ConstraintGlyphs()
	if len(glyphs) != 2 {
		t.Fatalf("got %d markers for 2 constraints on AddByTwoPoints geometry, want 2", len(glyphs))
	}
	// The horizontal acts on the bottom line's two endpoints, so its marker is their midpoint.
	for _, gl := range glyphs {
		if gl.Kind == SingleLineHorizontalKind && !closeTo(gl.At, math.P2(4, 0)) {
			t.Errorf("horizontal marker at %v, want the bottom edge's midpoint {4 0}", gl.At)
		}
	}
}
