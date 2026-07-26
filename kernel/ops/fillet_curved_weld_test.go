// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The shared test bed for the M5 curved-arm trihedral weld (T5.2–T5.4). It reuses b3CornerArms
// (fillet_curved_corner_solve_test.go) — the oracle-closed B3 corner from
// m5-weld-setback-retrim-derivation.md — and adds the three certified host-tangent points, the
// spherical-triangle vertices every weld rail must connect.

// b3TangentPoints returns the three certified host-tangent points T_W, T_K, T_N (derivation §A.2),
// kept exact via b3CornerCY=−√1500 so the rail endpoints hit them to machine precision:
//   - T_W = radial projection of C onto the wall R=50 (C_xy scaled 50/40), z=90;
//   - T_K = C + r·ẑ (foot on the cap z=100);
//   - T_N = foot of C onto the radial plane x=0.
func b3TangentPoints() (tW, tK, tN math.Point3) {
	tW = math.P3(12.5, 1.25*b3CornerCY, 90) // 1.25·C_xy: C_xy has radius 40, wall has radius 50
	tK = math.P3(10, b3CornerCY, 100)
	tN = math.P3(0, b3CornerCY, 90)
	return tW, tK, tN
}

// railOracle returns the two certified endpoints and the certified subtense of one arm's weld rail,
// discriminating the arms by surface type and axis: torus W∧K (90°), vertical cyl W∧N
// (arccos(−0.25)=104.478°), planar cyl K∧N (90°).
func railOracle(t *testing.T, a armSetback, tW, tK, tN math.Point3) ([2]math.Point3, float64) {
	t.Helper()
	switch s := a.arm.(type) {
	case geom.Torus:
		return [2]math.Point3{tW, tK}, stdmath.Pi / 2
	case geom.Cylinder:
		if stdmath.Abs(float64(s.AxisDir.Z())-1) < 0.5 { // ẑ axis → vertical W∧N arm
			return [2]math.Point3{tW, tN}, stdmath.Acos(-0.25)
		}
		return [2]math.Point3{tK, tN}, stdmath.Pi / 2 // ŷ axis → planar K∧N arm
	default:
		t.Fatalf("unexpected arm surface %T", a.arm)
		return [2]math.Point3{}, 0
	}
}

// TestCurvedSetbackRail_B3 drives the setback-rail constructor for all three B3 arms: each rail must
// be the corner sphere's great-circle arc (centre C, radius r, its plane through C) joining that
// arm's two certified host-tangent points, with the certified subtense (torus/planar 90°, cyl
// 104.478°). This is the T5.2 §A.2 weld rail.
func TestCurvedSetbackRail_B3(t *testing.T) {
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner (want ok)")
	}
	tW, tK, tN := b3TangentPoints()
	tol := res.Weld() * sphere.Radius
	for _, a := range w.arms {
		wantEnds, wantSubtense := railOracle(t, a, tW, tK, tN)
		assertSetbackRail(t, w, a, wantEnds, wantSubtense, tol)
	}
}

// assertSetbackRail builds one arm's rail and asserts it is the certified great-circle arc: a
// supporting circle centred at C with radius r whose plane passes through C, the two endpoints the
// arm's two host-tangent points (either order), and the certified subtense.
func assertSetbackRail(t *testing.T, w cornerWeld, a armSetback, wantEnds [2]math.Point3, wantSubtense, tol float64) {
	t.Helper()
	rail, ok := curvedSetbackRail(w, a)
	if !ok {
		t.Fatalf("curvedSetbackRail declined a certified arm (%T)", a.arm)
	}
	if d := rail.Center.DistanceTo(w.center); d > tol {
		t.Fatalf("rail centre off C by %.3e (want ≤%.1e) — not a great circle", d, tol)
	}
	if e := stdmath.Abs(rail.Radius - w.radius); e > tol {
		t.Fatalf("rail radius = %.9f, want r=%.9f ±%.1e", rail.Radius, w.radius, tol)
	}
	// (No separate "plane through C" check: |（C−rail.Center)·n̂| ≤ ‖C−rail.Center‖ by Cauchy–Schwarz
	// with n̂ unit, so the centre-distance check above already bounds it — it could never fail
	// independently. The rail plane's orientation is pinned by the endpoint + sweep checks below, and the
	// weld's exact-G1 seat is covered independently by TestCurvedRailG1_B3.)
	assertRailEnds(t, rail, wantEnds, tol)
	if s := stdmath.Abs(rail.SweepAngle); stdmath.Abs(s-wantSubtense) > 1e-4 {
		t.Fatalf("rail subtense = %.6f rad, want %.6f ±1e-4", s, wantSubtense)
	}
}

