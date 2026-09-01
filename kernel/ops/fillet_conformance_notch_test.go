// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestSimpleLoop2D covers the conformance-repair guard predicate: the boundary that broke it (the
// notched-face fillet loop, projected) self-intersects and must be rejected; a plain convex quad
// and a concave-but-simple polygon must be accepted.
func TestSimpleLoop2D(t *testing.T) {
	t.Parallel()
	// The real self-intersecting loop from the notched-box fillet (edge z=90,x∈[80,90] crosses
	// edge x=85,z∈[0,100] at (85,90)).
	selfInt := []math.Point2{math.P2(80, 100), math.P2(80, 90), math.P2(90, 90), math.P2(90, 100), math.P2(85, 100), math.P2(85, 0), math.P2(0, 0), math.P2(0, 100)}
	if tessellate.SimpleLoop2D(selfInt) {
		t.Error("self-intersecting loop reported as simple")
	}
	square := []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 10), math.P2(0, 10)}
	if !tessellate.SimpleLoop2D(square) {
		t.Error("convex square reported as non-simple")
	}
	// An L-shape (concave but simple) must pass.
	ell := []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 4), math.P2(4, 4), math.P2(4, 10), math.P2(0, 10)}
	if !tessellate.SimpleLoop2D(ell) {
		t.Error("simple concave L reported as non-simple")
	}
}

// TestConformancePreservesNotchedFaceArea regresses the body-tessellation collapse (corpus
// simple/Y2). Filleting the top edge of a notched prism (r=15) produces a planar face whose
// boundary self-intersects (a protruding notch pokes into the removed strip — the still-open Bug B
// in FilletEdges). conformCylConeFaces re-meshed that face with the boundary-faithful CDT, which
// collapsed it 8475→675 and dropped the whole body's measured area 61147→53337 (−12.6%), even
// though the B-rep and the per-face meshes are correct (OCCT area 61050). The guard keeps the robust
// initial mesh instead of the collapsed one; the measured body area must match OCCT within 1% and
// must equal the sum of the per-face tessellations (no face collapsed). Must never regress; when
// Bug B lands the loop becomes simple, the guard no longer fires, and this assertion still holds.
func TestConformancePreservesNotchedFaceArea(t *testing.T) {
	t.Parallel()
	body := importNotchedPrism(t)
	edge := notchTopEdge(t, body) // the filleted top edge (0,0,100)-(100,0,100)
	res, err := FilletEdges(body, [][]byte{edge.ReferenceKey()}, 15)
	if err != nil {
		t.Fatalf("fillet: %v", err)
	}
	q := PropertyQuality()
	bodyArea := BodyGeometryProperties(res, q).Area
	var faceSum float64
	for _, f := range res.Faces() {
		faceSum += y2FaceArea(tessellate.TessellateFace(f, q))
	}
	if rel := (bodyArea - 61050.1) / 61050.1; rel < -0.01 || rel > 0.01 {
		t.Fatalf("body area %.5g not within 1%% of OCCT 61050.1 (rel %+.2f%%) — conformance collapse regressed", bodyArea, rel*100)
	}
	if rel := (bodyArea - faceSum) / faceSum; rel < -0.005 || rel > 0.005 {
		t.Fatalf("body area %.5g diverges from per-face sum %.5g (rel %+.2f%%) — a face collapsed at body level", bodyArea, faceSum, rel*100)
	}
}

func importNotchedPrism(t *testing.T) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "box_notch_fillet.step"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import: %v (n=%d)", err, len(bodies))
	}
	return bodies[0]
}

// notchTopEdge returns the straight top edge (0,0,100)-(100,0,100) that the fillet rounds.
func notchTopEdge(t *testing.T, b *topo.Body) *topo.Edge {
	t.Helper()
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if (a.DistanceTo(math.P3(0, 0, 100)) < 1e-4 && c.DistanceTo(math.P3(100, 0, 100)) < 1e-4) ||
			(a.DistanceTo(math.P3(100, 0, 100)) < 1e-4 && c.DistanceTo(math.P3(0, 0, 100)) < 1e-4) {
			return e
		}
	}
	t.Fatal("top edge (0,0,100)-(100,0,100) not found")
	return nil
}

func y2FaceArea(m *Mesh) float64 {
	var a float64
	for k := 0; k+2 < len(m.Indices); k += 3 {
		p, q, r := m.Positions[m.Indices[k]], m.Positions[m.Indices[k+1]], m.Positions[m.Indices[k+2]]
		a += float64(p.VectorTo(q).Cross(p.VectorTo(r)).Length()) / 2
	}
	return a
}
