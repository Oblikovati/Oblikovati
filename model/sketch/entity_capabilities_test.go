// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"strings"
	"testing"

	"oblikovati.org/math"
)

// The entity capabilities added for #1624 (audit I1): every entity names its
// own kind and shape; circular kinds expose a radius; trim and projection
// rebind dispatch through the entity, not through consumer type switches.

func TestShapedEntityKindsAndPoints2D(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	populateEvery2DKind(t, sc, s)
	for _, e := range s.Entities() {
		se, ok := e.(ShapedEntity)
		if !ok {
			t.Errorf("entity %T does not implement ShapedEntity", e)
			continue
		}
		if se.Kind() == "" {
			t.Errorf("entity %T reports an empty kind", e)
		}
	}
	assertShape2DExamples(t, s)
}

// assertShape2DExamples pins the shape contract on representative kinds: the
// point orders the wire has always shipped.
func assertShape2DExamples(t *testing.T, s *Sketch) {
	t.Helper()
	line := s.Lines().Item(0)
	if got := line.ShapePoints(); len(got) != 2 || got[0] != line.A.Position() || got[1] != line.B.Position() {
		t.Errorf("line ShapePoints = %v, want [A B]", got)
	}
	circle := s.Circles().Item(0)
	if got := circle.ShapePoints(); len(got) != 1 || got[0] != circle.Center.Position() {
		t.Errorf("circle ShapePoints = %v, want [Center]", got)
	}
	if circle.ShapeRadius() != float64(circle.Radius) {
		t.Errorf("circle ShapeRadius = %v, want %v", circle.ShapeRadius(), circle.Radius)
	}
	arc := s.Arcs().Item(0)
	if got := arc.ShapePoints(); len(got) != 3 {
		t.Errorf("arc ShapePoints = %v, want [Center Start End]", got)
	}
	if arc.ShapeRadius() != float64(arc.Radius()) {
		t.Errorf("arc ShapeRadius = %v, want %v", arc.ShapeRadius(), arc.Radius())
	}
	if got := s.EquationCurves().Item(0).ShapePoints(); got != nil {
		t.Errorf("equation curve ShapePoints = %v, want nil (expression-derived)", got)
	}
}

func TestShapedEntityKindsAndPoints3D(t *testing.T) {
	c := NewSketches3D()
	s := c.Add()
	populateEvery3DKind(t, s)
	for _, e := range s.Entities() {
		se, ok := e.(ShapedEntity3D)
		if !ok {
			t.Errorf("3D entity %T does not implement ShapedEntity3D", e)
			continue
		}
		if se.Kind() == "" {
			t.Errorf("3D entity %T reports an empty kind", e)
		}
	}
	assertShape3DExamples(t, s)
}

// assertShape3DExamples pins the 3D shape contract: circular kinds carry their
// radius, free-form curves ship a display sample, splines split by fit flag.
func assertShape3DExamples(t *testing.T, s *Sketch3D) {
	t.Helper()
	kinds := map[EntityKind][]Entity{}
	for _, e := range s.Entities() {
		k := e.(ShapedEntity3D).Kind()
		kinds[k] = append(kinds[k], e)
	}
	if len(kinds[SplineKind]) != 1 || len(kinds[ControlPointSplineKind]) != 1 {
		t.Errorf("spline kinds = %d fit / %d control-point, want 1/1 (fit flag drives the kind)",
			len(kinds[SplineKind]), len(kinds[ControlPointSplineKind]))
	}
	circle := kinds[CircleKind][0].(*Circle3D)
	if circle.ShapeRadius() != float64(circle.Radius) {
		t.Errorf("3D circle ShapeRadius = %v, want %v", circle.ShapeRadius(), circle.Radius)
	}
	helix := kinds[HelicalKind][0].(*HelicalCurve3D)
	if helix.ShapeRadius() != float64(helix.StartRadius) {
		t.Errorf("helix ShapeRadius = %v, want start radius %v", helix.ShapeRadius(), helix.StartRadius)
	}
	if sample := kinds[SplineKind][0].(*Spline3D).ShapePoints3D(); len(sample) < 3 {
		t.Errorf("fit spline ShapePoints3D = %d points, want a display sample", len(sample))
	}
	ellipse := kinds[EllipseKind][0].(*Ellipse3D)
	if ellipse.ShapeRadius() != float64(ellipse.MajorRadius) {
		t.Errorf("3D ellipse ShapeRadius = %v, want major radius", ellipse.ShapeRadius())
	}
}

