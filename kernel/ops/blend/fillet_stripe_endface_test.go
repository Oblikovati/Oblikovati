// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Unit cover for the Oblikovati#2083 discriminator. The whole fix turns on one question — does the
// run-out section plane coincide with a face already at the corner? — so the pieces that answer it
// are pinned here, and the end-to-end behaviour in fillet_stripe_sharp_corner_test.go.

// squareFaceBody is a unit box, built the same way the fillet tests build theirs, so the helpers see
// real topology rather than a hand-wired stub.
func squareFaceBody(t *testing.T) *topo.Body {
	t.Helper()
	b := subd.ToBody(subd.Box(1, 1, 1), "box")
	if b == nil {
		t.Fatal("subd.ToBody returned no body")
	}
	return b
}

// vertexAt returns the body vertex at p.
func vertexAt(t *testing.T, b *topo.Body, p math.Point3) *topo.Vertex {
	t.Helper()
	for _, v := range b.Vertices() {
		if float64(v.Point().DistanceTo(p)) < 1e-9 {
			return v
		}
	}
	t.Fatalf("no vertex at %v", p)
	return nil
}

// planarFaceThrough returns the body's planar face lying in the plane through p with normal n.
// Selecting on the normal ALONE would not do: a box's opposite faces are parallel, so the first
// match is as likely to be the far one.
func planarFaceThrough(t *testing.T, b *topo.Body, p math.Point3, n math.Vector3) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || float64(pl.Normal().Cross(n).Length()) > 1e-9 {
			continue
		}
		if stdmath.Abs(float64(pl.Origin.VectorTo(p).Dot(pl.Normal()))) < 1e-9 {
			return f
		}
	}
	t.Fatalf("no planar face through %v with normal %v", p, n)
	return nil
}

// TestTerminalSectionPlaneComesFromTheCapArcPoints: the plane is read off the three points that
// already define the section arc, so it can never disagree with the arc the blend is bounded by.
func TestTerminalSectionPlaneComesFromTheCapArcPoints(t *testing.T) {
	t.Parallel()
	tm := stripeTerm{topA: math.P3(0.75, 1, 1), apex: math.P3(0.927, 1, 0.927), wallA: math.P3(1, 1, 0.75)}
	origin, n, ok := terminalSectionPlane(tm)
	if !ok {
		t.Fatal("a well-formed section produced no plane")
	}
	if float64(origin.DistanceTo(tm.topA)) > 1e-12 {
		t.Errorf("origin = %v, want the top foot %v", origin, tm.topA)
	}
	if d := stdmath.Abs(float64(n.Length()) - 1); d > 1e-12 {
		t.Errorf("|n| = %g, want unit", n.Length())
	}
	if d := stdmath.Abs(float64(n.Cross(math.V3(0, 1, 0)).Length())); d > 1e-9 {
		t.Errorf("normal %v is not the section plane's (y = 1)", n)
	}
}

// TestTerminalSectionPlaneRefusesACollapsedSection: three collinear section points define no plane,
// and returning one anyway would let a degenerate terminal claim a coincident face.
func TestTerminalSectionPlaneRefusesACollapsedSection(t *testing.T) {
	t.Parallel()
	tm := stripeTerm{topA: math.P3(0, 0, 0), apex: math.P3(1, 0, 0), wallA: math.P3(2, 0, 0)}
	if _, _, ok := terminalSectionPlane(tm); ok {
		t.Error("three collinear section points were accepted as a plane")
	}
}

// TestPlaneMatchesNeedsBothParallelAndCoincident: a parallel face at a different offset is a
// different plane, and a face through the same point at an angle is too. Either alone would let the
// wrong face be picked as the run-out's host.
func TestPlaneMatchesNeedsBothParallelAndCoincident(t *testing.T) {
	t.Parallel()
	b := squareFaceBody(t)
	top := planarFaceThrough(t, b, math.P3(0.5, 0.5, 1), math.V3(0, 0, 1))
	n := math.V3(0, 0, 1)
	if !planeMatches(top, math.P3(0.5, 0.5, 1), n, 1e-9) {
		t.Fatal("the top face did not match its own plane")
	}
	if planeMatches(top, math.P3(0.5, 0.5, 0.9), n, 1e-9) {
		t.Error("a plane offset by 0.1 matched: the coincidence test is not being applied")
	}
	if planeMatches(top, math.P3(0.5, 0.5, 1), math.V3(0, 1, 0), 1e-9) {
		t.Error("a perpendicular plane matched: the parallel test is not being applied")
	}
}

