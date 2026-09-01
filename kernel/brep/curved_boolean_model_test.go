// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Acceptance for the curved boolean face model + analytic imprint (M2 Phase 1,
// Oblikovati/Oblikovati#1334): facesOfAny flattens curved bodies the planar facesOf rejects,
// loop edges come out oriented along the loop, and curvedImprint returns the exact conic for
// the analytic surface pairs (deferring curved∩curved to Phase 2).

// cylSide returns the curvedFace whose surface is the cylindrical side of a SolidCylinder.
func cylSide(t *testing.T, faces []curvedFace) curvedFace {
	t.Helper()
	for _, f := range faces {
		if _, ok := f.surface.(geom.Cylinder); ok {
			return f
		}
	}
	t.Fatal("no cylindrical face found")
	return curvedFace{}
}

// TestFacesOfAnyFlattensCylinder: a SolidCylinder the planar facesOf rejects flattens here into
// three curvedFaces — two planar caps and one true cylindrical side — each carrying its surface,
// loops, and lineage.
func TestFacesOfAnyFlattensCylinder(t *testing.T) {
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	if _, ok := facesOf(cyl); ok {
		t.Fatal("planar facesOf should reject a cylinder (precondition for needing facesOfAny)")
	}
	faces := facesOfAny(cyl)
	if len(faces) != 3 {
		t.Fatalf("facesOfAny gave %d faces, want 3", len(faces))
	}
	planes, cylinders := 0, 0
	for _, f := range faces {
		switch f.surface.(type) {
		case geom.Plane:
			planes++
		case geom.Cylinder:
			cylinders++
		}
		if len(f.loops) == 0 {
			t.Errorf("face on %T has no boundary loops", f.surface)
		}
		if len(f.lineage.Tokens()) == 0 {
			t.Errorf("face on %T carries no lineage (reference key would not survive)", f.surface)
		}
	}
	if planes != 2 || cylinders != 1 {
		t.Errorf("got %d planar + %d cylindrical faces, want 2 + 1", planes, cylinders)
	}
}

// TestLoopEdgesWalkContiguously: a planar block face's loop edges must chain — each edge's
// oriented end meets the next edge's oriented start — so the loop is a closed contiguous ring.
func TestLoopEdgesWalkContiguously(t *testing.T) {
	box, err := SolidBlock(math.P3(0, 0, 0), math.P3(2, 3, 4), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	faces := facesOfAny(box)
	loop := faces[0].loops[0]
	n := len(loop.edges)
	if n < 3 {
		t.Fatalf("block face loop has %d edges, want ≥3", n)
	}
	for i := range n {
		end := loop.edges[i].end()
		start := loop.edges[(i+1)%n].start()
		if d := float64(end.DistanceTo(start)); d > 1e-9 {
			t.Errorf("edge %d end %v does not meet edge %d start %v (gap %g)", i, end, (i+1)%n, start, d)
		}
	}
}

// TestClosedCircleEdgeSpansDomain: a cylinder cap's boundary is a single closed-circle edge
// (start vertex == end vertex); its loopEdge must span the curve's whole [0,1] domain so the
// ring closes on itself rather than collapsing to a point.
func TestClosedCircleEdgeSpansDomain(t *testing.T) {
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	for _, f := range facesOfAny(cyl) {
		pl, ok := f.surface.(geom.Plane)
		if !ok {
			continue
		}
		_ = pl
		loop := f.loops[0]
		if len(loop.edges) != 1 {
			t.Fatalf("cap loop has %d edges, want 1 closed circle", len(loop.edges))
		}
		e := loop.edges[0]
		if _, isCircle := e.curve.(geom.Circle); !isCircle {
			t.Fatalf("cap boundary edge is %T, want geom.Circle", e.curve)
		}
		if stdmath.Abs(e.t1-e.t0) < 0.99 { // a full circle domain is [0,1]
			t.Errorf("closed circle edge spans only [%g,%g], want the full domain", e.t0, e.t1)
		}
	}
}

// TestCurvedImprintCylinderPlaneIsCircle: a cylinder side imprinted against a plane ⟂ its axis
// is the exact section circle — at the plane's height, with the cylinder's radius.
func TestCurvedImprintCylinderPlaneIsCircle(t *testing.T) {
	cyl, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 4)
	side := cylSide(t, facesOfAny(cyl))
	plane, _ := geom.NewPlane(math.P3(0, 0, 2.5), math.V3(0, 0, 1))
	curves, ok := curvedImprint(side, curvedFace{surface: plane}, geom.ResolutionForSize(1))
	if !ok || len(curves) != 1 {
		t.Fatalf("cylinder∩plane: handled=%v, %d curves; want handled, 1 circle", ok, len(curves))
	}
	c, isCircle := curves[0].(geom.Circle)
	if !isCircle {
		t.Fatalf("imprint is %T, want geom.Circle", curves[0])
	}
	if stdmath.Abs(c.Radius-3) > 1e-9 {
		t.Errorf("imprint radius %g, want 3", c.Radius)
	}
	if z := float64(c.Center.Z); stdmath.Abs(z-2.5) > 1e-9 {
		t.Errorf("imprint center z %g, want 2.5", z)
	}
}

