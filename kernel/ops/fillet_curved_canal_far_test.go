// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// W3c/F1 — the canal arms' GEOMETRIC far termini, oracle-pinned on the REAL N7 STEP body. W4 proved the
// reused "far end = host-loop crossing" identity invalid on the real body (the s_4 ruling runs to z=130,
// true runout z=80 — a 50u gap); F1 replaces it with the closed-form section rail∩F_far + armSurface∩F_far.
// These tests drive the REAL imported body (importCorpusSolid), never the truncated W2 fixture: they wire
// the fixture's arm surfaces + reflected centres onto the REAL edges + host faces so F_far is identified
// from real topology, then assert every terminus against the DRAWEXE-verified closed forms.

// n7RealArms returns the fixture weld + reflected centres + boundaries (all far-independent, resolved by
// the working W2 path) with each arm's edge and two host faces re-bound to the REAL N7 body — so the
// far-terminus machinery reads F_far from real topology. The arm SURFACES and centres are unchanged.
func n7RealArms(t *testing.T) (cornerWeld, []edgeFillet, []math.Point3, canalBoundaries, float64, Resolution) {
	t.Helper()
	w, fix, res := n7CornerFill(t)
	_, boundaries, centres, scale := n7CanalWeldInputs(t, w, fix, res)
	body := importCorpusSolid(t, "simple/N7")
	corner := vertexNear(t, body, math.P3(50, 0, 10))
	arms := make([]edgeFillet, len(fix))
	for i := range fix {
		arms[i] = bindArmToRealEdge(t, fix[i], corner)
	}
	return w, arms, centres, boundaries, scale, res
}

// bindArmToRealEdge finds the real N7 fillet edge at the corner matching the fixture arm (a torus arm →
// the arc edge; a cylinder arm → the straight edge whose direction matches the arm axis) and rebinds the
// arm's edge + its two host faces to the real body's, keeping the arm SURFACE.
func bindArmToRealEdge(t *testing.T, arm edgeFillet, corner *topo.Vertex) edgeFillet {
	t.Helper()
	for _, e := range corner.Edges() {
		if realEdgeMatchesArm(e, arm, corner) {
			faces := e.Faces()
			arm.edge, arm.a, arm.b = e, faces[0], faces[1]
			// Mirror production's armWithSurface: a torus arm's angle-zero Ref is re-pointed at the far
			// cut so curvedHostArc sweeps [0→φ*] from there (the fixture leaves the raw import Ref).
			if tor, ok := arm.armSurface.(geom.Torus); ok {
				arm.armSurface = alignTorusRefToFar(tor, farEndVertex(e, corner.Point()).Point())
			}
			return arm
		}
	}
	t.Fatalf("no real N7 edge matches arm %T", arm.armSurface)
	return arm
}

// realEdgeMatchesArm reports whether real edge e is the one carrying fixture arm's surface: a torus arm
// pairs with the arc edge; a cylinder arm with the straight edge whose far direction is parallel to the
// arm axis.
func realEdgeMatchesArm(e *topo.Edge, arm edgeFillet, corner *topo.Vertex) bool {
	_, isArc := e.Geometry().(geom.Arc3d)
	switch s := arm.armSurface.(type) {
	case geom.Torus:
		return isArc
	case geom.Cylinder:
		if isArc {
			return false
		}
		far := farEndVertex(e, corner.Point())
		dir, err := math.UnitVector3FromVector(corner.Point().VectorTo(far.Point()))
		if err != nil {
			return false
		}
		return stdmath.Abs(float64(dir.AsVector().Dot(s.AxisDir.AsVector()))) > 0.999
	}
	return false
}