// assertRailEnds checks the rail's two endpoints match the certified host-tangent pair (either order).
func assertRailEnds(t *testing.T, rail geom.Arc3d, want [2]math.Point3, tol float64) {
	t.Helper()
	p0, p1 := rail.PointAt(0), rail.PointAt(1)
	forward := p0.DistanceTo(want[0]) <= tol && p1.DistanceTo(want[1]) <= tol
	reverse := p0.DistanceTo(want[1]) <= tol && p1.DistanceTo(want[0]) <= tol
	if !forward && !reverse {
		t.Fatalf("rail endpoints (%v,%v) match neither ordering of (%v,%v) within %.1e", p0, p1, want[0], want[1], tol)
	}
}

// TestCurvedRailG1_B3 certifies the exact-G1 weld: along every arm's rail the arm normal (canal
// identity (P−m)/r) and the sphere normal (P−C)/r must coincide within res.Weld(). The NEGATIVE
// case proves the assertion bites — offsetting the arm surface centre by 0.1·r moves its moving
// ball-centre m off C, so ‖n_arm−n_sphere‖=‖C−m‖/r≈0.1 ≫ res.Weld(), and curvedRailG1 must reject.
func TestCurvedRailG1_B3(t *testing.T) {
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner")
	}
	for _, a := range w.arms {
		rail, ok := curvedSetbackRail(w, a)
		if !ok {
			t.Fatalf("curvedSetbackRail declined a certified arm (%T)", a.arm)
		}
		if !curvedRailG1(a.arm, rail, w.center, w.radius, res) {
			t.Fatalf("curvedRailG1 failed on a certified arm (%T) — normals should coincide exactly", a.arm)
		}
	}
	assertRailG1Bites(t, w, res)
}

// assertRailG1Bites is the mutation: rebuild the vertical cyl arm with its axis offset 0.1·r in x,
// keep the true rail, and require curvedRailG1 to reject. It also reports the observed normal
// mismatch magnitude so the bite is quantified, not just asserted.
func assertRailG1Bites(t *testing.T, w cornerWeld, res Resolution) {
	t.Helper()
	cyl := findVerticalCylArm(t, w)
	rail, ok := curvedSetbackRail(w, cyl)
	if !ok {
		t.Fatalf("curvedSetbackRail declined the cyl arm")
	}
	off := 0.1 * w.radius
	mutated := mustCylinder(t, math.P3(10+off, b3CornerCY, 0), math.V3(0, 0, 1), 10)
	if curvedRailG1(mutated, rail, w.center, w.radius, res) {
		t.Fatalf("curvedRailG1 accepted an arm centre offset 0.1·r (the G1 assertion did not bite)")
	}
	t.Logf("G1 mutation (0.1·r=%.3f offset): observed normal mismatch = %.6f (tol res.Weld()=%.3e)",
		off, observedRailMismatch(mutated, rail, w.center, w.radius), res.Weld())
}

// findVerticalCylArm returns the W∧N vertical cylinder arm (ẑ axis) from the solved corner.
func findVerticalCylArm(t *testing.T, w cornerWeld) armSetback {
	t.Helper()
	for _, a := range w.arms {
		if s, ok := a.arm.(geom.Cylinder); ok && stdmath.Abs(float64(s.AxisDir.Z())-1) < 0.5 {
			return a
		}
	}
	t.Fatalf("no vertical cyl arm in the solved corner")
	return armSetback{}
}

