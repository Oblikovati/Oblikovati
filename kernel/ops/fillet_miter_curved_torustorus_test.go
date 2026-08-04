// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	stepio "oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// syntheticTorusPair builds a HAND-SOLVABLE torus∧torus arm pair: two coaxial (axis ẑ) tori of minor
// radius r sharing the major-circle plane z=height, centres 30 apart in x, major radii majA/majB — the
// exact O9-shape family (torus-torus-miter-derivation.md), independent of any STEP import.
func syntheticTorusPair(t *testing.T, majA, majB, r, height float64) curvedMiterTorusPair {
	t.Helper()
	torA, err := geom.NewTorus(math.P3(50, 50, height), math.V3(0, 0, 1), majA, r)
	if err != nil {
		t.Fatalf("NewTorus A: %v", err)
	}
	torB, err := geom.NewTorus(math.P3(80, 50, height), math.V3(0, 0, 1), majB, r)
	if err != nil {
		t.Fatalf("NewTorus B: %v", err)
	}
	return curvedMiterTorusPair{torA: torA, torB: torB}
}

// TestTorusTorusStation_HandVerifiedSymmetricNumbers hand-verifies the closed form against an
// INDEPENDENTLY computed symmetric (boss∧boss, R_A′=R_B′=45) instance, per the coordinator's explicit
// "hand-verify before trusting" directive: separation d=30, half-separation a=15 (host axes 30 apart,
// equal radii puts the bisector foot exactly at the midpoint's x, matching x₀=65=50+15), so
// m*/sTop share x=65 and h=√(45²−15²)=√1800; sBot (ρ=50, the ORIGINAL host radius) has
// h=√(50²−15²)=√2275. These are exact closed-form values (do Carmo circle-chord geometry), not
// tolerance-fit constants — a mismatch beyond float64 epsilon is a bug, not a tolerance issue.
func TestTorusTorusStation_HandVerifiedSymmetricNumbers(t *testing.T) {
	pair := syntheticTorusPair(t, 45, 45, 5, 65)
	res := ResolutionForPoints([]math.Point3{pair.torA.Center, pair.torB.Center})
	near := math.P3(65, 0, 65) // biases nearerTorusTorusRoot toward the −y root, both hand-derived below
	wantH1800 := stdmath.Sqrt(1800)
	wantH2275 := stdmath.Sqrt(2275)

	center, ok := torusTorusStation(pair, 5, 0, 0, 0, near, res)
	assertTorusTorusStation(t, "center", center, ok, 65, 50-wantH1800, 65)

	sTop, ok := torusTorusStation(pair, 5, 5, 1, 1, near, res)
	assertTorusTorusStation(t, "sTop", sTop, ok, 65, 50-wantH1800, 70)

	sBot, ok := torusTorusStation(pair, 5, 0, 1, 1, near, res)
	assertTorusTorusStation(t, "sBot", sBot, ok, 65, 50-wantH2275, 65)
}

// assertTorusTorusStation asserts a station point to 1e-9 absolute — the closed form is exact, so this
// is a bug detector, not a tolerance-shaped pass.
func assertTorusTorusStation(t *testing.T, name string, p math.Point3, ok bool, x, y, z float64) {
	t.Helper()
	if !ok {
		t.Fatalf("%s: station declined, want a solution", name)
	}
	got := [3]float64{p.X, p.Y, p.Z}
	want := [3]float64{x, y, z}
	for i := range got {
		if stdmath.Abs(got[i]-want[i]) > 1e-9 {
			t.Fatalf("%s: got %v, want %v (component %d off by %g)", name, p, want, i, got[i]-want[i])
		}
	}
}

// TestTorusTorusCoaxial_RejectsAntiParallelAxes is the falsification proof for the axis-direction gate
// (package doc §"anti-parallel breaks the z² cancellation"): flipping torus B's axis must decline.
func TestTorusTorusCoaxial_RejectsAntiParallelAxes(t *testing.T) {
	pair := syntheticTorusPair(t, 45, 45, 5, 65)
	res := ResolutionForPoints([]math.Point3{pair.torA.Center, pair.torB.Center})
	if torusTorusCoaxial(pair.torA, pair.torB, res) != true {
		t.Fatalf("baseline (identical-direction axes) must accept")
	}
	flipped, err := geom.NewTorus(pair.torB.Center, math.V3(0, 0, -1), pair.torB.MajorRadius, pair.torB.MinorRadius)
	if err != nil {
		t.Fatalf("NewTorus flipped: %v", err)
	}
	if torusTorusCoaxial(pair.torA, flipped, res) {
		t.Fatalf("anti-parallel axis mutation must decline (z² cancellation does not hold)")
	}
}

