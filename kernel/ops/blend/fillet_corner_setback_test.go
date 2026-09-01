// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestConcaveDihedralSetbackExtendsRailByR is the L1-class positive: a 100³ box + 40³ boss whose four
// concave base edges are filleted r=5. The dihedral corner-setback pass must EXTEND each miter band's
// shared-face rail by exactly r past the raw endpoint and re-trim the box-top plane to it, so the
// top-plane hole recedes from the raw 40×40 boss footprint to a 50×50 square [35,85]² (each side ±r).
// A dropped setback leaves the reflected-seam [45,75]² hole; a wrong distance moves the corners off ±r.
func TestConcaveDihedralSetbackExtendsRailByR(t *testing.T) {
	t.Parallel()
	body := boxWithBoss(t)
	keys := concaveBaseEdgeKeys(body)
	if len(keys) != 4 {
		t.Fatalf("box+boss has %d concave base edges, want 4", len(keys))
	}
	res, err := FilletEdgesCorner(body, filletPicksFor(keys, 5), CornerMiter, FillConcaveOutward)
	if err != nil {
		t.Fatalf("fillet box+boss concave base edges: %v", err)
	}
	rep := validate.Validate(res)
	if !rep.Valid || !rep.Closed || !rep.HolesContained || !res.IsSolid() {
		t.Fatalf("set-back result not a watertight solid: valid=%v closed=%v holes=%v solid=%v",
			rep.Valid, rep.Closed, rep.HolesContained, res.IsSolid())
	}
	eps := tol.ForBody(res).Weld() // model-relative coincidence tolerance (M35), not a bare epsilon
	lo, hi := boxTopHoleExtent(t, res, eps)
	// raw boss footprint is [40,80]²; the setback = r = 5 recedes each side to [35,85].
	if !approxPoint(lo, math.P2(35, 35), eps) || !approxPoint(hi, math.P2(85, 85), eps) {
		t.Fatalf("box-top hole is [%.2f,%.2f]×… want [35,85]² (setback must extend the rail by exactly r=5); lo=%v hi=%v", lo.X, hi.X, lo, hi)
	}
}

// TestCornerSetbackGateFiresOnlyForConcaveOrthogonalPlanarDihedral pins the gate boundary: it fires for
// the L1-class corner (concave, orthogonal, planar shared face) and stays out of every other config — a
// convex miter (already correct, must stay byte-identical), a non-orthogonal concave miter (L7-skew /
// A8, a later slice), a curved-contact miter (families B/C), and a non-planar shared face. Each keeps
// the baseline body untouched.
func TestCornerSetbackGateFiresOnlyForConcaveOrthogonalPlanarDihedral(t *testing.T) {
	t.Parallel()
	shared := aPlanarFace(t)
	xAxis, yAxis, skew := math.V3(1, 0, 0), math.V3(0, 1, 0), math.V3(stdmath.Cos(stdmath.Pi/3), stdmath.Sin(stdmath.Pi/3), 0)
	for _, tc := range []struct {
		name string
		efA  edgeFillet
		efB  edgeFillet
		cm   cornerMiter
		want bool
	}{
		{"concave-orthogonal-planar", concaveArm(t, xAxis), concaveArm(t, yAxis), cornerMiter{shared: shared}, true},
		{"convex-orthogonal-planar", convexArm(t, xAxis), convexArm(t, yAxis), cornerMiter{shared: shared}, false},
		{"concave-non-orthogonal", concaveArm(t, xAxis), concaveArm(t, skew), cornerMiter{shared: shared}, false},
		{"concave-curved-contact", concaveArm(t, xAxis), concaveArm(t, yAxis), cornerMiter{shared: shared, curved: &curvedMiterCorner{}}, false},
		{"concave-non-planar-shared", concaveArm(t, xAxis), concaveArm(t, yAxis), cornerMiter{shared: aCylindricalFace(t)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := concaveOrthogonalDihedralMiter(&tc.efA, &tc.efB, &tc.cm); got != tc.want {
				t.Fatalf("gate fired=%v, want %v for %s", got, tc.want, tc.name)
			}
		})
	}
}

