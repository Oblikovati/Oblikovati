// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// coneCapRimFrustum is the named fixture for the convex closed cone-cap rim (the J1 topology): a frustum
// solid (bottom radius 100 at z=0, top radius 50 at z=200) whose top rim is a convex closed circular edge
// where the host cone meets its perpendicular cap plane — the edge the closed-band arm rounds into a full
// torus band.
func coneCapRimFrustum(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 200), 100, 50, "cone")
	if err != nil {
		t.Fatalf("SolidCylinderCone frustum fixture: %v", err)
	}
	return b
}

// topRimCircleEdge returns the fixture's top rim circle (a closed circular edge whose range-box centre is
// near topZ) — the convex cone-cap rim.
func topRimCircleEdge(t *testing.T, b *topo.Body, topZ float64) *topo.Edge {
	t.Helper()
	for _, e := range b.Edges() {
		if _, ok := e.Geometry().(geom.Circle); ok && stdmath.Abs(float64(e.RangeBox().Center().Z)-topZ) < 1e-3 {
			return e
		}
	}
	t.Fatalf("fixture has no top rim circle near z=%g", topZ)
	return nil
}

// TestClosedRimPick_DemuxesClosedRim asserts the P1 demux predicate: the two picks a CLOSED rim's single
// selection lands in the seam-vertex group (its StartVertex==EndVertex counts it twice) are recognized as
// one edge, not a 2-edge miter corner — so solveCorner takes no corner treatment and the rim reaches the
// closed-band arm assembly. A pick pair on an OPEN edge (distinct endpoints) is NOT a closed rim.
func TestClosedRimPick_DemuxesClosedRim(t *testing.T) {
	t.Parallel()
	b := coneCapRimFrustum(t)
	rim := topRimCircleEdge(t, b, 200)
	if rim.StartVertex().ID() != rim.EndVertex().ID() {
		t.Fatalf("fixture top rim edge %d is not closed (start %d != end %d)", rim.ID(), rim.StartVertex().ID(), rim.EndVertex().ID())
	}
	rimPick := filletPick{edge: rim, r0: 10, r1: 10}
	if !closedRimPick([]filletPick{rimPick, rimPick}) {
		t.Fatalf("closedRimPick did not recognize the closed rim %d counted twice as a non-miter", rim.ID())
	}
	seam := openSeamEdge(t, b)
	openPick := filletPick{edge: seam, r0: 10, r1: 10}
	if closedRimPick([]filletPick{rimPick, openPick}) {
		t.Fatalf("closedRimPick treated two DISTINCT edges (%d, %d) as a closed rim", rim.ID(), seam.ID())
	}
}

// openSeamEdge returns the fixture's cone seam (an OPEN line segment with distinct endpoints).
func openSeamEdge(t *testing.T, b *topo.Body) *topo.Edge {
	t.Helper()
	for _, e := range b.Edges() {
		if _, ok := e.Geometry().(geom.LineSegment); ok && e.StartVertex().ID() != e.EndVertex().ID() {
			return e
		}
	}
	t.Fatal("fixture has no open seam edge")
	return nil
}

// TestConeCapRimFillet_WeldsTorusBand drives the whole fillet path (demux + closed-band arm assembly) on the
// convex cone-cap rim and asserts a watertight torus-band solid: 4 faces (receded cone, receded cap disk,
// untouched bottom cap, one torus band), every edge 2-incident, valid + solid, watertight across tolerances.
func TestConeCapRimFillet_WeldsTorusBand(t *testing.T) {
	t.Parallel()
	b := coneCapRimFrustum(t)
	rim := topRimCircleEdge(t, b, 200)
	res, err := FilletEdgesCorner(b, []EdgeFilletRadii{{Key: rim.ReferenceKey(), R0: 10, R1: 10}}, CornerMiter, FillConcaveOutward)
	if err != nil {
		t.Fatalf("cone-cap rim fillet declined: %v", err)
	}
	if got := len(res.Faces()); got != 4 {
		t.Fatalf("cone-cap rim band has %d faces, want 4", got)
	}
	assertConeRimBandWatertight(t, res)
}

// assertConeRimBandWatertight checks the closed-band result is a watertight manifold solid carrying exactly
// one torus band (every edge 2-incident, valid + holes-contained + IsSolid). The full tessellation
// fold/area gate lives at model level (TestJ1ClosedRimTessellationFoldGate) on the real J1 body.
func assertConeRimBandWatertight(t *testing.T, res *topo.Body) {
	t.Helper()
	for _, e := range res.Edges() {
		if n := len(e.Uses()); n != 2 {
			t.Fatalf("cone-cap rim band edge %d is %d-incident, want exactly 2 (watertight manifold)", e.ID(), n)
		}
	}
	rep := validate.Validate(res)
	if !rep.Valid || !rep.HolesContained || !res.IsSolid() {
		t.Fatalf("cone-cap rim band not a valid solid: valid=%v holes=%v solid=%v issues=%v", rep.Valid, rep.HolesContained, res.IsSolid(), rep.Issues)
	}
	if bands := torusFaceCount(res); bands != 1 {
		t.Fatalf("cone-cap rim band has %d torus faces, want exactly 1", bands)
	}
}

// torusFaceCount counts the body's geom.Torus faces.
func torusFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Torus); ok {
			n++
		}
	}
	return n
}
