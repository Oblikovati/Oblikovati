// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestMixedTrihedralSetbackBuildsCornerTorus is the K9-class positive: the SAME 100³ box + 40³ boss as
// the dihedral tracer, but here the ONE mixed-sense trihedral corner at (80,40,100) is filleted — its
// two CONCAVE base edges (boss-wall∧box-top) plus the ONE CONVEX vertical boss edge. A mixed corner's
// rolling ball pivots AROUND the convex edge while staying tangent to the box top, so its centre traces
// a 90° arc of radius R=2r and the swept surface is a TORUS (axis = the convex fillet axis at (75,45),
// major R=2r=10, minor r=5), NOT the sphere solvePlanarBlend forces. The pass must build that torus
// (patch area (25π/2)(π−1)=84.10 for r=5), retract the three bands to its four contact arcs, and leave
// a watertight solid with ZERO corner spheres. A dropped torus leaves the oversized sphere (274.35).
func TestMixedTrihedralSetbackBuildsCornerTorus(t *testing.T) {
	t.Parallel()
	body := boxWithBoss(t)
	keys := edgeKeysAtVertex(body, math.P3(80, 40, 100))
	if len(keys) != 3 {
		t.Fatalf("mixed corner at (80,40,100) has %d edges, want 3 (2 concave base + 1 convex boss)", len(keys))
	}
	res, err := FilletEdgesCorner(body, filletPicksFor(keys, 5), CornerMiter, FillConcaveOutward)
	if err != nil {
		t.Fatalf("fillet mixed trihedral corner: %v", err)
	}
	rep := Validate(res)
	if !rep.Valid || !rep.Closed || !rep.HolesContained || !res.IsSolid() {
		t.Fatalf("mixed torus set-back result not a watertight solid: valid=%v closed=%v holes=%v solid=%v",
			rep.Valid, rep.Closed, rep.HolesContained, res.IsSolid())
	}
	assertCornerTorusNotSphere(t, res)
}

// assertCornerTorusNotSphere fails unless the body carries exactly one R=2r=10, minor=r=5 corner torus
// of trimmed area ≈84.10 =(25π/2)(π−1) AND zero corner spheres — the crux the P3 pass fixes (the
// baseline forces a radius-5 sphere of area 274.35). Areas use the body's own model scale for tolerance.
func assertCornerTorusNotSphere(t *testing.T, b *topo.Body) {
	t.Helper()
	eps := ResolutionForBody(b).Weld()
	tori, spheres := 0, 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Sphere); ok {
			spheres++
		}
		tor, ok := f.Geometry().(geom.Torus)
		if !ok {
			continue
		}
		tori++
		if stdmath.Abs(tor.MajorRadius-10) > eps || stdmath.Abs(tor.MinorRadius-5) > eps {
			t.Fatalf("corner torus major=%.6f minor=%.6f, want R=2r=10 minor=r=5", tor.MajorRadius, tor.MinorRadius)
		}
		if a := faceTriArea(f); stdmath.Abs(a-84.10) > 0.02*84.10 {
			t.Fatalf("corner torus trimmed area %.4f, want ≈84.10 (a forced sphere reads 274.35)", a)
		}
	}
	if tori != 1 || spheres != 0 {
		t.Fatalf("body has %d corner tori + %d spheres, want exactly 1 torus and 0 spheres (sphere not replaced?)", tori, spheres)
	}
}

// TestMixedSameSenseCornerStaysSphereNotTorus is the gate's real-body negative: the K6-class concave
// trihedral (box − pocket, three CONCAVE fillets) must NOT be recognised as a mixed corner — it keeps
// its sphere octant, never a torus. Proves the P3 gate declines a same-sense corner end-to-end.
func TestMixedSameSenseCornerStaysSphereNotTorus(t *testing.T) {
	t.Parallel()
	body := boxWithPocket(t)
	keys := pocketCornerEdgeKeys(t, body, math.P3(20, 20, 60))
	res, err := FilletEdgesCorner(body, filletPicksFor(keys, 5), CornerMiter, FillConcaveOutward)
	if err != nil {
		t.Fatalf("fillet concave trihedral corner: %v", err)
	}
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Torus); ok {
			t.Fatal("a same-sense (all-concave) trihedral corner produced a TORUS — the mixed-sense P3 gate leaked")
		}
	}
}