// TestCornerSetbackDeclinesConvexMiter confirms the unified accumulate reports fired=false (leaving the
// caller's baseline body byte-identical) when the only miter is convex — the shared-path do-no-harm
// guarantee. classifyMiterCorner tags a convex miter decline, so accumulate contributes no railWrite.
func TestCornerSetbackDeclinesConvexMiter(t *testing.T) {
	t.Parallel()
	fils := []edgeFillet{convexArm(t, math.V3(1, 0, 0)), convexArm(t, math.V3(0, 1, 0))}
	fils[0].c1.miter, fils[1].c1.miter = true, true
	v := fils[0].c1.vertex
	miters := map[uint64]*cornerMiter{v.ID(): {shared: aPlanarFace(t), vertex: v}}
	ctx := setbackCtx{fils: fils, blends: map[uint64]*cornerBlend{}, miters: miters, ends: miterCornerEnds(fils)}
	if accumulate(ctx).fired {
		t.Fatal("accumulate fired on a convex miter — the convex path must stay byte-identical")
	}
}

// TestConcaveTrihedralSetbackRetractsBandsToVoidSphere is the K6-class positive: a 100³ box with a blind
// rectangular pocket, whose ONE trihedral corner joins three concave fillets (a vertical wall edge + two
// floor edges) at three mutually-orthogonal planar faces. The trihedral corner-setback pass must flip the
// corner sphere from the material side (solvePlanarBlend's reflection) to the VOID side and RETRACT each
// band by exactly r to the void tangent circle. The pocket corner is (20,20,60); the void octant sphere
// centre is (25,25,65) (r inside the pocket from each face), and the vertical band's bottom rail recedes
// from the raw z=60 up to z=65 (=60+r). A dropped setback leaves the material-side sphere and the un-
// retracted band (bottom at z=60).
func TestConcaveTrihedralSetbackRetractsBandsToVoidSphere(t *testing.T) {
	t.Parallel()
	body := boxWithPocket(t)
	keys := pocketCornerEdgeKeys(t, body, math.P3(20, 20, 60))
	if len(keys) != 3 {
		t.Fatalf("pocket corner has %d concave edges, want 3 (a vertical wall edge + two floor edges)", len(keys))
	}
	res, err := FilletEdgesCorner(body, filletPicksFor(keys, 5), CornerMiter, FillConcaveOutward)
	if err != nil {
		t.Fatalf("fillet pocket corner concave edges: %v", err)
	}
	rep := validate.Validate(res)
	if !rep.Valid || !rep.Closed || !rep.HolesContained || !res.IsSolid() {
		t.Fatalf("trihedral set-back result not a watertight solid: valid=%v closed=%v holes=%v solid=%v",
			rep.Valid, rep.Closed, rep.HolesContained, res.IsSolid())
	}
	eps := tol.ForBody(res).Weld()
	assertVoidSphereCentre(t, res, math.P3(25, 25, 65), eps)
	// the vertical wall band (axis ∥z) retracts its corner rail by r: its lowest vertex sits at z=60+r=65.
	if zlo := verticalBandBottomZ(t, res); stdmath.Abs(zlo-65) > eps {
		t.Fatalf("vertical band bottom at z=%.4f, want 65 (=60+r: the trihedral setback must retract the band by exactly r=5)", zlo)
	}
}

