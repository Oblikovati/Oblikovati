// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// FR1 — the general far-runout engine skeleton (far-runout-engine-architecture.md ADR-1..4). These tests
// pin the three FR1 guarantees: (1) the perpendicular branch is byte-identical to today by CALL-GRAPH —
// armFarRunout's trim equals a direct farCrossSectionArc call with the same arguments; (2) the scope
// guard cappingFaceAtFarVertex finds the unique cap and declines the n-valent / ≥2-non-host setback
// regime; (3) the population probe on the REAL D5/B3 bodies proves the dispatch boundary is safe — B3's
// perpendicular caps read |n̂_cap·t̂|=1 to machine eps, D5's oblique cap reads 0.5, ~15 orders apart.

// perpFarFixture is a synthetic trihedral far vertex F=(0,0,100): a cylinder arm along ẑ (radius r), the
// arm edge along ẑ (hosts x=0 and y=0), and a cap plane z=100 ⊥ the spine. The two host-rail outer ends
// sit on the terminal circle. It gives full control over the perpendicular dispatch without a corner
// solve, so the call-graph identity is asserted in isolation.
type perpFarFixture struct {
	ef       edgeFillet
	w        cornerWeld
	h0, h1   endSeg
	cap      *topo.Face
	arm      geom.Cylinder
	r        float64
	farPoint math.Point3
}

func buildPerpFarFixture(t *testing.T) perpFarFixture {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "perp-far", 0))
	bld := topo.NewBuilder(true, lin)
	vNear := bld.AddVertex(math.P3(0, 0, 0), lin)
	vFar := bld.AddVertex(math.P3(0, 0, 100), lin)
	vA := bld.AddVertex(math.P3(0, 50, 100), lin)
	vB := bld.AddVertex(math.P3(50, 0, 100), lin)
	e := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(0, 0, 100)), vNear, vFar, lin) // arm edge ∥ ẑ
	e2 := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 100), math.P3(0, 50, 100)), vFar, vA, lin)
	e3 := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 100), math.P3(50, 0, 100)), vFar, vB, lin)
	faceA := bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(1, 0, 0)), lin, topo.OuterLoop(topo.Fwd(e), topo.Fwd(e2)))
	faceB := bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(0, 1, 0)), lin, topo.OuterLoop(topo.Fwd(e), topo.Fwd(e3)))
	cap := bld.AddFace(planeOn(t, math.P3(0, 0, 100), math.V3(0, 0, 1)), lin, topo.OuterLoop(topo.Fwd(e2), topo.Fwd(e3)))
	arm, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 10)
	if err != nil {
		t.Fatalf("cylinder arm: %v", err)
	}
	return perpFarFixture{
		ef:  edgeFillet{a: faceA, b: faceB, edge: e, armSurface: arm},
		w:   cornerWeld{center: math.P3(0, 0, 0), radius: 10},
		h0:  endSeg{from: math.P3(10, 0, 100)},
		h1:  endSeg{from: math.P3(0, 10, 100)},
		cap: cap, arm: arm, r: 10, farPoint: vFar.Point(),
	}
}

// TestArmFarRunout_PerpendicularCallGraphIdentity is the byte-identity gate (ADR-2): armFarRunout on a
// perpendicular far vertex returns the SAME cross-section arc farCrossSectionArc produces today, with the
// host rails passed through untouched — identity by call-graph, not tolerance.
func TestArmFarRunout_PerpendicularCallGraphIdentity(t *testing.T) {
	fx := buildPerpFarFixture(t)
	want, wok := farCrossSectionArc(fx.arm, fx.r, fx.h0.from, fx.h1.from)
	if !wok {
		t.Fatal("precondition: farCrossSectionArc must build the reference arc")
	}
	h0, h1, run, ok := armFarRunout(fx.ef, fx.w, fx.h0, fx.h1, ResolutionForSize(200))
	if !ok || run.regime != runoutPerpendicular {
		t.Fatalf("armFarRunout: ok=%v regime=%d, want ok+perpendicular", ok, run.regime)
	}
	if run.capping != fx.cap {
		t.Fatalf("run.capping = %v, want the cap face %v", run.capping, fx.cap.ID())
	}
	if h0 != fx.h0 || h1 != fx.h1 {
		t.Fatalf("perpendicular regime moved a host rail: %v/%v want %v/%v", h0, h1, fx.h0, fx.h1)
	}
	assertEndSegIdentical(t, run.trim, want)
	if run.feet != [2]math.Point3{fx.h0.from, fx.h1.from} {
		t.Fatalf("run.feet = %v, want the host-rail outer ends", run.feet)
	}
}

