// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// coneCornerCase is one cone-host trihedral corner from OCCT tests/blend/simple (CN3,
// cone-host-corner-derivation.md §3). It carries BOTH the exact synthetic geometry (host cone + the two
// r-offset host planes as {origin, material-outward normal} + the corner vertex) and the DRAWEXE-measured
// corner-ball CENTRE, so the same closed-form solve is pinned against the oracle from exact geometry AND
// (coneCornerImportedCentre) from the real imported STEP body. r = 10 in every case.
type coneCornerCase struct {
	name     string
	step     string
	apex     math.Point3
	tanAlpha float64
	p0Origin math.Point3  // plane-0 origin
	p0Out    math.Vector3 // plane-0 material-outward normal
	p1Origin math.Point3  // plane-1 origin
	p1Out    math.Vector3 // plane-1 material-outward normal
	vertex   math.Point3
	center   math.Point3 // DRAWEXE analytic corner-face centre, radius 10
}

// coneCornerCases are the four cone-host corners CN3 solves. Each cone opens downward (â = −ẑ); the two
// host planes are the r-offset cap/radial planes whose intersection is the offset line of §3's table.
var coneCornerCases = []coneCornerCase{
	{
		name: "C2", step: "simple/C2.step", apex: math.P3(0, 0, 270), tanAlpha: 1.0 / 3,
		p0Origin: math.P3(0, 0, 0), p0Out: math.V3(0, 0, -1),
		p1Origin: math.P3(0, 0, 0), p1Out: math.V3(0, 1, 0),
		vertex: math.P3(90, 0, 0), center: math.P3(75.4660749146, -10, 10),
	},
	{
		name: "C6", step: "simple/C6.step", apex: math.P3(0, 0, 270), tanAlpha: 1.0 / 3,
		p0Origin: math.P3(0, 0, 150), p0Out: math.V3(0, 0, 1),
		p1Origin: math.P3(0, 0, 0), p1Out: math.V3(1, 0, 0),
		vertex: math.P3(0, -40, 150), center: math.P3(-10, -31.2304660433, 140),
	},
	{
		name: "C8", step: "simple/C8.step", apex: math.P3(0, 0, 120), tanAlpha: 5.0 / 12,
		p0Origin: math.P3(0, 0, 0), p0Out: math.V3(1, 0, 0),
		p1Origin: math.P3(0, 0, 0), p1Out: math.V3(0, -1, 0),
		vertex: math.P3(0, 0, 120), center: math.P3(-10, 10, 60.0588745030),
	},
	{
		name: "D1", step: "simple/D1.step", apex: math.P3(0, 0, 120), tanAlpha: 5.0 / 12,
		p0Origin: math.P3(0, 0, 0), p0Out: math.V3(0, 0, -1),
		p1Origin: math.P3(0, 0, 0), p1Out: math.V3(0, 1, 0),
		vertex: math.P3(50, 0, 0), center: math.P3(stdmath.Sqrt(1125), -10, 10),
	},
}

// coneCornerR is the campaign fillet radius (r = 10 across the whole corpus).
const coneCornerR = 10.0

// coneCornerExactTol pins the centre solved from EXACT synthetic geometry to the DRAWEXE table (the table
// quotes ~10–12 digits). Set just above the achieved ~5e-11 real-import residual floor: 1e-10 still catches
// any sign/transcription error (which moves the centre by tens of units — root separation is ≥62 units
// everywhere) yet leaves no slack over the observed precision.
const coneCornerExactTol = 1e-10

// coneOf builds a case's host cone: apex on the axis, axis −ẑ, half-angle atan(tanα).
func coneOf(t *testing.T, c coneCornerCase) geom.Cone {
	t.Helper()
	co, err := geom.NewCone(c.apex, math.V3(0, 0, -1), stdmath.Atan(c.tanAlpha))
	if err != nil {
		t.Fatalf("%s cone: %v", c.name, err)
	}
	return co
}