// TestTrihedralSetbackGateFiresOnlyForConcaveOrthogonalPlanar pins the trihedral gate boundary: it fires
// for three CONCAVE fillets meeting three mutually-orthogonal PLANAR faces (K6/L4) and declines every
// other config — a mixed-sense corner (2 concave + 1 convex, the K9/M2 torus, P3), a non-orthogonal
// triple, a curved-host face, and a wrong valence — so each keeps its material-side sphere byte-identical.
func TestTrihedralSetbackGateFiresOnlyForConcaveOrthogonalPlanar(t *testing.T) {
	t.Parallel()
	fx, fy, fz := threeOrthogonalPlanarFaces(t)
	cyl := aCylindricalFace(t)
	for _, tc := range []struct {
		name  string
		fils  []edgeFillet
		faces [3]*topo.Face
		want  bool
	}{
		{"concave-orthogonal-planar", triArms(t, true, true, true), [3]*topo.Face{fx, fy, fz}, true},
		{"mixed-sense", triArms(t, true, true, false), [3]*topo.Face{fx, fy, fz}, false},
		{"non-orthogonal-parallel", triArms(t, true, true, true), [3]*topo.Face{fx, fy, fx}, false},
		{"curved-host-face", triArms(t, true, true, true), [3]*topo.Face{fx, fy, cyl}, false},
		{"wrong-valence", triArms(t, true, true, true)[:2], [3]*topo.Face{fx, fy, fz}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setTriArmFaces(tc.fils, tc.faces)
			v := fixtureCornerVertex
			sph, err := geom.NewSphere(math.P3(0, 0, 0), 5)
			if err != nil {
				t.Fatalf("NewSphere: %v", err)
			}
			cb := &cornerBlend{vertex: v, sphere: sph}
			_, ok := concaveTrihedralCornerFaces(v.ID(), cb, tc.fils)
			if ok != tc.want {
				t.Fatalf("gate fired=%v, want %v for %s", ok, tc.want, tc.name)
			}
			ctx := setbackCtx{fils: tc.fils, blends: map[uint64]*cornerBlend{v.ID(): cb}, miters: map[uint64]*cornerMiter{}, ends: miterCornerEnds(tc.fils)}
			if fired := accumulate(ctx).fired; fired != tc.want {
				t.Fatalf("accumulate fired=%v, want %v for %s (the void-sphere concaveSphere channel)", fired, tc.want, tc.name)
			}
		})
	}
}

// --- named fixtures -----------------------------------------------------------------------------------

// boxWithBoss builds the L1 solid: a 100³ box unioned with a 40³ boss seated on its top face.
func boxWithBoss(t *testing.T) *topo.Body {
	t.Helper()
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(100, 100, 100), "box")
	if err != nil {
		t.Fatalf("SolidBlock box: %v", err)
	}
	boss, err := brep.SolidBlock(math.P3(40, 40, 100), math.P3(80, 80, 140), "boss")
	if err != nil {
		t.Fatalf("SolidBlock boss: %v", err)
	}
	res, err := brep.Boolean(brep.Union, box, boss)
	if err != nil {
		t.Fatalf("brep.Boolean(brep.Union) box+boss: %v", err)
	}
	return res
}

// concaveBaseEdgeKeys returns the reference keys of the four concave horizontal base edges (the
// boss-wall-meets-box-top edges at z=100). The z coincidence uses the body's model-relative weld
// tolerance (M35), not a bare epsilon.
func concaveBaseEdgeKeys(b *topo.Body) [][]byte {
	eps := tol.ForBody(b).Weld()
	var keys [][]byte
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if stdmath.Abs(a.Z-100) < eps && stdmath.Abs(c.Z-100) < eps && ClassifyEdgeConvexity(e) == EdgeConcave {
			keys = append(keys, e.ReferenceKey())
		}
	}
	return keys
}

// filletPicksFor pairs each edge key with a constant radius r.
func filletPicksFor(keys [][]byte, r float64) []EdgeFilletRadii {
	picks := make([]EdgeFilletRadii, len(keys))
	for i, k := range keys {
		picks[i] = EdgeFilletRadii{Key: k, R0: r, R1: r}
	}
	return picks
}

// boxTopHoleExtent returns the min/max XY corners of the box-top (z=100) face's single hole loop. The
// z-plane match uses the caller's model-relative tolerance eps, not a bare epsilon.
func boxTopHoleExtent(t *testing.T, b *topo.Body, eps float64) (math.Point2, math.Point2) {
	t.Helper()
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || stdmath.Abs(pl.Normal().Z) < 0.99 || stdmath.Abs(pl.Origin.Z-100) > eps {
			continue
		}
		for _, l := range f.Loops() {
			if l.IsOuter() {
				continue
			}
			return loopXYExtent(l)
		}
	}
	t.Fatal("no box-top (z=100) plane with a hole loop")
	return math.Point2{}, math.Point2{}
}