// TestCurvedImprintSpherePlaneIsCircle: a sphere imprinted against an offset plane is the circle
// of radius sqrt(r²−d²) at the plane.
func TestCurvedImprintSpherePlaneIsCircle(t *testing.T) {
	sphere, _ := geom.NewSphere(math.P3(0, 0, 0), 5)
	plane, _ := geom.NewPlane(math.P3(0, 0, 3), math.V3(0, 0, 1))
	curves, ok := curvedImprint(curvedFace{surface: sphere}, curvedFace{surface: plane}, geom.ResolutionForSize(1))
	if !ok || len(curves) != 1 {
		t.Fatalf("sphere∩plane: handled=%v, %d curves; want handled, 1 circle", ok, len(curves))
	}
	c := curves[0].(geom.Circle)
	if want := stdmath.Sqrt(25 - 9); stdmath.Abs(c.Radius-want) > 1e-9 {
		t.Errorf("imprint radius %g, want %g", c.Radius, want)
	}
}

// TestCurvedImprintPlanePlaneIsLine: two non-parallel planar faces imprint to a line (the planar
// case still flows through the curved model).
func TestCurvedImprintPlanePlaneIsLine(t *testing.T) {
	box, _ := SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "box")
	faces := facesOfAny(box)
	a, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	b, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(1, 0, 0))
	_ = faces
	curves, ok := curvedImprint(curvedFace{surface: a}, curvedFace{surface: b}, geom.ResolutionForSize(1))
	if !ok || len(curves) != 1 {
		t.Fatalf("plane∩plane: handled=%v, %d curves; want handled, 1 line", ok, len(curves))
	}
	if _, isLine := curves[0].(geom.Line); !isLine {
		t.Errorf("imprint is %T, want geom.Line", curves[0])
	}
}

// TestCurvedImprintRuledPairIsExactSection: a cylinder∩cylinder pair is the ruled∩quadric closed form
// (#3489) — the two exact section loops, not the marched chains it used to defer to. The imprint must
// hand those on, since every edge stitched from them inherits their exactness.
func TestCurvedImprintRuledPairIsExactSection(t *testing.T) {
	a, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	b, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 2)
	curves, ok := curvedImprint(curvedFace{surface: a}, curvedFace{surface: b}, geom.ResolutionForSize(1))
	if !ok || len(curves) != 2 {
		t.Fatalf("cylinder∩cylinder: handled=%v, %d curves; want the 2 exact section loops", ok, len(curves))
	}
	for i, c := range curves {
		if _, isArc := c.(geom.RuledQuadricArc); !isArc {
			t.Errorf("imprint curve %d is %T, want the exact geom.RuledQuadricArc", i, c)
		}
	}
}

// TestCurvedImprintTorusPairDefers: a torus is neither a straight-ruled parametrisation nor an implicit
// quadric, so no closed form applies and curvedImprint must report handled=false (the caller routes the
// pair to the SSI tracer), NOT an empty "they don't cross" result.
func TestCurvedImprintTorusPairDefers(t *testing.T) {
	a, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 4, 1)
	b, _ := geom.NewCylinder(math.P3(4, 0, 0), math.V3(0, 0, 1), 0.5)
	if _, ok := curvedImprint(curvedFace{surface: a}, curvedFace{surface: b}, geom.ResolutionForSize(1)); ok {
		t.Error("torus∩cylinder should defer (handled=false) to the tracer, not be handled analytically")
	}
}