// armTermini re-derives one real arm's two host rails + terminal section at its reflected centre — the
// exact endSegs canalArmFace closes on — for the closed-form assertions.
func armTermini(t *testing.T, arm edgeFillet, centre math.Point3, w cornerWeld, scale float64, res Resolution) (endSeg, endSeg, endSeg) {
	t.Helper()
	set, ok := solveArmSetback(arm, centre, w.radius, scale, res)
	if !ok {
		t.Fatalf("arm %T: solveArmSetback declined at %v", arm.armSurface, centre)
	}
	wi := cornerWeld{center: centre, radius: w.radius, arms: []armSetback{set}}
	h0, h1, term, ok := canalArmRailsAndTerminal(arm, set, wi, res)
	if !ok {
		t.Fatalf("arm %T: canalArmRailsAndTerminal declined on the real body at %v", arm.armSurface, centre)
	}
	return h0, h1, term
}

// TestCanalFar_S4CylinderArmTerminus pins the s_4 arm (F_far = z=80 plane ⊥ spine): m_far = (45,50−√2000,
// 80), the two host feet, and the terminal cross-section arc sweep arccos(−1/9), all to res.Weld·scale.
func TestCanalFar_S4CylinderArmTerminus(t *testing.T) {
	w, arms, centres, _, scale, res := n7RealArms(t)
	i := cylinderArmAlong(t, arms, math.V3(0, 0, 1)) // s_4: axis ẑ
	tol := res.Weld() * scale
	_, _, term := armTermini(t, arms[i], centres[i], w, scale, res)
	arc, ok := term.curve.(geom.Arc3d)
	if !ok {
		t.Fatalf("s_4 terminal is %T, want geom.Arc3d (⊥ cross-section)", term.curve)
	}
	y := 50 - stdmath.Sqrt(2000)
	assertPointNear(t, "s_4 m_far / arc centre", arc.Center, math.P3(45, y, 80), tol)
	assertFeet(t, term, math.P3(50-50.0/9, 50-(10.0/9)*stdmath.Sqrt(2000), 80), math.P3(50, y, 80), tol, "s_4")
	assertScalarNear(t, "s_4 sweep arccos(-1/9)", arc.SweepAngle, stdmath.Acos(-1.0/9), tol)
}

// TestCanalFar_S10CylinderArmTerminus pins the s_10 arm (F_far = y=30 plane ⊥ spine): terminal arc centre
// (55,30,15), feet (50,30,15)/(55,30,10), sweep π/2 — exact.
func TestCanalFar_S10CylinderArmTerminus(t *testing.T) {
	w, arms, centres, _, scale, res := n7RealArms(t)
	i := cylinderArmAlong(t, arms, math.V3(0, 1, 0)) // s_10: axis ŷ
	tol := res.Weld() * scale
	_, _, term := armTermini(t, arms[i], centres[i], w, scale, res)
	arc, ok := term.curve.(geom.Arc3d)
	if !ok {
		t.Fatalf("s_10 terminal is %T, want geom.Arc3d", term.curve)
	}
	assertPointNear(t, "s_10 arc centre", arc.Center, math.P3(55, 30, 15), tol)
	assertFeet(t, term, math.P3(50, 30, 15), math.P3(55, 30, 10), tol, "s_10")
	assertScalarNear(t, "s_10 sweep π/2", arc.SweepAngle, stdmath.Pi/2, tol)
}