// observedRailMismatch is the max over rail samples of ‖(P−m)/r − (P−C)/r‖ = ‖C−m‖/r, the exact
// G1 error the derivation names — used only to report the mutation's bite magnitude.
func observedRailMismatch(arm geom.Surface, rail geom.Arc3d, center math.Point3, r float64) float64 {
	worst := 0.0
	for i := 0; i < 5; i++ {
		p := rail.PointAt(float64(i) / 4)
		m, ok := armBallCenter(arm, p)
		if !ok {
			continue
		}
		d := m.VectorTo(p).Scale(1 / r).Sub(center.VectorTo(p).Scale(1 / r)).Length()
		worst = stdmath.Max(worst, d)
	}
	return worst
}

// b3TorusArm returns the B3 torus arm surface from the solved corner (the W∧K rolling-ball torus).
func b3TorusArm(t *testing.T, w cornerWeld) geom.Torus {
	t.Helper()
	for _, a := range w.arms {
		if tor, ok := a.arm.(geom.Torus); ok {
			return tor
		}
	}
	t.Fatalf("no torus arm in the solved corner")
	return geom.Torus{}
}

// TestCurvedHostArc_B3 drives the circular host-rail emitter (T5.3 §B.1/B.2): the torus arm carves a
// circle of radius R=50 in the wall (spine plane z=90) and a circle of radius R−r=40 in the cap
// (plane z=100), each sweeping azimuth 0°→−75.522° from the y=0 cut to the sphere-side host-tangent
// point. This is the rail the straight-pull transformFace cannot represent.
func TestCurvedHostArc_B3(t *testing.T) {
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner")
	}
	tor := b3TorusArm(t, w)
	tW, tK, _ := b3TangentPoints()
	wall := mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 50)
	capPl := mustPlane(t, math.P3(0, 0, 100), math.V3(0, 0, 1))
	assertHostArc(t, mustHostArc(t, wall, tor, w, res), math.P3(0, 0, 90), 50, math.P3(50, 0, 90), tW)
	assertHostArc(t, mustHostArc(t, capPl, tor, w, res), math.P3(0, 0, 100), 40, math.P3(40, 0, 100), tK)
}

// TestFilletEdges_B3CurvedArmWeld is the T5.4 integration gate: rounding B3's three axis-aligned
// Plane∧Cylinder picks in ONE op must now assemble the nine result faces into a WATERTIGHT solid
// (§B.5) whose per-type faces reproduce the oracle. It is never gated on IsSolid alone — a wrong-sign
// arm welds inside-out and still passes IsSolid — so it pairs closure with Validate.HolesContained AND
// a per-surface-type tessellated-area faithfulness check: exactly one torus ≈960.008, one sphere
// ≈182.348, and the two cylinder arm faces ≈1641.13 and ≈608.367.
func TestFilletEdges_B3CurvedArmWeld(t *testing.T) {
	body, err := filletedCorpusEdges(t, "simple/B3", 10)
	if err != nil {
		t.Fatalf("B3 curved-arm weld errored (want a solid): %v", err)
	}
	if body == nil || !body.IsSolid() {
		t.Fatalf("B3 curved-arm weld is not a solid (IsSolid=false)")
	}
	rep := Validate(body)
	if !rep.Valid || !rep.HolesContained {
		t.Fatalf("B3 weld invalid: Valid=%v HolesContained=%v issues=%v", rep.Valid, rep.HolesContained, rep.Issues)
	}
	assertB3FaithfulFaces(t, body)
}

// assertB3FaithfulFaces checks each result face type reproduces its oracle-closed area (§B.5) via the
// TESSELLATED area — the only faithfulness that also catches an inside-out or mis-trimmed analytic face.
func assertB3FaithfulFaces(t *testing.T, body *topo.Body) {
	t.Helper()
	if n := countSurfaceFacesNear[geom.Torus](body, 960.008, 5); n != 1 {
		t.Fatalf("torus arm faces ≈960.008: got %d, want 1", n)
	}
	if n := countSurfaceFacesNear[geom.Sphere](body, 182.348, 2); n != 1 {
		t.Fatalf("sphere corner faces ≈182.348: got %d, want 1", n)
	}
	if n := countSurfaceFacesNear[geom.Cylinder](body, 1641.13, 5); n != 1 {
		t.Fatalf("cyl arm faces ≈1641.13: got %d, want 1", n)
	}
	if n := countSurfaceFacesNear[geom.Cylinder](body, 608.367, 5); n != 1 {
		t.Fatalf("cyl arm faces ≈608.367: got %d, want 1", n)
	}
}

