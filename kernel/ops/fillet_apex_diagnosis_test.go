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

// TestApexFilletMatchesOCCT regresses the G1 cluster-1a defect — now FIXED, and NOT by what the
// name "apex" first suggested.
//
// Symptom: filleting the revolution-axis apex edge of a partial revolved primitive was wrong —
// A9 (90° sector) and B4 (270° sector) both yielded area 19098.9 / vol 122853.2, removing ~73000
// vol³ where an r10 round should barely change the body (OCCT: 21308.8 and 44956.6). It looked
// like a corner-reconstruction defect at the revolution poles and was tentatively deferred to G5.
//
// Real root cause (found later, while chasing Q1): it was the curved-survivor-edge bug in
// transformLoop, not corner reconstruction. A partial primitive's radial cut faces border the
// quadric lateral face along ARC edges; rebuilding those faces, transformLoop dropped the arc
// curve (nil), and because both faces sharing each arc are transformed the shared edge collapsed
// to a straight chord — grossly deforming the sector and its measured volume. survivorCurve
// (fillet_faces.go) now carries the arc, correctly oriented to the loop traversal, so A9/B4/B8/C3/
// D2/D6 all match OCCT. (This also explains why M1 stayed correct: its shared arcs were oriented
// such that straightening barely moved the area.) Must never be made green by loosening it.
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