// coneCornerExactFixture wires a synthetic trihedral corner from a case's EXACT geometry — the host cone
// face plus the two r-offset host planes (material-outward normals) meeting at the corner vertex — driving
// the real solveBlend→coneHostCorner→solveConeBlend path without a full topological body (solveBlend reads
// only v.Point() and the three faces' Geometry()/Reversed()).
func coneCornerExactFixture(t *testing.T, c coneCornerCase) (*topo.Vertex, []*topo.Face) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "cone-corner", 0))
	bld := topo.NewBuilder(true, lin)
	v := bld.AddVertex(c.vertex, lin)
	coneFace := bld.AddFace(coneOf(t, c), lin) // NOT reversed: material inside the cone (convex-external)
	plane0 := bld.AddFace(planeOn(t, c.p0Origin, c.p0Out), lin)
	plane1 := bld.AddFace(planeOn(t, c.p1Origin, c.p1Out), lin)
	return v, []*topo.Face{coneFace, plane0, plane1}
}

// TestConeHostCornerCentre_Exact drives solveBlend on the exact synthetic geometry of every cone-host
// corner and asserts the analytic corner-ball centre matches the DRAWEXE table (§3) to ≤1e-9 — the
// cos²α-quadratic + nappe filter + nearer-vertex pick reproducing C2/C6/C8/D1's certified centres. Also
// asserts the host set is exactly {1 cone, 2 planes} and the corner sphere radius is r.
func TestConeHostCornerCentre_Exact(t *testing.T) {
	t.Parallel()
	for _, c := range coneCornerCases {
		t.Run(c.name, func(t *testing.T) {
			v, faces := coneCornerExactFixture(t, c)
			assertConeHostSet(t, faces)
			cb, err := solveBlend(nil, v, faces, coneCornerR)
			if err != nil {
				t.Fatalf("%s: solveBlend declined the cone-host corner: %v", c.name, err)
			}
			res := float64(cb.center.DistanceTo(c.center))
			t.Logf("%s corner centre %v, oracle %v, residual %.3e", c.name, cb.center, c.center, res)
			if res > coneCornerExactTol {
				t.Fatalf("%s: corner centre %v != oracle %v (residual %.3e > %g)", c.name, cb.center, c.center, res, coneCornerExactTol)
			}
			if cb.sphere.Radius != coneCornerR {
				t.Fatalf("%s: corner sphere radius %g, want %g", c.name, cb.sphere.Radius, coneCornerR)
			}
		})
	}
}

// TestConeHostCornerCentre_Imported drives solveBlend on the REAL imported STEP body of every cone-host
// corner (the SAME import path the corpus harness uses) and asserts the analytic centre matches the
// DRAWEXE table — the imported-geometry gate the brief requires (the closed form fed by the real cone +
// plane faces, not hardcoded). Residuals are logged so the achieved precision is on the record.
func TestConeHostCornerCentre_Imported(t *testing.T) {
	t.Parallel()
	for _, c := range coneCornerCases {
		t.Run(c.name, func(t *testing.T) {
			body := corpusFixture(t, c.step)
			v := vertexNearest(t, body, c.vertex)
			faces := facesAtVertex(v)
			assertConeHostSet(t, faces)
			cb, err := solveBlend(nil, v, faces, coneCornerR)
			if err != nil {
				t.Fatalf("%s: solveBlend declined the imported cone-host corner: %v", c.name, err)
			}
			res := float64(cb.center.DistanceTo(c.center))
			t.Logf("%s imported corner centre %v, oracle %v, residual %.3e", c.name, cb.center, c.center, res)
			if res > coneCornerExactTol {
				t.Fatalf("%s: imported corner centre %v != oracle %v (residual %.3e > %g)", c.name, cb.center, c.center, res, coneCornerExactTol)
			}
		})
	}
}

// TestConeTangentPointIdentity pins the exact cone-tangent-point identity (§3): the meridian foot T where
// the corner ball touches the host cone sits so that T − C has axial component r·sin α TOWARD the apex,
// i.e. (T − C)·â = −r·sin α (3.16227766 for C2/C6, 3.84615385 for C8/D1). A wrong ĝ (axis/radial swapped)
// or a wrong sign moves this by more than r, so the identity is a tight witness of coneTangentPoint.
func TestConeTangentPointIdentity(t *testing.T) {
	t.Parallel()
	for _, c := range coneCornerCases {
		t.Run(c.name, func(t *testing.T) {
			co := coneOf(t, c)
			tp := coneTangentPoint(co, c.center)
			sinA := stdmath.Sin(co.HalfAngle)
			axial := float64(c.center.VectorTo(tp).Dot(co.AxisDir.AsVector())) // (T − C)·â
			want := -coneCornerR * sinA
			t.Logf("%s (T−C)·â = %.12f, want %.12f (r·sinα toward apex)", c.name, axial, want)
			if stdmath.Abs(axial-want) > coneCornerExactTol {
				t.Fatalf("%s: (T−C)·â = %.12f, want %.12f (Δ %.3e)", c.name, axial, want, axial-want)
			}
			if d := float64(tp.DistanceTo(c.center)); stdmath.Abs(d-coneCornerR) > coneCornerExactTol {
				t.Fatalf("%s: |T−C| = %.12f, want r = %g (the foot must sit at radius r)", c.name, d, coneCornerR)
			}
		})
	}
}