// b3OracleVolume is the DRAWEXE vprops mass (density 1) of B3 — `pcylinder s 50 100 90;
// trotate 270; blend result s 10 s_1 10 s_4 10 s_5` — the self-contained OCCT
// tests/blend/simple/B3 script (checkprops -s 20559.5; vprops mass 190761).
const b3OracleVolume = 190761.0

// TestFilletEdges_B3VolumeRegression is the load-bearing WRONG-SIGN guard the T5.4 review identified as
// missing: the tessellated-AREA faithfulness check (assertB3FaithfulFaces) is orientation-BLIND — an
// inside-out (reversed-winding) face preserves its area magnitude and still passes IsSolid. Only the
// divergence-theorem VOLUME is orientation-sensitive, so a wrong-sign weld is caught here and nowhere
// else. The mutation (assertInsideOutTorusFailsVolume) proves the gate BITES: reversing only the torus
// arm face drops the volume ~31%, far outside the 1% band, while its area and IsSolid are unchanged.
func TestFilletEdges_B3VolumeRegression(t *testing.T) {
	body, err := filletedCorpusEdges(t, "simple/B3", 10)
	if err != nil {
		t.Fatalf("B3 curved-arm weld errored (want a solid): %v", err)
	}
	if body == nil || !body.IsSolid() {
		t.Fatalf("B3 curved-arm weld is not a solid (IsSolid=false)")
	}
	if rep := Validate(body); !rep.Valid || !rep.HolesContained {
		t.Fatalf("B3 weld invalid: Valid=%v HolesContained=%v issues=%v", rep.Valid, rep.HolesContained, rep.Issues)
	}
	mesh, _ := TessellateBody(body, DefaultQuality())
	if rel := relErrVol(bodyVolume(mesh), b3OracleVolume); rel > 0.01 {
		t.Fatalf("B3 tessellated volume %.2f, want %.1f ±1%% (rel %.4f) — a wrong-sign/mis-trimmed weld",
			bodyVolume(mesh), b3OracleVolume, rel)
	}
	assertInsideOutTorusFailsVolume(t, body)
}

// assertInsideOutTorusFailsVolume rebuilds the body mesh with ONLY the torus arm face wound inside-out
// and asserts the tessellated volume then leaves the 1% band — the proof that the volume gate catches
// an orientation defect the area/IsSolid gates miss (T5.4 review's "orientation-blind area" finding).
func assertInsideOutTorusFailsVolume(t *testing.T, body *topo.Body) {
	t.Helper()
	base, flipped := torusFlippedVolumes(body)
	if relErrVol(base, b3OracleVolume) > 0.01 {
		t.Fatalf("B3 base volume %.2f drifted from %.1f (harness mesh error)", base, b3OracleVolume)
	}
	if relErrVol(flipped, b3OracleVolume) <= 0.01 {
		t.Fatalf("inside-out torus volume %.2f still within 1%% of %.1f — the volume guard does not bite",
			flipped, b3OracleVolume)
	}
}

// torusFlippedVolumes returns the body's divergence-theorem volume as tessellated, and again with the
// torus arm face's triangle winding reversed (the inside-out mutation). It merges per-face meshes so a
// single face can be flipped in isolation — TessellateBody would hide it behind a whole-body merge.
func torusFlippedVolumes(body *topo.Body) (base, flipped float64) {
	merged, mutated := &Mesh{}, &Mesh{}
	for _, f := range body.Faces() {
		fm := TessellateFace(f, DefaultQuality())
		mergeMesh(merged, fm)
		if _, isTorus := f.Geometry().(geom.Torus); isTorus {
			mergeMesh(mutated, reversedWinding(fm))
			continue
		}
		mergeMesh(mutated, fm)
	}
	return bodyVolume(merged), bodyVolume(mutated)
}

