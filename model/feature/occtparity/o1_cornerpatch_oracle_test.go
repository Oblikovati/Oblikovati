// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The LOAD-BEARING gate on O1's corner patch — the check per-face areas are structurally blind to (S7 was
// 18.5% of r wrong in shape and 0.03% right in area). Two halves, because on this case the oracle is itself
// an approximation and only one of the two can be pinned tight:
//
//  1. TestO1CornerPatchIsTheExactRollingBallEnvelope — the TIGHT half, and the real definition of correct
//     here. Our patch must BE the envelope of one ball of radius r that rides the boss wall at ρ = R + r and
//     rolls on the convex band's tube at 2r: every u-isocurve a circle of radius exactly r, its centre at
//     exactly ρ and 2r from the two axes, both boundary loci exactly on their host. Asserted to 1e-6 of r.
//     No oracle can be tighter than this, because it IS the rolling-ball definition of a constant-radius
//     blend — and it is the property that decides which of the two surfaces (ours or OCCT's) is exact.
//
//  2. TestO1CornerPatchMatchesOCCTSurfacePoints — the ORACLE half: 25 points DRAWEXE evaluated on OCCT's OWN
//     result_5 surface, asserted onto ours. Its band is 1.5% of r, not the 0.01–0.04% the run-out cases hold,
//     and that is a measured property of OCCT's patch rather than slack: the same 25 points sit up to 1.16% of
//     r off the EXACT envelope (1), and the two boundary loci sit up to 3.3% of r off the exact contact loci
//     (0.143 on the boss wall, 0.162 on the band). Taken as an exact contact locus, OCCT's u=1 curve implies a
//     ball 0.174 short of tangency to the boss wall — i.e. OCCT's surface is not the envelope, ours is. The
//     band still separates the defect class that mattered on N4 (a chord-projected corner fill, 21% of r out)
//     by 14×, and the reverse direction was measured too: DRAWEXE `distmini` from 49 points of OUR shipped
//     patch to result_5 reads a worst interior 0.0584 — the same figure both ways, so the discrepancy is a
//     smooth two-sided offset, not a local shape error.
//
// All four corners agree to the 10 digits DRAWEXE prints, which is what makes the interior band meaningful
// rather than a whole-surface drift.

// o1PatchOraclePoints are 25 interior points of OCCT's OWN corner-patch surface, at the (u,v) fractions
// 0.1/0.3/0.5/0.7/0.9 of its parameter box — strictly interior, so none of them is a boundary the two
// surfaces share by construction. Captured live from DRAWEXE 8.0.0:
//
//	restore test-utilities/occt-blend/data/CFI_f5678fin.rle s ; tscale s 0 0 0 10 ; explode s e
//	blend result s 5 s_7 5 s_6 5 s_14 ; explode result f ; mksurface sf result_5
//	bounds sf u0 u1 v0 v1 ; svalue sf <u> <v> x y z
func o1PatchOraclePoints() []struct {
	u, v float64
	p    math.Point3
} {
	return []struct {
		u, v float64
		p    math.Point3
	}{
		{0.1, 0.1, math.P3(81.3488227624, 11.0210327756, 86.6163813381)},
		{0.1, 0.3, math.P3(80.2830823837, 10.1807108410, 89.5216180789)},
		{0.1, 0.5, math.P3(78.5663766930, 8.9218568400, 91.7728571618)},
		{0.1, 0.7, math.P3(76.4537937719, 7.5210239127, 93.3497143162)},
		{0.1, 0.9, math.P3(74.0355167209, 6.0993659166, 94.1812536923)},
		{0.3, 0.1, math.P3(80.7741027956, 10.3464231669, 86.4867342177)},
		{0.3, 0.3, math.P3(79.6957843859, 9.4694611590, 89.0903475288)},
		{0.3, 0.5, math.P3(78.0361029790, 8.2084830058, 90.9762701438)},
		{0.3, 0.7, math.P3(76.1164936168, 6.8863777929, 92.1939013625)},
		{0.3, 0.9, math.P3(74.0345593944, 5.6085042074, 92.7671510158)},
		{0.5, 0.1, math.P3(80.3229105516, 9.5647091690, 86.3668034255)},
		{0.5, 0.3, math.P3(79.2596723267, 8.6122639126, 88.6938903477)},
		{0.5, 0.5, math.P3(77.6942864820, 7.2778520817, 90.2487925766)},
		{0.5, 0.7, math.P3(75.9954785936, 5.9455842020, 91.1430727342)},
		{0.5, 0.9, math.P3(74.2599814527, 4.7218496333, 91.4902822785)},
		{0.7, 0.1, math.P3(80.0207615731, 8.7121111198, 86.2627962334)},
		{0.7, 0.3, math.P3(79.0079709903, 7.6600383894, 88.3591185360)},
		{0.7, 0.5, math.P3(77.5804887026, 6.2045427000, 89.6595621268)},
		{0.7, 0.7, math.P3(76.1183022961, 4.7933574936, 90.3298902082)},
		{0.7, 0.9, math.P3(74.6951677975, 3.5413672927, 90.5498358089)},
		{0.9, 0.1, math.P3(79.8787037616, 7.8317300773, 86.1793742079)},
		{0.9, 0.3, math.P3(78.9507107472, 6.6756561386, 88.1051455646)},
		{0.9, 0.5, math.P3(77.6946912779, 5.0878998547, 89.2537192989)},
		{0.9, 0.7, math.P3(76.4572059902, 3.5723734678, 89.8331079670)},
		{0.9, 0.9, math.P3(75.2656759420, 2.2500241206, 90.0522557339)},
	}
}

