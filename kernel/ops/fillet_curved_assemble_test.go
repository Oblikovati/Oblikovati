// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// curvedArmCorpusPicks maps an axis-aligned [Cylinder,Plane,Plane] corpus case to the GEOMETRIC
// midpoints of its picked edges (from occtparity/corpus.json). The rim arc's midpoint is the arc's
// geometric midpoint, not its chord midpoint, so curvedArmEdgeAt matches on either.
var curvedArmCorpusPicks = map[string][]math.Point3{
	"simple/B3": {
		math.P3(35.35533906, -35.35533906, 100), // top-rim arc: Cyl∧Plane axis⊥plane → torus
		math.P3(0, -50, 50),                     // vertical wall line: Cyl∧Plane axis∥plane → cylinder
		math.P3(0, -25, 100),                    // top radial segment: Plane∧Plane → cylinder
	},
	"simple/N1": {
		math.P3(85.85786438, 14.14213562, 100),
		math.P3(40, 0, 100),
		math.P3(80, 0, 50),
	},
	"simple/O1": {
		math.P3(80, 10, 55),
		math.P3(65.8113883, 2.565835097, 90),
		math.P3(80, -15, 90),
	},
	// M5 is the concave-BORE (roll-sense R3, R−r) trihedral corner of the same Gate-1 cluster as O1, still
	// unbuilt: it is here only so TestFilletEdges_M5DeclinesCleanly can pin the do-no-harm FLOOR on a corner
	// the ladder must not accept.
	"simple/M5": {
		math.P3(50, 20, 25),
		math.P3(32.67949192, 14.49489743, 50),
		math.P3(50, 10, 50),
	},
}

// curvedArmEdgeAt returns the body edge whose curve midpoint OR chord midpoint is nearest mid
// (within a generous fixture tolerance). A curved rim arc is matched by its true PointAt(0.5), a
// straight edge by either — the corpus locators store the geometric midpoint.
func curvedArmEdgeAt(b *topo.Body, mid math.Point3) *topo.Edge {
	var best *topo.Edge
	bestD := math.Scalar(1e18)
	for _, e := range b.Edges() {
		cands := []math.Point3{
			e.Geometry().PointAt(0.5),
			e.StartVertex().Point().Midpoint(e.EndVertex().Point()),
		}
		for _, p := range cands {
			if d := p.DistanceTo(mid); d < bestD {
				bestD, best = d, e
			}
		}
	}
	if bestD > 1e-2 {
		return nil
	}
	return best
}

// filletedCorpusEdges rounds ALL picked edges of an axis-aligned [Cylinder,Plane,Plane] corpus case
// at radius r in ONE FilletEdges op (the sibling of filletedCorpusEdge, which rounds a single edge).
// It resolves every pick from curvedArmCorpusPicks so the trihedral corner assembles from real arms.
func filletedCorpusEdges(t *testing.T, rel string, r float64) (*topo.Body, error) {
	t.Helper()
	b := importCorpusSolid(t, rel)
	mids, ok := curvedArmCorpusPicks[rel]
	if !ok {
		t.Fatalf("filletedCorpusEdges: no pick table for %s", rel)
	}
	keys := make([][]byte, 0, len(mids))
	for _, m := range mids {
		e := curvedArmEdgeAt(b, m)
		if e == nil {
			t.Fatalf("filletedCorpusEdges: edge near %v not found on %s", m, rel)
		}
		keys = append(keys, e.ReferenceKey())
	}
	return FilletEdges(b, keys, r)
}

// TestFilletEdges_B3NoPanic is the do-no-harm floor (Step 0): rounding B3's three axis-aligned
// Plane∧Cylinder picks must NEVER panic. Before the curved weld is built the op honest-rejects with a
// clean error (the unassembled curved arm cannot ride the planar setback path); once welded it returns
// a solid. What it must NOT do — and did before the gate — is nil-deref in applyRunoutSetback because
// the curved-arm edgeFillet carries a zero cyl/c0/c1. This test fails (panics) without the gate.
func TestFilletEdges_B3NoPanic(t *testing.T) {
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("B3 curved-arm fillet PANICKED (do-no-harm floor breached): %v", p)
		}
	}()
	body, err := filletedCorpusEdges(t, "simple/B3", 10)
	if err == nil && body == nil {
		t.Fatalf("B3: FilletEdges returned nil body and nil error")
	}
	// Until the weld lands this is a clean error; after the weld a valid solid. Either is acceptable
	// here — the invariant under test is NO PANIC. (Faithfulness is TestFilletEdges_B3CurvedArmWeld.)
}