// reversedWinding returns a copy of m with every triangle's winding reversed (b,c swapped) — an
// inside-out face. Positions/Normals are shared; only the index copy is mutated.
func reversedWinding(m *Mesh) *Mesh {
	idx := append([]int(nil), m.Indices...)
	for i := 0; i+2 < len(idx); i += 3 {
		idx[i+1], idx[i+2] = idx[i+2], idx[i+1]
	}
	return &Mesh{Positions: m.Positions, Normals: m.Normals, Indices: idx}
}

// relErrVol is the relative error |got−want|/|want| used by the volume gate (want ≠ 0 here).
func relErrVol(got, want float64) float64 { return stdmath.Abs(got-want) / stdmath.Abs(want) }

// TestFilletEdges_O1WeldsIntoASolid replaces the clean-DECLINE pin this case used to carry. O1 was in the
// ADR-0050 backlog as a Gate-1 concave-corner decliner ("edge borders a curved cylinder face not yet
// supported"), and the pin existed to prove the whole op reached the do-no-harm floor rather than shipping a
// partial solid. Slice 2 of the general corner-weld layer WELDS it (cornerweld_class_o1.go): a boss cylinder
// fused to a protruding box, filleted r=5 on the three edges at (80,10,90) — two CONCAVE arms terminating at
// the corner plus one CONVEX planar band running past it, closed by the rolling-ball canal of a ball riding
// the boss wall at R+r and rolling on the band's tube at 2r.
//
// So the assertion inverts: O1 must now produce a solid, and the per-case gate in
// model/feature/occtparity/o1_cornerweld_layer_test.go is what pins its 12 faces, its 65104.9 area and its
// twelve per-face reconciliations against DRAWEXE. Kept here (rather than deleted) because the decline path
// this case used to exercise is still the floor for every corner the ladder does NOT recognise, and a
// regression that re-declines O1 should read as "the O1 builder stopped recognising its class", not as a
// silently-removed test.
//
// TestFilletEdges_M5DeclinesCleanly keeps that floor pinned on a case the ladder still does NOT recognise:
// M5 is the concave-BORE (roll-sense regime R3, R−r) trihedral corner of the same Gate-1 cluster, tracked for
// a later slice. It must honest-reject — never panic, never a partial or wrong-sign solid — and it doubles as
// the guard that adding the O1 builder did not make a DIFFERENT concave corner accept wrongly (the
// class-disjointness matrix in fillet_curved_mixed_o1_test.go proves that on role signatures; this proves it
// end to end on a real body).
func TestFilletEdges_M5DeclinesCleanly(t *testing.T) {
	assertCurvedCornerDeclinesCleanly(t, "simple/M5", 5)
}

// N1 USED to be pinned alongside it as the sibling decliner: its wall is a CONCAVE bore (radius 20, material
// OUTSIDE) and the pre-M5 convex-external-only arm builder / planar corner solver placed the corner at
// R−r (INSIDE the bore, wrong material side), so the station gate correctly declined. The
// corner-blend-weld R+r bore-wall foundation (Pieces 1+2) now solves N1 at R+r and welds it into the
// watertight 11-face solid the DRAWEXE oracle expects (area 58091.9), so N1 moved to the green gate
// (TestOCCTBlendSimple/N1 + the N1 fingerprint pin); it is no longer a clean-decline case.
func TestFilletEdges_O1WeldsIntoASolid(t *testing.T) {
	body, err := filletedCorpusEdges(t, "simple/O1", 5)
	if err != nil {
		t.Fatalf("simple/O1: FilletEdges declined (%v) — the O1 class builder no longer recognises its corner", err)
	}
	if body == nil {
		t.Fatal("simple/O1: FilletEdges returned no error and no body")
	}
	if got := len(body.Faces()); got != 12 {
		t.Fatalf("simple/O1 welded %d faces, want the DRAWEXE oracle's 12", got)
	}
}

