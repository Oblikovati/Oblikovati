// SPDX-License-Identifier: GPL-2.0-only

package blend_test

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestFilletPreservesCurvedSurvivorEdge regresses the Q1 area defect (corpus simple/Q1).
//
// Filleting the (0,0,1)-(0,1,1) edge of a prism that borders a cylindrical face put the fillet's
// end-cap face (perpendicular to the edge) next to the cylinder: that cap shares an Arc3d edge with
// the cylinder. Rebuilding the face, transformLoop dropped the curve of every "survivor" edge
// (passed nil), and since BOTH faces sharing the arc are transformed, the shared edge collapsed to
// a straight LineSegment — bulging the planar cap and inflating the body area +3.4% (12.273 vs OCCT
// 11.869). The fix carries a curved survivor edge's geometry through the rebuild (survivorCurve).
//
// Asserts the fix two ways: the arc survives on the cap face, and the body area matches OCCT within
// 1%. Must never regress; a straight survivor edge is unaffected (a LineSegment curve equals nil).
func TestFilletPreservesCurvedSurvivorEdge(t *testing.T) {
	t.Parallel()
	body := importPrismCylBorder(t)
	edge := edgeBetween(t, body, math.P3(0, 0, 1), math.P3(0, 1, 1))
	res, err := blend.FilletEdges(body, [][]byte{edge.ReferenceKey()}, 0.2)
	if err != nil {
		t.Fatalf("fillet: %v", err)
	}
	if !hasArcEdge(res) {
		t.Error("no Arc3d edge survived the fillet — a curved neighbour edge was straightened")
	}
	area := query.BodyGeometryProperties(res, ops.PropertyQuality()).Area
	if rel := (area - 11.8686) / 11.8686; rel < -0.01 || rel > 0.01 {
		t.Fatalf("filleted area %.5g, want OCCT 11.8686 within 1%% (rel %+.2f%%) — curved survivor edge straightened", area, rel*100)
	}
}

func importPrismCylBorder(t *testing.T) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "prism_cyl_border.step"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import: %v (n=%d)", err, len(bodies))
	}
	return bodies[0]
}

// edgeBetween returns the edge whose endpoints match p and q (either order).
func edgeBetween(t *testing.T, b *topo.Body, p, q math.Point3) *topo.Edge {
	t.Helper()
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if (a.DistanceTo(p) < 1e-6 && c.DistanceTo(q) < 1e-6) || (a.DistanceTo(q) < 1e-6 && c.DistanceTo(p) < 1e-6) {
			return e
		}
	}
	t.Fatalf("edge %v-%v not found", p, q)
	return nil
}

// hasArcEdge reports whether the body has any arc (non-straight) edge.
func hasArcEdge(b *topo.Body) bool {
	for _, e := range b.Edges() {
		if _, ok := e.Geometry().(geom.Arc3d); ok {
			return true
		}
	}
	return false
}
