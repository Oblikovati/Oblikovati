// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestExtractObstacleIsClosedValence4 proves extractObstacle turns a real T6 ObstacleFeature into a
// 4-sided RailLoop whose sides chain end-to-start (RailLoop.Closed) — the precondition coons4Provider
// requires before it will even attempt a fill.
func TestExtractObstacleIsClosedValence4(t *testing.T) {
	t.Parallel()
	of := newT6Obstacle(t)
	loop, ok := extractObstacle(of)
	if !ok {
		t.Fatal("extractObstacle declined T6")
	}
	if loop.Valence() != 4 {
		t.Fatalf("valence = %d, want 4", loop.Valence())
	}
	if !loop.Closed(blendScale().Weld()) {
		t.Fatal("loop not closed")
	}
}

// TestExtractObstacleResolvesToCoons4 proves the extracted loop fills via the general coons4 tier and
// passes the F2 probe (the corrected, non-folding sign).
func TestExtractObstacleResolvesToCoons4(t *testing.T) {
	t.Parallel()
	of := newT6Obstacle(t)
	loop, _ := extractObstacle(of)
	fill, rails, sides, ok := coons4Fill(loop)
	if !ok {
		t.Fatal("coons4Fill declined the extracted obstacle loop")
	}
	if !ribbonSeamNonFolding(fill, rails, sides, blendScale()) {
		t.Fatal("extracted obstacle loop folds under coons4")
	}
}

// TestExtractObstacleAdjacentsPositionallyExact hardens obstacleAdjacents beyond
// TestExtractObstacleResolvesToCoons4: the F2 probe only checks the fill's cross-derivative SIGN, so a
// subtly wrong wing-cylinder radius/axis or an offset/rotated wall plane can still pass it while sitting
// in the wrong place in space (nothing else on this path checks POSITION). Proven live: inflating
// wingCylinder's reconstructed radius by 10% left both TestExtractObstacleIsClosedValence4 and
// TestExtractObstacleResolvesToCoons4 green (see task-5-report.md, "Fix: positional surface exactness").
//
// Two independent, model-relative checks per G1 side (wall, wingL, wingR):
//  1. NORMAL AGREEMENT (tangent-plane match): obstaclePatchNeighbours' ribbons (g.wall/g.wingL/g.wingR)
//     are the ALREADY-CERTIFIED G1 neighbour surfaces TestObstacleT6RibbonNonFolding proves fold-free;
//     their VMin-edge S_u×S_v is the true host normal at the rail (obstaclePatchNeighbours' doc: the
//     ribbon is first-order-exact to its neighbour there). obstacleAdjacents' reconstructed analytic
//     surface must report a NormalAt at the same point that is tightly parallel to it. The check is on
//     |dot|, not dot: wingL's and wingR's ribbon-normal sign flips relative to each other (wingDir's
//     per-node sign, see corner_blend_obstacle.go) while Cylinder.NormalAt always reports the fixed
//     "outward radial" convention, so an overall sign mismatch is an artifact of two independently
//     chosen conventions, not a reconstruction defect — only a MISALIGNED tangent plane (wrong
//     radius/axis) is. This alone would NOT catch a pure radius error (Cylinder.NormalAt's angle is
//     radius-independent), which is exactly why check 2 exists.
//  2. ON-SURFACE: the ORIGINAL exact node geometry (of.WallLine's own endpoints, of.WingStart/WingEnd's
//     own arc midpoint — the exact points wingCylinder's doc says the reconstruction is built FROM) must
//     round-trip through the reconstructed surface's ParamAt→PointAt within the model weld. Sampling the
//     exact source geometry (not the RebuildCurve-refit rail, which carries ~1e-4-scale fit noise of its
//     own, unrelated to obstacleAdjacents) isolates a genuine offset/wrong-radius/wrong-axis
//     reconstruction bug: PointAt(ParamAt(q)) only reproduces q when the surface's radius (cylinder) or
//     in-plane basis (plane) truly matches, so a 10% radius inflation opens a ~0.1×Radius gap here.
func TestExtractObstacleAdjacentsPositionallyExact(t *testing.T) {
	t.Parallel()
	of := newT6Obstacle(t)
	wall, wingL, wingR, _, ok := obstacleAdjacents(of)
	if !ok {
		t.Fatal("obstacleAdjacents declined T6")
	}
	assertRibbonNormalsAgree(t, of, wall, wingL, wingR)
	assertExactGeometryOnSurface(t, of, wall, wingL, wingR)
}

