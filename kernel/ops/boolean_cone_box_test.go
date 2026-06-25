// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// cone ∩ box / cone − box with a box face PARALLEL to the cone axis now stay exact: the cut is the
// analytic hyperbola the plane carves from the cone, not faceted CSG (Oblikovati/Oblikovati#1372).
// The frustum has tanα=0.3 (bottom r=3 at z=0, top r=6 at z=10); the box face x=2 (|D|=2 < bottom r)
// cuts every cross-section, the arc-band case.

func coneBoxFrustum(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6, "frustum")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	return b
}

// TestConeIntersectBoxExact keeps the +x wedge of the frustum (x ≥ 2): an exact cone arc-band face,
// two cap segments and the planar lid, watertight, with no faceted soup.
func TestConeIntersectBoxExact(t *testing.T) {
	frustum := coneBoxFrustum(t)
	box, err := brep.SolidBlock(math.P3(2, -20, -5), math.P3(20, 20, 15), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	res, err := ops.Boolean(ops.Intersect, frustum, box)
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	assertConeBoxExact(t, res)
}

// TestConeCutBoxExact removes the +x wedge (the box spans the frustum on every side but x), leaving
// the cone minus a flat — still an exact cone arc-band face.
func TestConeCutBoxExact(t *testing.T) {
	frustum := coneBoxFrustum(t)
	box, err := brep.SolidBlock(math.P3(2, -20, -5), math.P3(20, 20, 15), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	res, err := ops.Boolean(ops.Cut, frustum, box)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	assertConeBoxExact(t, res)
}

// assertConeBoxExact pins the result on the exact analytic path: a watertight solid carrying at least
// one analytic cone face and no non-analytic (tessellated) face.
func assertConeBoxExact(t *testing.T, res *topo.Body) {
	t.Helper()
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("result is not a watertight solid: %+v", v)
	}
	cones, nonAnalytic := 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone:
			cones++
		case geom.Plane, geom.Cylinder, geom.Sphere, geom.Torus:
		default:
			nonAnalytic++
		}
	}
	if cones == 0 {
		t.Errorf("result has no analytic cone face across %d faces — it fell back to CSG", len(res.Faces()))
	}
	if nonAnalytic > 0 {
		t.Errorf("result has %d non-analytic faces — not the exact path", nonAnalytic)
	}
}

// TestConeIntersectBoxVolume cross-checks the kept +x wedge volume against a dense numeric integral of
// the frustum cross-section area beyond x=2 (independent of the B-rep), so a wrong cut shape is caught.
func TestConeIntersectBoxVolume(t *testing.T) {
	frustum := coneBoxFrustum(t)
	box, _ := brep.SolidBlock(math.P3(2, -20, -5), math.P3(20, 20, 15), "box")
	res, err := ops.Boolean(ops.Intersect, frustum, box)
	if err != nil {
		t.Fatalf("Boolean: %v", err)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := frustumCapBeyondPlaneVolume(0.3, 0, 10, 2)
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("kept wedge volume %.4f vs numeric %.4f (rel %.4f > 0.03)", got, want, rel)
	}
}

// Vertex-inside-band cone ∩/− box (Oblikovati/Oblikovati#1374): the box face x=4 has |D|=4 between the
// bottom radius (3) and top radius (6), so the flat fades out before the small rim. Intersect keeps a
// single tongue narrowing to the hyperbola vertex; Cut keeps an annulus notched down to that vertex.
// Both must stay exact (an analytic cone face, no faceted soup), not fall back to CSG.

