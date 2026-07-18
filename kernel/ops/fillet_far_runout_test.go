// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// FR1 — the general far-runout engine skeleton (far-runout-engine-architecture.md ADR-1..4). These tests
// pin the FR1 guarantees: (1) the perpendicular branch is byte-identical to today by CALL-GRAPH —
// armFarRunout's trim equals a direct farCrossSectionArc call with the same arguments; (2) the admission
// gate cappingFaceAtFarVertex finds the unique cap, declines the ≥2-non-host setback regime, AND declines
// when a SECOND picked/filleted edge ends at the far vertex (fillet-fillet interference — the pick guard,
// review finding b); (3) the OBLIQUE regime is classified correctly on distinct valid feet, which kills
// the "any plane ⇒ perpendicular" mutant (review finding c); (4) the FULL-population probe over B3's box
// corners (every one perpendicular to machine eps) plus D5 AND E4 meridians (oblique, 0.5) certifies the
// FR3 call-site flip introduces no latent regression — ~9+ orders apart, sinFloor (1e-6) safely between.

// perpFarFixture is a synthetic trihedral far vertex F=(0,0,100): a cylinder arm along ẑ (radius r), the
// arm edge along ẑ (hosts x=0 and y=0), and a cap plane z=100 ⊥ the spine. The two host-rail outer ends
// sit on the terminal circle. It gives full control over the perpendicular dispatch without a corner
// solve, so the call-graph identity is asserted in isolation.
type perpFarFixture struct {
	ef       edgeFillet
	w        cornerWeld
	h0, h1   endSeg
	cap      *topo.Face
	capEdgeA *topo.Edge // the faceA∧cap sharp edge ending at F — a candidate SECOND picked edge (pick-guard test)
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
		cap: cap, capEdgeA: e2, arm: arm, r: 10, farPoint: vFar.Point(),
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
	h0, h1, run, ok, reason := armFarRunout(fx.ef, fx.w, fx.h0, fx.h1, onlyArmFilleted(fx.ef), ResolutionForSize(200))
	if !ok || run.regime != runoutPerpendicular {
		t.Fatalf("armFarRunout: ok=%v regime=%d reason=%q, want ok+perpendicular", ok, run.regime, reason)
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

// onlyArmFilleted is the ordinary single-fillet-at-this-corner pick set: exactly the arm's own edge is
// picked, so the second-picked-edge guard never fires (the common admissible case).
func onlyArmFilleted(ef edgeFillet) map[uint64]bool {
	return map[uint64]bool{ef.edge.ID(): true}
}

// TestCappingFaceAtFarVertex_UniqueCap: the synthetic trihedral vertex yields exactly the cap face.
func TestCappingFaceAtFarVertex_UniqueCap(t *testing.T) {
	fx := buildPerpFarFixture(t)
	far := farEndVertex(fx.ef.edge, fx.w.center)
	got, ok, reason := cappingFaceAtFarVertex(far, fx.ef, onlyArmFilleted(fx.ef))
	if !ok || got != fx.cap {
		t.Fatalf("cappingFaceAtFarVertex = (%v, %v, %q), want the unique cap %v", got, ok, reason, fx.cap.ID())
	}
}

// TestCappingFaceAtFarVertex_DeclinesNValent: a far vertex with TWO non-host transverse faces (a second
// capping face meeting at F — the setback regime) must decline, never pick one arbitrarily.
func TestCappingFaceAtFarVertex_DeclinesNValent(t *testing.T) {
	ef, far := buildNValentFarVertex(t)
	if got, ok, _ := cappingFaceAtFarVertex(far, ef, onlyArmFilleted(ef)); ok {
		t.Fatalf("cappingFaceAtFarVertex accepted an n-valent far vertex (returned %v); want decline", got.ID())
	}
}

// TestCappingFaceAtFarVertex_DeclinesSecondPickedEdge is the pick-guard regression (review finding b). The
// face-count guard alone does NOT catch this: the arm edge e₁ (hosts a=faceA,b=faceB) and the cap edge
// e₂=faceA∧cap both end at F, so the non-host set is exactly {cap} (count 1 — the face-count guard would
// ACCEPT). But when e₂ is ALSO a picked/filleted edge, cap is a live host of fillet e₂ (fillet-fillet
// interference, out of scope) and the engine must decline. Proven by flipping ONLY the pick set: accept
// when the arm is the sole pick, decline the instant e₂ joins it.
func TestCappingFaceAtFarVertex_DeclinesSecondPickedEdge(t *testing.T) {
	fx := buildPerpFarFixture(t)
	far := farEndVertex(fx.ef.edge, fx.w.center)
	if _, ok, reason := cappingFaceAtFarVertex(far, fx.ef, onlyArmFilleted(fx.ef)); !ok {
		t.Fatalf("control: cappingFaceAtFarVertex should ACCEPT when only the arm edge is picked; reason=%q", reason)
	}
	withSecond := map[uint64]bool{fx.ef.edge.ID(): true, fx.capEdgeA.ID(): true}
	got, ok, reason := cappingFaceAtFarVertex(far, fx.ef, withSecond)
	if ok {
		t.Fatalf("cappingFaceAtFarVertex accepted a far vertex where a SECOND picked edge %d ends (got cap %v); want decline",
			fx.capEdgeA.ID(), got.ID())
	}
	if !strings.Contains(reason, "second filleted edge") {
		t.Fatalf("decline reason %q does not name the second filleted edge obstruction", reason)
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

// TestIntersectArmCapping_DeclinesInvalidFeet: FR2 implements the port, but zero/placeholder feet are not
// on the arm's section ellipse, so the §0 on-arm certificate declines — the oblique branch still floors
// honestly on invalid feet (do-no-harm), as it must until FR3 supplies authoritative closed-form feet.
func TestIntersectArmCapping_DeclinesInvalidFeet(t *testing.T) {
	arm, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 10)
	pl := planeOn(t, math.P3(0, 0, 100), math.V3(1, 0, 1))
	if c, ok := intersectArmCapping(arm, pl, [2]math.Point3{{}, {}}, 10, ResolutionForSize(200)); ok || c != nil {
		t.Fatalf("intersectArmCapping returned (%v, %v) on placeholder feet, want (nil, false)", c, ok)
	}
}

// obliqueTorusFarFixture is a synthetic OBLIQUE far vertex: a torus arm (spine = major circle radius 50
// in z=0, tube radius 10) terminating at F=(50,0,10), capped by a plane whose normal (0,1,1)/√2 is
// oblique to the spine tangent ŷ there (|n̂·t̂|=1/√2 → 1−0.707≈0.293, decisively oblique). The two feet
// are DISTINCT valid points on the terminal tube section — (50,0,10) and (60,0,0) — chosen so that
// farCrossSectionArc SUCCEEDS on them: that is what makes the "any plane ⇒ perpendicular" mutant
// detectable (under the mutant the fixture would take the fast-path and build an arc, mis-reporting
// perpendicular; the correct code routes it oblique and floors).
type obliqueTorusFarFixture struct {
	ef     edgeFillet
	w      cornerWeld
	h0, h1 endSeg
}

func buildObliqueTorusFarFixture(t *testing.T) obliqueTorusFarFixture {
	t.Helper()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 50, 10)
	if err != nil {
		t.Fatalf("torus arm: %v", err)
	}
	lin := topo.NewLineage(topo.Tok("test", "oblique-torus-far", 0))
	bld := topo.NewBuilder(true, lin)
	vNear := bld.AddVertex(math.P3(50, -50, 10), lin)
	vFar := bld.AddVertex(math.P3(50, 0, 10), lin) // F
	vA := bld.AddVertex(math.P3(50, 0, 60), lin)
	vB := bld.AddVertex(math.P3(90, 0, 10), lin)
	e := bld.AddEdge(geom.NewLineSegment(math.P3(50, -50, 10), math.P3(50, 0, 10)), vNear, vFar, lin) // arm edge ∥ ŷ
	e2 := bld.AddEdge(geom.NewLineSegment(math.P3(50, 0, 10), math.P3(50, 0, 60)), vFar, vA, lin)
	e3 := bld.AddEdge(geom.NewLineSegment(math.P3(50, 0, 10), math.P3(90, 0, 10)), vFar, vB, lin)
	faceA := bld.AddFace(planeOn(t, math.P3(50, 0, 10), math.V3(1, 0, 0)), lin, topo.OuterLoop(topo.Fwd(e), topo.Fwd(e2)))
	faceB := bld.AddFace(planeOn(t, math.P3(50, 0, 10), math.V3(0, 0, 1)), lin, topo.OuterLoop(topo.Fwd(e), topo.Fwd(e3)))
	bld.AddFace(planeOn(t, math.P3(50, 0, 10), math.V3(0, 1, 1)), lin, topo.OuterLoop(topo.Fwd(e2), topo.Fwd(e3))) // oblique cap
	return obliqueTorusFarFixture{
		ef: edgeFillet{a: faceA, b: faceB, edge: e, armSurface: tor},
		w:  cornerWeld{center: math.P3(50, -50, 10), radius: 10},
		h0: endSeg{from: math.P3(50, 0, 10)}, // top of the terminal tube section
		h1: endSeg{from: math.P3(60, 0, 0)},  // outer equator of the same section — distinct, valid
	}
}

// TestArmFarRunout_ObliqueRegime kills the "any plane ⇒ perpendicular" mutant (review finding c). On a
// realistic oblique far vertex with DISTINCT valid feet, armFarRunout must classify runoutOblique and
// floor (ok=false, the FR1 port stub). The mutation evidence: deleting the planePerpToDir angle check in
// farRunoutIsPerpendicular (so ANY plane ⇒ perpendicular) makes this fixture take the fast-path — because
// its feet are valid, farCrossSectionArc SUCCEEDS and armFarRunout returns ok=true/regime=perpendicular,
// FAILING both asserts below. Restoring the check greens it. (The pre-existing D5 decline test could NOT
// catch this: it feeds degenerate coincident feet, so farCrossSectionArc fails regardless of regime.)
func TestArmFarRunout_ObliqueRegime(t *testing.T) {
	fx := buildObliqueTorusFarFixture(t)
	// Precondition: the feet are DISTINCT and valid — farCrossSectionArc succeeds on them, so a
	// mis-classification as perpendicular would actually build an arc (that is the mutant this test kills).
	if _, ok := farCrossSectionArc(fx.ef.armSurface, fx.w.radius, fx.h0.from, fx.h1.from); !ok {
		t.Fatal("precondition: farCrossSectionArc must succeed on the distinct oblique feet (else the mutant is undetectable)")
	}
	_, _, run, ok, reason := armFarRunout(fx.ef, fx.w, fx.h0, fx.h1, onlyArmFilleted(fx.ef), ResolutionForSize(300))
	if run.regime != runoutOblique {
		t.Fatalf("armFarRunout classified regime=%d, want runoutOblique (the 'any plane ⇒ perpendicular' mutant survives)", run.regime)
	}
	if ok {
		t.Fatalf("oblique far vertex must FLOOR in FR1 (port stub) but returned ok=true; reason=%q", reason)
	}
}

// TestFarRunoutPopulationProbe is the ADR-2 population certificate — the FR3 call-site-flip regression
// gate. It classifies the WHOLE perpendicular population (every cylinder-arm box-corner far vertex on B3,
// not one witness) and asserts each is PERPENDICULAR to machine eps (1−|n̂_cap·t̂| ≤ 1e-12), while the
// oblique witnesses (D5 AND E4 meridian far vertices) are OBLIQUE (≥ 1e-3). The two populations sit ~9+
// orders apart with sinFloor (1e-6) safely between them, so wiring armRailBundle onto armFarRunout (FR3)
// can never make a current green silently evaluate the oblique branch, nor an oblique case take the arc.
func TestFarRunoutPopulationProbe(t *testing.T) {
	perp := allPerpendicularFarGaps(t, "simple/B3")
	if len(perp) == 0 {
		t.Fatal("B3: no perpendicular far vertices collected — the probe would be vacuous")
	}
	worst := 0.0
	for i, g := range perp {
		if g > worst {
			worst = g
		}
		if g > 1e-12 {
			t.Fatalf("B3 far vertex #%d gap %.3e exceeds 1e-12 — not machine-exact perpendicular", i, g)
		}
	}
	oblique := map[string]float64{"D5": meridianObliqueGap(t, "simple/D5"), "E4": meridianObliqueGap(t, "simple/E4")}
	t.Logf("population probe: B3 %d perpendicular far vertices (worst 1−|n̂·t̂|=%.3e) ; D5=%.3e ; E4=%.3e",
		len(perp), worst, oblique["D5"], oblique["E4"])
	for name, g := range oblique {
		if g < 1e-3 {
			t.Fatalf("%s meridian gap %.3e is below 1e-3 — not safely in the oblique population", name, g)
		}
	}
}

// allPerpendicularFarGaps collects 1−|n̂_cap·t̂_spine| at EVERY cylinder-arm plane-capped far vertex of a
// corpus box body — the whole perpendicular population, not a single witness. Each box corner's cap plane
// is ⊥ the arm spine, so every gap is 0 to machine eps; the FR3 flip must keep all of them perpendicular.
func allPerpendicularFarGaps(t *testing.T, name string) []float64 {
	body := importCorpusSolid(t, name)
	var gaps []float64
	for _, e := range body.Edges() {
		ef, handled, err := cylinderArmEdge(body, e, filletPick{edge: e, r0: 10, r1: 10})
		if !handled || err != nil || ef.armSurface == nil {
			continue
		}
		far := farEndVertex(e, math.P3(0, 0, 0))
		if g, ok := capNormalSpineGap(far, ef); ok {
			gaps = append(gaps, g)
		}
	}
	return gaps
}

// meridianObliqueGap builds a body's meridian (sphere∧longitude-plane y=0) torus arm and returns
// 1−|n̂_cap·t̂| at its far (top-cap) vertex — the oblique witness (0.5 in the oracle geometry). It also
// asserts that far vertex DECLINES through armFarRunout: the FR1 port is a stub, so the oblique branch
// floors honestly (the byte-neutral guarantee the whole slice rests on).
func meridianObliqueGap(t *testing.T, name string) float64 {
	body := importCorpusSolid(t, name)
	mer := meridianEdge(t, body)
	ef, handled, err := sphereArmEdge(body, mer, filletPick{edge: mer, r0: 10, r1: 10})
	if !handled || err != nil || ef.armSurface == nil {
		t.Fatalf("%s meridian: sphereArmEdge handled=%v err=%v arm=%v", name, handled, err, ef.armSurface)
	}
	far := mer.EndVertex() // the +z cap end
	g, ok := capNormalSpineGap(far, ef)
	if !ok {
		t.Fatalf("%s meridian: no plane capping face at the far vertex", name)
	}
	w := cornerWeld{center: mer.StartVertex().Point(), radius: 10}
	if _, _, _, obOK, _ := armFarRunout(ef, w, endSeg{from: far.Point()}, endSeg{from: far.Point()}, onlyArmFilleted(ef), ResolutionForSize(300)); obOK {
		t.Fatalf("%s oblique far vertex must DECLINE in FR1 (the port is a stub) — it floored non-honestly", name)
	}
	return g
}

// capNormalSpineGap returns 1−|n̂_cap·t̂_spine| at far vertex `far` of arm ef: it finds the capping face,
// requires it be a plane, and dots its normal with the arm's spine tangent. ok=false if there is no
// unique plane cap or no spine tangent.
func capNormalSpineGap(far *topo.Vertex, ef edgeFillet) (float64, bool) {
	capping, ok, _ := cappingFaceAtFarVertex(far, ef, onlyArmFilleted(ef))
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