// coneCornerCertFixture returns C8's certified centre plus the exact cone, host planes, and model-relative
// resolution that coneCornerConsistent runs against — the shared setup for the certificate accept/reject
// proof.
func coneCornerCertFixture(t *testing.T) (math.Point3, geom.Cone, [2]*topo.Face, Resolution) {
	t.Helper()
	c := caseByName(t, "C8")
	v, faces := coneCornerExactFixture(t, c)
	co := coneOf(t, c)
	planes := [2]*topo.Face{faces[1], faces[2]}
	return c.center, co, planes, coneCornerResolution(v, co, planes)
}

// TestConeCornerConsistent_RejectsPerturbedCentre is the certificate regression fence: coneCornerConsistent
// must ACCEPT C8's certified corner centre and REJECT it once nudged 1e-3 (≈3000× the ~3e-7 weld) off — both
// off a host plane (along plane-0's normal, which breaks that plane's |dist| = r arm) and off the host cone
// (along +ẑ, which leaves both plane distances at r but moves the exact signed cone distance). Mutation
// witness: forcing coneCornerConsistent → true fails the two reject arms.
func TestConeCornerConsistent_RejectsPerturbedCentre(t *testing.T) {
	t.Parallel()
	center, co, planes, res := coneCornerCertFixture(t)
	if !coneCornerConsistent(center, co, planes, coneCornerR, 1, res) {
		t.Fatalf("certified C8 centre %v rejected by the certificate; want accept", center)
	}
	offPlane := center.TranslateBy(math.V3(1, 0, 0).Scale(1e-3)) // along plane-0's normal: breaks a plane distance
	if coneCornerConsistent(offPlane, co, planes, coneCornerR, 1, res) {
		t.Fatalf("centre %v nudged 1e-3 off plane-0 accepted by the certificate; want reject", offPlane)
	}
	offCone := center.TranslateBy(math.V3(0, 0, 1).Scale(1e-3)) // parallel to both planes: breaks only the cone distance
	if coneCornerConsistent(offCone, co, planes, coneCornerR, 1, res) {
		t.Fatalf("centre %v nudged 1e-3 off the host cone accepted by the certificate; want reject", offCone)
	}
}

// assertConeHostSet checks the corner vertex bounds exactly one cone host and two planes — the
// recognizer's precondition (coneHostCorner), so a locator drift is caught here, not downstream.
func assertConeHostSet(t *testing.T, faces []*topo.Face) {
	t.Helper()
	nCo, nPl := 0, 0
	for _, f := range faces {
		switch f.Geometry().(type) {
		case geom.Cone:
			nCo++
		case geom.Plane:
			nPl++
		}
	}
	if nCo != 1 || nPl != 2 || len(faces) != 3 {
		t.Fatalf("corner host set = %d faces (%d cone, %d plane), want 1 cone + 2 planes", len(faces), nCo, nPl)
	}
}

// caseByName returns the named cone-corner case (for the targeted nappe/qa-sign/degeneracy proofs).
func caseByName(t *testing.T, name string) coneCornerCase {
	t.Helper()
	for _, c := range coneCornerCases {
		if c.name == name {
			return c
		}
	}
	t.Fatalf("no cone-corner case %q", name)
	return coneCornerCase{}
}