// loopXYExtent returns the XY bounding-box corners of a loop's vertices.
func loopXYExtent(l *topo.Loop) (math.Point2, math.Point2) {
	lo := math.P2(stdmath.Inf(1), stdmath.Inf(1))
	hi := math.P2(stdmath.Inf(-1), stdmath.Inf(-1))
	for _, u := range l.EdgeUses() {
		p := u.Edge().StartVertex().Point()
		lo = math.P2(stdmath.Min(lo.X, p.X), stdmath.Min(lo.Y, p.Y))
		hi = math.P2(stdmath.Max(hi.X, p.X), stdmath.Max(hi.Y, p.Y))
	}
	return lo, hi
}

// boxWithPocket builds the K6-class solid: a 100³ box minus a blind rectangular pocket cut from the top,
// x∈[20,50], y∈[20,50], floor at z=60. Its bottom corners are concave trihedral vertices.
func boxWithPocket(t *testing.T) *topo.Body {
	t.Helper()
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(100, 100, 100), "box")
	if err != nil {
		t.Fatalf("SolidBlock box: %v", err)
	}
	tool, err := brep.SolidBlock(math.P3(20, 20, 60), math.P3(50, 50, 101), "pocket") // 101 > 100: clean cut through the top
	if err != nil {
		t.Fatalf("SolidBlock pocket: %v", err)
	}
	res, err := brep.Boolean(brep.Difference, box, tool)
	if err != nil {
		t.Fatalf("brep.Boolean(brep.Difference) box−pocket: %v", err)
	}
	return res
}

// pocketCornerEdgeKeys returns the reference keys of the concave edges touching the pocket corner p (the
// vertical wall edge + the two floor edges), matched within the body's model-relative weld tolerance.
func pocketCornerEdgeKeys(t *testing.T, b *topo.Body, p math.Point3) [][]byte {
	t.Helper()
	eps := tol.ForBody(b).Weld()
	var keys [][]byte
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if (a.DistanceTo(p) < eps || c.DistanceTo(p) < eps) && ClassifyEdgeConvexity(e) == EdgeConcave {
			keys = append(keys, e.ReferenceKey())
		}
	}
	return keys
}

// assertVoidSphereCentre fails unless the body carries exactly one radius-5 corner sphere at want — the
// VOID-side octant point (proof the trihedral pass flipped the reflected material-side sphere).
func assertVoidSphereCentre(t *testing.T, b *topo.Body, want math.Point3, eps float64) {
	t.Helper()
	found := 0
	for _, f := range b.Faces() {
		sph, ok := f.Geometry().(geom.Sphere)
		if !ok || sph.Radius > 10 {
			continue
		}
		found++
		if d := sph.Center.DistanceTo(want); d > eps {
			t.Fatalf("corner sphere centre %v, want void-side %v (off by %.4f — sphere not flipped to the void side)", sph.Center, want, d)
		}
	}
	if found != 1 {
		t.Fatalf("body has %d corner spheres, want exactly 1 void-side octant patch", found)
	}
}

// verticalBandBottomZ returns the lowest vertex z of the radius-5 z-axis fillet cylinder (the vertical
// wall band), whose retracted corner rail marks the setback station.
func verticalBandBottomZ(t *testing.T, b *topo.Body) float64 {
	t.Helper()
	zlo := stdmath.Inf(1)
	for _, f := range b.Faces() {
		cy, ok := f.Geometry().(geom.Cylinder)
		if !ok || stdmath.Abs(cy.AxisDir.AsVector().Z) < 0.99 {
			continue
		}
		for _, v := range f.Vertices() {
			zlo = stdmath.Min(zlo, v.Point().Z)
		}
	}
	if stdmath.IsInf(zlo, 1) {
		t.Fatal("no vertical (z-axis) fillet cylinder in the pocket result")
	}
	return zlo
}

// threeOrthogonalPlanarFaces returns three real planar faces with mutually ORTHOGONAL outward normals
// (+x, +y, +z), each from its own unit box so the ids are distinct — the gate's orthogonal-triple input.
func threeOrthogonalPlanarFaces(t *testing.T) (fx, fy, fz *topo.Face) {
	t.Helper()
	return unitBoxFace(t, math.V3(1, 0, 0)), unitBoxFace(t, math.V3(0, 1, 0)), unitBoxFace(t, math.V3(0, 0, 1))
}

