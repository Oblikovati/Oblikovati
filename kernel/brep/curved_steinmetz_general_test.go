// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Steinmetz through the GENERAL (u,v)-arrangement pipeline (#1403, approach A). The bicylinder is built by
// splitting the self-intersecting imprint into four open elliptical arcs and trimming each cylinder side —
// the angular-next-edge tracer separating the lobes — instead of the bespoke loop→body assembler. The
// result must match the bespoke topology: a watertight four-face solid of cylinder faces and elliptical-arc
// edges meeting at the two pinch vertices.

// TestSteinmetzIntersectGeneralWatertight pins that the general intersect produces the same watertight
// four-face bicylinder as the bespoke constructor: cylinder faces, every edge used exactly twice.
func TestSteinmetzIntersectGeneralWatertight(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)

	res, ok := SteinmetzIntersectGeneral(cx, cz, nil)
	if !ok {
		t.Fatal("general Steinmetz intersect declined; want a four-face bicylinder")
	}
	if !res.IsSolid() {
		t.Fatalf("general Steinmetz result is not a solid: %+v", res)
	}
	for _, f := range res.Faces() {
		if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
			t.Errorf("face surface %T is not a cylinder (the analytic surface must survive)", f.Geometry())
		}
	}
	for _, e := range res.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold — no free/non-manifold edge)", e.Lineage(), uses)
		}
		if _, isArc := e.Geometry().(geom.EllipticalArc); !isArc {
			t.Errorf("edge %v geometry %T is not an elliptical arc", e.Lineage(), e.Geometry())
		}
	}
	if n := len(res.Faces()); n != 4 {
		t.Errorf("general Steinmetz has %d faces, want 4 (two lobes per cylinder)", n)
	}
	if n := len(res.Edges()); n != 4 {
		t.Errorf("general Steinmetz has %d edges, want 4 (two ellipses, each split at the pinch points)", n)
	}
	if n := len(res.Vertices()); n != 2 {
		t.Errorf("general Steinmetz has %d vertices, want 2 (the pinch points)", n)
	}
}

// TestSteinmetzGeneralEdgesAnchoredToVertices pins the well-formedness invariant a boolean must never
// violate: every stored edge curve starts at its StartVertex and ends at its EndVertex (PointAt(0)≈start,
// PointAt(1)≈end for a forward use). A lobe loop walks its shared elliptical arc in the arc's DECREASING
// parameter direction (t0=1→t1=0); if edgeCurveFor keeps the arc's original forward parameterisation, the
// stored curve's PointAt(0) lands at the FAR pinch — 2R away from StartVertex — so the face's discretised
// boundary jumps across the solid, the (u,v) loop self-intersects, and the tessellator falls back to a
// mis-oriented plane patch (volume 37 vs 144). Re-anchoring the sub-arc to [t0,t1] restores the invariant.
func TestSteinmetzGeneralEdgesAnchoredToVertices(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)

	res, ok := SteinmetzIntersectGeneral(cx, cz, nil)
	if !ok {
		t.Fatal("general Steinmetz intersect declined")
	}
	const tol = 1e-6
	for _, e := range res.Edges() {
		c := e.Geometry()
		if d := float64(c.PointAt(0).DistanceTo(e.StartVertex().Point())); d > tol {
			t.Errorf("edge curve PointAt(0) is %g from StartVertex (arc parameterised opposite to its vertices)", d)
		}
		if d := float64(c.PointAt(1).DistanceTo(e.EndVertex().Point())); d > tol {
			t.Errorf("edge curve PointAt(1) is %g from EndVertex", d)
		}
	}
}

// TestSteinmetzGeneralDeclinesUnequalRadius pins that the general path declines the non-Steinmetz case (so
// kernel/ops keeps the clean crossing-cylinder pipeline for unequal radii).
func TestSteinmetzGeneralDeclinesUnequalRadius(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := SteinmetzIntersectGeneral(cx, cz, nil); ok {
		t.Error("general Steinmetz must decline unequal radii (ok=false)")
	}
}

// nearEqualSteinmetz builds the canonical perpendicular pair with the z-axis cylinder's radius offset by dr
// from the x-axis cylinder's radius 3 — the near-pinch band the snap targets (#1780).
func nearEqualSteinmetz(dr float64) (cx, cz *topo.Body) {
	cx, _ = SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	cz, _ = SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3+dr, 12)
	return cx, cz
}

// TestSteinmetzIntersectGeneralNearEqualSnaps pins the #1780 snap: cylinders whose radii differ by less than
// the model-relative ceiling build the SAME watertight four-face bicylinder as the exactly-equal case, and —
// the load-bearing check — EVERY wall face carries the snapped MEAN radius, not either operand's true radius.
// If the snap routed only into the imprint arcs and left the walls at their true radii, the two z-cylinder
// lobes would report 3+dr and this fails: it is the regression guard for the holistic-routing pitfall.
func TestSteinmetzIntersectGeneralNearEqualSnaps(t *testing.T) {
	cx, cz := nearEqualSteinmetz(0)
	ceil := geom.ResolutionForBox(cx.RangeBox().Union(cz.RangeBox())).Stitch()
	dr := 0.5 * ceil // strictly within the snap ceiling
	cx, cz = nearEqualSteinmetz(dr)

	body, ok := SteinmetzIntersectGeneral(cx, cz, nil)
	if !ok {
		t.Fatalf("near-equal (|Δr|=%.3g ≤ ceiling %.3g) must snap and build the bicylinder", dr, ceil)
	}
	if !body.IsSolid() || len(body.Faces()) != 4 || len(body.Edges()) != 4 || len(body.Vertices()) != 2 {
		t.Fatalf("snapped topology F/E/V = %d/%d/%d, want 4/4/2 (identical to the equal case)",
			len(body.Faces()), len(body.Edges()), len(body.Vertices()))
	}
	wantR := 3 + dr/2 // the mean; both walls AND arcs must be rebuilt here
	for _, f := range body.Faces() {
		cyl, isCyl := f.Geometry().(geom.Cylinder)
		if !isCyl {
			t.Errorf("face %T is not a cylinder", f.Geometry())
			continue
		}
		if d := stdmath.Abs(cyl.Radius - wantR); d > 1e-12 {
			t.Errorf("wall radius %.12f, want the mean %.12f — a wall left at a true radius means only the arcs snapped", cyl.Radius, wantR)
		}
	}
}