// coneCornerQuadratic rebuilds a case's cos²α host-tangency quadratic (coefficients + offset line + u/d/â)
// exactly as coneHostCornerCenter does — the shared setup for the nappe-filter and qa-sign proofs.
func coneCornerQuadratic(t *testing.T, c coneCornerCase) (qa, qb, qc float64, u, d, ahat math.Vector3, p0 math.Point3, co geom.Cone, res Resolution) {
	t.Helper()
	v, faces := coneCornerExactFixture(t, c)
	co = coneOf(t, c)
	planes := [2]*topo.Face{faces[1], faces[2]}
	res = coneCornerResolution(v, co, planes)
	var ok bool
	p0, d, ok = planePairLine(planes, coneCornerR, v.Point())
	if !ok {
		t.Fatalf("%s: planePairLine failed", c.name)
	}
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	ahat = co.AxisDir.AsVector()
	apexPrime := co.Apex.TranslateBy(ahat.Scale(coneCornerR / sinA))
	u = apexPrime.VectorTo(p0)
	qa, qb, qc = coneQuadCoeffs(u, d, ahat, cosA*cosA)
	return qa, qb, qc, u, d, ahat, p0, co, res
}

// TestConeNappeFilter_C8LoadBearing is the nappe-filter mutation proof (§3, "Numerical pitfalls"). C8's
// host-tangency quadratic has TWO real, well-separated roots; the one nearer the corner vertex (the apex)
// is the WRONG-nappe tangency (z ≈ 127.94 > A′z = 94). Without the nappe filter, nearer-vertex alone picks
// that wrong root; WITH it, only the opening-side root (z ≈ 60.06) survives. This test proves the filter
// changes the answer — deleting it flips C8 to the wrong centre — and that the filtered answer is correct.
func TestConeNappeFilter_C8LoadBearing(t *testing.T) {
	t.Parallel()
	c := caseByName(t, "C8")
	qa, qb, qc, u, d, ahat, p0, _, res := coneCornerQuadratic(t, c)
	roots, ok := coneRealRoots(qa, qb, qc, d, res)
	if !ok || len(roots) != 2 {
		t.Fatalf("C8: want two real roots, got %v (ok=%v)", roots, ok)
	}
	tNoFilter, _ := nearerKeptRoot(roots, p0, d, c.vertex) // no nappe filter: nearer-vertex over BOTH roots
	cNoFilter := p0.TranslateBy(d.Scale(tNoFilter))
	tFilter, okF := coneCornerParam(qa, qb, qc, u, d, ahat, p0, c.vertex, res) // production path (filtered)
	if !okF {
		t.Fatalf("C8: coneCornerParam rejected after the nappe filter")
	}
	cFilter := p0.TranslateBy(d.Scale(tFilter))
	if float64(cFilter.DistanceTo(c.center)) > coneCornerExactTol {
		t.Fatalf("C8: filtered centre %v != oracle %v", cFilter, c.center)
	}
	if float64(cNoFilter.DistanceTo(c.center)) < 1.0 {
		t.Fatalf("C8: without the nappe filter the pick %v already matches the oracle — the filter is not load-bearing in this fixture", cNoFilter)
	}
	if float64(u.Add(d.Scale(tNoFilter)).Dot(ahat)) > 0 {
		t.Fatalf("C8: the no-filter pick %v is on the opening-side nappe — expected the WRONG (far) nappe", cNoFilter)
	}
	t.Logf("C8 filtered %v (correct); no-filter %v (wrong nappe, w·â=%.3f)", cFilter, cNoFilter, float64(u.Add(d.Scale(tNoFilter)).Dot(ahat)))
}

// TestConeQuadCoeffs_C8NegativeQa asserts the qa<0 branch: C8's offset line runs PARALLEL to the cone
// axis, so qa = cos²α·|d|² − (d·â)² = −sin²α·|d|² < 0 (the cos²α-scaled coefficients admit a negative
// leading term, unlike a Euclidean tangency). The solve must still find the centre — which the exact-centre
// test already confirms — but this pins that C8 genuinely exercises the negative-qa quadratic branch.
func TestConeQuadCoeffs_C8NegativeQa(t *testing.T) {
	t.Parallel()
	c := caseByName(t, "C8")
	qa, _, _, _, d, _, _, co, _ := coneCornerQuadratic(t, c)
	if qa >= 0 {
		t.Fatalf("C8: qa = %v, want < 0 (line ∥ axis ⇒ qa = −sin²α·|d|²)", qa)
	}
	sinA := stdmath.Sin(co.HalfAngle)
	want := -sinA * sinA * float64(d.Dot(d))
	if stdmath.Abs(qa-want) > coneCornerExactTol {
		t.Fatalf("C8: qa = %.12f, want −sin²α·|d|² = %.12f", qa, want)
	}
}

