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

// TestConcaveDihedralSetbackExtendsRailByR is the L1-class positive: a 100³ box + 40³ boss whose four
// concave base edges are filleted r=5. The dihedral corner-setback pass must EXTEND each miter band's
// shared-face rail by exactly r past the raw endpoint and re-trim the box-top plane to it, so the
// top-plane hole recedes from the raw 40×40 boss footprint to a 50×50 square [35,85]² (each side ±r).
// A dropped setback leaves the reflected-seam [45,75]² hole; a wrong distance moves the corners off ±r.
func TestConcaveDihedralSetbackExtendsRailByR(t *testing.T) {
	body := boxWithBoss(t)
	keys := concaveBaseEdgeKeys(body)
	if len(keys) != 4 {
		t.Fatalf("box+boss has %d concave base edges, want 4", len(keys))
	}
	res, err := FilletEdgesCorner(body, filletPicksFor(keys, 5), CornerMiter, FillConcaveOutward)
	if err != nil {
		t.Fatalf("fillet box+boss concave base edges: %v", err)
	}
	rep := Validate(res)
	if !rep.Valid || !rep.Closed || !rep.HolesContained || !res.IsSolid() {
		t.Fatalf("set-back result not a watertight solid: valid=%v closed=%v holes=%v solid=%v",
			rep.Valid, rep.Closed, rep.HolesContained, res.IsSolid())
	}
	eps := ResolutionForBody(res).Weld() // model-relative coincidence tolerance (M35), not a bare epsilon
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

// TestApplyCornerSetbackDeclinesConvex confirms the pass reports fired=false (leaving the caller's
// baseline body byte-identical) when the only miter is convex — the shared-path do-no-harm guarantee.
func TestApplyCornerSetbackDeclinesConvex(t *testing.T) {
	fils := []edgeFillet{convexArm(t, math.V3(1, 0, 0)), convexArm(t, math.V3(0, 1, 0))}
	fils[0].c1.miter, fils[1].c1.miter = true, true
	v := fils[0].c1.vertex
	miters := map[uint64]*cornerMiter{v.ID(): {shared: aPlanarFace(t), vertex: v}}
	if _, fired := applyCornerSetback(fils, miters); fired {
		t.Fatal("applyCornerSetback fired on a convex miter — the convex path must stay byte-identical")
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
	res, err := Boolean(Join, box, boss)
	if err != nil {
		t.Fatalf("Boolean(Join) box+boss: %v", err)
	}
	return res
}

// concaveBaseEdgeKeys returns the reference keys of the four concave horizontal base edges (the
// boss-wall-meets-box-top edges at z=100). The z coincidence uses the body's model-relative weld
// tolerance (M35), not a bare epsilon.
func concaveBaseEdgeKeys(b *topo.Body) [][]byte {
	eps := ResolutionForBody(b).Weld()
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
// tolerance (ResolutionForBody().Weld()), not a bare epsilon.
func approxPoint(a, b math.Point2, eps float64) bool {
	return stdmath.Abs(a.X-b.X) < eps && stdmath.Abs(a.Y-b.Y) < eps
}