// TestSteinmetzGeneralDeclinesResidualBand pins that a radius gap ABOVE the snap ceiling (but still inside the
// near-pinch band, |Δr| < 2.5e-4·r) is NOT snapped: the neck √(2R·Δr) is resolvable two-loop geometry, so the
// general Steinmetz path declines and the boolean keeps the deterministic faceted route (#1780 Direction 2).
func TestSteinmetzGeneralDeclinesResidualBand(t *testing.T) {
	cx, cz := nearEqualSteinmetz(0)
	ceil := geom.ResolutionForBox(cx.RangeBox().Union(cz.RangeBox())).Stitch()
	dr := 4 * ceil // above the ceiling, still ≪ 2.5e-4·3
	cx, cz = nearEqualSteinmetz(dr)
	if _, ok := SteinmetzIntersectGeneral(cx, cz, nil); ok {
		t.Errorf("residual band (|Δr|=%.3g > ceiling %.3g) must decline the snap (ok=false)", dr, ceil)
	}
}

// TestSteinmetzGeneralDeclinesNonCrossing pins that equal-radius cylinders whose axes do NOT intersect
// (offset/skew) are not the Steinmetz case, so the general path declines (steinmetzFrame fails).
func TestSteinmetzGeneralDeclinesNonCrossing(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 5), math.V3(1, 0, 0), 3, 12) // offset in z, axes do not meet
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := SteinmetzIntersectGeneral(cx, cz, nil); ok {
		t.Error("non-crossing equal-radius cylinders should decline (ok=false)")
	}
}

// steinmetzCylinders returns the canonical radius-3, length-12 perpendicular test pair (axes x and z).
func steinmetzCylinders() (cx, cz *topo.Body) {
	cx, _ = SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	cz, _ = SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	return cx, cz
}

// assertWatertightAnalytic fails unless res is a solid whose every edge is used exactly twice and whose faces
// are exactly wantCyl cylinders + wantPlane planes — the shared shape of the Steinmetz cut/join checks.
func assertWatertightAnalytic(t *testing.T, res *topo.Body, wantCyl, wantPlane int) {
	t.Helper()
	if !res.IsSolid() {
		t.Fatalf("result is not a solid: %+v", res)
	}
	for _, e := range res.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
		c := e.Geometry()
		if d := float64(c.PointAt(0).DistanceTo(e.StartVertex().Point())); d > 1e-6 {
			t.Errorf("edge curve PointAt(0) is %g from StartVertex (arc parameterised opposite to its vertices)", d)
		}
	}
	cyls, planes := 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			cyls++
		case geom.Plane:
			planes++
		default:
			t.Errorf("face surface %T is not analytic", f.Geometry())
		}
	}
	if cyls != wantCyl || planes != wantPlane {
		t.Errorf("got %d cylinder + %d planar faces, want %d + %d", cyls, planes, wantCyl, wantPlane)
	}
}

// TestSteinmetzCutGeneralWatertight pins that the general cut (target − tool) produces the same watertight
// six-face bitten solid as the bespoke constructor: two target bands (kept OUTSIDE the tool) + two reversed
// tool lobes (the saddle bite) + two target caps. The two outside bands come from the wrapping-band emission,
// which for the pinched Steinmetz saddle must recognise each band wraps the full azimuth (op-aware wrapsAllU).
func TestSteinmetzCutGeneralWatertight(t *testing.T) {
	cx, cz := steinmetzCylinders()
	res, ok := SteinmetzCutGeneral(cx, cz, nil)
	if !ok {
		t.Fatal("general Steinmetz cut declined; want a six-face bitten solid")
	}
	assertWatertightAnalytic(t, res, 4, 2) // two A bands + two B lobes; two caps
}

// TestSteinmetzJoinGeneralWatertight pins that the general join (a ∪ b) produces the same watertight
// eight-face union solid as the bespoke constructor: two outside bands + two caps per cylinder, the two
// cylinders meeting along the shared intersection ellipses (no lobes, no reversal).
func TestSteinmetzJoinGeneralWatertight(t *testing.T) {
	cx, cz := steinmetzCylinders()
	res, ok := SteinmetzJoinGeneral(cx, cz, nil)
	if !ok {
		t.Fatal("general Steinmetz join declined; want an eight-face union solid")
	}
	assertWatertightAnalytic(t, res, 4, 4) // two bands per cylinder; two caps per cylinder
}

// TestSteinmetzCutJoinGeneralDeclineUnequalRadius pins that both general cut and join decline the
// non-Steinmetz case, so kernel/ops keeps the clean crossing-cylinder pipeline for unequal radii.
func TestSteinmetzCutJoinGeneralDeclineUnequalRadius(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := SteinmetzCutGeneral(cx, cz, nil); ok {
		t.Error("general Steinmetz cut must decline unequal radii (ok=false)")
	}
	if _, ok := SteinmetzJoinGeneral(cx, cz, nil); ok {
		t.Error("general Steinmetz join must decline unequal radii (ok=false)")
	}
}
