// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

// TestCopyCarriesInternalConstraintAndDimension: copying a line whose horizontal constraint and
// distance dimension both reference only that line carries both onto the target, and the copy
// stays solvable there (#1083 — #151's second acceptance criterion).
func TestCopyCarriesInternalConstraintAndDimension(t *testing.T) {
	src := NewSketches().Add(XYPlane())
	l := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(40, 0))
	src.GeometricConstraints().AddHorizontal(l.A, l.B)
	if _, err := src.DimensionConstraints().AddDistance(l.A, l.B, "40 mm"); err != nil {
		t.Fatalf("source dimension: %v", err)
	}

	dst := NewSketches().Add(XYPlane())
	clones, _ := dst.CopyEntitiesWithConstraints(src, []Entity{l}, math.V2(0, 50))
	if len(clones) != 1 {
		t.Fatalf("clones = %d, want 1", len(clones))
	}
	if got := dst.GeometricConstraints().Count(); got != 1 {
		t.Errorf("carried geometric constraints = %d, want 1", got)
	}
	if got := dst.DimensionConstraints().Count(); got != 1 {
		t.Errorf("carried dimensions = %d, want 1", got)
	}
	if res := dst.Solve(); !res.Converged {
		t.Errorf("copied sketch did not solve: %+v", res)
	}
}

// TestCopyDropsConstraintWithExternalOperand: a parallel constraint references two lines; copying
// only one of them drops the constraint (external operand), but copying both carries it (#1083).
func TestCopyDropsConstraintWithExternalOperand(t *testing.T) {
	src := NewSketches().Add(XYPlane())
	l1 := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(40, 0))
	l2 := src.Lines().AddByTwoPoints(math.P2(0, 10), math.P2(40, 10))
	src.GeometricConstraints().AddParallel(l1, l2)

	only := NewSketches().Add(XYPlane())
	only.CopyEntitiesWithConstraints(src, []Entity{l1}, math.V2(0, 0))
	if got := only.GeometricConstraints().Count(); got != 0 {
		t.Errorf("constraint with an external operand carried = %d, want 0 (dropped)", got)
	}

	both := NewSketches().Add(XYPlane())
	both.CopyEntitiesWithConstraints(src, []Entity{l1, l2}, math.V2(0, 0))
	if got := both.GeometricConstraints().Count(); got != 1 {
		t.Errorf("constraint with both operands copied carried = %d, want 1", got)
	}
}

// TestCopyDimensionMintsFreshParameter: when source and target share a parameter store (as the
// sketches of one part do), a copied driving dimension takes a fresh name rather than colliding
// with the source's, and keeps its expression (#1083).
func TestCopyDimensionMintsFreshParameter(t *testing.T) {
	store := param.NewParameters()
	src := NewSketches().Add(XYPlane())
	src.SetParameters(store)
	dst := NewSketches().Add(XYPlane())
	dst.SetParameters(store)

	l := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(40, 0))
	d, err := src.DimensionConstraints().AddDistance(l.A, l.B, "40 mm")
	if err != nil {
		t.Fatalf("source dimension: %v", err)
	}
	srcName := d.Parameter().Name()

	dst.CopyEntitiesWithConstraints(src, []Entity{l}, math.V2(0, 50))
	if dst.DimensionConstraints().Count() != 1 {
		t.Fatalf("carried dimensions = %d, want 1", dst.DimensionConstraints().Count())
	}
	nd := dst.DimensionConstraints().Item(0)
	if nd.Parameter().Name() == srcName {
		t.Errorf("copied dimension reused source parameter name %q; want a fresh one", srcName)
	}
	if got := nd.Parameter().Expression(); got != "40 mm" {
		t.Errorf("copied dimension expression = %q, want %q", got, "40 mm")
	}
}

