// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestG1Cluster1cTriage locks the diagnosis that routed the four G1 "restored solid" area-parity
// cases (simple/{W2,H6,Y2,Q1}) to their true owners, so the reclassification is evidence-backed
// and cannot silently rot (roadmap 2026-07-11-occt-blend-greening-roadmap.md, "G1 1c triage"):
//
//   - W2, H6 → G6 (curved neighbour): the picked edge is a geom.Arc3d bordering a geom.Cylinder,
//     so this is a fillet-adjacent-to-a-curved-face blend, not a planar defect. Stays FAIL(area)
//     on the gate until G6 builds the curved-neighbour corner.
//   - Y2 → a body-tessellation bug, NOT a fillet defect: the built B-rep is correct (sum of the
//     per-face tessellations is within 0.16% of OCCT's 61050), but ops.BodyGeometryProperties
//     under-measures the assembled body by ~12.6%. Split out as a tessellation issue (CLAUDE.md:
//     tessellation correctness outranks feature work); the corpus gate measures via the body path,
//     so this case fails on measurement, not geometry.
//   - Q1 → a genuine planar single-edge fillet error (+3.41%), kept as the lone G1 residual: the
//     picked edge and both its endpoints are incident only to planes (no curved neighbour), on an
//     irregular non-axis-aligned prism, so the defect is in the planar fillet run-out itself.
//
// This test asserts the load-bearing facts behind each route (edge/neighbour kinds; the Y2
// per-face-vs-body split). It does NOT assert OCCT parity — that is the gate's job, and these
// stay red there until their owning package fixes them.
func TestG1Cluster1cTriage(t *testing.T) {
	dir := CorpusFixtureDir()

	t.Run("W2_H6_curved_neighbour_to_G6", func(t *testing.T) {
		for _, c := range []string{"W2", "H6"} {
			e := locate1c(t, dir, c)
			assertEdgeIsArc(t, c, e)
			if !hasCylinderNeighbour(e) {
				t.Errorf("simple/%s: expected a cylinder neighbour face (curved-neighbour → G6), got %v",
					c, neighbourKinds(e))
			}
		}
	})

	t.Run("Y2_is_body_tessellation_bug_not_fillet", func(t *testing.T) {
		e := locate1c(t, dir, "Y2")
		res := fillet1c(t, dir, "Y2", e)
		sum := sumFaceTess(res)
		body := ops.BodyGeometryProperties(res, ops.PropertyQuality()).Area
		occt := caseArea(t, "Y2")
		// The geometry is right (per-face sum ≈ OCCT) but the body tessellation drops area.
		if rel := (sum - occt) / occt; rel < -0.01 || rel > 0.01 {
			t.Errorf("Y2: per-face-tess sum %.5g not within 1%% of OCCT %.5g (rel %+.2f%%) — geometry, not just measurement, is off", sum, occt, rel*100)
		}
		if body >= sum*0.99 {
			t.Errorf("Y2: body-tess %.5g no longer under-measures the correct per-face sum %.5g — tessellation bug may be FIXED; re-evaluate the split", body, sum)
		}
	})

	t.Run("Q1_is_genuine_planar_fillet_residual", func(t *testing.T) {
		e := locate1c(t, dir, "Q1")
		if hasCylinderNeighbour(e) {
			t.Errorf("Q1: picked edge unexpectedly has a curved neighbour %v — would reroute to G6", neighbourKinds(e))
		}
		for _, v := range []*topo.Vertex{e.StartVertex(), e.EndVertex()} {
			if vertexTouchesCurved(v) {
				t.Errorf("Q1: endpoint %v touches a curved face — the fillet run-out is a curved-neighbour case, reroute to G6", v.Point())
			}
		}
	})
}

// locate1c imports a simple/<case> fixture and locates its single picked edge.
func locate1c(t *testing.T, dir, c string) *topo.Edge {
	t.Helper()
	var rec Record
	for _, r := range Corpus() {
		if r.Grid == "simple" && r.Case == c {
			rec = r
		}
	}
	body, err := importInput(filepath.Join(dir, rec.InputStep))
	if err != nil {
		t.Fatalf("simple/%s import: %v", c, err)
	}
	e, err := locateEdge(body, rec.Picks[0].Locator, importTol(body))
	if err != nil {
		t.Fatalf("simple/%s locate: %v", c, err)
	}
	return e
}

// fillet1c fillets the located edge at the case's recorded radius.
func fillet1c(t *testing.T, dir, c string, e *topo.Edge) *topo.Body {
	t.Helper()
	var rec Record
	for _, r := range Corpus() {
		if r.Grid == "simple" && r.Case == c {
			rec = r
		}
	}
	body, _ := importInput(filepath.Join(dir, rec.InputStep))
	e2, _ := locateEdge(body, rec.Picks[0].Locator, importTol(body))
	res, err := ops.FilletEdges(body, [][]byte{e2.ReferenceKey()}, rec.Picks[0].Radius)
	if err != nil {
		t.Fatalf("simple/%s fillet: %v", c, err)
	}
	return res
}

func caseArea(t *testing.T, c string) float64 {
	t.Helper()
	for _, r := range Corpus() {
		if r.Grid == "simple" && r.Case == c {
			return r.ExpectedArea
		}
	}
	t.Fatalf("no record simple/%s", c)
	return 0
}

// assertEdgeIsArc fails unless e's curve is a non-linear (arc/circle) 3D curve.
func assertEdgeIsArc(t *testing.T, c string, e *topo.Edge) {
	t.Helper()
	kind := fmt.Sprintf("%T", e.Geometry())
	if kind == "geom.LineSegment" || kind == "geom.Line" {
		t.Errorf("simple/%s: expected a curved (arc) edge for a curved-neighbour case, got %s", c, kind)
	}
}

func neighbourKinds(e *topo.Edge) []string {
	kinds := make([]string, 0, 2)
	for _, f := range e.Faces() {
		kinds = append(kinds, fmt.Sprintf("%T", f.Geometry()))
	}
	return kinds
}

func hasCylinderNeighbour(e *topo.Edge) bool {
	for _, k := range neighbourKinds(e) {
		if k == "geom.Cylinder" || k == "geom.Cone" || k == "geom.Sphere" || k == "geom.Torus" {
			return true
		}
	}
	return false
}

func vertexTouchesCurved(v *topo.Vertex) bool {
	for _, e := range v.Edges() {
		if hasCylinderNeighbour(e) {
			return true
		}
	}
	return false
}

func sumFaceTess(b *topo.Body) float64 {
	var sum float64
	for _, f := range b.Faces() {
		m := ops.TessellateFace(f, ops.PropertyQuality())
		for k := 0; k+2 < len(m.Indices); k += 3 {
			p, q, r := m.Positions[m.Indices[k]], m.Positions[m.Indices[k+1]], m.Positions[m.Indices[k+2]]
			sum += float64(p.VectorTo(q).Cross(p.VectorTo(r)).Length()) / 2
		}
	}
	return sum
}