// TestConeRealRoots_LinearFallback exercises the qa≈0 branch: when the leading coefficient vanishes (the
// offset line lies on the offset cone's ruling-direction cone) the quadratic degenerates to the single
// linear root t = −qc/qb, and a vanishing qb too honest-rejects (line on the offset cone). Driven directly
// on the coefficients (this regime is not in the corpus — the corpus is all quadratic).
func TestConeRealRoots_LinearFallback(t *testing.T) {
	t.Parallel()
	d := math.V3(1, 0, 0)
	res := ResolutionForSize(100)
	roots, ok := coneRealRoots(0, 2, -4, d, res) // 2t − 4 = 0 → t = 2
	if !ok || len(roots) != 1 || stdmath.Abs(roots[0]-2) > coneCornerExactTol {
		t.Fatalf("linear fallback: got roots %v (ok=%v), want [2]", roots, ok)
	}
	if _, ok := coneRealRoots(0, 0, -4, d, res); ok {
		t.Fatalf("qa≈0 and qb≈0 (line on the offset cone): want reject, got accept")
	}
}

// TestConeRealRoots_GrazingRejects exercises the grazing branch: two coalescing roots whose point-space
// separation falls below the band are a geometric degeneracy, not an imprecise solve — honest-reject.
func TestConeRealRoots_GrazingRejects(t *testing.T) {
	t.Parallel()
	d := math.V3(1, 0, 0)
	res := ResolutionForSize(1e5) // band = curvedCornerBandK·res.Weld() ≈ 4e-4
	x0 := 1 - 1e-12
	// qa=1, qb=0, qc=x0²−1 → a near-double root pair with separation ≈ 2.83e-6, far inside the band.
	if _, ok := coneRealRoots(1, 0, x0*x0-1, d, res); ok {
		t.Fatalf("grazing near-double roots accepted; want reject")
	}
	if _, ok := coneRealRoots(1, 0, -1, d, res); !ok { // separation 2.0 ≫ band: a clean pair
		t.Fatalf("well-separated roots rejected; want accept")
	}
}

// coneConcaveFixture wires C2's geometry but with the cone face REVERSED — material OUTSIDE the cone, a
// concave conical bore (s = −1, the CN5 follow-on this R4-wave change adds: A′ = A − r/sinα·â).
func coneConcaveFixture(t *testing.T) (*topo.Vertex, []*topo.Face) {
	t.Helper()
	c := caseByName(t, "C2")
	lin := topo.NewLineage(topo.Tok("test", "cone-corner-concave", 0))
	bld := topo.NewBuilder(true, lin)
	v := bld.AddVertex(c.vertex, lin)
	coneFace := bld.AddReversedFace(coneOf(t, c), lin) // material OUTSIDE the cone: concave bore
	plane0 := bld.AddFace(planeOn(t, c.p0Origin, c.p0Out), lin)
	plane1 := bld.AddFace(planeOn(t, c.p1Origin, c.p1Out), lin)
	return v, []*topo.Face{coneFace, plane0, plane1}
}

// TestConeHostCornerConcave_Solves is the CN5 regression (R4 wave, simple/I4): a Reversed cone face
// (material outside, s = −1) now SOLVES via A′ = A − r/sinα·â instead of honest-rejecting. The
// centre is checked against an INDEPENDENT hand computation (not solveBlend's own machinery): both
// plane distances must be exactly r, and the (radial, axial) decomposition about the TRUE apex must
// satisfy the 45°-cone identity radial = axial·tanα directly from the C2 case's tanα = 1/3 (mirrored
// here — coneConcaveFixture reuses C2's tanα).
func TestConeHostCornerConcave_Solves(t *testing.T) {
	t.Parallel()
	v, faces := coneConcaveFixture(t)
	cb, err := solveBlend(nil, v, faces, coneCornerR)
	if err != nil {
		t.Fatalf("concave cone-host corner declined: %v, want a solved sphere (CN5)", err)
	}
	c := caseByName(t, "C2")
	assertConcaveConeCentreIndependent(t, cb.center, coneOf(t, c), c)
}