// TestCopyCarriesEveryConstraintAndDimensionKind builds a sketch holding one of every carryable
// 2D geometric constraint and dimension kind, then copies the whole sketch: each relation's
// operands are inside the copied set, so all carry over (#1083). This exercises every carry path
// and dimension-dispatch branch.
func TestCopyCarriesEveryConstraintAndDimensionKind(t *testing.T) {
	src := NewSketches().Add(XYPlane())
	g := src.GeometricConstraints()
	dc := src.DimensionConstraints()

	pa := src.Points().Add(math.P2(0, 0))
	pb := src.Points().Add(math.P2(1, 0))
	l1 := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	l2 := src.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(4, 1))
	l3 := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 4))
	c1 := src.Circles().Add(src.Points().Add(math.P2(2, 2)), 1)
	c2 := src.Circles().Add(src.Points().Add(math.P2(5, 2)), 1)
	arc := src.Arcs().Add(src.Points().Add(math.P2(8, 0)), src.Points().Add(math.P2(9, 0)), src.Points().Add(math.P2(8, 1)), true)
	ell := src.Ellipses().AddWithCenter(src.Points().Add(math.P2(12, 0)), math.V2(1, 0), 2, 1)

	g.AddCoincident(pa, pb)
	g.AddHorizontal(l1.A, l1.B)
	g.AddVertical(l3.A, l3.B)
	g.AddLineHorizontal(l1) // single-line forms (#1871)
	g.AddLineVertical(l3)
	g.AddMidpoint(pa, l1)
	g.AddPointOnLine(pb, l2)
	g.AddPointOnCircle(pa, c1)
	g.AddParallel(l1, l2)
	g.AddPerpendicular(l1, l3)
	g.AddCollinear(l1, l2)
	g.AddEqualLength(l1, l2)
	g.AddConcentric(c1, c2)
	g.AddEqualRadius(c1, c2)
	g.AddCircularTangent(c1, c2)
	g.AddTangent(l1, c1)
	g.AddSymmetry(pa, pb, l3)
	// Ellipse-axis relations (#1879): a line aligned to the ellipse's major/minor axis.
	eop := func(major bool) AxisOperand { op, _ := EllipseAxisOf(ell, major); return op }
	g.AddEllipseParallel(eop(true), LineAxis(l1))
	g.AddEllipsePerpendicular(eop(false), LineAxis(l2))
	g.AddEllipseCollinear(eop(true), LineAxis(l3))
	g.AddFix(pa)

	// The six kinds the pre-#1637 switch silently dropped (TextBoxAnchor is the
	// documented skip, exercised in TestCopyWarnsOnTextBoxAnchorConstraint).
	sp1 := src.Splines().AddByPoints([]math.Point2{math.P2(20, 0), math.P2(21, 1), math.P2(22, 0)}, false)
	sp2 := src.Splines().AddByPoints([]math.Point2{math.P2(22, 0), math.P2(23, -1), math.P2(24, 0)}, false)
	j1, j2, ok := NearestSmoothJoin(sp1, sp2)
	if !ok {
		t.Fatal("no smooth join between the two splines")
	}
	g.AddSmooth(sp1, sp2, j1, j2)
	g.AddGround(l2)
	g.AddOffset(l1, l2, 1)
	g.AddPatternLink(pa, pb)
	if _, err := g.AddCustom("test-addin", "tag", []Entity{l1}); err != nil {
		t.Fatalf("source custom constraint: %v", err)
	}

	// add spreads a (dimension, error) factory result, failing fast on a build error.
	add := func(_ *DimensionConstraint, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("source dimension: %v", err)
		}
	}
	add(dc.AddDistance(pa, pb, "1 mm"))
	add(dc.AddAngle(l1, l3, "90 deg"))
	add(dc.AddRadius(c1, "1 mm"))
	add(dc.AddDiameter(c2, "2 mm"))
	add(dc.AddArcLength(arc, "1 mm"))
	add(dc.AddOffsetDim(pa, l2, "1 mm"))
	add(dc.AddThreePointAngle(pa, pb, l1.B, "45 deg"))
	add(dc.AddEllipseRadius(ell, "2 mm"))
	add(dc.AddTangentDistance(l1, c1, false, "1 mm"))

	wantC, wantD := g.Count(), dc.Count()

	dst := NewSketches().Add(XYPlane())
	_, warns := dst.CopyEntitiesWithConstraints(src, src.Entities(), math.V2(100, 0))
	if len(warns) != 0 {
		t.Errorf("copy warnings = %q, want none (every kind here is carryable)", warns)
	}
	if got := dst.GeometricConstraints().Count(); got != wantC {
		t.Errorf("carried geometric constraints = %d, want %d (every kind)", got, wantC)
	}
	if got := dst.DimensionConstraints().Count(); got != wantD {
		t.Errorf("carried dimensions = %d, want %d (every kind)", got, wantD)
	}
}

