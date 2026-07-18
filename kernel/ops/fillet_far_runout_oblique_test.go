// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// FR3 — the oblique far termination wired live into the arm-weld (far-runout-engine-architecture.md
// ADR-4). These tests pin: (1) the engine's authoritative feet == the FR2 closed-form D5 feet, ordered to
// the arm's hosts; (2) obliqueRunout builds the analytic spiric trim AND re-terminates both host rails on
// the feet, so trim.endpoints == feet == rail-outer-ends (the shared-edge identity); (3) the bite sampler
// develops an oblique analytic trim by its true curvature (not a chord) while the Arc3d path is verbatim;
// (4) the far-bite host router keys an oblique runout by capping identity, a perpendicular one by surface
// membership. The D5 geometry is the FR2 fixture (d5MeridianArm) — the DRAWEXE-pinned meridian torus arm.

// d5EdgeFillet wraps D5's meridian arm as an edgeFillet with real topo host faces (sphere = ef.a, longitude
// plane = ef.b) — the shape obliqueRunout/armRunoutFeet consume. Only the faces' Geometry() is read.
func d5EdgeFillet(t *testing.T, tor geom.Torus, sphere geom.Sphere, lonPlane geom.Plane) edgeFillet {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "d5-oblique", 0))
	bld := topo.NewBuilder(true, lin)
	return edgeFillet{a: stubFace(bld, lin, sphere), b: stubFace(bld, lin, lonPlane), armSurface: tor}
}

// stubFace builds a minimal face carrying surf (a throwaway two-edge loop; only Geometry() is used here).
func stubFace(bld *topo.Builder, lin topo.Lineage, surf geom.Surface) *topo.Face {
	v0 := bld.AddVertex(math.P3(0, 0, 0), lin)
	v1 := bld.AddVertex(math.P3(1, 0, 0), lin)
	e0 := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), v0, v1, lin)
	e1 := bld.AddEdge(geom.NewLineSegment(math.P3(1, 0, 0), math.P3(0, 0, 0)), v1, v0, lin)
	return bld.AddFace(surf, lin, topo.OuterLoop(topo.Fwd(e0), topo.Fwd(e1)))
}

// TestArmRunoutFeet_D5MatchesOracle: the engine's feet (armSprings ∩ cap, ordered to ef.a/ef.b) equal the
// DRAWEXE-pinned D5 feet — sphere foot on ef.a, plane foot on ef.b.
func TestArmRunoutFeet_D5MatchesOracle(t *testing.T) {
	tor, sphere, lonPlane, cap := d5MeridianArm(t)
	ef := d5EdgeFillet(t, tor, sphere, lonPlane)
	near := math.P3(-(tor.MajorRadius + tor.MinorRadius), 10, d5CapZ)
	feet, ok, reason := armRunoutFeet(ef, cap, near, near, 10, ResolutionForSize(300))
	if !ok {
		t.Fatalf("armRunoutFeet declined D5's oblique arm: %s", reason)
	}
	want := d5Feet(t, tor, sphere, lonPlane, cap) // [sphere, plane]
	if d := float64(feet[0].DistanceTo(want[0])); d > 1e-9 {
		t.Fatalf("foot[0] (ef.a=sphere) %v off the sphere foot %v by %.3e", feet[0], want[0], d)
	}
	if d := float64(feet[1].DistanceTo(want[1])); d > 1e-9 {
		t.Fatalf("foot[1] (ef.b=plane) %v off the plane foot %v by %.3e", feet[1], want[1], d)
	}
}