// TestTorusTorusCoaxial_RejectsNonCoplanarMajorCircles is the falsification proof for the coplanarity
// gate: shifting torus B's centre off torus A's major-circle-plane height must decline.
func TestTorusTorusCoaxial_RejectsNonCoplanarMajorCircles(t *testing.T) {
	pair := syntheticTorusPair(t, 45, 45, 5, 65)
	res := ResolutionForPoints([]math.Point3{pair.torA.Center, pair.torB.Center})
	shifted, err := geom.NewTorus(pair.torB.Center.TranslateBy(math.V3(0, 0, 1)), pair.torB.AxisDir.AsVector(), pair.torB.MajorRadius, pair.torB.MinorRadius)
	if err != nil {
		t.Fatalf("NewTorus shifted: %v", err)
	}
	if torusTorusCoaxial(pair.torA, shifted, res) {
		t.Fatalf("non-coplanar major-circle mutation must decline")
	}
}

// TestTorusTorusCoaxial_AcceptsUnequalMajorRadii documents the corrected scope (package doc): an
// unequal major-radius pair (O9's REAL boss+notch construction, 45 vs 55) is NOT gated out at
// recognition — torusTorusPhysicalSign certifies the per-arm sign later instead. An earlier version of
// this recognizer wrongly gated on equal major radii and rejected O9's own real fixture.
func TestTorusTorusCoaxial_AcceptsUnequalMajorRadii(t *testing.T) {
	pair := syntheticTorusPair(t, 45, 55, 5, 65)
	res := ResolutionForPoints([]math.Point3{pair.torA.Center, pair.torB.Center})
	if !torusTorusCoaxial(pair.torA, pair.torB, res) {
		t.Fatalf("unequal major radii must NOT be gated at recognition (certified later by sign)")
	}
}

// TestTorusTorusHostSign_CertifiesBossAndBore hand-builds a boss (R′+r=R_host) and a bore (R′−r=R_host)
// arm against a REAL cylinder∧plane edge (via a synthetic topo body) and asserts each sign
// independently — and that a torus whose major radius matches NEITHER sign declines.
func TestTorusTorusHostSign_CertifiesBossAndBore(t *testing.T) {
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(50, 0, 0)})
	e, cyl := syntheticCylinderPlaneEdge(t, 50)
	boss, err := geom.NewTorus(cyl.Origin, cyl.AxisDir.AsVector(), 45, 5) // R′=R−r=45
	if err != nil {
		t.Fatalf("NewTorus boss: %v", err)
	}
	if sign, ok := torusTorusHostSign(e, boss, res); !ok || sign != 1 {
		t.Fatalf("boss arm: got sign=%v ok=%v, want +1,true", sign, ok)
	}
	bore, err := geom.NewTorus(cyl.Origin, cyl.AxisDir.AsVector(), 55, 5) // R′=R+r=55
	if err != nil {
		t.Fatalf("NewTorus bore: %v", err)
	}
	if sign, ok := torusTorusHostSign(e, bore, res); !ok || sign != -1 {
		t.Fatalf("bore arm: got sign=%v ok=%v, want -1,true", sign, ok)
	}
	neither, err := geom.NewTorus(cyl.Origin, cyl.AxisDir.AsVector(), 30, 5) // matches neither R±r
	if err != nil {
		t.Fatalf("NewTorus neither: %v", err)
	}
	if _, ok := torusTorusHostSign(e, neither, res); ok {
		t.Fatalf("a major radius matching neither R±r must decline, not silently pick a sign")
	}
}

// syntheticCylinderPlaneEdge builds a bare synthetic Cylinder∧Plane edge (a vertical ruling at
// (radius,0,z), axis ẑ) so torusTorusHostSign can be exercised without a full STEP import —
// cylinderPlaneEdge only reads the edge's two FACE geometries, not its own curve, so a line-ruling
// edge is a faithful, minimal stand-in for the cap-circle edges the real fixtures use (mirrors
// fillet_arm_concave_test.go's concaveBoreFixture, the established pattern for this kind of fixture).
func syntheticCylinderPlaneEdge(t *testing.T, radius float64) (*topo.Edge, geom.Cylinder) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "torustorus-hostsign", 0))
	bld := topo.NewBuilder(true, lin)
	lo := bld.AddVertex(math.P3(radius, 0, 0), lin)
	hi := bld.AddVertex(math.P3(radius, 0, 50), lin)
	e := bld.AddEdge(geom.NewLineSegment(math.P3(radius, 0, 0), math.P3(radius, 0, 50)), lo, hi, lin)
	cyl := cylAxis(0, 0, 1, radius)
	pl := planeWithNormal(1, 0, 0)
	bld.AddFace(cyl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e)))
	bld.AddFace(pl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e)))
	bld.Build()
	return e, cyl
}

