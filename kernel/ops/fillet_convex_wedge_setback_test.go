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

// TestConvexWedgeSetbackDistanceIsRCotHalfTheta validates the derivation formula the convex corner's
// material-side sphere realizes (geometry-derivation §1/§4): two filleted edges meeting at interior
// angle θ in their shared host plane set the band back by s = r·cot(θ/2). An ORTHOGONAL box corner
// (θ=90°) gives s=r exactly; A8's WEDGE (top-plane edge pair, cos θ = 0.37139) gives the non-orthogonal
// s ≈ 14.770 the root-cause named ("A8 sets back differently"). This pins the identity so a future
// non-orthogonal corner slice cannot silently regress the s=r orthogonal case.
func TestConvexWedgeSetbackDistanceIsRCotHalfTheta(t *testing.T) {
	const r = 10.0
	for _, tc := range []struct {
		name  string
		cos   float64 // cosine of the interior edge-pair angle θ in the shared plane
		wantS float64
	}{
		{"orthogonal-box-theta-90", 0, r},           // θ=90° ⇒ cot45° = 1 ⇒ s = r
		{"a8-wedge-top-edge-pair", 0.37139, 14.770}, // A8's non-orthogonal wedge ⇒ s > r
	} {
		s := r * cotHalfFromCos(tc.cos)
		if stdmath.Abs(s-tc.wantS) > 1e-3 {
			t.Fatalf("%s: s = r·cot(θ/2) = %.4f, want %.4f", tc.name, s, tc.wantS)
		}
	}
}

// cotHalfFromCos returns cot(θ/2) from cos θ via the half-angle identity cot(θ/2)=√((1+cosθ)/(1−cosθ)).
func cotHalfFromCos(c float64) float64 { return stdmath.Sqrt((1 + c) / (1 - c)) }

// TestAllBodyFacesPlanarSeparatesWedgeFromCurved pins the scope boundary that keeps the pass on
// polyhedral wedges (A8/A6) and OFF curved-host bodies (F7's elliptical prism, whose oblique band runs
// off against a curved neighbour so the flat-plane pierce would move it AWAY from OCCT).
func TestAllBodyFacesPlanarSeparatesWedgeFromCurved(t *testing.T) {
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(100, 100, 100), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	if !allBodyFacesPlanar(box) {
		t.Fatal("a box body must be all-planar (the wedge scope)")
	}
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 20, 50)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	if allBodyFacesPlanar(cyl) {
		t.Fatal("a cylinder body must NOT be all-planar (curved-host exclusion, F7)")
	}
}

// TestConvexWedgeSenseAndPlanarPredicates pins the two per-corner discriminants: allFacesPlanar accepts
// a planar triple and rejects a curved face; anyConcaveBand fires the moment ANY band is concave (so a
// mixed-sense corner — the P3 torus — never reaches this convex-only pass).
func TestConvexWedgeSenseAndPlanarPredicates(t *testing.T) {
	box, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(100, 100, 100), "box")
	planes := wedgePlanarTriple(box)
	if !allFacesPlanar(planes) {
		t.Fatal("three box faces must be all-planar")
	}
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 20, 50)
	if allFacesPlanar([]*topo.Face{cylinderSideFace(t, cyl)}) {
		t.Fatal("a cylinder side face must fail allFacesPlanar")
	}
	if anyConcaveBand([]cornerBand{{concave: false}, {concave: false}, {concave: false}}) {
		t.Fatal("three convex bands are same-sense CONVEX, not concave")
	}
	if !anyConcaveBand([]cornerBand{{concave: false}, {concave: true}, {concave: false}}) {
		t.Fatal("a mixed-sense corner (one concave band) must be rejected by the convex-only pass")
	}
}

