// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestApexFilletHonestlyRejected pins the G1 cluster-1a decision.
//
// Diagnosis (verified): filleting the revolution-axis APEX edge of a partial revolved
// primitive currently ships a SILENTLY WRONG solid. A9 (90° sector, apex edge convex) and B4
// (270° sector, apex edge concave) both yield area 19098.9 / vol 122853.2 — the fillet ignores
// the dihedral sign and removes ~73000 vol³ where an r10 round should barely change the body
// (OCCT expects 21308.8 and 44956.6). The built faces are the right TYPES (a cylindrical fillet
// strip + trimmed planes, 6 faces) but grossly wrong EXTENT, identically for convex and
// concave. Root cause is the corner reconstruction at the two axis-pole vertices — the general
// high-valence-at-a-revolution-pole corner problem (ADR-0050 Phase 6 / greening package G5),
// not a trivial sign fix.
//
// DECISION: an interim honest-rejection guard lands here (G1); the real blend is deferred to
// G5. This test asserts the honest rejection — it is RED until the apex guard lands (plan
// Task 3) and green after. The floor is "never a silent wrong solid".
func TestApexFilletHonestlyRejected(t *testing.T) {
	for _, c := range []string{"A9", "B4"} {
		body := importPartCyl(t, c)
		apex := edgeNearestMid(t, body, math.P3(0, 0, 50))
		_, err := ops.FilletEdges(body, [][]byte{apex.ReferenceKey()}, 10)
		if err == nil {
			t.Fatalf("%s: apex fillet returned a solid; it must be honestly rejected (never a silent wrong solid)", c)
		}
		if !strings.Contains(err.Error(), "apex") {
			t.Fatalf("%s: rejection reason %q should name the revolution-axis apex", c, err.Error())
		}
	}
}

// importPartCyl loads a partial-cylinder fixture (OCCT input for simple/<case>, oracle-exported).
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