// assertConcaveConeCentreIndependent recomputes, WITHOUT calling coneHostCornerCenter/
// coneCornerConsistent, the two invariants a valid concave (s=−1) corner centre must satisfy:
// |dist| = r from both host planes, and the perpendicular point-to-line distance from
// (axial,radial) — the centre's meridian-plane coordinates about the TRUE apex — to the cone's
// generator line (through the origin at angle α) equals −r. Point-to-line-through-origin distance
// is the standard 2D identity |x·sinθ − y·cosθ| (independently re-derived here, not copied from
// coneSignedDistance): with x=axial, y=radial, θ=α, the SIGNED form is axial·sinα − radial·cosα.
func assertConcaveConeCentreIndependent(t *testing.T, centre math.Point3, co geom.Cone, c coneCornerCase) {
	t.Helper()
	for _, pl := range []struct {
		origin math.Point3
		out    math.Vector3
	}{{c.p0Origin, c.p0Out}, {c.p1Origin, c.p1Out}} {
		d := float64(pl.origin.VectorTo(centre).Dot(pl.out))
		if stdmath.Abs(stdmath.Abs(d)-coneCornerR) > coneCornerExactTol {
			t.Fatalf("plane distance = %.9f, want |±%g|", d, coneCornerR)
		}
	}
	assertConeMeridianDistance(t, centre, co, -coneCornerR)
}

// assertConeMeridianDistance is the shared independent (point-to-generator-line) check: the SIGNED
// distance axial·sinα−radial·cosα from centre to the cone's true generator, about the TRUE apex,
// must equal wantSigned (+r convex / −r concave).
func assertConeMeridianDistance(t *testing.T, centre math.Point3, co geom.Cone, wantSigned float64) {
	t.Helper()
	w := co.Apex.VectorTo(centre)
	axial := float64(w.Dot(co.AxisDir.AsVector()))
	radial := stdmath.Sqrt(float64(w.Dot(w)) - axial*axial)
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	if got := axial*sinA - radial*cosA; stdmath.Abs(got-wantSigned) > 1e-6 {
		t.Fatalf("meridian signed distance = %.9f, want %.9f (centre %v)", got, wantSigned, centre)
	}
}

// TestConeCornerConsistent_SignSensitive is the mutation witness for coneCornerConsistent's sign
// parameter (the CN5 addition): the true concave (s=−1) centre must PASS its own certificate and
// FAIL the certificate checked at the wrong sign (s=+1) — proving the s·r term is load-bearing, not
// a no-op (a mutant that drops the sign, e.g. hardcoding +r, would accept both and this test would
// catch it).
func TestConeCornerConsistent_SignSensitive(t *testing.T) {
	t.Parallel()
	v, faces := coneConcaveFixture(t)
	cb, err := solveBlend(nil, v, faces, coneCornerR)
	if err != nil {
		t.Fatalf("concave cone-host corner declined: %v", err)
	}
	co, _, planes, ok := coneHostCorner(faces)
	if !ok {
		t.Fatalf("coneConcaveFixture is not a [cone,plane,plane] host set")
	}
	res := coneCornerResolution(v, co, planes)
	if !coneCornerConsistent(cb.center, co, planes, coneCornerR, -1, res) {
		t.Fatalf("true concave centre %v rejected at its own sign s=-1", cb.center)
	}
	if coneCornerConsistent(cb.center, co, planes, coneCornerR, 1, res) {
		t.Fatalf("concave centre %v accepted at the WRONG sign s=+1; want reject", cb.center)
	}
}

