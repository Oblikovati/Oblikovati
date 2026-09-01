// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CN4b-1 — the cone-ruling canal arm's FAR-RUNOUT CAP (cone-host-corner-derivation.md §4). These tests
// build from the REAL imported C2/C6/D1 ruling geometry and pin: the two exact host springs cross the cap
// plane at the exact feet (plane foot on the radial plane, cone foot on the cone, both ON the cap, ≤1e-9);
// the ⊥-axis cap trim lies on the cap plane with its endpoints AT the feet; the D1 snout is the exact
// vertex circle (centre (0,−10,70), r=10 in {x=0}, ends (0,0,70)/(0,−250/13,120−600/13)) with the ⊥ guard
// mutation-proven; obliqueRunout welds the three coincident identities; and the far-capped arm face meshes
// FOLD-FREE. It greens no corpus case (the corner weld is CN4b-2).

const coneFarExactTol = 1e-9

// coneFarFixture is a rim/snout fixture and its DRAWEXE-pinned corner centre (the ruling whose canal spine
// passes through C is the corner's arm).
type coneFarFixture struct {
	name   string
	center math.Point3
	snout  bool
}

func coneFarFixtures() []coneFarFixture {
	return []coneFarFixture{
		{"C2", math.P3(75.4660749146, -10, 10), false},
		{"C6", math.P3(-10, -31.2304660433, 140), false},
		{"D1", math.P3(33.541019662496844, -10, 10), true},
	}
}

// coneFarSetup is one fixture's arm + far cap ready for the runout engine.
type coneFarSetup struct {
	ef    edgeFillet
	sp    coneCanalSpine
	cap   geom.Plane
	far   *topo.Vertex
	weld  cornerWeld
	res   Resolution
	snout bool
}

// setupConeFar imports a fixture, selects the corner's ruling arm (spine through C), and finds its single
// non-host far cap plane (z-plane ⊥ axis for C2/C6; the far radial plane for the D1 snout).
func setupConeFar(t *testing.T, fx coneFarFixture) coneFarSetup {
	t.Helper()
	body := importSimpleFixture(t, fx.name)
	res := ResolutionForBody(body)
	ef := findCornerRulingFillet(t, body, fx.center, res)
	far := farEndVertex(ef.edge, fx.center)
	cap, ok := coneFarCapPlane(ef, far)
	if !ok {
		t.Fatalf("%s: no single non-host far cap plane at the far vertex", fx.name)
	}
	return coneFarSetup{ef: ef, sp: *ef.armCanalSpine, cap: cap, far: far,
		weld: cornerWeld{center: fx.center, radius: coneArmR}, res: res, snout: fx.snout}
}

// findCornerRulingFillet returns the canal-arm edgeFillet of the corner's ruling — the Cone∧Plane ruling
// whose exact hyperbola spine passes through the corner centre C (stationOf succeeds).
func findCornerRulingFillet(t *testing.T, body *topo.Body, c math.Point3, res Resolution) edgeFillet {
	t.Helper()
	for _, e := range body.Edges() {
		co, pl, _, _, ok := conePlaneEdge(e)
		if !ok || classifyConeArm(co, pl, coneRadiusAt(co, edgeMidpoint(e)), res) != coneClassRuling {
			continue
		}
		ef, handled, err := coneArmEdge(body, e, filletPick{edge: e, r0: coneArmR, r1: coneArmR})
		if !handled || err != nil || ef.armCanalSpine == nil {
			continue
		}
		if _, ok := ef.armCanalSpine.stationOf(c, res.Size(), res.Weld()); ok {
			return ef
		}
	}
	t.Fatalf("no corner ruling arm whose spine passes through %v", c)
	return edgeFillet{}
}

// coneFarCapPlane returns the UNIQUE non-host plane at the far vertex — the cap that closes the arm.
func coneFarCapPlane(ef edgeFillet, far *topo.Vertex) (geom.Plane, bool) {
	var found geom.Plane
	n := 0
	for _, f := range facesAround(far) {
		if f.ID() == ef.a.ID() || f.ID() == ef.b.ID() {
			continue
		}
		if pl, ok := f.Geometry().(geom.Plane); ok {
			found, n = pl, n+1
		}
	}
	return found, n == 1
}

