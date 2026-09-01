// SPDX-License-Identifier: GPL-2.0-only

package brepfixture

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The #2009 "starved rail" strip: a single-span exact circular arc extruded to a height, carried as
// a rational B-spline whose weights are BUNCHED so the parametrisation is fast at one end and slow
// at the other. Its two axial edges are exact straight segments, which a naive discretizer gives
// two points regardless of length — the starvation the densifier exists to fix. Tessellation,
// picking and the metric-scale memo are all specified against it, in different packages.

// The bunched strip's parameters, shared so every package measures the SAME fixture.
const (
	BunchedStripRadius = 2.0
	BunchedStripSweep  = 1.0
	BunchedStripHeight = 30.0
	BunchedStripBunch  = 10.0
)

// cylindricalStripSurfaceBunched is cylindricalStripSurface with the SAME exact circular-arc shape
// (bunch=1 reproduces it exactly) but a non-uniform v-speed near v=0 when bunch>1 — reproducing the
// recon's real-fixture pathology ("coons parameterization bunches hard toward the rail", §2: local
// |∂P/∂u| ≈ 100× the domain mean) WITHOUT changing the surface's geometric shape or its closed-form
// area (area is parameterization-invariant). This exploits the classical fact that a conic's
// rational quadratic Bezier representation is non-unique: for control points (P0,P1,P2) and weights
// (w0,w1,w2), only the projective ratio w1²/(w0·w2) fixes the conic's SHAPE — asymmetrically
// rescaling w0→w0/bunch, w2→w2·bunch preserves that ratio (same shape, verified exactly on-circle by
// the caller) while concentrating parameter range away from v=0 (so a small Δv near the v=0 rail —
// where the interior grid's first off-boundary column lands — covers an anomalously LARGE 3D
// distance: exactly the local-metric spike that forces discretizeEdge's starved 2-point rail against
// a saturated interior grid into one giant off-chord triangle).
func CylindricalStripSurface(tb testing.TB, r, sweep, h, bunch float64) geom.BSplineSurface {
	tb.Helper()
	if sweep <= 0 || sweep >= stdmath.Pi {
		tb.Fatalf("CylindricalStripSurface: sweep=%g must be in (0, pi) for a single-span exact arc", sweep)
	}
	alpha := sweep / 2
	cosA, sinA := stdmath.Cos(alpha), stdmath.Sin(alpha)
	p0 := math.P3(r*cosA, -r*sinA, 0)
	p1 := math.P3(r/cosA, 0, 0)
	p2 := math.P3(r*cosA, r*sinA, 0)
	ctrl := [][]math.Point3{
		{p0, p1, p2},
		{math.P3(p0.X, p0.Y, h), math.P3(p1.X, p1.Y, h), math.P3(p2.X, p2.Y, h)},
	}
	w0, w2 := 1/bunch, bunch // ratio w1^2/(w0*w2) = cos^2(alpha) unchanged: same conic, different speed
	w := [][]float64{{w0, cosA, w2}, {w0, cosA, w2}}
	s, err := geom.NewBSplineSurface(1, 2, ctrl, w, []float64{0, 0, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		tb.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

// CylindricalStripBody wraps the strip surface in a single-face body. The two axial (u) edges are
// its EXACT straight rails (geom.LineSegment — the #2009 starvation trigger, since a straight edge
// discretizes to just 2 points regardless of length); the two angular (v) edges are the surface's
// EXACT circular arcs, which are already sagitta-adaptive. Returns the body and its two rails.
//
// Example: body, rails := brepfixture.CylindricalStripBody(t, s, r, sweep, h)
func CylindricalStripBody(tb testing.TB, s geom.BSplineSurface, r, sweep, h float64) (body *topo.Body, rails [2]*topo.Edge) {
	tb.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("fixture", "cylstrip", 0)))
	c00, c10, c11, c01 := s.PointAt(0, 0), s.PointAt(1, 0), s.PointAt(1, 1), s.PointAt(0, 1)
	v := make([]*topo.Vertex, 4)
	for i, c := range []math.Point3{c00, c10, c11, c01} {
		v[i] = bld.AddVertex(c, topo.NewLineage(topo.Tok("fixture", "v", i)))
	}
	arcLo, arcHi := stripArcs(tb, r, sweep, h)
	railV0 := bld.AddEdge(geom.NewLineSegment(c00, c10), v[0], v[1], topo.NewLineage(topo.Tok("fixture", "e", 0)))
	arcU1 := bld.AddEdge(arcHi, v[1], v[2], topo.NewLineage(topo.Tok("fixture", "e", 1)))
	railV1 := bld.AddEdge(geom.NewLineSegment(c11, c01), v[2], v[3], topo.NewLineage(topo.Tok("fixture", "e", 2)))
	arcU0 := bld.AddEdge(arcLo, v[0], v[3], topo.NewLineage(topo.Tok("fixture", "e", 3)))
	bld.AddFace(s, topo.NewLineage(topo.Tok("fixture", "face", 0)),
		topo.OuterLoop(topo.Fwd(railV0), topo.Fwd(arcU1), topo.Fwd(railV1), topo.Rev(arcU0)))
	return bld.Build(), [2]*topo.Edge{railV0, railV1}
}

// stripArcs builds the strip's two exact bounding arcs, at z=0 and z=h.
func stripArcs(tb testing.TB, r, sweep, h float64) (lo, hi geom.Arc3d) {
	tb.Helper()
	alpha := sweep / 2
	lo, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r, -alpha, sweep)
	if err != nil {
		tb.Fatalf("strip arc at z=0 (r=%g, sweep=%g): %v", r, sweep, err)
	}
	hi, err = geom.NewArc3d(math.P3(0, 0, h), math.V3(0, 0, 1), math.V3(1, 0, 0), r, -alpha, sweep)
	if err != nil {
		tb.Fatalf("strip arc at z=%g (r=%g, sweep=%g): %v", h, r, sweep, err)
	}
	return lo, hi
}
