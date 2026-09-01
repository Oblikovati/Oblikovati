// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// TestG1Cluster1cTriage records where the four G1 "restored solid" area-parity cases
// (simple/{W2,H6,Y2,Q1}) ended up (roadmap 2026-07-11-occt-blend-greening-roadmap.md). Y2 and Q1
// are now FIXED; W2/H6 are genuinely G6. The subtests assert the current, load-bearing facts.
//
//   - W2, H6 → G6 (curved neighbour): the picked edge is a geom.Arc3d bordering a geom.Cylinder —
//     a fillet ON a curved-face-adjacent edge, not a planar defect. Still FAIL(area) until G6.
//   - Y2 → FIXED by the conformance-repair guard (Bug A). The B-rep was always correct (per-face
//     tess sum ≈ OCCT); query.BodyGeometryProperties used to under-measure it ~12.6% because the
//     boundary-faithful CDT collapsed a face on a self-intersecting loop. Now body ≈ sum ≈ OCCT.
//     (The self-intersecting loop itself — Bug B — is a tracked, harmless topology blemish.)
//   - Q1 → FIXED by survivorCurve. Its PICKED edge is planar (asserted below), which is exactly why
//     the first read "planar run-out bug" was wrong: the real defect was a curved SURVIVOR edge on
//     the end-cap face (bordering the solid's cylinder) being straightened. Now matches OCCT.
func TestG1Cluster1cTriage(t *testing.T) {
	t.Parallel()
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

	t.Run("Y2_fixed_body_tess_matches_geometry", func(t *testing.T) {
		e := locate1c(t, dir, "Y2")
		res := fillet1c(t, dir, "Y2", e)
		sum := sumFaceTess(res)
		body := query.BodyGeometryProperties(res, ops.PropertyQuality()).Area
		occt := caseArea(t, "Y2")
		if rel := (body - occt) / occt; rel < -0.01 || rel > 0.01 {
			t.Errorf("Y2: body area %.5g not within 1%% of OCCT %.5g (rel %+.2f%%) — Bug A conformance guard regressed", body, occt, rel*100)
		}
		if rel := (body - sum) / sum; rel < -0.005 || rel > 0.005 {
			t.Errorf("Y2: body-tess %.5g diverges from per-face sum %.5g — a face collapsed again", body, sum)
		}
	})

	t.Run("Q1_fixed_picked_edge_was_planar", func(t *testing.T) {
		e := locate1c(t, dir, "Q1")
		res := fillet1c(t, dir, "Q1", e)
		// The picked edge is planar (defect was on a survivor edge elsewhere, not this run-out).
		if hasCylinderNeighbour(e) {
			t.Errorf("Q1: picked edge unexpectedly has a curved neighbour %v", neighbourKinds(e))
		}
		for _, v := range []*topo.Vertex{e.StartVertex(), e.EndVertex()} {
			if vertexTouchesCurved(v) {
				t.Errorf("Q1: picked-edge endpoint %v touches a curved face", v.Point())
			}
		}
		body := query.BodyGeometryProperties(res, ops.PropertyQuality()).Area
		if occt := caseArea(t, "Q1"); (body-occt)/occt < -0.01 || (body-occt)/occt > 0.01 {
			t.Errorf("Q1: body area %.5g not within 1%% of OCCT %.5g — survivorCurve fix regressed", body, occt)
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
	res, err := blend.FilletEdges(body, [][]byte{e2.ReferenceKey()}, rec.Picks[0].Radius)
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
	return slices.ContainsFunc(v.Edges(), hasCylinderNeighbour)
}

func sumFaceTess(b *topo.Body) float64 {
	var sum float64
	for _, f := range b.Faces() {
		m := tessellate.TessellateFace(f, ops.PropertyQuality())
		for k := 0; k+2 < len(m.Indices); k += 3 {
			p, q, r := m.Positions[m.Indices[k]], m.Positions[m.Indices[k+1]], m.Positions[m.Indices[k+2]]
			sum += float64(p.VectorTo(q).Cross(p.VectorTo(r)).Length()) / 2
		}
	}
	return sum
}