// TestObliqueRunout_D5SpiricAndReterminatedRails is the ADR-4 gate: obliqueRunout builds the spiric trim
// through the feet and re-terminates BOTH host rails on those same feet — trim.endpoints == feet == rail
// outer ends. The host rails are the true contact-circle arcs (sphere / plane), so the re-termination lands
// each rail's outer end on its foot.
func TestObliqueRunout_D5SpiricAndReterminatedRails(t *testing.T) {
	tor, sphere, lonPlane, cap := d5MeridianArm(t)
	ef := d5EdgeFillet(t, tor, sphere, lonPlane)
	capFace := stubCapFace(t, cap)
	res := ResolutionForSize(300)
	h0 := contactArcRail(t, sphereContactCircleOf(t, sphere, tor, res), tor)
	h1 := contactArcRail(t, capContactCircleOf(t, lonPlane, tor, res), tor)

	h0p, h1p, run, ok, reason := obliqueRunout(ef, capFace, h0, h1, 10, res)
	if !ok {
		t.Fatalf("obliqueRunout declined D5's oblique arm: %s", reason)
	}
	if _, isSpiric := run.trim.curve.(geom.SpiricArc); !isSpiric {
		t.Fatalf("run.trim.curve is %T, want geom.SpiricArc (the analytic section, not a chord)", run.trim.curve)
	}
	// ADR-4: the three coincident identities.
	assertCoincident(t, "trim.from == foot[0] == h0'.outer", run.trim.from, run.feet[0], h0p.from)
	assertCoincident(t, "trim.to   == foot[1] == h1'.outer", run.trim.to, run.feet[1], h1p.from)
}

// assertCoincident fails unless all three points coincide to machine eps (the shared-edge identity).
func assertCoincident(t *testing.T, what string, a, b, c math.Point3) {
	t.Helper()
	if float64(a.DistanceTo(b)) > 1e-9 || float64(a.DistanceTo(c)) > 1e-9 {
		t.Fatalf("%s: %v / %v / %v not coincident", what, a, b, c)
	}
}

// stubCapFace wraps the cap plane in a minimal topo face (obliqueRunout reads only capping.Geometry()).
func stubCapFace(t *testing.T, cap geom.Plane) *topo.Face {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "d5-cap", 0))
	return stubFace(topo.NewBuilder(true, lin), lin, cap)
}

// sphereContactCircleOf / capContactCircleOf expose the production contact circles for the rail fixtures.
func sphereContactCircleOf(t *testing.T, sphere geom.Sphere, tor geom.Torus, res Resolution) geom.Circle {
	t.Helper()
	c, r, ok := sphereContactCircle(sphere, tor, res)
	if !ok {
		t.Fatal("sphereContactCircle declined the D5 sphere host")
	}
	return geom.Circle{Center: c, Normal: tor.AxisDir, RefDir: tor.Ref, Radius: r}
}

func capContactCircleOf(t *testing.T, pl geom.Plane, tor geom.Torus, res Resolution) geom.Circle {
	t.Helper()
	c, r, ok := capContactCircle(pl, tor, res)
	if !ok {
		t.Fatal("capContactCircle declined the D5 longitude-plane host")
	}
	return geom.Circle{Center: c, Normal: tor.AxisDir, RefDir: tor.Ref, Radius: r}
}

// contactArcRail builds a host contact-arc rail (a small sub-arc of the contact circle); reterminateRail
// re-sweeps it foot→tHost, so only the underlying circle and a tHost on it matter here.
func contactArcRail(t *testing.T, circle geom.Circle, tor geom.Torus) endSeg {
	t.Helper()
	arc, err := geom.NewArc3d(circle.Center, circle.Normal.AsVector(), circle.RefDir.AsVector(), circle.Radius, 2.5, 0.4)
	if err != nil {
		t.Fatalf("contact arc: %v", err)
	}
	return endSeg{from: arc.PointAt(0), to: arc.PointAt(1), curve: arc, mid: arc.PointAt(0.5), arc: true}
}