// TestConeCanalSprings_ExactFeet pins the two exact host springs and their cap crossings: springCapFoot
// lands the plane foot on the radial plane and the cone foot on the cone, BOTH on the cap plane, each at
// ball distance r from its exact spine centre — all to ≤1e-9 (closed form, matching CN2's spine tests).
func TestConeCanalSprings_ExactFeet(t *testing.T) {
	t.Parallel()
	for _, fx := range coneFarFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			s := setupConeFar(t, fx)
			springs, ok := armSprings(s.ef, s.ef.a.Geometry(), s.ef.b.Geometry(), coneArmR)
			if !ok {
				t.Fatal("armSprings declined the canal arm")
			}
			sA, sB := springsForHosts(s.ef, springs)
			near := s.far.Point()
			footA, okA := springCapFoot(sA, s.cap, near, s.res)
			footB, okB := springCapFoot(sB, s.cap, near, s.res)
			if !okA || !okB {
				t.Fatalf("springCapFoot declined (A=%v B=%v)", okA, okB)
			}
			assertFeetExact(t, s, footA, footB)
		})
	}
}

// assertFeetExact checks each foot is on the cap plane and on its host at ball radius r (≤1e-9). footA is
// on ef.a, footB on ef.b; the plane host's foot lies on the radial plane, the cone host's on the cone.
func assertFeetExact(t *testing.T, s coneFarSetup, footA, footB math.Point3) {
	t.Helper()
	assertOnCapPlane(t, "footA", s.cap, footA)
	assertOnCapPlane(t, "footB", s.cap, footB)
	planeFoot, coneFoot := footA, footB
	if _, aIsPlane := s.ef.a.Geometry().(geom.Plane); !aIsPlane {
		planeFoot, coneFoot = footB, footA
	}
	if d := stdmath.Abs(float64(s.sp.apex.VectorTo(planeFoot).Dot(s.sp.nOut))); d > coneFarExactTol {
		t.Fatalf("plane foot %v off the radial plane by %g", planeFoot, d)
	}
	assertFootOnConeIncident(t, s.sp, coneFoot)
	xf := float64(s.sp.apex.VectorTo(planeFoot).Dot(s.sp.ePerp))
	if d := stdmath.Abs(float64(planeFoot.DistanceTo(s.sp.center(xf))) - coneArmR); d > coneFarExactTol {
		t.Fatalf("plane foot %v is %g off radius r from its spine centre", planeFoot, d)
	}
}

// assertOnCapPlane checks p lies on the cap plane to ≤1e-9.
func assertOnCapPlane(t *testing.T, what string, cap geom.Plane, p math.Point3) {
	t.Helper()
	if d := stdmath.Abs(float64(cap.Origin.VectorTo(p).Dot(cap.Normal()))); d > coneFarExactTol {
		t.Fatalf("%s %v is %g off the cap plane", what, p, d)
	}
}

// assertFootOnConeIncident checks the cone foot lies ON the host cone: the exact signed cone distance
// (w·â)·sinα − |w⊥|·cosα is 0 within 1e-9. (The |T−m|=r envelope identity is separately pinned by CN2's
// TestConeCanalSpine_ExactFeet and holds by construction of coneFoot.)
func assertFootOnConeIncident(t *testing.T, sp coneCanalSpine, cT math.Point3) {
	t.Helper()
	w := sp.apex.VectorTo(cT)
	axial := float64(w.Dot(sp.axis))
	perp := float64(w.Sub(sp.axis.Scale(axial)).Length())
	if d := axial*sp.sinA - perp*sp.cosA; stdmath.Abs(d) > coneFarExactTol {
		t.Fatalf("cone foot %v off the host cone by %g", cT, d)
	}
}

