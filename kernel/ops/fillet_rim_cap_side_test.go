// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The spool the Reversed-cap closed-rim guard is falsified on. A base disc, a thin neck, and an
// OVERHANGING head, all coaxial about +z:
//
//	base  r ≤ 27, z ∈ [0, 17]
//	neck  r ≤  5, z ∈ [17, 20]
//	head  r ≤ 30, z ∈ [20, 25]
//
// The picked rim is the head's lower rim (r = 30, z = 20) between the head wall and the head's
// underside annulus (r ∈ [5, 30], material-outward normal −z). The edge is CONVEX.
//
// ★ WHY THIS SHAPE AND NOT A PLAIN BOSS. solveRim gates its seat on a probe at the ball centre,
// radius cylR−r off the axis. On a plain boss a wrong-side seat puts that probe in open air and the
// existing probe rejects it — which is exactly why simple/K1 and simple/Z1, the corpus's two live
// Reversed-cap rims, never shipped a bad band. Here the BASE carries material at radius 26 below the
// cap, so a wrong-side seat's probe lands INSIDE the solid and passes. The stored normal is then the
// only thing deciding the side, and if it is Reversed the band is built a whole 2r below where it
// belongs — with its cylinder-tangent rail (r = 30) hanging 3 units outside the base (r = 27).
const (
	spoolBaseR, spoolBaseTop     = 27.0, 17.0
	spoolNeckR                   = 5.0
	spoolHeadR, spoolCapZ        = 30.0, 20.0
	spoolHeadTop, spoolRimRadius = 25.0, 4.0
)

// spoolWithOverhangingHead builds the fixture solid as an analytic surface of revolution.
func spoolWithOverhangingHead(t *testing.T) *topo.Body {
	t.Helper()
	mer := []math.Point2{
		math.P2(0, 0), math.P2(spoolBaseR, 0), math.P2(spoolBaseR, spoolBaseTop),
		math.P2(spoolNeckR, spoolBaseTop), math.P2(spoolNeckR, spoolCapZ),
		math.P2(spoolHeadR, spoolCapZ), math.P2(spoolHeadR, spoolHeadTop), math.P2(0, spoolHeadTop),
	}
	b, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "spool")
	if b == nil || err != nil {
		t.Fatalf("spool fixture: SolidOfRevolution = %v, %v; want a body", b, err)
	}
	return b
}

// headUnderside returns the fixture's cap face — the head's underside annulus at z = spoolCapZ.
func headUnderside(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if ok && stdmath.Abs(float64(pl.Origin.Z)-spoolCapZ) < 1e-9 {
			return f
		}
	}
	t.Fatalf("spool fixture has no planar face at z=%g", spoolCapZ)
	return nil
}

// headLowerRim returns the closed circular edge at (r = spoolHeadR, z = spoolCapZ) — the pick.
func headLowerRim(t *testing.T, b *topo.Body) *topo.Edge {
	t.Helper()
	for _, e := range b.Edges() {
		if !isClosedCircularEdge(e) {
			continue
		}
		p := e.Geometry().PointAt(0)
		r := stdmath.Hypot(float64(p.X), float64(p.Y))
		if stdmath.Abs(r-spoolHeadR) < 1e-9 && stdmath.Abs(float64(p.Z)-spoolCapZ) < 1e-9 {
			return e
		}
	}
	t.Fatalf("spool fixture has no closed rim at r=%g z=%g", spoolHeadR, spoolCapZ)
	return nil
}

// withStoredNormalFlipped rebuilds b with ONE planar face stored the other way up: its plane's U and V
// axes swapped (so Plane.Normal() = V×U is the exact negation of U×V) and the face marked Reversed, which
// leaves the material side — and therefore the solid — untouched. This is simple/Z1's condition
// synthesised: "a plain convex rim whose cap is simply stored bottom-up" (fillet_rim_concave.go).
func withStoredNormalFlipped(b *topo.Body, target *topo.Face) *topo.Body {
	bld := topo.NewBuilder(b.IsSolid(), b.Lineage())
	verts := map[*topo.Vertex]*topo.Vertex{}
	for _, v := range b.Vertices() {
		verts[v] = bld.AddVertex(v.Point(), v.Lineage())
	}
	edges := map[*topo.Edge]*topo.Edge{}
	for _, e := range b.Edges() {
		edges[e] = bld.AddEdge(e.Geometry(), verts[e.StartVertex()], verts[e.EndVertex()], e.Lineage())
	}
	for _, f := range b.Faces() {
		addFlippedOrCopiedFace(bld, f, target, edges)
	}
	return bld.Build()
}