func coneVertexInsideBox(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(math.P3(4, -20, -5), math.P3(20, 20, 15), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}

// TestConeIntersectBoxVertexInsideExact keeps the +x tongue (x ≥ 4) — an exact cone face narrowing to
// the hyperbola vertex, with the small rim wholly dropped.
func TestConeIntersectBoxVertexInsideExact(t *testing.T) {
	res, err := ops.Boolean(ops.Intersect, coneBoxFrustum(t), coneVertexInsideBox(t))
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	assertConeBoxExact(t, res)
}

// TestConeCutBoxVertexInsideExact removes the +x tongue, leaving the cone minus a flat that fades out
// before the small rim — an exact notched-annulus cone face.
func TestConeCutBoxVertexInsideExact(t *testing.T) {
	res, err := ops.Boolean(ops.Cut, coneBoxFrustum(t), coneVertexInsideBox(t))
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	assertConeBoxExact(t, res)
}

// TestConeVertexInsideVolumes cross-checks both pieces against the independent numeric integral: the
// kept tongue equals the cross-section area beyond x=4, and the notched complement equals the full
// frustum (210π) minus that tongue. The tongue pinches to the hyperbola vertex, a sharply-curved cusp
// whose chord error at display density is ~3%; it converges to the analytic value as the mesh refines
// (proving the B-rep is exact, not just close), so this verifies the cut SHAPE at a fine quality where
// chord error is negligible — the geometry is right, independent of display tessellation density.
func TestConeVertexInsideVolumes(t *testing.T) {
	tongue := frustumCapBeyondPlaneVolume(0.3, 0, 10, 4)
	full := 210 * stdmath.Pi
	fine := ops.Quality{ChordTolerance: 0.005, AngleTolerance: 5 * stdmath.Pi / 180}
	cases := []struct {
		name string
		op   ops.PartFeatureOperation
		want float64
	}{
		{"intersect tongue", ops.Intersect, tongue},
		{"cut notched annulus", ops.Cut, full - tongue},
	}
	for _, c := range cases {
		res, err := ops.Boolean(c.op, coneBoxFrustum(t), coneVertexInsideBox(t))
		if err != nil {
			t.Fatalf("%s: Boolean: %v", c.name, err)
		}
		got := ops.BodyGeometryProperties(res, fine).Volume
		if rel := stdmath.Abs(got-c.want) / c.want; rel > 0.01 {
			t.Errorf("%s volume %.4f vs numeric %.4f (rel %.4f > 0.01)", c.name, got, c.want, rel)
		}
	}
}

// Oblique cone ∩/− box (Oblikovati/Oblikovati#1375): a frustum tilted so its axis is (0,0.6,0.8) cut by
// the axis-aligned box face z=4 — a plane tilted steeper than the generators, so the section is a closed
// ELLIPSE wholly within the band. Both directions must stay exact (an analytic cone face, the elliptical
// cut edge), not fall back to CSG, and match OCC getMass.

func obliqueConeBoxFrustum(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 6, 8), 3, 6, "frustum") // axis (0,0.6,0.8)
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	return b
}

func obliqueConeBox(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(math.P3(-20, -20, 4), math.P3(20, 20, 30), "box") // only the z=4 face cuts
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}

// TestConeIntersectBoxObliqueExact keeps the frustum above z=4 — an exact cone band bounded below by the
// elliptical lid, no faceted soup.
func TestConeIntersectBoxObliqueExact(t *testing.T) {
	res, err := ops.Boolean(ops.Intersect, obliqueConeBoxFrustum(t), obliqueConeBox(t))
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	assertConeBoxExact(t, res)
	if !resultHasEllipticalEdge(res) {
		t.Error("result carries no elliptical edge — the oblique cut was not imprinted as an ellipse")
	}
}

// TestConeCutBoxObliqueExact removes the frustum above z=4, leaving the lower piece — also exact.
func TestConeCutBoxObliqueExact(t *testing.T) {
	res, err := ops.Boolean(ops.Cut, obliqueConeBoxFrustum(t), obliqueConeBox(t))
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	assertConeBoxExact(t, res)
	if !resultHasEllipticalEdge(res) {
		t.Error("result carries no elliptical edge — the oblique cut was not imprinted as an ellipse")
	}
}

// resultHasEllipticalEdge reports whether any edge of the body stores an analytic ellipse/elliptical arc.
func resultHasEllipticalEdge(b *topo.Body) bool {
	for _, e := range b.Edges() {
		switch e.Geometry().(type) {
		case geom.EllipseFull, geom.EllipticalArc:
			return true
		}
	}
	return false
}

// Oblique cone ∩/− box, HYPERBOLIC tilt (Oblikovati/Oblikovati#1375): a frustum tilted so its axis is
// (0.2,0,0.98) — shallower than the box's x=2 face relative to that axis — so the section is a HYPERBOLA
// (vertex below the band, arms crossing both rims). Both directions must stay exact (an analytic cone
// face, the hyperbolic cut edge), not fall back to CSG, and match OCC getMass. This is also the case that
// composes only because the cap-seam crossing fix (the near-axial plane slices both caps through centre).

func obliqueHyperbolaFrustum(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(2, 0, 9.797958971), 3, 6, "frustum") // axis (0.2,0,0.98), h=10
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	return b
}

func obliqueHyperbolaBox(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(math.P3(2, -20, -20), math.P3(40, 20, 40), "box") // only the x=2 face cuts
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}

