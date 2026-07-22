// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The double-bitten shared-face cap-bridge (miter piece B). On P5 the CYLINDER arm's far shared-contact
// foot (48.148,0.034,60) is an INTERIOR fresh cut the fillet makes mid-wall on the shared cylinder — NOT
// on any original loop edge — so farPathSegs cannot anchor there. sharedRailAnchor detects the off-loop
// foot (pointOnLoop=false) and bridges it to the arm's FAR VERTEX (65,2.303,60), an on-loop point, with
// ONE exact Arc3d on the shared cylinder (capBridgeArc). The TORUS arm's far foot (65,97.697,145) is
// interior to an original edge (e2), so pointOnLoop=true and NO bridge is spliced — byte-identical.
// These fixtures pin both branches against the DRAWEXE-verified P5 geometry (curved-miter-closure §3).

// p5SharedCylinder is P5's shared bore: geom.Cylinder R=50, axis ẑ through (50,50,0) — the face both
// picked edges belong to and the surface the cap-bridge arc rides.
func p5SharedCylinder(t *testing.T) geom.Cylinder {
	t.Helper()
	cyl, err := geom.NewCylinderWithRef(math.P3(50, 50, 0), math.V3(0, 0, 1), math.V3(0, 1, 0), 50)
	if err != nil {
		t.Fatalf("P5 shared cylinder: %v", err)
	}
	return cyl
}

// p5SharedLoopSegs is a faithful slice of P5's shared-cylinder ORIGINAL loop (curved-miter-recon §1a):
// the back boss-line e2 (vertical, z 150→60), the front boss-line e8 = the picked cyl edge (vertical,
// z 60→150), and the exposed top-rim arc e1 (the MAJOR R=50 arc through (0,50,150)). It carries the two
// points the anchor test probes: the torus far foot lies INTERIOR to e2; the cyl far foot lies on NONE.
func p5SharedLoopSegs(t *testing.T) []endSeg {
	t.Helper()
	e1arc, err := geom.Arc3dByThreePoints(math.P3(65, 2.303039929, 150), math.P3(0, 50, 150), math.P3(65, 97.696960071, 150))
	if err != nil {
		t.Fatalf("P5 top-rim arc e1: %v", err)
	}
	return []endSeg{
		{from: math.P3(65, 97.696960071, 150), to: math.P3(65, 97.696960071, 60)}, // e2 back boss-line
		{from: math.P3(65, 2.303039929, 60), to: math.P3(65, 2.303039929, 150)},   // e8 front boss-line (picked)
		{from: math.P3(65, 2.303039929, 150), to: math.P3(65, 97.696960071, 150), curve: e1arc, mid: e1arc.PointAt(0.5), arc: true},
	}
}

// TestPointOnLoopSelectsCapBridgeBranch pins the anchor-branch discriminator: the torus arm's far foot
// is interior to an original edge (on-loop → no bridge, byte-identical), while the cyl arm's far foot is
// a mid-wall fresh cut (off-loop → cap-bridge). pointOnLoop must agree with farPathSegs's own
// insertSplits/indexOfSegFrom mechanism so an on-loop foot is never mis-routed through a spurious bridge.
func TestPointOnLoopSelectsCapBridgeBranch(t *testing.T) {
	segs := p5SharedLoopSegs(t)
	tol := 1e-6 * 50
	torusFoot := math.P3(65, 97.696960071, 145) // interior to e2 (z 150→60)
	if !pointOnLoop(segs, torusFoot, tol) {
		t.Fatalf("torus far foot %v must be ON the original loop (interior to e2) → no cap-bridge", torusFoot)
	}
	cylFoot := math.P3(48.148148148, 0.034305321, 60) // mid-wall interior fresh cut — on NO edge
	if pointOnLoop(segs, cylFoot, tol) {
		t.Fatalf("cyl far foot %v must be OFF the original loop (mid-wall fresh cut) → needs a cap-bridge", cylFoot)
	}
}