// TestTorusTorusSeamBottomCertified_RejectsMismatch is the falsification proof for the hard gate
// (torus-torus-miter-derivation.md §4): a correctly-derived sBot certifies, and the SAME point nudged
// by a distance well above weld tolerance does not.
func TestTorusTorusSeamBottomCertified_RejectsMismatch(t *testing.T) {
	pair := o9RealTorusPair(t)
	res := ResolutionForPoints([]math.Point3{pair.torA.Center, pair.torB.Center})
	vp := math.P3(65, 2.3, 65)
	sBot, ok := torusTorusStation(pair, 5, 0, 1, -1, vp, res)
	if !ok {
		t.Fatalf("sBot station declined")
	}
	if !torusTorusSeamBottomCertified(pair, sBot, vp, res) {
		t.Fatalf("the genuinely-derived sBot must certify")
	}
	nudged := sBot.TranslateBy(math.V3(1, 0, 0)) // 1mm, far above weld tolerance at this model scale
	if torusTorusSeamBottomCertified(pair, nudged, vp, res) {
		t.Fatalf("a 1mm-off sBot mutation must be REJECTED by the hard gate, not silently accepted")
	}
}

// o9RealTorusPair rebuilds O9's own two arm tori (R_A′=45 boss, R_B′=55 bore, minor r=5, centres 30
// apart, both height 65) directly, WITH their own synthetic host Cylinder∧Plane edges (both R=50,
// matching O9's real cylinders) so torusTorusSeamBottomCertified's cylinderPlaneEdge lookups resolve —
// the synthetic mirror of the REAL fixture's numbers (verified against the imported STEP body in
// TestSolveCurvedMiterTorusPair_O9RealFixture below), so the pure closed-form tests do not depend on
// STEP import succeeding.
func o9RealTorusPair(t *testing.T) curvedMiterTorusPair {
	t.Helper()
	pair := syntheticTorusPair(t, 45, 55, 5, 65)
	pair.edgeA, _ = syntheticCylinderPlaneEdgeAt(t, pair.torA.Center, 50)
	pair.edgeB, _ = syntheticCylinderPlaneEdgeAt(t, pair.torB.Center, 50)
	return pair
}

// syntheticCylinderPlaneEdgeAt is syntheticCylinderPlaneEdge with the cylinder's axis line through an
// arbitrary origin instead of the world origin — needed once the fixture has two DIFFERENT axis
// positions (O9's torus pair), unlike the single-torus torusTorusHostSign tests above.
func syntheticCylinderPlaneEdgeAt(t *testing.T, axisPoint math.Point3, radius float64) (*topo.Edge, geom.Cylinder) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "torustorus-hostsign-at", 0))
	bld := topo.NewBuilder(true, lin)
	p0 := axisPoint.TranslateBy(math.V3(radius, 0, 0))
	p1 := p0.TranslateBy(math.V3(0, 0, 50))
	lo := bld.AddVertex(p0, lin)
	hi := bld.AddVertex(p1, lin)
	e := bld.AddEdge(geom.NewLineSegment(p0, p1), lo, hi, lin)
	cyl, err := geom.NewCylinder(axisPoint, math.V3(0, 0, 1), radius)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	pl, err := geom.NewPlane(p0, math.V3(1, 0, 0))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	bld.AddFace(cyl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e)))
	bld.AddFace(pl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e)))
	bld.Build()
	return e, cyl
}

// importO9 loads the real simple/O9 fixture the corpus scoreboard uses.
func importO9(t *testing.T) *topo.Body {
	t.Helper()
	path := filepath.Join("..", "..", "model", "feature", "occtparity", "fixtures", "simple", "O9.step")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	bodies, _, err := stepio.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil {
		t.Fatalf("import %s: %v", path, err)
	}
	if len(bodies) != 1 {
		t.Fatalf("%s: got %d bodies, want 1", path, len(bodies))
	}
	return bodies[0]
}

