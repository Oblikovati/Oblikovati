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

// apexCase pairs a partial-primitive fixture with OCCT's reference filleted area.
type apexCase struct {
	name string
	area float64
}

// TestApexFilletMatchesOCCT pins the G1 cluster-1a defect and its RESOLVED scope.
//
// Filleting the revolution-axis apex edge of a partial revolved primitive is geometrically
// wrong: A9 (90° sector, apex convex) and B4 (270° sector, apex concave) both yield area
// 19098.9 / vol 122853.2 — the fillet ignores the dihedral and removes ~73000 vol³ where an r10
// round should barely change the body (OCCT: 21308.8 and 44956.6). The built faces are the
// right TYPES (a cylindrical fillet strip + trimmed planes) but grossly wrong EXTENT.
//
// Scope decision (revised after investigation, superseding an earlier "interim guard" plan):
// this is NOT interim-guardable. simple/M1 fillets a structurally-identical apex edge (a small
// fused partial cylinder) and is CORRECT to -0.29% — so the engine can fillet an apex edge, a
// clean structural predicate cannot separate the good case from the bad (they are topologically
// identical), and any rejection guard keyed on apex-detection would wrongly reject M1's valid
// fillet. It is a real corner-reconstruction geometry bug whose magnitude scales with the
// sector geometry (M1 -0.29%, A9 -10%, B4/C3 -57%). The fix belongs to greening package G5
// (corner reconstruction at revolution-axis poles); this test is the regression target and is
// RED until G5 makes the apex fillet match OCCT. It must never be made green by loosening it.
func TestApexFilletMatchesOCCT(t *testing.T) {
	for _, c := range []apexCase{{"A9", 21308.8}, {"B4", 44956.6}} {
		body := importPartCyl(t, c.name)
		apex := edgeNearestMid(t, body, math.P3(0, 0, 50))
		res, err := ops.FilletEdges(body, [][]byte{apex.ReferenceKey()}, 10)
		if err != nil {
			t.Fatalf("%s: apex fillet errored: %v", c.name, err)
		}
		got := ops.BodyGeometryProperties(res, ops.PropertyQuality()).Area
		if rel := (got - c.area) / c.area; rel < -0.01 || rel > 0.01 {
			t.Fatalf("%s: apex fillet area %.1f != OCCT %.1f (rel %.2f%%) — G5 corner-reconstruction fix pending", c.name, got, c.area, rel*100)
		}
	}
}

// importPartCyl loads a partial-primitive fixture (OCCT input for simple/<case>, oracle-exported).
func importPartCyl(t *testing.T, name string) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "partcyl_"+name+".step"))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import %s: %v (bodies=%d)", name, err, len(bodies))
	}
	return bodies[0]
}

// edgeNearestMid returns the body edge whose midpoint is closest to p.
func edgeNearestMid(t *testing.T, b *topo.Body, p math.Point3) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestD := math.Scalar(1e18)
	for _, e := range b.Edges() {
		if d := topo.DescribeEdge(e).Midpoint.DistanceTo(p); d < bestD {
			bestD, best = d, e
		}
	}
	if best == nil {
		t.Fatal("no edges on body")
	}
	return best
}