// assertCurvedCornerDeclinesCleanly requires FilletEdges to honest-reject rel at radius r: a non-nil
// error, a nil body (no partial solid shipped), and no panic — the do-no-harm floor.
func assertCurvedCornerDeclinesCleanly(t *testing.T, rel string, r float64) {
	t.Helper()
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("%s curved-arm fillet PANICKED (do-no-harm floor breached): %v", rel, p)
		}
	}()
	body, err := filletedCorpusEdges(t, rel, r)
	if err == nil {
		t.Fatalf("%s: FilletEdges returned no error — an unsupported corner must honest-reject, not ship a solid", rel)
	}
	if body != nil {
		t.Fatalf("%s: FilletEdges returned a non-nil body alongside the decline error (partial solid): %v", rel, err)
	}
}

// TestFilletEdges_B3UnconsumedPickDeclines is the I-2 do-no-harm regression (M5 Slice A whole-branch
// review): the curved weld consumes ONLY the arms meeting the shared trihedral vertex, so a pick that
// sits outside that corner must make the WHOLE op decline — never a solid with the extra edge left
// unrounded. The fixture is B3's three certified corner picks PLUS B3's bottom radial segment
// (mid (25,0,0)), a planar edge that shares no vertex with the corner, so it reaches the weld and would
// (pre-guard) be silently dropped into an otherwise-valid returned solid. Asserts a clean decline
// (non-nil error, nil body, no panic). This FAILS against the pre-guard code (which returned a solid).
func TestFilletEdges_B3UnconsumedPickDeclines(t *testing.T) {
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("B3 + unconsumed pick PANICKED (do-no-harm floor breached): %v", p)
		}
	}()
	body, err := b3PicksPlusExtra(t, math.P3(25, 0, 0)) // bottom radial segment, disjoint from the top corner
	if err == nil {
		t.Fatalf("FilletEdges returned no error — an unconsumed pick must decline, not ship a solid with it unrounded")
	}
	if body != nil {
		t.Fatalf("FilletEdges returned a non-nil body alongside the decline error (partial solid): %v", err)
	}
}

// b3PicksPlusExtra fillets B3's three certified curved-corner picks plus one extra edge located at
// `extra`, all in one op — the fixture for the unconsumed-pick decline (I-2). It reuses the corpus
// locator (curvedArmEdgeAt) so the extra edge is resolved by geometric midpoint like the corner picks.
func b3PicksPlusExtra(t *testing.T, extra math.Point3) (*topo.Body, error) {
	t.Helper()
	b := importCorpusSolid(t, "simple/B3")
	mids := append(append([]math.Point3(nil), curvedArmCorpusPicks["simple/B3"]...), extra)
	keys := make([][]byte, 0, len(mids))
	for _, m := range mids {
		e := curvedArmEdgeAt(b, m)
		if e == nil {
			t.Fatalf("b3PicksPlusExtra: edge near %v not found", m)
		}
		keys = append(keys, e.ReferenceKey())
	}
	return FilletEdges(b, keys, 10)
}

// TestCurvedWeldFaceProvenance is the ADR-0043 provenance regression (Important #4): a trimmed arm face
// must carry its GENERATING filleted edge's lineage as parent (via filletEdgeProvenance — the same helper
// the planar cyl faces use), so the blend's edges/faces get a stable topological name that survives an
// upstream edit instead of a build-order name that renumbers. The corner SPHERE face, generated by the
// trihedral VERTEX (not a single edge), carries no single-edge parent by design (mirrors spherePatchFace).
func TestCurvedWeldFaceProvenance(t *testing.T) {
	ef := armFilletWithLineage(t, topo.NewLineage(topo.Tok("test", "armedge", 7)))
	rails := armRails{segs: []endSeg{
		{from: math.P3(0, 0, 0), to: math.P3(1, 0, 0)},
		{from: math.P3(1, 0, 0), to: math.P3(1, 1, 0)},
		{from: math.P3(1, 1, 0), to: math.P3(0, 0, 0)},
	}}
	ff := curvedArmTrimmedFace(rails, ef)
	if len(ff.parent.Tokens()) == 0 {
		t.Fatalf("arm face carries an empty (build-order) lineage — ADR-0043 provenance not set")
	}
	if want := filletEdgeProvenance(ef.edge); ff.parent.KeyString() != want.KeyString() {
		t.Fatalf("arm face parent = %q, want the generating edge's provenance %q", ff.parent.KeyString(), want.KeyString())
	}
	assertSphereFaceHasNoSingleEdgeParent(t)
}