// TestConeAlphaInBand_NearCylinderRejects is the α-limit regression: a near-cylinder cone (sin α below the
// model-relative band k·res.Weld()/|A−p₀|) must fail coneAlphaInBand (its apex shift r/sinα blows up — a
// true cylinder host takes the M5 path), while a healthy cone passes. Driven directly, since NewCone's open
// interval keeps a genuinely-degenerate α out of the corpus.
func TestConeAlphaInBand_NearCylinderRejects(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(1e6) // weld ≈ 1e-3, so band = coneAlphaBandCoef·weld/|A−p₀|
	p0 := math.P3(1, 0, 0)        // |A − p₀| = 1 ⇒ band ≈ 3e-3
	small, _ := geom.NewCone(math.P3(0, 0, 0), math.V3(0, 0, -1), 1e-4)
	sinS, cosS := stdmath.Sincos(small.HalfAngle)
	if coneAlphaInBand(small, p0, sinS, cosS, res) {
		t.Fatalf("near-cylinder cone (sinα=%.1e < band): coneAlphaInBand accepted; want reject", sinS)
	}
	healthy := coneOf(t, caseByName(t, "C2"))
	sinH, cosH := stdmath.Sincos(healthy.HalfAngle)
	if !coneAlphaInBand(healthy, math.P3(0, 0, 0), sinH, cosH, ResolutionForSize(300)) {
		t.Fatalf("healthy cone (sinα=%.3f): coneAlphaInBand rejected; want accept", sinH)
	}
}

// TestConeHostCornerUnsupportedMixDeclines is the do-no-harm ordering regression: a corner whose host set
// is NOT {1 cone, 2 planes} (here two cones + one plane) must NOT reach solveConeBlend — coneHostCorner
// returns ok=false and solveBlend falls through to solvePlanarBlend, declining with the exact string.
func TestConeHostCornerUnsupportedMixDeclines(t *testing.T) {
	t.Parallel()
	c := caseByName(t, "C2")
	lin := topo.NewLineage(topo.Tok("test", "cone-corner-mix", 0))
	bld := topo.NewBuilder(true, lin)
	v := bld.AddVertex(c.vertex, lin)
	cone0 := bld.AddFace(coneOf(t, c), lin)
	cone1 := bld.AddFace(coneOf(t, c), lin)
	plane0 := bld.AddFace(planeOn(t, c.p0Origin, c.p0Out), lin)
	faces := []*topo.Face{cone0, cone1, plane0}
	if _, _, _, ok := coneHostCorner(faces); ok {
		t.Fatalf("coneHostCorner accepted a 2-cone + 1-plane host set; want ok=false")
	}
	if _, err := solveBlend(nil, v, faces, coneCornerR); err == nil || err.Error() != "fillet: corner face must be planar" {
		t.Fatalf("unsupported cone mix: got err %v, want %q", err, "fillet: corner face must be planar")
	}
}

// TestConeHostCornerConcave_I4Imported drives the REAL imported simple/I4 fixture (R4 wave, radius
// 10) through the real solveBlend→coneHostCorner→solveConeBlend path — the corpus's actual concave
// conical-bore corner, not a synthetic mirror — and checks the centre with the SAME independent
// (radial=axial·tanα, |plane dist|=r) identity assertConcaveConeCentreIndependent uses for C2.
// I4's corner still declines end-to-end in the corpus scoreboard (fillet_conearm.go's concave-cone
// ARM guard is a separate, unowned file — see the R4 wave report) — this test certifies the CORNER
// PATCH capability this file owns, independent of that companion gap.
func TestConeHostCornerConcave_I4Imported(t *testing.T) {
	t.Parallel()
	body := corpusFixture(t, "simple/I4.step")
	v := vertexNearest(t, body, math.P3(-200, 250, 0))
	faces := facesAtVertex(v)
	co, coneFace, planes, ok := coneHostCorner(faces)
	if !ok {
		t.Fatalf("I4 corner is not a [cone,plane,plane] host set: %d faces", len(faces))
	}
	if sgn, ok := coneCornerMaterialSign(co, coneFace); !ok || sgn >= 0 {
		t.Fatalf("I4 material sign = %v (ok=%v), want a genuine concave (<0) fixture", sgn, ok)
	}
	cb, err := solveBlend(nil, v, faces, 10)
	if err != nil {
		t.Fatalf("I4 concave cone-host corner declined: %v", err)
	}
	for _, f := range planes {
		pl := f.Geometry().(geom.Plane)
		d := stdmath.Abs(float64(pl.Origin.VectorTo(cb.center).Dot(outwardPlaneNormal(f, pl))))
		if stdmath.Abs(d-10) > 1e-6 {
			t.Fatalf("I4 centre %v is %.9f from a host plane, want 10", cb.center, d)
		}
	}
	assertConeMeridianDistance(t, cb.center, co, -10)
}