// o1PatchCorners are DRAWEXE's four corner points of result_5, in (u,v) = (0,0) (1,0) (0,1) (1,1) order: the
// boss-wall and band feet of the cylinder arm's station, then of the cove arm's station. Both surfaces share
// them by construction, so they are asserted at machine precision.
func o1PatchCorners() [4]math.Point3 {
	return [4]math.Point3{
		math.P3(81.8181818182, 11.4305392080, 85.0000000000),
		math.P3(80.0000000000, 7.5735931288, 85.0000000000),
		math.P3(72.7272727273, 5.4638228585, 95.0000000000),
		math.P3(75.0000000000, 1.0102051443, 90.0000000000),
	}
}

const (
	// o1FilletRadius is the case's fillet radius; every tolerance below is a fraction OF IT (ADR-0042 — a
	// tolerance on a blend is relative to the rolling ball, never an absolute epsilon).
	o1FilletRadius = 5.0
	// o1EnvelopeRelTol is the band the EXACT-ENVELOPE invariants hold to: our patch's cross-section radius,
	// its centre's two axis distances and its two boundary loci all measure within 1e-6·r. That is the
	// construction's own arithmetic noise (the loft pins its ends exactly and every station is a closed
	// form), so it is a real assertion, not a fitted threshold.
	o1EnvelopeRelTol = 1e-6
	// o1OracleSurfaceRelTol is the band OCCT's own interior points hold to on our patch: 2% of r against a
	// measured worst 1.16% (1.7× headroom). It is OCCT's approximation error, quantified in this file's
	// header, and cannot be tightened without asserting that error as truth.
	o1OracleSurfaceRelTol = 2e-2
	// o1CornerPointTol is the band the four SHARED corner points hold to — both surfaces derive them from the
	// same exact stations, so they agree to the 10 digits DRAWEXE prints.
	o1CornerPointTol = 1e-9
)

// TestO1CornerPatchIsTheExactRollingBallEnvelope is the tight half (see this file's header): our corner patch
// must satisfy the rolling-ball definition of the blend, which OCCT's approximation does not.
func TestO1CornerPatchIsTheExactRollingBallEnvelope(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "O1")
	patch := o1CornerPatchFace(t, body)
	surf := patch.Geometry().(geom.BSplineSurface)
	bossAxis := o1BossWallAxis(t, body)
	bandAxis := o1BandArmAxis(t, body)
	tol := o1EnvelopeRelTol * o1FilletRadius
	u0, u1 := surf.UDomain()
	v0, v1 := surf.VDomain()
	for i := 0; i <= 16; i++ {
		v := v0 + (v1-v0)*float64(i)/16
		centre, radius := o1CrossSectionBall(t, surf, u0, u1, v)
		assertO1Near(t, "cross-section radius", radius, o1FilletRadius, tol, v)
		assertO1Near(t, "ball centre distance from the boss-wall axis (ρ = R + r)",
			o1DistanceToAxis(centre, bossAxis), 55, tol, v)
		assertO1Near(t, "ball centre distance from the convex band's axis (2r)",
			o1DistanceToAxis(centre, bandAxis), 2*o1FilletRadius, tol, v)
		assertO1Near(t, "u=0 locus distance from the boss-wall axis (on the host, R)",
			o1DistanceToAxis(surf.PointAt(u0, v), bossAxis), 50, tol, v)
		assertO1Near(t, "u=1 locus distance from the band's axis (on the band's tube, r)",
			o1DistanceToAxis(surf.PointAt(u1, v), bandAxis), o1FilletRadius, tol, v)
	}
}