// TestMixedTorusGateBoundary pins the two predicates the P3 gate is built from: splitMixedSense (a MIXED
// sense split, either way round, with the minority-sense band as the pivot) and orthogonalPlanarTriple
// (three mutually-perpendicular planar hosts). Together they decline every non-mixed config — same-sense
// (all concave / all convex), a wrong valence, a non-orthogonal (parallel) triple, and a curved host — so
// each keeps its baseline sphere. Both mixed signatures are accepted and the pivot is asserted to be the
// odd one out: 2cc+1cvx pivots on the convex band (K9/M2/L6), 1cc+2cvx on the concave one (B5/C4/D7).
func TestMixedTorusGateBoundary(t *testing.T) {
	t.Parallel()
	fx, fy, fz := threeOrthogonalPlanarFaces(t)
	for _, tc := range []struct {
		name        string
		senses      []bool // per-band concavity (true = concave)
		wantMix     bool
		wantPivotCC bool // the pivot band's expected sense when wantMix
	}{
		{"mixed 2concave+1convex", []bool{true, true, false}, true, false},
		{"mixed 2convex+1concave", []bool{true, false, false}, true, true},
		{"same-sense all concave", []bool{true, true, true}, false, false},
		{"same-sense all convex", []bool{false, false, false}, false, false},
		{"wrong valence 2", []bool{true, false}, false, false},
		{"wrong valence 4", []bool{true, true, false, false}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pivot, pair, ok := splitMixedSense(bandsWithSenses(tc.senses))
			if ok != tc.wantMix {
				t.Fatalf("splitMixedSense ok=%v, want %v for %s", ok, tc.wantMix, tc.name)
			}
			if !ok {
				return
			}
			if pivot.concave != tc.wantPivotCC {
				t.Fatalf("pivot concave=%v, want %v (the pivot must be the MINORITY sense)", pivot.concave, tc.wantPivotCC)
			}
			if len(pair) != 2 || pair[0].concave != !tc.wantPivotCC || pair[1].concave != !tc.wantPivotCC {
				t.Fatalf("pair senses %v, want both %v", pair, !tc.wantPivotCC)
			}
		})
	}
	assertOrthoTriple(t, "orthogonal planar", []*topo.Face{fx, fy, fz}, true)
	assertOrthoTriple(t, "non-orthogonal parallel", []*topo.Face{fx, fy, fx}, false)
	assertOrthoTriple(t, "curved host", []*topo.Face{fx, fy, aCylindricalFace(t)}, false)
}

// assertOrthoTriple checks orthogonalPlanarTriple's verdict on a three-face set.
func assertOrthoTriple(t *testing.T, name string, faces []*topo.Face, want bool) {
	t.Helper()
	if got := orthogonalPlanarTriple(faces); got != want {
		t.Fatalf("orthogonalPlanarTriple=%v, want %v for %s", got, want, name)
	}
}

// bandsWithSenses builds fake cornerBands with the given concavity flags — the splitMixedSense input
// (it reads only the concave flag and the count).
func bandsWithSenses(senses []bool) []cornerBand {
	out := make([]cornerBand, len(senses))
	for i, c := range senses {
		out[i] = cornerBand{concave: c}
	}
	return out
}

// edgeKeysAtVertex returns the reference keys of every edge touching point p (within the body's
// model-relative weld tolerance) — the three converging edges of the mixed corner.
func edgeKeysAtVertex(b *topo.Body, p math.Point3) [][]byte {
	eps := ResolutionForBody(b).Weld()
	var keys [][]byte
	for _, e := range b.Edges() {
		if e.StartVertex().Point().DistanceTo(p) < eps || e.EndVertex().Point().DistanceTo(p) < eps {
			keys = append(keys, e.ReferenceKey())
		}
	}
	return keys
}

// faceTriArea sums the Property-quality tessellation triangle areas of face f — the trimmed corner
// patch's area (a torus patch has no closed-form area from the body alone; its mesh area is measured).
func faceTriArea(f *topo.Face) float64 {
	m := tessellate.TessellateFace(f, PropertyQuality())
	if m == nil {
		return 0
	}
	var area float64
	for i := 0; i+2 < len(m.Indices); i += 3 {
		p, q, r := m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]]
		area += 0.5 * p.VectorTo(q).Cross(p.VectorTo(r)).Length()
	}
	return area
}