// TestCanalFar_S5TorusArmSpiricTerminus pins the s_5 arm (F_far = x=80 plane ∥ torus axis): the terminal
// is the SPIRIC section (geom.SpiricArc, NOT an arc), pointwise ON the derivation's P(v) form to the
// oracle tol; its endpoints are the wall foot (80,10,5) and the extended cap foot (80,50−√1125,10).
func TestCanalFar_S5TorusArmSpiricTerminus(t *testing.T) {
	w, arms, centres, _, scale, res := n7RealArms(t)
	i := torusArmIndex(t, arms)
	tol := res.Weld() * scale
	_, _, term := armTermini(t, arms[i], centres[i], w, scale, res)
	sa, ok := term.curve.(geom.SpiricArc)
	if !ok {
		t.Fatalf("s_5 terminal is %T, want geom.SpiricArc (torus∩plane section, not an arc)", term.curve)
	}
	assertFeet(t, term, math.P3(80, 10, 5), math.P3(80, 50-stdmath.Sqrt(1125), 10), tol, "s_5")
	const spiricTol = 5.5e-7 // OCCT's own approximation tol; ours is exact-on-surface (derivation §4)
	for kk := 0; kk <= 10; kk++ {
		s := float64(kk) / 10
		v := sa.V0 + s*(sa.V1-sa.V0) // the tube angle at parameter s (derivation's P(v))
		want := math.P3(80, 50-stdmath.Sqrt((45+5*stdmath.Cos(v))*(45+5*stdmath.Cos(v))-900), 5+5*stdmath.Sin(v))
		if d := float64(sa.PointAt(s).DistanceTo(want)); d > spiricTol {
			t.Fatalf("s_5 spiric at v=%.4f is %.3e off P(v) (tol %.1e)", v, d, spiricTol)
		}
	}
}

// TestCanalFar_HostRailFeetShared is the shared-edge identity gate: each arm's two host rails END exactly
// at the terminal section's two feet — so the arm face, the host rails and (F2) the F_far bite chain on
// the SAME points. Also asserts the s_5 CAP rail carries the +δ_cap extension (its far foot reaches the
// cap section (80,50−√1125,10), beyond the far-vertex azimuth).
func TestCanalFar_HostRailFeetShared(t *testing.T) {
	w, arms, centres, _, scale, res := n7RealArms(t)
	tol := res.Weld() * scale
	for i := range arms {
		h0, h1, term := armTermini(t, arms[i], centres[i], w, scale, res)
		d0 := float64(h0.from.DistanceTo(term.from))
		d1 := float64(h1.from.DistanceTo(term.to))
		if d0 > tol || d1 > tol {
			t.Fatalf("arm %d (%T): host rail feet %v/%v not shared with terminal %v/%v (off %.3e/%.3e)",
				i, arms[i].armSurface, h0.from, h1.from, term.from, term.to, d0, d1)
		}
	}
}

// TestCanalFar_S5CapRailExtension isolates the cap-rail extension: without it the cap rail would end at
// the far-vertex azimuth (the wall crossing (80,16.20…,10)); with the δ_cap = asin(d/ρ)−asin(d/(ρ+r))
// extension it reaches the cap section (80,50−√1125,10) = (80,16.4589…,10), 0.0862 rad (3.88u) further.
func TestCanalFar_S5CapRailExtension(t *testing.T) {
	w, arms, centres, _, scale, res := n7RealArms(t)
	i := torusArmIndex(t, arms)
	tol := res.Weld() * scale
	h0, h1, _ := armTermini(t, arms[i], centres[i], w, scale, res)
	cap := h0
	if _, isPlane := arms[i].b.Geometry().(geom.Plane); isPlane {
		cap = h1
	}
	assertPointNear(t, "s_5 cap-rail extended far foot", cap.from, math.P3(80, 50-stdmath.Sqrt(1125), 10), tol)
	// Discriminator: WITHOUT the extension the cap rail would end at the far-vertex azimuth on the ρ=45
	// circle — the far-vertex direction (0.6,−0.8) from the axis: (50+45·0.6, 50−45·0.8, 10) = (77,14,10).
	// The δ_cap = asin(d/ρ)−asin(d/(ρ+r)) = 0.0862 rad extension moves it 3.88u to the cap section end.
	unextended := math.P3(77, 14, 10)
	if d := float64(cap.from.DistanceTo(unextended)); stdmath.Abs(d-3.88) > 0.02 {
		t.Fatalf("s_5 cap rail extension = %.3fu from the un-extended azimuth foot %v, want 3.88 (δ_cap·ρ)", d, unextended)
	}
}