// obstacleNormalDotFloor is the unit-dot slack for the ribbon-vs-reconstructed-surface tangent-plane
// check: a dot product of two unit vectors is scale-invariant (not a length), so this is NOT
// model-scaled like a weld tolerance — it is set one order of magnitude above the measured T6
// floating-point floor (|dot| ~ 1-1e-8, from the ribbon rail's RebuildCurve fit not landing its sample
// point EXACTLY on the true cylinder/plane) so it only trips on a genuine tangent-plane misalignment.
const obstacleNormalDotFloor = 1e-6

// assertRibbonNormalsAgree runs check 1 (see TestExtractObstacleAdjacentsPositionallyExact doc) for all
// three G1 sides against the certified obstaclePatchNeighbours ribbons.
func assertRibbonNormalsAgree(t *testing.T, of *ObstacleFeature, wall, wingL, wingR geom.Surface) {
	t.Helper()
	g, ok := obstaclePatchNeighbours(of)
	if !ok {
		t.Fatal("obstaclePatchNeighbours declined T6 (its ribbons are the certified normal reference)")
	}
	assertRibbonNormalAgrees(t, "wall", g.c0, g.wall, wall)
	assertRibbonNormalAgrees(t, "wingL", g.d0, g.wingL, wingL)
	assertRibbonNormalAgrees(t, "wingR", g.d1, g.wingR, wingR)
}

// assertRibbonNormalAgrees samples rail's domain midpoint, reads the certified ribbon's VMin-edge normal
// there, and requires adj's own NormalAt at that same 3D point to be tightly parallel to it.
func assertRibbonNormalAgrees(t *testing.T, label string, rail geom.BSplineCurve, ribbon geom.BSplineSurface, adj geom.Surface) {
	t.Helper()
	lo, hi := rail.Domain()
	pt := rail.PointAt((lo + hi) / 2)
	vMin, _ := ribbon.VDomain()
	hostNormal := ribbon.NormalAt((lo+hi)/2, vMin)
	adjNormal := adj.NormalAt(adj.ParamAt(pt))
	if dot := hostNormal.Dot(adjNormal); stdmath.Abs(dot) < 1-obstacleNormalDotFloor {
		t.Errorf("%s: reconstructed surface normal not parallel to the certified ribbon normal: |dot|=%.9f (host=%v adj=%v)",
			label, stdmath.Abs(dot), hostNormal, adjNormal)
	}
}

// assertExactGeometryOnSurface runs check 2 (see TestExtractObstacleAdjacentsPositionallyExact doc) for
// all three G1 sides against the ORIGINAL exact rail geometry (never the RebuildCurve-refit rail).
func assertExactGeometryOnSurface(t *testing.T, of *ObstacleFeature, wall, wingL, wingR geom.Surface) {
	t.Helper()
	weld := blendScale().Weld()
	lo, hi := of.WallLine.Domain()
	assertPointOnSurface(t, "wall.A", of.WallLine.PointAt(lo), wall, weld)
	assertPointOnSurface(t, "wall.D", of.WallLine.PointAt(hi), wall, weld)
	assertPointOnSurface(t, "wingL.mid", curveMidpoint(of.WingStart), wingL, weld)
	assertPointOnSurface(t, "wingR.mid", curveMidpoint(of.WingEnd), wingR, weld)
}

// curveMidpoint returns c's domain-midpoint position.
func curveMidpoint(c geom.Curve3) math.Point3 {
	lo, hi := c.Domain()
	return c.PointAt((lo + hi) / 2)
}

// assertPointOnSurface requires pt to round-trip through adj's ParamAt→PointAt within tol — the
// on-surface check that catches an offset origin, wrong radius, or misrotated axis (see
// TestExtractObstacleAdjacentsPositionallyExact doc, check 2).
func assertPointOnSurface(t *testing.T, label string, pt math.Point3, adj geom.Surface, tol float64) {
	t.Helper()
	onSurf := adj.PointAt(adj.ParamAt(pt))
	if dist := onSurf.DistanceTo(pt); dist > tol {
		t.Errorf("%s: reconstructed surface does not contain the source point: dist=%.9g > weld %.9g (pt=%v, onSurf=%v)",
			label, dist, tol, pt, onSurf)
	}
}
