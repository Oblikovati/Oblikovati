// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"reflect"
	"testing"

	"oblikovati.org/math"
)

// The constraint codec registry guards (#1625, audit I2), mirroring the entity
// registry guards (#1624): the registered kind sets must equal the persisted
// vocabulary exactly, a half or duplicate registration must panic, and one
// instance of EVERY registered kind must round-trip to an identical recipe —
// so a kind added to one codec half fails here at CI, not at the user's save.
// That drift class shipped #1574 (Symmetry enumerable but not creatable) and
// left Equal3D failing every save at runtime until this registry.

// persistedConstraintVocabulary2D / 3D are the exact constraint kind sets a
// recipe may contain, one line per kind. Adding a constraint type means
// registering its codec AND extending this list AND giving the round-trip
// tests below an instance — all in this package, all enforced together.
var (
	persistedConstraintVocabulary2D = []ConstraintKind{
		CoincidentKind, HorizontalKind, VerticalKind,
		SingleLineHorizontalKind, SingleLineVerticalKind,
		PointOnLineKind, MidpointKind, PointOnCircleKind,
		ParallelKind, PerpendicularKind, CollinearKind, EqualLengthKind,
		ConcentricKind, EqualRadiusKind, CircularTangentKind, TangentKind,
		SymmetryKind, LineSymmetryKind, CircularSymmetryKind, ArcMidpointKind,
		EllipseParallelKind, EllipsePerpendicularKind, EllipseCollinearKind,
		EllipseHorizontalKind, EllipseVerticalKind,
		FixKind, SmoothKind,
		GroundKind, OffsetKind, PatternLinkKind, TextBoxAnchorKind, CustomKind,
	}
	persistedConstraintVocabulary3D = []ConstraintKind{
		CoincidentKind, Collinear3PointKind, ConcentricKind, EqualKind,
		ParallelKind, PerpendicularKind, MidpointKind, GroundKind,
		ParallelToXAxisKind, ParallelToYAxisKind, ParallelToZAxisKind,
		ParallelToXYKind, ParallelToXZKind, ParallelToYZKind,
		TangentKind, SmoothKind, SplineFitPointsKind, HelicalJoinKind, BendKind,
	}
)

func TestConstraintCodecRegistryMatchesVocabulary(t *testing.T) {
	assertConstraintKindSetsEqual(t, "2D", registeredConstraintKinds2D(), persistedConstraintVocabulary2D)
	assertConstraintKindSetsEqual(t, "3D", registeredConstraintKinds3D(), persistedConstraintVocabulary3D)
}

func assertConstraintKindSetsEqual(t *testing.T, family string, got, want []ConstraintKind) {
	t.Helper()
	gotSet, wantSet := constraintKindSet(got), constraintKindSet(want)
	for k := range wantSet {
		if !gotSet[k] {
			t.Errorf("%s constraint kind %q is in the persisted vocabulary but has no codec", family, k)
		}
	}
	for k := range gotSet {
		if !wantSet[k] {
			t.Errorf("%s constraint kind %q has a codec but is not in the persisted vocabulary list", family, k)
		}
	}
}

func constraintKindSet(kinds []ConstraintKind) map[ConstraintKind]bool {
	out := make(map[ConstraintKind]bool, len(kinds))
	for _, k := range kinds {
		out[k] = true
	}
	return out
}

func TestRegisterConstraintCodecRejectsBadRegistration(t *testing.T) {
	complete2D := constraintCodec{
		encode: func(Constraint) (ConstraintData, error) { return ConstraintData{}, nil },
		decode: func(*sketchRestorer, ConstraintData) error { return nil },
	}
	complete3D := constraintCodec3D{
		encode: func(Constraint) (Constraint3DRow, error) { return Constraint3DRow{}, nil },
		decode: func(*Sketch3D, Constraint3DRow, []*Point3D, map[int]Entity) error { return nil },
	}
	mustPanic(t, "2D empty kind", func() { registerConstraintCodec("", complete2D) })
	mustPanic(t, "2D encode-only", func() { registerConstraintCodec("testHalf", constraintCodec{encode: complete2D.encode}) })
	mustPanic(t, "2D decode-only", func() { registerConstraintCodec("testHalf", constraintCodec{decode: complete2D.decode}) })
	mustPanic(t, "2D duplicate", func() { registerConstraintCodec(CoincidentKind, complete2D) })
	mustPanic(t, "3D empty kind", func() { registerConstraintCodec3D("", complete3D) })
	mustPanic(t, "3D encode-only", func() { registerConstraintCodec3D("testHalf", constraintCodec3D{encode: complete3D.encode}) })
	mustPanic(t, "3D duplicate", func() { registerConstraintCodec3D(BendKind, complete3D) })
}