// unitBoxFace builds a fresh unit box and returns the planar face whose material-outward normal is n.
func unitBoxFace(t *testing.T, n math.Vector3) *topo.Face {
	t.Helper()
	b, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "b")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if ok && outwardPlaneNormal(f, pl).Dot(n) > 0.99 {
			return f
		}
	}
	t.Fatalf("unit box has no planar face with outward normal %v", n)
	return nil
}

// triArms builds up to three fake blend-corner edgeFillets sharing fixtureCornerVertex, with the given
// per-arm concavity (flip) — the gate reads only flip and the corner's blend/vertex/a/b (set by
// setTriArmFaces).
func triArms(t *testing.T, flips ...bool) []edgeFillet {
	t.Helper()
	out := make([]edgeFillet, len(flips))
	for i, flip := range flips {
		out[i] = edgeFillet{flip: flip, c1: corner{vertex: fixtureCornerVertex, blend: true}}
	}
	return out
}

// setTriArmFaces assigns the three faces to the arms PAIRWISE (arm i borders faces[i], faces[i+1]), so a
// full triple covers all three distinct faces — the real trihedral adjacency the gate collects. The
// edgeFillet a/b mirror the corner faces so cornerBandsAt (the mixed-sense classifier's band source)
// reads the same adjacency as blendCornerFaces (the concave-sphere gate's corner-face source).
func setTriArmFaces(fils []edgeFillet, faces [3]*topo.Face) {
	for i := range fils {
		fils[i].a, fils[i].c1.a = faces[i%3], faces[i%3]
		fils[i].b, fils[i].c1.b = faces[(i+1)%3], faces[(i+1)%3]
	}
}

// concaveArm is a fake concave (flip) constant edgeFillet whose cylinder axis is dir — the material half
// of an L1-class miter arm the gate inspects (it reads only flip/varying/cyl.AxisDir).
func concaveArm(t *testing.T, dir math.Vector3) edgeFillet {
	return armWithAxis(t, dir, true)
}

// convexArm is a fake convex (non-flip) constant edgeFillet — the already-correct miter the gate must skip.
func convexArm(t *testing.T, dir math.Vector3) edgeFillet {
	return armWithAxis(t, dir, false)
}

// armWithAxis builds a fake constant edgeFillet with a radius-5 cylinder along dir and the given flip,
// its c1 a miter corner at a shared fixture vertex (so two arms pair on the same corner id).
func armWithAxis(t *testing.T, dir math.Vector3, flip bool) edgeFillet {
	t.Helper()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), dir, 5)
	if err != nil {
		t.Fatalf("NewCylinder axis %v: %v", dir, err)
	}
	return edgeFillet{cyl: cyl, flip: flip, c1: corner{vertex: fixtureCornerVertex, miter: true}}
}

// fixtureCornerVertex is the single shared corner vertex the fake miter arms meet at (a package-level
// builder vertex — topo vertices are only minted through a Builder). Its identity pairs the two arms.
var fixtureCornerVertex = topo.NewBuilder(true, topo.NewLineage(topo.Tok("test", "corner", 0))).
	AddVertex(math.P3(80, 40, 100), topo.NewLineage(topo.Tok("test", "v", 0)))

// aPlanarFace returns a real planar topo face (a box side) for the gate's shared-face plane check.
func aPlanarFace(t *testing.T) *topo.Face {
	t.Helper()
	b, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "b")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Plane); ok {
			return f
		}
	}
	t.Fatal("SolidBlock has no planar face")
	return nil
}

// aCylindricalFace returns a real cylindrical topo face for the gate's non-planar-shared rejection.
func aCylindricalFace(t *testing.T) *topo.Face {
	t.Helper()
	b, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1, 2)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return f
		}
	}
	t.Fatal("SolidCylinder has no cylindrical face")
	return nil
}

// approxPoint reports whether two 2D points coincide within eps — the caller's model-relative weld
// tolerance (tol.ForBody().Weld()), not a bare epsilon.
func approxPoint(a, b math.Point2, eps float64) bool {
	return stdmath.Abs(a.X-b.X) < eps && stdmath.Abs(a.Y-b.Y) < eps
}