// TestCanalFar_ThreeArmFacesBuildOnRealBody is THE W4-blocker-gone deliverable: all three canalArmFace
// loops now CLOSE on the REAL N7 body (each builds, every junction chains through the W2 chainOnto gates).
func TestCanalFar_ThreeArmFacesBuildOnRealBody(t *testing.T) {
	w, arms, centres, boundaries, scale, res := n7RealArms(t)
	faces, reason := canalArmFaces(arms, w, boundaries, centres, scale, res)
	if reason != "" {
		t.Fatalf("canalArmFaces declined the REAL N7 body (W4 blocker not gone): %s", reason)
	}
	if len(faces) != len(arms) {
		t.Fatalf("built %d arm faces, want %d", len(faces), len(arms))
	}
	for i := range faces {
		if len(faces[i].loops) != 1 || len(faces[i].loops[0].pts) < 4 {
			t.Fatalf("arm %d face did not close into one ≥4-point loop", i)
		}
	}
}

// TestCanalFar_LoopCrossingMutationFails is the discriminator: on the REAL s_4 arm the OLD loop-crossing
// path (canalArmHostRails → armRulingEnd) DECLINES — it runs to the wall rim z=130, and runoutAgrees
// rejects it against the far vertex z=80 (the exact W4 blocker). The F1 geometric terminus builds the
// same arm at z=80. Proof the F_far section, not a loop crossing, is what closes the loop on the real body.
func TestCanalFar_LoopCrossingMutationFails(t *testing.T) {
	w, arms, centres, _, scale, res := n7RealArms(t)
	i := cylinderArmAlong(t, arms, math.V3(0, 0, 1)) // s_4
	set, ok := solveArmSetback(arms[i], centres[i], w.radius, scale, res)
	if !ok {
		t.Fatalf("s_4: solveArmSetback declined")
	}
	wi := cornerWeld{center: centres[i], radius: w.radius, arms: []armSetback{set}}
	// The OLD path: the loop-crossing rails decline on the real windowed wall (the W4 blocker).
	if _, _, ok := canalArmHostRails(arms[i], set, wi, res); ok {
		t.Fatal("expected the OLD loop-crossing rails to DECLINE on the real s_4 wall (runout z=80 is not a loop crossing)")
	}
	// The F1 path: the geometric F_far terminus BUILDS, at the true runout z=80.
	_, _, term, ok := canalArmRailsAndTerminal(arms[i], set, wi, res)
	if !ok {
		t.Fatal("s_4 geometric F_far terminus must build where the loop-crossing path could not")
	}
	if arc, isArc := term.curve.(geom.Arc3d); !isArc || stdmath.Abs(float64(arc.Center.Z)-80) > res.Weld()*scale {
		t.Fatalf("s_4 terminal must be the z=80 cross-section arc, got %T at z=%.3f", term.curve, arc.Center.Z)
	}
}