// TestConeCanalCapTrim_OnPlane pins the ⊥-axis cap trim (C2/C6): every sampled point lies on the cap plane
// (≤ res.Weld·r) and the trim's endpoints ARE the two feet (the shared-edge identity).
func TestConeCanalCapTrim_OnPlane(t *testing.T) {
	t.Parallel()
	for _, fx := range coneFarFixtures() {
		if fx.snout {
			continue
		}
		t.Run(fx.name, func(t *testing.T) {
			s := setupConeFar(t, fx)
			feet := canalFeet(t, s)
			trim, ok := intersectArmCapping(s.ef, s.cap, feet, coneArmR, s.res)
			if !ok {
				t.Fatal("intersectArmCapping declined the ⊥-axis cap trim")
			}
			tol := s.res.Weld() * coneArmR
			for i := 0; i <= 40; i++ {
				p := trim.PointAt(float64(i) / 40)
				if d := stdmath.Abs(float64(s.cap.Origin.VectorTo(p).Dot(s.cap.Normal()))); d > tol {
					t.Fatalf("trim sample %d %v is %g off the cap plane (tol %g)", i, p, d, tol)
				}
			}
			assertEndsAtFeet(t, trim, feet)
		})
	}
}

// canalFeet returns the two runout feet ordered (ef.a, ef.b) via the engine's own armRunoutFeet.
func canalFeet(t *testing.T, s coneFarSetup) [2]math.Point3 {
	t.Helper()
	near := s.far.Point()
	feet, ok, reason := armRunoutFeet(s.ef, s.cap, near, near, coneArmR, s.res)
	if !ok {
		t.Fatalf("armRunoutFeet declined: %s", reason)
	}
	return feet
}

// assertEndsAtFeet checks the trim's endpoints coincide with feet[0]/feet[1] to ≤1e-9.
func assertEndsAtFeet(t *testing.T, trim geom.Curve3, feet [2]math.Point3) {
	t.Helper()
	lo, hi := trim.Domain()
	if d := float64(trim.PointAt(lo).DistanceTo(feet[0])); d > coneFarExactTol {
		t.Fatalf("trim start %v != foot[0] %v (%g)", trim.PointAt(lo), feet[0], d)
	}
	if d := float64(trim.PointAt(hi).DistanceTo(feet[1])); d > coneFarExactTol {
		t.Fatalf("trim end %v != foot[1] %v (%g)", trim.PointAt(hi), feet[1], d)
	}
}

// TestConeCanalSnout_Exact pins D1's snout cap: the terminal characteristic arc is the exact circle centre
// (0,−10,70) r=10 in {x=0}, with ends bit-exactly on the axis edge (0,0,70) and the far ruling foot
// T*=(0,−250/13,120−600/13). The ⊥ guard is mutation-proven: perturbing the far-plane normal off the
// vertex spine tangent makes the snout honest-reject.
func TestConeCanalSnout_Exact(t *testing.T) {
	t.Parallel()
	s := setupConeFar(t, coneFarFixtures()[2]) // D1
	feet := canalFeet(t, s)
	trim, ok := intersectArmCapping(s.ef, s.cap, feet, coneArmR, s.res)
	if !ok {
		t.Fatal("intersectArmCapping declined the D1 snout")
	}
	arc, ok := trim.(geom.Arc3d)
	if !ok {
		t.Fatalf("snout trim is %T, want geom.Arc3d", trim)
	}
	assertSnoutCircle(t, arc)
	assertSnoutEnds(t, arc)
	assertSnoutGuardRejectsTilt(t, s, feet)
	assertSnoutGuardRejectsOffset(t, s, feet)
}

// assertSnoutCircle checks the snout arc's circle: centre (0,−10,70), radius 10, in the plane {x=0}.
func assertSnoutCircle(t *testing.T, arc geom.Arc3d) {
	t.Helper()
	if d := float64(arc.Center.DistanceTo(math.P3(0, -10, 70))); d > coneFarExactTol {
		t.Fatalf("snout centre %v != (0,−10,70) by %g", arc.Center, d)
	}
	if stdmath.Abs(arc.Radius-10) > coneFarExactTol {
		t.Fatalf("snout radius %g != 10", arc.Radius)
	}
	if d := 1 - stdmath.Abs(float64(arc.Normal.AsVector().Dot(math.V3(1, 0, 0)))); d > coneFarExactTol {
		t.Fatalf("snout circle not in {x=0}: |n·x̂| off by %g", d)
	}
}