// assertEndSegIdentical asserts two endSegs are the byte-identical construction: same from/to/mid and the
// same curve sampled at 11 parameters (the call-graph-identity certificate — exact equality, no tol).
func assertEndSegIdentical(t *testing.T, got, want endSeg) {
	t.Helper()
	if got.from != want.from || got.to != want.to || got.mid != want.mid || got.arc != want.arc {
		t.Fatalf("endSeg endpoints differ: got {%v→%v mid %v arc %v} want {%v→%v mid %v arc %v}",
			got.from, got.to, got.mid, got.arc, want.from, want.to, want.mid, want.arc)
	}
	for k := 0; k <= 10; k++ {
		s := float64(k) / 10
		if got.curve.PointAt(s) != want.curve.PointAt(s) {
			t.Fatalf("trim curve differs at s=%.1f: %v vs %v (not call-graph-identical)", s, got.curve.PointAt(s), want.curve.PointAt(s))
		}
	}
}

// TestCappingFaceAtFarVertex_UniqueCap: the synthetic trihedral vertex yields exactly the cap face.
func TestCappingFaceAtFarVertex_UniqueCap(t *testing.T) {
	fx := buildPerpFarFixture(t)
	far := farEndVertex(fx.ef.edge, fx.w.center)
	got, ok := cappingFaceAtFarVertex(far, fx.ef)
	if !ok || got != fx.cap {
		t.Fatalf("cappingFaceAtFarVertex = (%v, %v), want the unique cap %v", got, ok, fx.cap.ID())
	}
}

// TestCappingFaceAtFarVertex_DeclinesNValent: a far vertex with TWO non-host transverse faces (a second
// capping face / picked edge meeting at F — the setback regime) must decline, never pick one arbitrarily.
func TestCappingFaceAtFarVertex_DeclinesNValent(t *testing.T) {
	ef, far := buildNValentFarVertex(t)
	if got, ok := cappingFaceAtFarVertex(far, ef); ok {
		t.Fatalf("cappingFaceAtFarVertex accepted an n-valent far vertex (returned %v); want decline", got.ID())
	}
}

// buildNValentFarVertex is a far vertex F with 2 host faces AND 2 non-host transverse cap planes — the
// out-of-scope setback regime (a second picked edge / ≥2 non-host faces). Both caps are ⊥ the ẑ edge.
func buildNValentFarVertex(t *testing.T) (edgeFillet, *topo.Vertex) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "nvalent-far", 0))
	bld := topo.NewBuilder(true, lin)
	vNear := bld.AddVertex(math.P3(0, 0, 0), lin)
	vFar := bld.AddVertex(math.P3(0, 0, 100), lin)
	spoke := func(to math.Point3) *topo.Edge {
		return bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 100), to), vFar, bld.AddVertex(to, lin), lin)
	}
	e := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(0, 0, 100)), vNear, vFar, lin)
	e2, e3, e4 := spoke(math.P3(0, 50, 100)), spoke(math.P3(50, 0, 100)), spoke(math.P3(0, -50, 100))
	faceA := bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(1, 0, 0)), lin, topo.OuterLoop(topo.Fwd(e), topo.Fwd(e2)))
	faceB := bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(0, 1, 0)), lin, topo.OuterLoop(topo.Fwd(e), topo.Fwd(e3)))
	bld.AddFace(planeOn(t, math.P3(0, 0, 100), math.V3(0, 0, 1)), lin, topo.OuterLoop(topo.Fwd(e2), topo.Fwd(e4))) // cap1 ⊥ ẑ
	bld.AddFace(planeOn(t, math.P3(0, 0, 100), math.V3(0, 0, 1)), lin, topo.OuterLoop(topo.Fwd(e3), topo.Fwd(e4))) // cap2 ⊥ ẑ
	arm, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 10)
	return edgeFillet{a: faceA, b: faceB, edge: e, armSurface: arm}, vFar
}

// TestIntersectArmCapping_StubDeclines: the FR1 port stub declines (FR2 implements it). armFarRunout's
// oblique branch therefore floors honestly.
func TestIntersectArmCapping_StubDeclines(t *testing.T) {
	arm, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 10)
	pl := planeOn(t, math.P3(0, 0, 100), math.V3(1, 0, 1))
	if c, ok := intersectArmCapping(arm, pl, [2]math.Point3{{}, {}}, 10, ResolutionForSize(200)); ok || c != nil {
		t.Fatalf("intersectArmCapping stub returned (%v, %v), want (nil, false) until FR2", c, ok)
	}
}