// TestBuildCurvedMiterTorusPair_O9RealFixture is the REAL-fixture recognizer test: locates O9's own
// two torus-arm edges by geometry (centroid/length, mirroring the corpus harness's own locateEdge) and
// asserts buildCurvedMiterTorusPair recognizes them with the EXACT major radii the corner-patch layer
// (fillet_twocyl_corner.go) independently derives (45 boss, 55 bore — cut s1 s2, not fuse).
func TestBuildCurvedMiterTorusPair_O9RealFixture(t *testing.T) {
	body := importO9(t)
	e8 := locateEdgeForTest(t, body, math.P3(24.57553393, 50, 70), 187.5220562)
	e16 := locateEdgeForTest(t, body, math.P3(42.33267569, 50, 70), 126.602109)
	res := ResolutionForBody(body)
	pair, ok := buildCurvedMiterTorusPair([]filletPick{{edge: e8, r0: 5, r1: 5}, {edge: e16, r0: 5, r1: 5}}, 5, res)
	if !ok {
		t.Fatalf("buildCurvedMiterTorusPair declined on O9's real torus-arm edge pair")
	}
	if stdmath.Abs(pair.torA.MajorRadius-45) > 1e-6 || stdmath.Abs(pair.torB.MajorRadius-55) > 1e-6 {
		t.Fatalf("got major radii (%g, %g), want (45, 55) — O9 is `cut s1 s2`, a boss+notch mix", pair.torA.MajorRadius, pair.torB.MajorRadius)
	}
	signA, signB, ok := torusTorusPhysicalSign(pair, res)
	if !ok || signA != 1 || signB != -1 {
		t.Fatalf("got signA=%v signB=%v ok=%v, want +1,-1,true (boss∧bore)", signA, signB, ok)
	}
}

// TestBuildCurvedMiterArms_DeclinesOnTorusTorusPair proves buildCurvedMiterArms (family B/C) does NOT
// also match a torus∧torus pick set — the do-no-harm disjointness buildCurvedMiterTorusPair's own doc
// asserts (its torIdx/cylIdx bookkeeping structurally cannot both be set from two geom.Torus arms).
func TestBuildCurvedMiterArms_DeclinesOnTorusTorusPair(t *testing.T) {
	body := importO9(t)
	e8 := locateEdgeForTest(t, body, math.P3(24.57553393, 50, 70), 187.5220562)
	e16 := locateEdgeForTest(t, body, math.P3(42.33267569, 50, 70), 126.602109)
	res := ResolutionForBody(body)
	ps := []filletPick{{edge: e8, r0: 5, r1: 5}, {edge: e16, r0: 5, r1: 5}}
	if _, ok := buildCurvedMiterArms(ps, 5, res); ok {
		t.Fatalf("buildCurvedMiterArms must decline a torus∧torus pick set (family D's own scope)")
	}
}

// TestSolveCurvedMiterTorusPair_O9RealFixture drives the full family-D solve against O9's real miter
// vertex and independently re-checks sTop/sBot against the hand-derived asymmetric-branch numbers
// (torus-torus-miter-derivation.md, the mixed boss+bore branch): sBot at ρ=50 (both, the ORIGINAL host
// radius) reproduces the SAME (√2275-offset) point family as the symmetric case since ρ_A=ρ_B=50
// there regardless of R′_A≠R′_B; sTop's x is NOT 65 (the pair is no longer symmetric) — computed here
// via intersectCoplanarCircles directly as an independent cross-check, not by calling
// torusTorusStation on itself.
func TestSolveCurvedMiterTorusPair_O9RealFixture(t *testing.T) {
	body := importO9(t)
	e8 := locateEdgeForTest(t, body, math.P3(24.57553393, 50, 70), 187.5220562)
	e16 := locateEdgeForTest(t, body, math.P3(42.33267569, 50, 70), 126.602109)
	res := ResolutionForBody(body)
	pair, ok := buildCurvedMiterTorusPair([]filletPick{{edge: e8, r0: 5, r1: 5}, {edge: e16, r0: 5, r1: 5}}, 5, res)
	if !ok {
		t.Fatalf("buildCurvedMiterTorusPair declined")
	}
	v := miterVertexForTest(t, e8, e16)
	shared := sharedFace(e8, e16)
	if shared == nil {
		t.Fatalf("e8/e16 share no face")
	}
	corner, err := solveCurvedMiterTorusPair(v, shared, pair, 5, res)
	if err != nil {
		t.Fatalf("solveCurvedMiterTorusPair: %v", err)
	}
	if len(corner.seam) < 2 {
		t.Fatalf("seam too short: %d points", len(corner.seam))
	}
	sBot := corner.seam[len(corner.seam)-1]
	wantSBotOffset := stdmath.Sqrt(2275.0)
	gotSBotOffset := stdmath.Abs(sBot.Y - 50)
	if stdmath.Abs(gotSBotOffset-wantSBotOffset) > 1e-6 {
		t.Fatalf("sBot y-offset from 50: got %v, want %v (independent hand-derived √2275)", gotSBotOffset, wantSBotOffset)
	}
	if stdmath.Abs(sBot.X-65) > 1e-6 {
		t.Fatalf("sBot.X: got %v, want 65 (ρ_A=ρ_B=cylRadius=50 at sBot, the equal-radius bisector regardless of R′_A≠R′_B)", sBot.X)
	}
	// Independent cross-check of sTop against intersectCoplanarCircles called directly (not via
	// torusTorusStation), using the ASYMMETRIC radii (45, 55) at D=0.
	p1, p2, ok := intersectCoplanarCircles(pair.torA.Center, 45, pair.torB.Center, 55, pair.torA.AxisDir.AsVector(), res)
	if !ok {
		t.Fatalf("independent sTop cross-check: intersectCoplanarCircles declined")
	}
	wantSTop := nearerTorusTorusRoot(v.Point(), p1, p2).TranslateBy(math.V3(0, 0, 5))
	sTop := corner.seam[0]
	if float64(sTop.DistanceTo(wantSTop)) > 1e-6 {
		t.Fatalf("sTop: got %v, want %v (independent cross-check)", sTop, wantSTop)
	}
}