// assertSnoutEnds checks the arc runs bit-exactly between the axis edge and the far ruling foot.
func assertSnoutEnds(t *testing.T, arc geom.Arc3d) {
	t.Helper()
	tStar := math.P3(0, -250.0/13.0, 120-600.0/13.0)
	ends := [2]math.Point3{arc.PointAt(0), arc.PointAt(1)}
	axis, ruling := math.P3(0, 0, 70), tStar
	okFwd := ends[0].DistanceTo(axis) <= coneFarExactTol && ends[1].DistanceTo(ruling) <= coneFarExactTol
	okRev := ends[0].DistanceTo(ruling) <= coneFarExactTol && ends[1].DistanceTo(axis) <= coneFarExactTol
	if !okFwd && !okRev {
		t.Fatalf("snout ends %v/%v are not {(0,0,70), %v}", ends[0], ends[1], tStar)
	}
}

// assertSnoutGuardRejectsTilt is the condition-(a) mutation witness: a far radial plane whose normal is
// tilted off the vertex spine tangent (a non-90° wedge) makes canalCappingTrim honest-reject the snout,
// with the reason naming condition (a).
func assertSnoutGuardRejectsTilt(t *testing.T, s coneFarSetup, feet [2]math.Point3) {
	t.Helper()
	tilted, err := geom.NewPlane(s.cap.Origin, math.V3(1, 0.3, 0))
	if err != nil {
		t.Fatalf("tilted plane: %v", err)
	}
	_, ok, reason := canalCappingTrim(s.sp, tilted, feet, coneArmR, s.res)
	if ok || !strings.Contains(reason, "cond (a)") {
		t.Fatalf("snout guard cond (a) miss: ok=%v reason=%q (want reject naming cond (a))", ok, reason)
	}
}

// assertSnoutGuardRejectsOffset is the condition-(b) mutation witness: a radial plane PARALLEL to the
// vertex tangent (normal = ê) but OFFSET so it does not contain the vertex circle centre (an {x=c≠0}
// plane) must honest-reject, with the reason naming condition (b) and its incidence.
func assertSnoutGuardRejectsOffset(t *testing.T, s coneFarSetup, feet [2]math.Point3) {
	t.Helper()
	offset, err := geom.NewPlane(s.cap.Origin.TranslateBy(s.sp.ePerp.Scale(5)), s.sp.ePerp)
	if err != nil {
		t.Fatalf("offset plane: %v", err)
	}
	_, ok, reason := canalCappingTrim(s.sp, offset, feet, coneArmR, s.res)
	if ok || !strings.Contains(reason, "cond (b)") {
		t.Fatalf("snout guard cond (b) miss: ok=%v reason=%q (want reject naming cond (b))", ok, reason)
	}
}

// TestConeCanalReject_CarriesValues proves the canal declines carry the offending values (CLAUDE.md
// "exception messages must include the offending value"): a cap BELOW the hyperbola vertex names ρ and r,
// and an obliquely-posed cap names |n̂·â|.
func TestConeCanalReject_CarriesValues(t *testing.T) {
	t.Parallel()
	s := setupConeFar(t, coneFarFixtures()[2]) // D1
	spring := coneCanalSpring{spine: s.sp, lo: 0, hi: 40, onCone: false}
	below, err := geom.NewPlane(s.sp.apex.TranslateBy(s.sp.axis.Scale(40)), s.sp.axis)
	if err != nil {
		t.Fatalf("below-vertex cap: %v", err)
	}
	if _, ok, reason := spring.canalCapFoot(below, s.far.Point(), s.res); ok ||
		!strings.Contains(reason, "ρ=") || !strings.Contains(reason, "r=") {
		t.Fatalf("ρ<r reject must carry ρ and r: ok=%v reason=%q", ok, reason)
	}
	oblique, err := geom.NewPlane(s.cap.Origin, math.V3(1, 0, 1))
	if err != nil {
		t.Fatalf("oblique cap: %v", err)
	}
	if _, ok, reason := spring.canalCapFoot(oblique, s.far.Point(), s.res); ok ||
		!strings.Contains(reason, "|n̂·â|=") {
		t.Fatalf("oblique-cap reject must carry |n̂·â|: ok=%v reason=%q", ok, reason)
	}
}