// TestBiteArcBulge_SamplesObliqueTrim: the bite sampler develops an analytic section trim (a spiric) by its
// true curvature (interior points ON the curve), keeps the Arc3d path verbatim (arcInteriorPoints), and a
// straight bite contributes no bulge — the generalization that lets an oblique bite share the arm's far edge.
func TestBiteArcBulge_SamplesObliqueTrim(t *testing.T) {
	tor, sphere, lonPlane, cap := d5MeridianArm(t)
	feet := d5Feet(t, tor, sphere, lonPlane, cap)
	trim, ok := intersectArmCapping(tor, cap, feet, 10, ResolutionForSize(300))
	if !ok {
		t.Fatal("precondition: the D5 spiric trim must build")
	}
	bite := endSeg{from: feet[0], to: feet[1], curve: trim, mid: trim.PointAt(0.5)} // arc==false: analytic section
	pts := biteArcBulge(bite, feet[0])
	if len(pts) != biteArcSamples-1 {
		t.Fatalf("oblique bite bulge sampled %d points, want %d interior samples", len(pts), biteArcSamples-1)
	}
	for i, p := range pts {
		if signedDistTorus(tor, p) > 1e-9 || stdmath.Abs(float64(p.Z)-d5CapZ) > 1e-9 {
			t.Fatalf("bulge point %d %v is not ON the spiric trim (off-torus/off-cap)", i, p)
		}
	}
	// Arc3d path verbatim, straight path empty.
	arc, _ := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 10, 0, stdmath.Pi/2)
	if got := biteArcBulge(endSeg{from: arc.PointAt(0), to: arc.PointAt(1), curve: arc, arc: true}, arc.PointAt(0)); len(got) != biteArcSamples-1 {
		t.Fatalf("arc bite bulge sampled %d, want %d (arcInteriorPoints verbatim)", len(got), biteArcSamples-1)
	}
	if got := biteArcBulge(endSeg{from: math.P3(0, 0, 0), to: math.P3(1, 0, 0)}, math.P3(0, 0, 0)); got != nil {
		t.Fatalf("straight bite must contribute no bulge, got %v", got)
	}
}

// TestReterminateRail_StraightRuling covers the cylinder-arm branch (no corpus case reaches it yet, since
// D9's cylinder arm floors earlier at its host contact rail): a straight ruling re-clips to foot→tHost when
// the foot lies on the ruling line, and declines when it does not (never snapping an off-line foot).
func TestReterminateRail_StraightRuling(t *testing.T) {
	rail := endSeg{from: math.P3(0, 0, 0), to: math.P3(0, 0, 10)} // ruling along ẑ
	onLine := math.P3(0, 0, -3)                                   // beyond the segment but ON the line
	got, ok := reterminateRail(rail, onLine, 1e-6)
	if !ok || got.from != onLine || got.to != rail.to {
		t.Fatalf("straight re-termination = (%v,%v), want from=%v to=%v", got, ok, onLine, rail.to)
	}
	if _, ok := reterminateRail(rail, math.P3(2, 0, -3), 1e-6); ok {
		t.Fatal("a foot off the ruling line must decline (do-no-harm), not snap")
	}
}

// TestFarBiteOnHost_RoutesObliqueByCapping: an oblique runout bites the face that IS its capping (identity,
// tolerance-free), never a face its far endpoints merely lie on; a perpendicular/unclassified runout still
// routes by surface membership (verbatim).
func TestFarBiteOnHost_RoutesObliqueByCapping(t *testing.T) {
	lin := topo.NewLineage(topo.Tok("test", "farbite", 0))
	bld := topo.NewBuilder(true, lin)
	capFace := stubFace(bld, lin, planeOn(t, math.P3(0, 0, 100), math.V3(0, 0, 1)))
	other := stubFace(bld, lin, planeOn(t, math.P3(0, 0, 100), math.V3(0, 0, 1)))
	far := endSeg{from: math.P3(1, 0, 100), to: math.P3(0, 1, 100)}
	oblique := armRails{far: far, runout: armRunout{regime: runoutOblique, capping: capFace}}
	if !farBiteOnHost(capFace, capFace.Geometry(), oblique, 1e-6) {
		t.Fatal("oblique runout must bite its OWN capping face (identity routing)")
	}
	if farBiteOnHost(other, other.Geometry(), oblique, 1e-6) {
		t.Fatal("oblique runout must NOT bite a different face even when the feet lie on its surface")
	}
	perp := armRails{far: far, runout: armRunout{regime: runoutPerpendicular, capping: capFace}}
	if !farBiteOnHost(other, other.Geometry(), perp, 1e-6) {
		t.Fatal("perpendicular runout must route by surface membership (both feet on the surface)")
	}
}