// miterVertexForTest returns the vertex shared by e8's OWN far end (not the trihedral corner they
// share with the third edge) — i.e. the SECOND vertex e8 and e16 have in common, mirroring how
// solveCorner's own vertex grouping finds the miter (as opposed to the blend) corner. Since a real
// fixture may have edges shared at TWO vertices (the trihedral corner AND the miter), this returns
// whichever shared vertex has valence-2 in {e8,e16} but NOT also touched by a third face-count-3 host —
// simplified here to: the shared vertex FARTHER from both edges' own trihedral-corner shared vertex is
// not computable without the third edge, so this picks the shared vertex nearer the fixture's own
// known miter location (65, ~97.7, 70) rather than the corner's (65, ~2.3, 70).
func miterVertexForTest(t *testing.T, e8, e16 *topo.Edge) *topo.Vertex {
	t.Helper()
	shared := sharedVerticesForTest(e8, e16)
	if len(shared) != 2 {
		t.Fatalf("e8/e16 share %d vertices, want 2 (trihedral corner + miter)", len(shared))
	}
	miterY := math.P3(65, 97.7, 70)
	if shared[0].Point().DistanceTo(miterY) <= shared[1].Point().DistanceTo(miterY) {
		return shared[0]
	}
	return shared[1]
}

// sharedVerticesForTest returns the vertices common to both edges.
func sharedVerticesForTest(e1, e2 *topo.Edge) []*topo.Vertex {
	var out []*topo.Vertex
	for _, v1 := range e1.Vertices() {
		for _, v2 := range e2.Vertices() {
			if v1.ID() == v2.ID() {
				out = append(out, v1)
			}
		}
	}
	return out
}

// locateEdgeForTest re-finds a body edge by its arc-length centroid/length, mirroring
// model/feature/occtparity's own locateEdge (kept local — that package is a layer above kernel/ops).
func locateEdgeForTest(t *testing.T, b *topo.Body, centroid math.Point3, length float64) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestD := stdmath.Inf(1)
	for _, e := range b.Edges() {
		c := e.Geometry()
		lo, hi := c.Domain()
		var sum math.Vector3
		var l float64
		prev := c.PointAt(lo)
		for i := 1; i <= 64; i++ {
			tt := lo + (hi-lo)*float64(i)/64
			p := c.PointAt(tt)
			seg := float64(p.DistanceTo(prev))
			sum = sum.Add(prev.Midpoint(p).AsVector().Scale(seg))
			l += seg
			prev = p
		}
		if l == 0 || stdmath.Abs(l-length) > 0.01*length+1e-6 {
			continue
		}
		cen := sum.Scale(1 / l).AsPoint()
		if d := float64(cen.DistanceTo(centroid)); d < bestD {
			bestD, best = d, e
		}
	}
	if best == nil {
		t.Fatalf("locateEdgeForTest: no edge near centroid %v length %v", centroid, length)
	}
	return best
}