// TestDerivedCurve3DShapeIsIdentityOnly: the surface-derived curves have no
// enumeration geometry of their own (they re-evaluate on recompute), and the
// included kinds are always construction geometry.
func TestDerivedCurve3DShapeIsIdentityOnly(t *testing.T) {
	derived := []ShapedEntity3D{
		&IntersectionCurve3D{}, &SilhouetteCurve3D{}, &ProjectToSurfaceCurve3D{},
		&OnFaceCurve3D{}, &OffsetCurve3{},
	}
	for _, e := range derived {
		if e.Kind() == "" {
			t.Errorf("%T reports an empty kind", e)
		}
		if pts := e.ShapePoints3D(); pts != nil {
			t.Errorf("%T ShapePoints3D = %v, want nil (recompute-derived)", e, pts)
		}
	}
	if !(&IncludedCurve3D{}).IsConstruction() {
		t.Error("an included curve must always be construction geometry")
	}
	if (&IncludedCurve3D{}).Kind() != IncludedCurveKind || (&IncludedPoint3D{}).Kind() != IncludedPointKind {
		t.Error("included kinds must name themselves")
	}
}

func TestRadiusDOFDrivesTheEntity(t *testing.T) {
	c := NewSketches3D()
	s := c.Add()
	circle := s.AddCircle3D(math.P3(0, 0, 0), mustUnit3(t, 0, 0, 1), 2)
	helix := s.AddHelix3D(math.P3(0, 0, 0), mustUnit3(t, 0, 0, 1), 1, 0.5, 0, 3, false)
	*circle.RadiusDOF() = 5
	if circle.Radius != 5 {
		t.Errorf("circle radius after DOF write = %v, want 5", circle.Radius)
	}
	*helix.RadiusDOF() = 4
	if helix.StartRadius != 4 {
		t.Errorf("helix start radius after DOF write = %v, want 4", helix.StartRadius)
	}
}

// TestTrimCurveAtDispatchesByEntity: the single trim seam both drivers share
// (#1624) — trimmable kinds route to their trim, everything else is refused
// with the offending type named.
func TestTrimCurveAtDispatchesByEntity(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	// Two crossing lines so the trim has a crossing to cut at.
	l := s.Lines().AddByTwoPoints(math.P2(-1, 0), math.P2(1, 0))
	s.Lines().AddByTwoPoints(math.P2(0, -1), math.P2(0, 1))
	if _, err := s.TrimCurveAt(l, math.P2(0.5, 0)); err != nil {
		t.Fatalf("TrimCurveAt(line): %v", err)
	}
	text := s.texts.AddStyled(math.P2(0, 0), "x", 1, 0, TextHJustify(0), TextVJustify(0), "sans", 0)
	_, err := s.TrimCurveAt(text, math.P2(0, 0))
	if err == nil || !strings.Contains(err.Error(), "TextBox") {
		t.Errorf("TrimCurveAt(text) = %v, want a refusal naming the type", err)
	}
}

// fakeProjectionResolver resolves fixed descriptors for the rebind test.
type fakeProjectionResolver struct {
	pointSrc PointSource
	curveSrc CurveSource
}

func (f *fakeProjectionResolver) PointProjectionSource(kind, _ string) (PointSource, bool) {
	return f.pointSrc, kind == "vertex"
}

func (f *fakeProjectionResolver) CurveProjectionSource(kind, _ string, _ Plane) (CurveSource, bool) {
	return f.curveSrc, kind == "edge"
}

// TestRebindProjectionReattachesSources: the sketch re-attaches each restored projection's live
// source through the resolver — projected points (entities) and curve projections (Projection
// records driving reference entities, ADR-0055 phase 3) alike; unknown descriptors stay frozen.
func TestRebindProjectionReattachesSources(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	pp := s.RestoreProjectedPoint(nextID(), math.P2(1, 1), "vertex", "V1")
	pc := s.RestoreProjection(s.addReferencePolyline([]math.Point2{math.P2(0, 0), math.P2(1, 0)}), "edge", "E1")
	frozen := s.RestoreProjection(s.addReferencePolyline([]math.Point2{math.P2(2, 0), math.P2(3, 0)}), "mystery", "X")
	res := &fakeProjectionResolver{
		pointSrc: &fakePointSource{id: "V1", pos: math.P3(1, 1, 0)},
		curveSrc: &fakeCurveSource{id: "E1", pts: []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)}},
	}
	s.RebindProjections(res)
	if !pp.Linked() {
		t.Error("projected point must relink through the resolver")
	}
	if !pc.Linked() {
		t.Error("projected curve must relink through the resolver")
	}
	if frozen.Linked() {
		t.Error("an unresolvable descriptor must stay frozen")
	}
}