// TestCanalFar_ArcBranchGuardDeclinesObliqueFeet is the do-no-harm guard proof (F1 review, Finding 1):
// farCrossSectionArc 3-point-fits SOME arc through ANY two feet, so if the spiric branch is bypassed a
// torus arm whose F_far is NOT ⊥ its spine reaches the arc branch and SILENTLY snaps a wrong surface. The
// reviewer's mutation (spiric branch disabled) sent the s_5 feet — the wall foot (80,10,5) at r=5 and the
// cap foot (80,50−√1125,10) at 6.33 from the far ball centre (77,14,5) — through farCrossSectionArc and the
// arm loop STILL closed. feetOnFarCrossSection now GATES the branch: it declines feet that are not a true
// radius-r cross-section, and canalTerminalSection (arc branch, reached here via an OBLIQUE F_far that
// takes neither the spiric nor the ⊥-cylinder path) declines with it.
func TestCanalFar_ArcBranchGuardDeclinesObliqueFeet(t *testing.T) {
	tor, err := geom.NewTorus(math.P3(50, 50, 5), math.V3(0, 0, 1), 45, 5)
	if err != nil {
		t.Fatalf("build torus: %v", err)
	}
	const r, tol = 5.0, 7.5e-6
	mFar := math.P3(77, 14, 5) // torusBallCenter of the wall foot — the far cross-section centre
	wallFoot := math.P3(80, 10, 5)
	capFoot := math.P3(80, 50-stdmath.Sqrt(1125), 10) // the true s_5 cap foot: 6.33 from m_far, NOT r
	if d := float64(mFar.DistanceTo(capFoot)); stdmath.Abs(d-6.328) > 0.01 {
		t.Fatalf("precondition: cap foot is %.3f from m_far, want the reviewer's 6.33", d)
	}
	// The guard: the mutation feet are NOT a radius-r cross-section → decline (no snapped arc).
	if feetOnFarCrossSection(tor, r, wallFoot, capFoot, tol) {
		t.Fatalf("guard admitted feet at {%.3f,%.3f} from m_far as an r=%.1f cross-section",
			float64(mFar.DistanceTo(wallFoot)), float64(mFar.DistanceTo(capFoot)), r)
	}
	// A TRUE cross-section (both feet at r from m_far, same tube meridian) still builds.
	if !feetOnFarCrossSection(tor, r, wallFoot, math.P3(77, 14, 10), tol) {
		t.Fatal("guard rejected a genuine radius-r cross-section (both feet at r from m_far)")
	}
	// End-to-end: an OBLIQUE F_far (|n̂·axis|=√.5) bypasses the spiric branch, so canalTerminalSection
	// reaches the arc branch and DECLINES the mutation feet rather than snapping the 5/6.33 arc.
	oblique, err := geom.NewPlane(math.P3(80, 10, 5), math.V3(1, 0, 1))
	if err != nil {
		t.Fatalf("build oblique plane: %v", err)
	}
	if _, ok := canalTerminalSection(tor, r, oblique, endSeg{from: wallFoot}, endSeg{from: capFoot}, tol); ok {
		t.Fatal("canalTerminalSection built a terminal through non-cross-section feet on the arc branch")
	}
}

// cylinderArmAlong returns the index of the cylinder arm whose axis is parallel to dir, or fails.
func cylinderArmAlong(t *testing.T, arms []edgeFillet, dir math.Vector3) int {
	t.Helper()
	u, err := math.UnitVector3FromVector(dir)
	if err != nil {
		t.Fatalf("bad dir %v", dir)
	}
	for i := range arms {
		if c, ok := arms[i].armSurface.(geom.Cylinder); ok {
			if stdmath.Abs(float64(c.AxisDir.AsVector().Dot(u.AsVector()))) > 0.999 {
				return i
			}
		}
	}
	t.Fatalf("no cylinder arm with axis ∥ %v", dir)
	return -1
}

// assertFeet asserts the terminal endSeg's from/to are p0/p1 in EITHER order (the two host feet), within tol.
func assertFeet(t *testing.T, term endSeg, p0, p1 math.Point3, tol float64, name string) {
	t.Helper()
	direct := float64(term.from.DistanceTo(p0)) <= tol && float64(term.to.DistanceTo(p1)) <= tol
	swapped := float64(term.from.DistanceTo(p1)) <= tol && float64(term.to.DistanceTo(p0)) <= tol
	if !direct && !swapped {
		t.Fatalf("%s feet: terminal %v→%v does not match {%v,%v} (tol %.3e)", name, term.from, term.to, p0, p1, tol)
	}
}

// assertScalarNear asserts got is within tol of want.
func assertScalarNear(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if d := stdmath.Abs(got - want); d > tol {
		t.Fatalf("%s: %.10f vs %.10f (off %.3e, tol %.3e)", name, got, want, d, tol)
	}
}