// TestEndOvershootsRunoffDetectsTab pins the load-bearing overshoot guard — the discriminant between
// A8/A6 (an OBLIQUE band overshoots its run-off plane by a tab OUTSIDE the body → clip) and a case whose
// rail already sits inside the plane (no clip, byte-identical). A blend end has no run-off plane at all.
func TestEndOvershootsRunoffDetectsTab(t *testing.T) {
	box, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(100, 100, 100), "box")
	xf := planeFaceWithOutwardNormal(t, box, math.V3(-1, 0, 0)) // x=0 face, material x>0 ⇒ outward (−1,0,0)
	v := aVertexOn(xf)
	base := corner{a: xf, b: xf, endFace: xf, vertex: v}

	tab := base
	tab.ta, tab.tb = math.P3(-5, 50, 50), math.P3(0, 50, 50) // ta at x=−5 is OUTSIDE the x=0 plane
	if !endOvershootsRunoff(&tab) {
		t.Fatal("a rail at x=−5 past the x=0 face is a tab and must be flagged")
	}
	inside := base
	inside.ta, inside.tb = math.P3(5, 50, 50), math.P3(5, 60, 50) // both inside the body
	if endOvershootsRunoff(&inside) {
		t.Fatal("a rail inside the body must NOT read as a tab (keeps the case byte-identical)")
	}
	blendEnd := tab
	blendEnd.blend = true // opposFarPlane rejects a blend corner: no run-off plane
	if endOvershootsRunoff(&blendEnd) {
		t.Fatal("a blend end has no run-off plane and must never overshoot")
	}
}

// TestObliqueRunoffRailClipsPerpendicularNoop pins the clip primitive on A8's exact numbers: the
// oblique band's overshooting top rail (−9.285,96.286,100) slides along its axis to PIERCE the x=0
// run-off plane at (0,73.07,100) — OCCT's station — while a PERPENDICULAR box rail already on the plane
// does not move (t=0, the s=r orthogonal termination). railPierce is the shared runout primitive the
// convex-wedge pass reuses.
func TestObliqueRunoffRailClipsPerpendicularNoop(t *testing.T) {
	n := math.V3(-1, 0, 0) // outward normal of the x=0 run-off plane
	q := math.P3(0, 0, 0)  // a point on it

	perp, ok := railPierce(math.P3(0, 50, 50), math.V3(1, 0, 0), q, n) // axis ⊥ plane, rail already on it
	if !ok || perp.DistanceTo(math.P3(0, 50, 50)) > 1e-9 {
		t.Fatalf("perpendicular rail on the plane must not move: got %v (ok=%v)", perp, ok)
	}
	obl, ok := railPierce(math.P3(-9.285, 96.286, 100), math.V3(-0.371391, 0.928477, 0), q, n)
	if !ok || obl.DistanceTo(math.P3(0, 73.074, 100)) > 1e-2 {
		t.Fatalf("oblique tab rail must clip to the x=0 pierce (0,73.07,100): got %v (ok=%v)", obl, ok)
	}
}

// planarFaces returns up to n planar faces of the body.
func wedgePlanarTriple(b *topo.Body) []*topo.Face {
	var out []*topo.Face
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Plane); ok {
			out = append(out, f)
			if len(out) == 3 {
				break
			}
		}
	}
	return out
}

// cylinderSideFace returns the body's cylindrical side face.
func cylinderSideFace(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return f
		}
	}
	t.Fatal("body has no cylindrical side face")
	return nil
}

// planeFaceWithOutwardNormal returns the planar face whose material-outward normal matches n.
func planeFaceWithOutwardNormal(t *testing.T, b *topo.Body, n math.Vector3) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		nf, ok := planeNormal(f)
		if ok && nf.Dot(n) > 0.999 {
			return f
		}
	}
	t.Fatalf("body has no planar face with outward normal %v", n)
	return nil
}

// aVertexOn returns any vertex of the face's first loop (it lies on the face's plane).
func aVertexOn(f *topo.Face) *topo.Vertex {
	return f.Loops()[0].EdgeUses()[0].Edge().StartVertex()
}