// TestConeIntersectBoxObliqueHyperbolaExact keeps the +x wedge (x ≥ 2) of the tilted frustum — an exact
// cone arc-band bounded by the hyperbola arms, no faceted soup.
func TestConeIntersectBoxObliqueHyperbolaExact(t *testing.T) {
	res, err := ops.Boolean(ops.Intersect, obliqueHyperbolaFrustum(t), obliqueHyperbolaBox(t))
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	assertConeBoxExact(t, res)
	if !resultHasHyperbolicEdge(res) {
		t.Error("result carries no hyperbolic edge — the oblique cut was not imprinted as a hyperbola")
	}
}

// TestConeCutBoxObliqueHyperbolaExact removes the +x wedge, leaving the rest — also exact.
func TestConeCutBoxObliqueHyperbolaExact(t *testing.T) {
	res, err := ops.Boolean(ops.Cut, obliqueHyperbolaFrustum(t), obliqueHyperbolaBox(t))
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	assertConeBoxExact(t, res)
	if !resultHasHyperbolicEdge(res) {
		t.Error("result carries no hyperbolic edge — the oblique cut was not imprinted as a hyperbola")
	}
}

// resultHasHyperbolicEdge reports whether any edge of the body stores an analytic hyperbolic arc.
func resultHasHyperbolicEdge(b *topo.Body) bool {
	for _, e := range b.Edges() {
		if _, ok := e.Geometry().(geom.HyperbolicArc); ok {
			return true
		}
	}
	return false
}

// Parabolic boundary-tilt cone ∩/− box (Oblikovati/Oblikovati#1375): a frustum whose axis is tilted by
// its own half-angle has one vertical generator, so the box's x=2 face is PARALLEL to it — the section is
// a parabola (the limit between the elliptic and hyperbolic tilts). Both directions must stay exact (an
// analytic cone face, the parabolic cut edge) and match OCC getMass.

func parabolaFrustum(t *testing.T) *topo.Body {
	t.Helper()
	a := stdmath.Atan(0.3)
	top := math.P3(math.Scalar(stdmath.Sin(a)*10), 0, math.Scalar(stdmath.Cos(a)*10)) // axis tilted by α
	b, err := brep.SolidCylinderCone(math.P3(0, 0, 0), top, 3, 6, "frustum")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	return b
}

func parabolaBox(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(math.P3(2, -20, -20), math.P3(40, 20, 40), "box") // only the x=2 face cuts
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}

// TestConeIntersectBoxParabolaExact keeps the +x wedge of the tilted frustum — an exact cone arc-band
// bounded by the parabola arms.
func TestConeIntersectBoxParabolaExact(t *testing.T) {
	res, err := ops.Boolean(ops.Intersect, parabolaFrustum(t), parabolaBox(t))
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	assertConeBoxExact(t, res)
	if !resultHasParabolicEdge(res) {
		t.Error("result carries no parabolic edge — the boundary-tilt cut was not imprinted as a parabola")
	}
}

// TestConeCutBoxParabolaExact removes the +x wedge — also exact.
func TestConeCutBoxParabolaExact(t *testing.T) {
	res, err := ops.Boolean(ops.Cut, parabolaFrustum(t), parabolaBox(t))
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	assertConeBoxExact(t, res)
	if !resultHasParabolicEdge(res) {
		t.Error("result carries no parabolic edge — the boundary-tilt cut was not imprinted as a parabola")
	}
}

// resultHasParabolicEdge reports whether any edge of the body stores an analytic parabolic arc.
func resultHasParabolicEdge(b *topo.Body) bool {
	for _, e := range b.Edges() {
		if _, ok := e.Geometry().(geom.ParabolicArc); ok {
			return true
		}
	}
	return false
}

// frustumCapBeyondPlaneVolume integrates the area of each circular cross-section lying at x ≥ xCut for
// a cone of slope tanα over z∈[z0,z1] (apex at z=z0−r0/tanα). A circle of radius r centred on the axis
// has area r²·(θ − sinθ·cosθ) beyond a chord at distance xCut, θ = arccos(xCut/r) (0 when xCut ≥ r).
func frustumCapBeyondPlaneVolume(tanA, z0, z1, xCut float64) float64 {
	const steps = 20000
	sum := 0.0
	for i := 0; i < steps; i++ {
		z := z0 + (z1-z0)*(float64(i)+0.5)/steps
		r := (z - (z0 - 3/tanA)) * tanA // radius grows linearly; bottom r=3 at z0
		sum += circleAreaBeyondChord(r, xCut)
	}
	return sum * (z1 - z0) / steps
}

func circleAreaBeyondChord(r, x float64) float64 {
	if x >= r {
		return 0
	}
	theta := stdmath.Acos(x / r)
	return r * r * (theta - stdmath.Sin(theta)*stdmath.Cos(theta))
}