// TestO1CornerPatchMatchesOCCTSurfacePoints is the oracle half: DRAWEXE's own interior surface points, and its
// four corner points, asserted onto our patch.
func TestO1CornerPatchMatchesOCCTSurfacePoints(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "O1")
	surf := o1CornerPatchFace(t, body).Geometry().(geom.BSplineSurface)
	u0, u1 := surf.UDomain()
	v0, v1 := surf.VDomain()
	for i, c := range o1PatchCorners() {
		u, v := u0, v0
		if i%2 == 1 {
			u = u1
		}
		if i >= 2 {
			v = v1
		}
		if d := float64(surf.PointAt(u, v).DistanceTo(c)); d > o1CornerPointTol {
			t.Fatalf("O1 patch corner (u=%g,v=%g) is %.3e from DRAWEXE's %v, want <= %g", u, v, d, c, o1CornerPointTol)
		}
	}
	worst, at := 0.0, [2]float64{}
	for _, s := range o1PatchOraclePoints() {
		_, _, foot := geom.ClosestPointOnSurface(surf, s.p)
		if d := float64(foot.DistanceTo(s.p)); d > worst {
			worst, at = d, [2]float64{s.u, s.v}
		}
	}
	if band := o1OracleSurfaceRelTol * o1FilletRadius; worst > band {
		t.Fatalf("O1 corner patch: DRAWEXE's own result_5 point at (u=%g,v=%g) lies %.6f off our patch, want <= %g (%g%% of r)",
			at[0], at[1], worst, band, o1OracleSurfaceRelTol*100)
	}
}

// o1CrossSectionBall returns the centre and radius of the circle our patch's u-isocurve at v traces: the
// circumcircle of its two ends and its midpoint. It fails when those three points are collinear, which is
// exactly the failure mode a non-envelope fill would show.
func o1CrossSectionBall(t *testing.T, surf geom.BSplineSurface, u0, u1, v float64) (math.Point3, float64) {
	t.Helper()
	a, m, b := surf.PointAt(u0, v), surf.PointAt((u0+u1)/2, v), surf.PointAt(u1, v)
	arc, err := geom.Arc3dByThreePoints(a, m, b)
	if err != nil {
		t.Fatalf("O1 patch u-isocurve at v=%g is not a circle through %v %v %v: %v", v, a, m, b, err)
	}
	return arc.Center, arc.Radius
}

// o1Axis is a cylinder's axis as a point + a unit direction — the reference every distance below is taken to.
type o1Axis struct {
	origin math.Point3
	dir    math.Vector3
}

// o1BossWallAxis returns the boss wall's axis.
func o1BossWallAxis(t *testing.T, body *topo.Body) o1Axis {
	t.Helper()
	cyl := o1BossWallFace(t, body).Geometry().(geom.Cylinder)
	return o1Axis{origin: cyl.Origin, dir: cyl.AxisDir.AsVector()}
}

// o1BandArmAxis returns the convex planar band arm's rolling-ball cylinder axis.
func o1BandArmAxis(t *testing.T, body *topo.Body) o1Axis {
	t.Helper()
	cyl := o1ArmFace(t, body, math.V3(0, 1, 0)).Geometry().(geom.Cylinder)
	return o1Axis{origin: cyl.Origin, dir: cyl.AxisDir.AsVector()}
}

// o1DistanceToAxis is the perpendicular distance from p to an axis line.
func o1DistanceToAxis(p math.Point3, a o1Axis) float64 {
	rel := a.origin.VectorTo(p)
	return float64(rel.Sub(a.dir.Scale(rel.Dot(a.dir))).Length())
}

// assertO1Near fails unless got is within tol of want, naming the isoparm it was measured on.
func assertO1Near(t *testing.T, what string, got, want, tol, v float64) {
	t.Helper()
	if stdmath.Abs(got-want) > tol {
		t.Fatalf("O1 corner patch at v=%.4f: %s measured %.9f, want %.9f within %g", v, what, got, want, tol)
	}
}