// TestCopyDropsDimensionWithExternalOperand: a distance dimension between a copied line's
// endpoint and a point left behind is silently dropped (#1083 — the dimension acceptance case).
func TestCopyDropsDimensionWithExternalOperand(t *testing.T) {
	src := NewSketches().Add(XYPlane())
	l := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(40, 0))
	outside := src.Points().Add(math.P2(40, 30))
	if _, err := src.DimensionConstraints().AddDistance(l.A, outside, "50 mm"); err != nil {
		t.Fatalf("source dimension: %v", err)
	}

	dst := NewSketches().Add(XYPlane())
	dst.CopyEntitiesWithConstraints(src, []Entity{l}, math.V2(0, 0)) // copy the line, not the point
	if got := dst.DimensionConstraints().Count(); got != 0 {
		t.Errorf("dimension with an external operand carried = %d, want 0 (dropped)", got)
	}
}

// TestCopyDimensionPreservesDrivenAndLimits: a copied dimension keeps its driven role and any
// drive limits (#1083).
func TestCopyDimensionPreservesDrivenAndLimits(t *testing.T) {
	src := NewSketches().Add(XYPlane())
	l := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(40, 0))
	d, err := src.DimensionConstraints().AddDistance(l.A, l.B, "40 mm")
	if err != nil {
		t.Fatalf("source dimension: %v", err)
	}
	d.SetDriven(true)
	d.SetLimits(10, 80)

	dst := NewSketches().Add(XYPlane())
	dst.CopyEntitiesWithConstraints(src, []Entity{l}, math.V2(0, 50))
	nd := dst.DimensionConstraints().Item(0)
	if !nd.Driven() {
		t.Error("copied dimension lost its driven role")
	}
	if lim := nd.Limits(); !lim.Enabled || lim.Min != 10 || lim.Max != 80 {
		t.Errorf("copied dimension limits = %+v, want {10 80 true}", lim)
	}
}

// TestCopyCarriesCurveAndAngleRelations exercises the curve- and line-pair carry paths:
// concentric circles and an angle dimension between two lines all survive a whole-sketch copy
// (#1083).
func TestCopyCarriesCurveAndAngleRelations(t *testing.T) {
	src := NewSketches().Add(XYPlane())
	c1 := src.Circles().Add(src.Points().Add(math.P2(0, 0)), 5)
	c2 := src.Circles().Add(src.Points().Add(math.P2(0, 0)), 9)
	src.GeometricConstraints().AddConcentric(c1, c2)
	src.GeometricConstraints().AddEqualRadius(c1, c2)

	l1 := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	l2 := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 10))
	if _, err := src.DimensionConstraints().AddAngle(l1, l2, "90 deg"); err != nil {
		t.Fatalf("source angle dimension: %v", err)
	}

	dst := NewSketches().Add(XYPlane())
	dst.CopyEntitiesWithConstraints(src, src.Entities(), math.V2(100, 0))
	if got := dst.GeometricConstraints().Count(); got != 2 {
		t.Errorf("carried geometric constraints = %d, want 2 (concentric + equal-radius)", got)
	}
	if got := dst.DimensionConstraints().Count(); got != 1 {
		t.Errorf("carried dimensions = %d, want 1 (angle)", got)
	}
}