// TestPlaneMatchesOffsetIsModelRelative: the offset test is a DISTANCE, so it must scale with the
// model (ADR-0042). A fixed epsilon would reject a real coincidence on a large part.
func TestPlaneMatchesOffsetIsModelRelative(t *testing.T) {
	t.Parallel()
	b := squareFaceBody(t)
	top := planarFaceThrough(t, b, math.P3(0.5, 0.5, 1), math.V3(0, 0, 1))
	n := math.V3(0, 0, 1)
	if !planeMatches(top, math.P3(0.5, 0.5, 1+5e-4), n, 1e-3) {
		t.Error("an offset inside the given weld was rejected")
	}
	if planeMatches(top, math.P3(0.5, 0.5, 1+5e-3), n, 1e-3) {
		t.Error("an offset outside the given weld was accepted")
	}
}

// TestCoplanarFaceAtSkipsTheStripesOwnFaces: the shared face and the wall are excluded by hand,
// because at a run-out they meet the corner too and one of them can be parallel to the section.
func TestCoplanarFaceAtSkipsTheStripesOwnFaces(t *testing.T) {
	t.Parallel()
	b := squareFaceBody(t)
	corner := vertexAt(t, b, math.P3(1, 1, 1))
	side := planarFaceThrough(t, b, math.P3(0.5, 1, 0.5), math.V3(0, 1, 0))
	top := planarFaceThrough(t, b, math.P3(0.5, 0.5, 1), math.V3(0, 0, 1))
	wall := planarFaceThrough(t, b, math.P3(1, 0.5, 0.5), math.V3(1, 0, 0))
	if got := coplanarFaceAt(corner, math.P3(1, 1, 1), math.V3(0, 1, 0), top, wall, 1e-9); got != side {
		t.Errorf("coplanarFaceAt found %v, want the y=1 side face", got)
	}
	if got := coplanarFaceAt(corner, math.P3(1, 1, 1), math.V3(0, 1, 0), top, side, 1e-9); got != nil {
		t.Error("the excluded face was returned anyway")
	}
	if got := coplanarFaceAt(corner, math.P3(1, 1, 1), math.V3(1, 1, 1), top, wall, 1e-9); got != nil {
		t.Error("a plane matching no face at the corner still found one")
	}
}

// TestEdgeSharedByFindsTheSingleCommonEdge: the two boundary edges to cut back are named by the face
// pair they separate, so this lookup has to be exact — and must decline rather than guess.
func TestEdgeSharedByFindsTheSingleCommonEdge(t *testing.T) {
	t.Parallel()
	b := squareFaceBody(t)
	corner := vertexAt(t, b, math.P3(1, 1, 1))
	top := planarFaceThrough(t, b, math.P3(0.5, 0.5, 1), math.V3(0, 0, 1))
	side := planarFaceThrough(t, b, math.P3(0.5, 1, 0.5), math.V3(0, 1, 0))
	e := edgeSharedBy(corner, top, side)
	if e == nil {
		t.Fatal("no edge found between the top and the y=1 side at their shared corner")
	}
	for _, v := range e.Vertices() {
		if float64(v.Point().Z) != 1 || float64(v.Point().Y) != 1 {
			t.Errorf("edge endpoint %v is not on the top/side intersection line", v.Point())
		}
	}
	far := planarFaceThrough(t, b, math.P3(0.5, 0, 0.5), math.V3(0, 1, 0))
	if got := edgeSharedBy(corner, top, far); got != nil {
		t.Error("an edge was reported between two faces that do not meet at this corner")
	}
}

// TestSoleCoplanarFaceDeclinesTwoCandidates: two faces in the section plane is not the simple sharp
// run-out this path rebuilds, so it must hand back to the flat cap rather than pick one. The pair
// here is the top face of two unit boxes side by side — genuinely distinct faces, genuinely coplanar.
func TestSoleCoplanarFaceDeclinesTwoCandidates(t *testing.T) {
	t.Parallel()
	left, right := squareFaceBody(t), squareFaceBody(t)
	a := planarFaceThrough(t, left, math.P3(0.5, 0.5, 1), math.V3(0, 0, 1))
	b := planarFaceThrough(t, right, math.P3(0.5, 0.5, 1), math.V3(0, 0, 1))
	origin, n := math.P3(0.5, 0.5, 1), math.V3(0, 0, 1)
	if got := soleCoplanarFace([]*topo.Face{a}, origin, n, nil, nil, 1e-9); got != a {
		t.Fatal("a single candidate was not returned")
	}
	if got := soleCoplanarFace([]*topo.Face{a, b}, origin, n, nil, nil, 1e-9); got != nil {
		t.Error("two coplanar candidates produced a pick instead of a refusal")
	}
	if got := soleCoplanarFace([]*topo.Face{a, b}, origin, n, b, nil, 1e-9); got != a {
		t.Error("excluding one of the two candidates should leave exactly one")
	}
}