// TestConeCanalFarRunout_Identities is the ADR-4 gate for the canal arm: the far runout builds the analytic
// cap trim through the authoritative feet and re-terminates BOTH host rails on those same feet —
// trim.endpoints == feet == rail outer ends. C2/C6 route end-to-end through armFarRunout (its ⊥-axis cap
// is transverse to the ruling); the D1 SNOUT cap is PARALLEL to the ruling tangent (the general
// capping-face finder cannot classify a tangent-parallel cap — a CN4b-2 routing concern), so it is driven
// through obliqueRunout directly with the far radial cap. Both must return regime=runoutOblique.
func TestConeCanalFarRunout_Identities(t *testing.T) {
	t.Parallel()
	for _, fx := range coneFarFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			s := setupConeFar(t, fx)
			feet := canalFeet(t, s)
			h0 := straightRail(feet[0], s.weld.center)
			h1 := straightRail(feet[1], s.weld.center)
			h0p, h1p, run, ok, reason := runConeFar(t, s, h0, h1)
			if !ok {
				t.Fatalf("%s far runout declined: %s", fx.name, reason)
			}
			if run.regime != runoutOblique {
				t.Fatalf("%s regime %d, want runoutOblique", fx.name, run.regime)
			}
			assertCoincident(t, "trim.from == foot[0] == h0'.outer", run.trim.from, run.feet[0], h0p.from)
			assertCoincident(t, "trim.to   == foot[1] == h1'.outer", run.trim.to, run.feet[1], h1p.from)
		})
	}
}

// runConeFar drives the far runout: armFarRunout end-to-end for a transverse cap (C2/C6), or obliqueRunout
// directly for the snout's tangent-parallel radial cap (D1).
func runConeFar(t *testing.T, s coneFarSetup, h0, h1 endSeg) (endSeg, endSeg, armRunout, bool, string) {
	t.Helper()
	if s.snout {
		return obliqueRunout(s.ef, stubCapFace(t, s.cap), h0, h1, coneArmR, s.res)
	}
	return armFarRunout(s.ef, s.weld, h0, h1, map[uint64]bool{s.ef.edge.ID(): true}, s.res)
}

// straightRail is a straight incoming host rail whose OUTER end is the foot (so reterminateRail's ruling
// branch re-clips it to foot→inner with the foot trivially on its own line).
func straightRail(foot, inner math.Point3) endSeg {
	return endSeg{from: foot, to: inner}
}

// TestConeCanalFarCap_FoldFree is the highest-priority tessellation gate (CLAUDE.md): the far-capped canal
// arm face — the arm patch below its new far-cap trim — meshes to a positive area with NO fold edges
// (validate.FoldEdgeCount == 0). The far boundary is the cap trim (⊥-axis swept polyline for C2/C6, the snout arc
// for D1); the near boundary is a near iso-cut. A folded trim (the x_f-sweep defect the ψ-sweep fixes)
// would crease the mesh here.
func TestConeCanalFarCap_FoldFree(t *testing.T) {
	t.Parallel()
	for _, fx := range coneFarFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			s := setupConeFar(t, fx)
			feet := canalFeet(t, s)
			trim, ok := intersectArmCapping(s.ef, s.cap, feet, coneArmR, s.res)
			if !ok {
				t.Fatal("intersectArmCapping declined the far-cap trim")
			}
			surf, ok := s.ef.armSurface.(geom.BSplineSurface)
			if !ok {
				t.Fatalf("arm is %T, want geom.BSplineSurface", s.ef.armSurface)
			}
			m := meshFarCapFace(t, surf, trim, feet)
			if m == nil || m.TriangleCount() == 0 {
				t.Fatalf("%s far-cap face produced no mesh", fx.name)
			}
			if n := validate.FoldEdgeCount(m); n != 0 {
				t.Fatalf("%s far-cap face meshed with %d fold edges; want 0", fx.name, n)
			}
			if a := validate.MeshArea(m); a <= 0 || stdmath.IsInf(a, 0) || stdmath.IsNaN(a) {
				t.Fatalf("%s far-cap face area %g; want finite positive", fx.name, a)
			}
		})
	}
}