// TestFarRunoutPopulationProbe is the ADR-2 population certificate on the REAL corpus bodies: every
// current far vertex is PERPENDICULAR (1−|n̂_cap·t̂| ≤ 1e-12, machine-exact for the axis-aligned box),
// and D5's meridian arm far vertex is OBLIQUE (1−|n̂_cap·t̂| ≥ 1e-3). The two populations are ≥9 orders
// apart with sinFloor (1e-6) safely between them — so the perpendicular fast-path can never silently
// evaluate an oblique case, and no oblique case can silently take the arc.
func TestFarRunoutPopulationProbe(t *testing.T) {
	perp := b3PerpendicularGap(t)
	obliq := d5MeridianGap(t)
	t.Logf("population probe: B3 perpendicular 1−|n̂·t̂|=%.3e ; D5 meridian oblique 1−|n̂·t̂|=%.3e", perp, obliq)
	if perp > 1e-12 {
		t.Fatalf("B3 perpendicular gap %.3e exceeds 1e-12 — not machine-exact perpendicular", perp)
	}
	if obliq < 1e-3 {
		t.Fatalf("D5 meridian gap %.3e is below 1e-3 — not safely in the oblique population", obliq)
	}
}

// b3PerpendicularGap builds a B3 cylinder arm on its real edge and returns 1−|n̂_cap·t̂_spine| at its far
// vertex (a box cap ⊥ the spine → 0 to machine eps).
func b3PerpendicularGap(t *testing.T) float64 {
	body := importCorpusSolid(t, "simple/B3")
	for _, e := range body.Edges() {
		ef, handled, err := cylinderArmEdge(body, e, filletPick{edge: e, r0: 10, r1: 10})
		if !handled || err != nil || ef.armSurface == nil {
			continue
		}
		far := farEndVertex(e, math.P3(0, 0, 0))
		if g, ok := capNormalSpineGap(far, ef); ok {
			return g
		}
	}
	t.Fatal("B3: no cylinder-arm edge with a perpendicular trihedral far vertex")
	return 0
}

// d5MeridianGap builds the D5 meridian (sphere∧longitude-plane y=0) torus arm and returns 1−|n̂_cap·t̂|
// at its far vertex — the oblique cap (0.5 in the oracle geometry).
func d5MeridianGap(t *testing.T) float64 {
	body := importCorpusSolid(t, "simple/D5")
	mer := meridianEdge(t, body)
	ef, handled, err := sphereArmEdge(body, mer, filletPick{edge: mer, r0: 10, r1: 10})
	if !handled || err != nil || ef.armSurface == nil {
		t.Fatalf("D5 meridian: sphereArmEdge handled=%v err=%v arm=%v", handled, err, ef.armSurface)
	}
	far := mer.EndVertex() // the +z cap end
	g, ok := capNormalSpineGap(far, ef)
	if !ok {
		t.Fatal("D5 meridian: no plane capping face at the far vertex")
	}
	if _, _, _, obOK := armFarRunout(ef, cornerWeld{center: mer.StartVertex().Point(), radius: 10}, endSeg{from: far.Point()}, endSeg{from: far.Point()}, ResolutionForSize(300)); obOK {
		t.Fatal("D5 oblique far vertex must DECLINE in FR1 (the port is a stub) — it floored non-honestly")
	}
	return g
}

// capNormalSpineGap returns 1−|n̂_cap·t̂_spine| at far vertex `far` of arm ef: it finds the capping face,
// requires it be a plane, and dots its normal with the arm's spine tangent. ok=false if there is no
// unique plane cap or no spine tangent.
func capNormalSpineGap(far *topo.Vertex, ef edgeFillet) (float64, bool) {
	capping, ok := cappingFaceAtFarVertex(far, ef)
	if !ok {
		return 0, false
	}
	pl, isPlane := capping.Geometry().(geom.Plane)
	tan, tok := armSpineTangentAtFar(ef.armSurface, far.Point())
	if !isPlane || !tok {
		return 0, false
	}
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return 0, false
	}
	return 1 - stdmath.Abs(float64(n.AsVector().Dot(tan.AsVector()))), true
}

// meridianEdge finds D5's longitude-plane (y=0) meridian edge: a sphere∧plane edge whose plane normal is
// ±ŷ. That arm's far vertex on the top/bottom cap is the oblique population's witness.
func meridianEdge(t *testing.T, body *topo.Body) *topo.Edge {
	t.Helper()
	for _, e := range body.Edges() {
		_, pl, _, _, ok := spherePlaneEdge(e)
		if !ok {
			continue
		}
		n, _ := math.UnitVector3FromVector(pl.Normal())
		v := n.AsVector()
		if stdmath.Abs(float64(v.Y)) > 0.99 && stdmath.Abs(float64(v.X)) < 1e-2 && stdmath.Abs(float64(v.Z)) < 1e-2 {
			return e
		}
	}
	t.Fatal("D5: no meridian (y=0 longitude-plane) sphere∧plane edge")
	return nil
}