// TestCapBridgeArcOnSharedCylinder pins the cap-bridge geometry against DRAWEXE (curved-miter-recon §3):
// the arc rides the shared cylinder's z=60 latitude circle (centre (50,50,60), R=50), runs foot→farVertex
// exactly, and is a genuine MINOR Arc3d (≈19.6°) — never a chord — so the shared-cyl↔z=60-cap weld stays
// watertight.
func TestCapBridgeArcOnSharedCylinder(t *testing.T) {
	cyl := p5SharedCylinder(t)
	foot := math.P3(48.148148148, 0.034305321, 60)
	farVtx := math.P3(65, 2.303039929, 60)
	tol := 1e-6 * 50
	seg, ok := capBridgeArc(cyl, foot, farVtx, tol)
	if !ok {
		t.Fatal("cap-bridge on the shared cylinder declined for two co-latitude z=60 points")
	}
	if !seg.arc || float64(seg.from.DistanceTo(foot)) > tol || float64(seg.to.DistanceTo(farVtx)) > tol {
		t.Fatalf("cap-bridge must be an arc foot→farVtx, got arc=%v %v->%v", seg.arc, seg.from, seg.to)
	}
	arc, isArc := seg.curve.(geom.Arc3d)
	if !isArc {
		t.Fatalf("cap-bridge curve must be a geom.Arc3d, got %T", seg.curve)
	}
	if float64(arc.Center.DistanceTo(math.P3(50, 50, 60))) > tol || stdmath.Abs(arc.Radius-50) > tol {
		t.Fatalf("cap-bridge arc must ride the z=60 latitude circle (50,50,60)/R50, got %v/R%.4f", arc.Center, arc.Radius)
	}
	if stdmath.Abs(arc.SweepAngle) >= stdmath.Pi {
		t.Fatalf("cap-bridge must be the MINOR arc (|sweep|<π), got %.4f rad", arc.SweepAngle)
	}
}

// TestCapBridgeArcDeclines pins the do-no-harm guards: a non-cylinder shared face, a pair NOT co-latitude
// on the axis (an oblique/non-planar far cap), and a point off the cylinder radius each floor honestly —
// so the bridge is spliced ONLY when it exactly closes the shared-cyl↔cap seam.
func TestCapBridgeArcDeclines(t *testing.T) {
	cyl := p5SharedCylinder(t)
	tol := 1e-6 * 50
	foot := math.P3(48.148148148, 0.034305321, 60)
	farVtx := math.P3(65, 2.303039929, 60)
	plane, err := geom.NewPlane(math.P3(0, 0, 60), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("z=60 plane: %v", err)
	}
	if _, ok := capBridgeArc(plane, foot, farVtx, tol); ok {
		t.Fatal("cap-bridge must decline on a non-cylinder shared face")
	}
	if _, ok := capBridgeArc(cyl, foot, math.P3(65, 2.303039929, 61), tol); ok {
		t.Fatal("cap-bridge must decline when foot and far vertex are not co-latitude (different axial height)")
	}
	if _, ok := capBridgeArc(cyl, math.P3(45, 5, 60), farVtx, tol); ok {
		t.Fatal("cap-bridge must decline when a point is not at the cylinder radius")
	}
}

// TestCoLatitudeOnCyl pins the latitude guard directly: two points at the same axial height and radius are
// co-latitude; the same points at different heights are not.
func TestCoLatitudeOnCyl(t *testing.T) {
	cyl := p5SharedCylinder(t)
	tol := 1e-6 * 50
	foot := math.P3(48.148148148, 0.034305321, 60)
	farVtx := math.P3(65, 2.303039929, 60)
	center := projectOntoAxis(foot, cyl.Origin, cyl.AxisDir)
	if !coLatitudeOnCyl(cyl, center, foot, farVtx, tol) {
		t.Fatal("two z=60 points at R=50 must be co-latitude on the shared cylinder")
	}
	if coLatitudeOnCyl(cyl, center, foot, math.P3(65, 2.303039929, 90), tol) {
		t.Fatal("points at different axial heights must NOT be co-latitude")
	}
}