// TestEvery2DConstraintKindRoundTripsIdentically saves a sketch holding one
// constraint of every registered 2D kind, restores it, saves again, and
// demands the two recipes match — the enumerating closure over the registry.
func TestEvery2DConstraintKindRoundTripsIdentically(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	populateEvery2DConstraintKind(s)

	first, err := sc.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	assertConstraintRowsCoverVocabulary(t, constraintKindsIn(first), persistedConstraintVocabulary2D)

	fresh := NewSketches()
	if err := fresh.ApplyRecipe(first); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	second, err := fresh.MarshalRecipe()
	if err != nil {
		t.Fatalf("re-MarshalRecipe: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("recipe changed across a round trip:\nfirst:  %+v\nsecond: %+v", first, second)
	}
}

// populateEvery2DConstraintKind creates one constraint of each persisted 2D
// kind through the same factories the tools use. The text box goes first: its
// anchor constraint is auto-created at entity-add time, matching the restore
// order (entities before constraint rows).
func populateEvery2DConstraintKind(s *Sketch) {
	s.texts.AddStyled(math.P2(7, 7), "note", 0.5, 0, TextHJustify(0), TextVJustify(0), "sans", 0)
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(2, 0))
	l2 := s.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(2, 1))
	c1 := s.Circles().AddByCenterRadius(math.P2(5, 0), 1)
	c2 := s.Circles().AddByCenterRadius(math.P2(8, 0), 1)
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 5), math.P2(1, 5), math.P2(0, 6), true)
	g := s.GeometricConstraints()

	g.AddCoincident(l1.A, l2.A)
	g.AddHorizontal(l1.A, l1.B)
	g.AddVertical(l2.A, l2.B)
	g.AddLineHorizontal(l1)
	g.AddLineVertical(l2)
	g.AddPointOnLine(l2.A, l1)
	g.AddMidpoint(l2.B, l1)
	g.AddPointOnCircle(l1.B, c1)
	g.AddParallel(l1, l2)
	g.AddPerpendicular(l1, l2)
	g.AddCollinear(l1, l2)
	g.AddEqualLength(l1, l2)
	g.AddConcentric(c1, c2)
	g.AddEqualRadius(c1, c2)
	g.AddCircularTangent(c1, c2)
	g.AddTangent(l1, c1)
	g.AddSymmetry(l1.A, l1.B, l2)
	// Entity symmetry (#1870) and arc midpoint (#1872): dedicated geometry so the round-trip
	// exercises each new codec.
	sl1 := s.Lines().AddByTwoPoints(math.P2(10, 0), math.P2(11, 1))
	sl2 := s.Lines().AddByTwoPoints(math.P2(13, 0), math.P2(12, 1))
	axis := s.Lines().AddByTwoPoints(math.P2(11.5, -1), math.P2(11.5, 2))
	g.AddLineSymmetry(sl1, sl2, axis)
	g.AddCircularSymmetry(c1, c2, axis)
	g.AddMidpointToArc(s.Points().Add(math.P2(0, 6)), arc)
	// Ellipse-axis relations (#1879): one instance of each kind, mixing major/minor selectors.
	ell := s.Ellipses().Add(math.P2(20, 0), math.V2(1, 0), 3, 1)
	ellOp := func(major bool) AxisOperand { op, _ := EllipseAxisOf(ell, major); return op }
	g.AddEllipseParallel(ellOp(true), LineAxis(l1))
	g.AddEllipsePerpendicular(ellOp(false), LineAxis(l2))
	g.AddEllipseCollinear(ellOp(true), LineAxis(l1))
	g.AddEllipseHorizontal(ellOp(true))
	g.AddEllipseVertical(ellOp(false))
	g.AddFix(l1.A)
	g.AddSmooth(l1, arc, l1.B, arc.Start)
	g.AddGroundPoints(l2.A, l2.B)
	g.AddOffset(l1, l2, 1)
	g.AddPatternLink(l1.A, l2.A)
	// Anchor the tag to a line AND a bare point: points persist in the Points
	// table, so the restore fallback for point-anchored tags is exercised too
	// (surfaced by the #1625 constraintseam live test).
	mustAddCustom(g, l1, l1.A)
}

// mustAddCustom adds a tag constraint; the factory only errors on an empty
// client id, which this call never passes.
func mustAddCustom(g *GeometricConstraints, ents ...Entity) {
	if _, err := g.AddCustom("test.client", "tag", ents); err != nil {
		panic(err)
	}
}

// TestEvery3DConstraintKindRoundTripsIdentically is the 3D closure. It pins
// the #1625 Equal3D regression: creatable and enumerable over the wire but
// missing from both serialize switches, it failed every save at runtime.
func TestEvery3DConstraintKindRoundTripsIdentically(t *testing.T) {
	c := NewSketches3D()
	s := c.Add()
	populateEvery3DConstraintKind(t, s)

	first, err := c.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("MarshalRecipe3D: %v", err)
	}
	assertConstraintRowsCoverVocabulary(t, constraintKinds3DIn(first), persistedConstraintVocabulary3D)

	fresh := NewSketches3D()
	if err := fresh.ApplyRecipe3D(first); err != nil {
		t.Fatalf("ApplyRecipe3D: %v", err)
	}
	second, err := fresh.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("re-MarshalRecipe3D: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("3D recipe changed across a round trip:\nfirst:  %+v\nsecond: %+v", first, second)
	}
}