// TestSoleEdgeBoundingDeclinesTwoCandidates is the same refusal on the edge lookup: given two edges
// bounding the same face pair, neither is the one to cut back.
func TestSoleEdgeBoundingDeclinesTwoCandidates(t *testing.T) {
	t.Parallel()
	b := squareFaceBody(t)
	corner := vertexAt(t, b, math.P3(1, 1, 1))
	top := planarFaceThrough(t, b, math.P3(0.5, 0.5, 1), math.V3(0, 0, 1))
	side := planarFaceThrough(t, b, math.P3(0.5, 1, 0.5), math.V3(0, 1, 0))
	e := edgeSharedBy(corner, top, side)
	if e == nil {
		t.Fatal("the fixture has no shared edge to duplicate")
	}
	if got := soleEdgeBounding([]*topo.Edge{e, e}, top, side); got != nil {
		t.Error("two matching edges produced a pick instead of a refusal")
	}
}

// TestCurveBetweenRestrictsACurvedBoundary is the case the box fixtures cannot show: their end-face
// boundaries are straight, so a remnant carrying the WHOLE parent curve would still tessellate to the
// same chord. On a CURVED boundary it does not — tessellate.SampleEdgeCurve walks the curve's whole domain and
// only snaps the two end samples, so an untrimmed remnant would be drawn along the original sweep.
func TestCurveBetweenRestrictsACurvedBoundary(t *testing.T) {
	t.Parallel()
	// A quarter circle of radius 1 in the xy plane, from (1,0,0) to (0,1,0).
	arc, err := geom.Arc3dByThreePoints(math.P3(1, 0, 0), math.P3(stdmath.Sqrt2/2, stdmath.Sqrt2/2, 0), math.P3(0, 1, 0))
	if err != nil {
		t.Fatal(err)
	}
	foot, far := arc.PointAt(0.5*(dom(arc))), math.P3(0, 1, 0) // cut back to the arc's own midpoint
	sub, err := curveBetween(arc, foot, far, 1e-9)
	if err != nil {
		t.Fatalf("curveBetween: %v", err)
	}
	if d := float64(sub.PointAt(0).DistanceTo(foot)); d > 1e-9 {
		t.Errorf("the remnant starts %g from the section foot", d)
	}
	if d := float64(sub.PointAt(1).DistanceTo(far)); d > 1e-9 {
		t.Errorf("the remnant ends %g from the far vertex", d)
	}
	// Its own midpoint must be the midpoint of the SURVIVING span (67.5°), not of the whole arc (45°).
	want := math.P3(stdmath.Cos(3*stdmath.Pi/8), stdmath.Sin(3*stdmath.Pi/8), 0)
	if d := float64(sub.PointAt(0.5).DistanceTo(want)); d > 1e-9 {
		t.Errorf("the remnant's midpoint is %v, want %v — it still spans the whole parent arc",
			sub.PointAt(0.5), want)
	}
}

// TestCurveBetweenRefusesAPointOffTheCurve: a foot that is not on the boundary means the corner is
// not the one this path assumed, and a silent nearest-parameter guess would misdraw the face.
func TestCurveBetweenRefusesAPointOffTheCurve(t *testing.T) {
	t.Parallel()
	seg := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))
	if _, err := curveBetween(seg, math.P3(0.5, 5, 0), math.P3(1, 0, 0), 1e-9); err == nil {
		t.Error("a point 5 off the line was accepted as a cut-back foot")
	}
}

// dom is the parameter span of a curve, so a fixture can name a point part-way along it.
func dom(c geom.Curve3) float64 {
	lo, hi := c.Domain()
	return lo + hi
}

// TestResolveStripeEndsIsANoOpForAClosedLoop: a closed stripe has no terminals at all, so the
// detection must not read term[] — which is unset there.
func TestResolveStripeEndsIsANoOpForAClosedLoop(t *testing.T) {
	t.Parallel()
	ends := resolveStripeEnds(&tangentStripe{closed: true}, 1e-9)
	for t2, e := range ends {
		if e.active() {
			t.Errorf("terminal %d of a closed loop was classified active", t2)
		}
	}
}