// meshFarCapFace builds the arm's far-cap region (the trim as the far boundary, a near iso-cut below it)
// as a UV loop on the arm surface and tessellates it through the production fold-driven trim path.
func meshFarCapFace(t *testing.T, surf geom.BSplineSurface, trim geom.Curve3, feet [2]math.Point3) *Mesh {
	t.Helper()
	uvs := invertTrimUV(t, surf, trim)
	vCut := interiorCutV(surf, uvs)
	outerUV := make([]math.Point2, 0, len(uvs)+2)
	outerUV = append(outerUV, uvs...)
	outerUV = append(outerUV, math.P2(uvs[len(uvs)-1].X, math.Scalar(vCut)), math.P2(uvs[0].X, math.Scalar(vCut)))
	outer3D := make([]math.Point3, len(outerUV))
	for i, p := range outerUV {
		outer3D[i] = surf.PointAt(float64(p.X), float64(p.Y))
	}
	su, sv := tessellate.MetricScale(surf)
	return tessellate.FoldDrivenPatch(surf, su, sv, DefaultQuality(), outer3D, outerUV, nil, nil)
}

// invertTrimUV maps the trim's 3D samples to arm (u,v) parameters by a grid search plus local refine, so
// the cap trim can bound a UV face patch. The trim rides the exact envelope within weld of the arm.
func invertTrimUV(t *testing.T, surf geom.BSplineSurface, trim geom.Curve3) []math.Point2 {
	t.Helper()
	const n = 24
	out := make([]math.Point2, n+1)
	for i := 0; i <= n; i++ {
		out[i] = invertArmUV(surf, trim.PointAt(float64(i)/n))
	}
	return out
}

// invertArmUV finds the (u,v) on the arm surface nearest p — a 60×60 coarse grid then a shrinking local
// refine. Adequate for a test-side face boundary (the mesh needs a monotone loop, not machine precision).
func invertArmUV(surf geom.BSplineSurface, p math.Point3) math.Point2 {
	u0, u1 := surf.UDomain()
	v0, v1 := surf.VDomain()
	bu, bv := gridNearestUV(surf, p, u0, u1, v0, v1, 60)
	step := 0.5 * stdmath.Max((u1-u0)/60, (v1-v0)/60)
	for range 40 {
		bu, bv = gridNearestUV(surf, p, stdmath.Max(u0, bu-step), stdmath.Min(u1, bu+step),
			stdmath.Max(v0, bv-step), stdmath.Min(v1, bv+step), 6)
		step *= 0.5
	}
	return math.P2(math.Scalar(bu), math.Scalar(bv))
}

// gridNearestUV returns the (u,v) of the closest surface grid sample to p over [ulo,uhi]×[vlo,vhi].
func gridNearestUV(surf geom.BSplineSurface, p math.Point3, ulo, uhi, vlo, vhi float64, n int) (float64, float64) {
	bu, bv, best := ulo, vlo, stdmath.Inf(1)
	for i := 0; i <= n; i++ {
		u := ulo + (uhi-ulo)*float64(i)/float64(n)
		if v, d := nearestVOnColumn(surf, p, u, vlo, vhi, n); d < best {
			bu, bv, best = u, v, d
		}
	}
	return bu, bv
}

// nearestVOnColumn returns the v (and its distance) of the closest sample to p along the u=const column.
func nearestVOnColumn(surf geom.BSplineSurface, p math.Point3, u, vlo, vhi float64, n int) (float64, float64) {
	bv, best := vlo, stdmath.Inf(1)
	for j := 0; j <= n; j++ {
		v := vlo + (vhi-vlo)*float64(j)/float64(n)
		if d := float64(surf.PointAt(u, v).DistanceTo(p)); d < best {
			bv, best = v, d
		}
	}
	return bv, best
}

// interiorCutV is the v of the near iso-cut: 30 % of the way from the trim's mean v toward the arm's
// INTERIOR (the far domain end from the trim), so the cap patch lies between the trim and the cut whether
// the cap sits at the v=1 far end (C2/C6) or the v=0 vertex end (the D1 snout, clamped to lo).
func interiorCutV(surf geom.BSplineSurface, uvs []math.Point2) float64 {
	v0, v1 := surf.VDomain()
	mean := 0.0
	for _, p := range uvs {
		mean += float64(p.Y) / float64(len(uvs))
	}
	interior := v0
	if stdmath.Abs(mean-v1) > stdmath.Abs(mean-v0) {
		interior = v1
	}
	return mean + 0.3*(interior-mean)
}