// addFlippedOrCopiedFace adds f verbatim, except the target planar face, which is added with its stored
// plane normal negated and Reversed set.
func addFlippedOrCopiedFace(bld *topo.Builder, f, target *topo.Face, edges map[*topo.Edge]*topo.Edge) {
	specs := make([]topo.LoopSpec, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		uses := make([]topo.Use, 0, len(l.EdgeUses()))
		for _, u := range l.EdgeUses() {
			uses = append(uses, topo.Use{Edge: edges[u.Edge()], Reversed: u.Reversed()})
		}
		if l.IsOuter() {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	pl, isPlane := f.Geometry().(geom.Plane)
	if f != target || !isPlane {
		if f.Reversed() {
			bld.AddReversedFace(f.Geometry(), f.Lineage(), specs...)
			return
		}
		bld.AddFace(f.Geometry(), f.Lineage(), specs...)
		return
	}
	bld.AddReversedFace(geom.Plane{Origin: pl.Origin, UAxis: pl.VAxis, VAxis: pl.UAxis}, f.Lineage(), specs...)
}

// spoolVolume is the fixture's CLOSED-FORM volume with the rim rounded: the three revolved discs less
// the corner the blend removes, by Pappus on the corner region (a square of side r minus its quarter
// disc, swept about the axis at its own centroid radius).
func spoolVolume(r float64) float64 {
	discs := stdmath.Pi * (spoolBaseR*spoolBaseR*spoolBaseTop +
		spoolNeckR*spoolNeckR*(spoolCapZ-spoolBaseTop) +
		spoolHeadR*spoolHeadR*(spoolHeadTop-spoolCapZ))
	rc := spoolHeadR - r                                          // ball-centre radius
	square := r * r * (rc + r/2)                                  // ∫ρ dA over the r×r corner square
	quarter := stdmath.Pi * r * r / 4 * (rc + 4*r/(3*stdmath.Pi)) // …over the quarter disc
	return discs - 2*stdmath.Pi*(square-quarter)
}

// TestReversedCapClosedRimSeatsOnTheMaterialSide is the falsification of the solveRim cap-normal guard.
// The fixture's cap is stored bottom-up, so solveRim's historic `inward := pl.Normal().Negate()` seats the
// rolling ball a full 2r BELOW where it belongs, and — unlike simple/K1 and simple/Z1 — the wrong seat's
// own material probe passes, because the base carries material at the ball-centre radius. Without the
// guard the band is built into the void below the head: its cylinder-tangent rail sits at r = 30 where the
// solid is only r = 27. With it the seat is the one the edge's convexity and the true outward normal
// agree on, and the built solid matches the closed form.
func TestReversedCapClosedRimSeatsOnTheMaterialSide(t *testing.T) {
	plain := spoolWithOverhangingHead(t)
	b := withStoredNormalFlipped(plain, headUnderside(t, plain))
	if !headUnderside(t, b).Reversed() {
		t.Fatalf("fixture cap face is not stored Reversed — the hazard is not reproduced")
	}
	rim := headLowerRim(t, b)
	rf, err := resolveRim(b, rim.ReferenceKey(), spoolRimRadius)
	if err != nil {
		t.Fatalf("resolveRim on the Reversed-cap spool: %v", err)
	}
	assertRimBandOnTheMaterialSide(t, b, rf)
	assertSpoolBuilds(t, b, rim)
}

// assertRimBandOnTheMaterialSide checks the solved band against the closed form AND against the solid:
// the torus centre must be r ABOVE the cap (inside the head), and every point of the cylinder-tangent
// rail must lie ON the head wall — probed just inside it, since a point exactly on the surface is a
// winding-number coin flip.
func assertRimBandOnTheMaterialSide(t *testing.T, b *topo.Body, rf *rimFillet) {
	t.Helper()
	tor, ok := rf.band.(geom.Torus)
	if !ok {
		t.Fatalf("band is %T, want geom.Torus", rf.band)
	}
	wantZ := spoolCapZ + spoolRimRadius
	if stdmath.Abs(float64(tor.Center.Z)-wantZ) > 1e-9 || stdmath.Abs(tor.MajorRadius-(spoolHeadR-spoolRimRadius)) > 1e-9 {
		t.Errorf("band torus centre %v R=%g, want centre z=%g (the material side of the cap) R=%g",
			tor.Center, tor.MajorRadius, wantZ, spoolHeadR-spoolRimRadius)
	}
	for i := range 8 {
		p := rf.cylTan.PointAt(float64(i) / 8 * 2 * stdmath.Pi)
		inset := math.P3(p.X*0.99, p.Y*0.99, p.Z)
		if !PointInsideBody(b, inset) {
			t.Errorf("cyl-tangent rail point %v is not on the solid (probe %v outside) — the band was built into the void",
				p, inset)
		}
	}
}

// assertSpoolBuilds drives the SHIPPED entry point and checks the result is a valid solid whose volume
// matches the closed form.
func assertSpoolBuilds(t *testing.T, b *topo.Body, rim *topo.Edge) {
	t.Helper()
	out, err := FilletCylinderRim(b, rim.ReferenceKey(), spoolRimRadius)
	if err != nil {
		t.Fatalf("FilletCylinderRim on the spool: %v", err)
	}
	if rep := Validate(out); !rep.Valid || !out.IsSolid() {
		t.Fatalf("filleted spool is not a valid solid: %+v", rep.Issues)
	}
	got := BodyGeometryProperties(out, PropertyQuality()).Volume
	want := spoolVolume(spoolRimRadius)
	if rel := stdmath.Abs(got-want) / want; rel > 2e-3 {
		t.Errorf("filleted spool volume %g, closed form %g (rel %.4f%%)", got, want, rel*100)
	}
}

// TestUnreversedCapClosedRimIsUnchangedByTheGuard is the other direction: the guard must not flip a rim
// whose stored normal was already the outward one. The SAME spool, unflipped, must solve to the SAME band
// — centre, radii, and both tangent rails — as the Reversed one.
func TestUnreversedCapClosedRimIsUnchangedByTheGuard(t *testing.T) {
	plain := spoolWithOverhangingHead(t)
	if headUnderside(t, plain).Reversed() {
		t.Fatalf("the unflipped fixture's cap is already Reversed — the control is not a control")
	}
	got, err := resolveRim(plain, headLowerRim(t, plain).ReferenceKey(), spoolRimRadius)
	if err != nil {
		t.Fatalf("resolveRim on the plain spool: %v", err)
	}
	flipped := withStoredNormalFlipped(plain, headUnderside(t, plain))
	want, err := resolveRim(flipped, headLowerRim(t, flipped).ReferenceKey(), spoolRimRadius)
	if err != nil {
		t.Fatalf("resolveRim on the Reversed-cap spool: %v", err)
	}
	assertSameBand(t, got, want)
}

// assertSameBand fails unless two solved rim fillets carry the same torus and the same tangent rails.
func assertSameBand(t *testing.T, got, want *rimFillet) {
	t.Helper()
	g, gok := got.band.(geom.Torus)
	w, wok := want.band.(geom.Torus)
	if !gok || !wok {
		t.Fatalf("bands are %T and %T, want geom.Torus", got.band, want.band)
	}
	if g.Center.DistanceTo(w.Center) > 1e-12 || stdmath.Abs(g.MajorRadius-w.MajorRadius) > 1e-12 ||
		stdmath.Abs(g.MinorRadius-w.MinorRadius) > 1e-12 {
		t.Errorf("band torus %v R=%g r=%g, want %v R=%g r=%g",
			g.Center, g.MajorRadius, g.MinorRadius, w.Center, w.MajorRadius, w.MinorRadius)
	}
	for i := range 8 {
		u := float64(i) / 8 * 2 * stdmath.Pi
		if d := got.cylTan.PointAt(u).DistanceTo(want.cylTan.PointAt(u)); d > 1e-12 {
			t.Errorf("cyl-tangent rail differs by %g at u=%g", d, u)
		}
		if d := got.capTan.PointAt(u).DistanceTo(want.capTan.PointAt(u)); d > 1e-12 {
			t.Errorf("cap-tangent rail differs by %g at u=%g", d, u)
		}
	}
}