// populateEvery3DConstraintKind creates one constraint of each persisted 3D kind.
func populateEvery3DConstraintKind(t *testing.T, s *Sketch3D) {
	t.Helper()
	uz := mustUnit3(t, 0, 0, 1)
	a := s.AddPoint3D(math.P3(0, 0, 3))
	b := s.AddPoint3D(math.P3(1, 0, 3))
	c := s.AddPoint3D(math.P3(2, 0, 3))
	l1 := s.AddLine3D(math.P3(0, 0, 0), math.P3(2, 0, 0))
	l2 := s.AddLine3D(math.P3(0, 1, 0), math.P3(2, 1, 0))
	c1 := s.AddCircle3D(math.P3(0, 0, 2), uz, 1.5)
	c2 := s.AddCircle3D(math.P3(4, 0, 2), uz, 1.5)
	arc := s.AddArc3D(math.P3(0, 5, 0), math.P3(1, 5, 0), math.P3(0, 6, 0), true)
	h := s.AddHelix3D(math.P3(0, 0, 2), uz, 1.5, 0.5, 0, 3, false)
	sp := s.AddSpline3D([]math.Point3{{X: 0, Y: 8, Z: 0}, {X: 1, Y: 9, Z: 0}, {X: 2, Y: 8, Z: 1}}, false, true)
	g := s.GeometricConstraints3D()

	g.Add(NewCoincident3D(a, b))
	g.Add(NewCollinear3D(a, b, c))
	g.Add(NewConcentric3D(a, b))
	g.Add(mustEqual3D(t, c1, c2))
	g.Add(NewParallel3D(l1, l2))
	g.Add(NewPerpendicular3D(l1, l2))
	g.Add(NewMidpoint3D(a, l1))
	g.Add(NewGround3D(a))
	g.Add(NewParallelToXAxis3D(l1))
	g.Add(NewParallelToYAxis3D(l1))
	g.Add(NewParallelToZAxis3D(l1))
	g.Add(NewParallelToXYPlane3D(l2))
	g.Add(NewParallelToXZPlane3D(l2))
	g.Add(NewParallelToYZPlane3D(l2))
	g.Add(NewTangent3D(l1, arc, l1.B, arc.Start))
	g.Add(NewSmooth3D(arc, sp, arc.End, sp.Points[0]))
	addSplineFit3D(t, g, sp, s.AddPoint3D(math.P3(0, 8, 0)))
	addHelical3D(t, g, h, c1)
	if _, err := s.AddBend3D(s.AddLine3D(math.P3(6, 0, 0), math.P3(9, 0, 0)), s.AddLine3D(math.P3(9, 0, 0), math.P3(9, 3, 0)), 0.5); err != nil {
		t.Fatalf("AddBend3D: %v", err)
	}
}

func addSplineFit3D(t *testing.T, g *GeometricConstraints, sp *Spline3D, p *Point3D) {
	t.Helper()
	c, err := NewSplineFitPoints3D(sp, p)
	if err != nil {
		t.Fatalf("NewSplineFitPoints3D: %v", err)
	}
	g.Add(c)
}

func addHelical3D(t *testing.T, g *GeometricConstraints, h *HelicalCurve3D, c *Circle3D) {
	t.Helper()
	hc, err := NewHelical3D(h, c)
	if err != nil {
		t.Fatalf("NewHelical3D: %v", err)
	}
	g.Add(hc)
}

// assertConstraintRowsCoverVocabulary demands the marshalled recipe exercises
// every kind in the vocabulary — a codec the round-trip does not cover is a
// hole in the closure.
func assertConstraintRowsCoverVocabulary(t *testing.T, got map[ConstraintKind]bool, want []ConstraintKind) {
	t.Helper()
	for _, k := range want {
		if !got[k] {
			t.Errorf("round-trip recipe has no %q constraint row — the closure does not cover its codec", k)
		}
	}
}

func constraintKindsIn(data []SketchData) map[ConstraintKind]bool {
	out := map[ConstraintKind]bool{}
	for _, sd := range data {
		for _, cd := range sd.Constraints {
			out[ConstraintKind(cd.Kind)] = true
		}
	}
	return out
}

func constraintKinds3DIn(data []SketchData3D) map[ConstraintKind]bool {
	out := map[ConstraintKind]bool{}
	for _, sd := range data {
		for _, cd := range sd.Constraints {
			out[ConstraintKind(cd.Kind)] = true
		}
	}
	return out
}
