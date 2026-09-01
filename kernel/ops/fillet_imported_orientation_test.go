// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestFilletConvexEdgeOnImportedBox regresses the orientation bug where edgePlanarFaces read a
// face's raw plane normal instead of its material-outward normal. A STEP-imported 100³ box has
// a reversed face with an inward plane normal; the picked (100,0,50) edge is plainly convex, yet
// the rolling-ball centre was computed on the wrong side and the fillet failed with "edge is not
// convex". Filleting it must now succeed and land at OCCT's reference area (59527.9 ± 1%).
func TestFilletConvexEdgeOnImportedBox(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("testdata", "box100_oriented.step"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(bodies) != 1 {
		t.Fatalf("imported %d bodies, want 1", len(bodies))
	}
	body := bodies[0]

	edge := edgeNearest(t, body, math.P3(100, 0, 50))
	res, err := ops.FilletEdges(body, [][]byte{edge.ReferenceKey()}, 10)
	if err != nil {
		t.Fatalf("fillet of a convex imported-box edge failed: %v", err)
	}
	area := ops.BodyGeometryProperties(res, ops.PropertyQuality()).Area
	if rel := (area - 59527.9) / 59527.9; rel < -0.01 || rel > 0.01 {
		t.Fatalf("filleted area %.1f, want 59527.9 within 1%% (rel %.4f)", area, rel)
	}
}

// edgeNearest returns the body edge whose midpoint is closest to p.
func edgeNearest(t *testing.T, b *topo.Body, p math.Point3) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestD := math.Scalar(1e18)
	for _, e := range b.Edges() {
		if d := topo.DescribeEdge(e).Midpoint.DistanceTo(p); d < bestD {
			bestD, best = d, e
		}
	}
	if best == nil {
		t.Fatalf("no edges on body")
	}
	return best
}