// armFilletWithLineage builds an edgeFillet whose edge carries lin and whose arm surface is a torus —
// the minimum curvedArmTrimmedFace reads to derive its ADR-0043 provenance.
func armFilletWithLineage(t *testing.T, lin topo.Lineage) edgeFillet {
	t.Helper()
	bld := topo.NewBuilder(true, topo.Lineage{})
	v0, v1 := bld.AddVertex(math.P3(0, 0, 0), lin), bld.AddVertex(math.P3(1, 0, 0), lin)
	e := bld.AddEdge(geom.NewLineSegment(v0.Point(), v1.Point()), v0, v1, lin)
	tor, err := geom.NewTorusWithRef(math.P3(0, 0, 90), math.V3(0, 0, 1), math.V3(1, 0, 0), 40, 10)
	if err != nil {
		t.Fatalf("build torus arm: %v", err)
	}
	return edgeFillet{edge: e, armSurface: tor}
}

// assertSphereFaceHasNoSingleEdgeParent certifies the corner sphere face is vertex-generated: it must
// carry an empty parent lineage (no filletEdgeProvenance edge-name applies), matching spherePatchFace.
func assertSphereFaceHasNoSingleEdgeParent(t *testing.T) {
	t.Helper()
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner")
	}
	sf, ok := curvedSphereFace(w, sphere)
	if !ok {
		t.Fatalf("curvedSphereFace declined the certified B3 corner")
	}
	if n := len(sf.parent.Tokens()); n != 0 {
		t.Fatalf("sphere face carries a %d-token parent, want none (the corner sphere is vertex-generated)", n)
	}
}

// TestCurvedCornerFace_B3ByteIdentical, assertFilletFaceIdentical and assertLoopIdentical (the ADR-2
// Step-1 strangler byte-identity golden) moved to fillet_curved_weld_assert_test.go — this file crossed
// the 500-line cap (CLAUDE.md) once this task's golden helpers landed.

// mustHostArc builds a host arc or fails the test.
func mustHostArc(t *testing.T, host geom.Surface, tor geom.Torus, w cornerWeld, res Resolution) geom.Arc3d {
	t.Helper()
	arc, ok := curvedHostArc(host, tor, w, res)
	if !ok {
		t.Fatalf("curvedHostArc declined host %T", host)
	}
	return arc
}

// assertHostArc checks a host arc against its certified circle: centre, radius, a plane normal along
// ẑ (the arc lies in a constant-z plane), the far endpoint at azimuth 0, the near endpoint at the
// host-tangent point, and the certified −75.522° sweep.
func assertHostArc(t *testing.T, arc geom.Arc3d, center math.Point3, radius float64, far, near math.Point3) {
	t.Helper()
	if d := arc.Center.DistanceTo(center); float64(d) > 1e-6 {
		t.Fatalf("arc centre %v off %v by %.3e", arc.Center, center, d)
	}
	if e := stdmath.Abs(arc.Radius - radius); e > 1e-6 {
		t.Fatalf("arc radius = %.9f, want %.9f", arc.Radius, radius)
	}
	if z := stdmath.Abs(float64(arc.Normal.Z()) - 1); z > 1e-9 {
		t.Fatalf("arc normal %v not ±ẑ (arc not in a constant-z plane)", arc.Normal)
	}
	if d := arc.PointAt(0).DistanceTo(far); float64(d) > 1e-4 {
		t.Fatalf("arc PointAt(0) = %v, want far end %v (Δ=%.3e)", arc.PointAt(0), far, d)
	}
	if d := arc.PointAt(1).DistanceTo(near); float64(d) > 1e-4 {
		t.Fatalf("arc PointAt(1) = %v, want host-tangent %v (Δ=%.3e)", arc.PointAt(1), near, d)
	}
	if s := stdmath.Abs(arc.SweepAngle) - 75.522*stdmath.Pi/180; stdmath.Abs(s) > 1e-3 {
		t.Fatalf("arc sweep = %.6f rad, want 75.522° (Δ=%.3e)", arc.SweepAngle, s)
	}
}
