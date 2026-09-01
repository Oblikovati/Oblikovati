// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestCutEdgeScallopClipsEdge: a cylinder drilled through a plate whose circle CLIPS the +x edge (center
// (4,0), r2, plate x∈[-5,5]) cuts a watertight, ORIENTABLE scallop notch — genus 0 (chi=2), the analytic
// cylinder wall preserved. Exercises the partial-drill planeUV path end-to-end (#1591 Slice A').
func TestCutEdgeScallopClipsEdge(t *testing.T) {
	t.Parallel()
	plate, _ := SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	drill, _ := SolidCylinder(math.P3(4, 0, -1), math.V3(0, 0, 1), 2, 4) // through the slab, circle clips x=5
	res, ok := CutEdgeScallop(plate, drill)
	if !ok {
		t.Fatal("edge-scallop cut declined; want plate − edge-clipping drill")
	}
	free, inconsistent := 0, 0
	for _, e := range res.Edges() {
		uses := e.Uses()
		if len(uses) != 2 {
			free++
			continue
		}
		if uses[0].Reversed() == uses[1].Reversed() {
			inconsistent++
		}
	}
	t.Logf("faces=%d edges=%d free=%d inconsistent=%d solid=%v shells=%d chi=%d",
		len(res.Faces()), len(res.Edges()), free, inconsistent, res.IsSolid(), len(res.Shells()), res.EulerCharacteristic())
	if free != 0 {
		t.Errorf("scallop has %d free edges (want 0 — watertight)", free)
	}
	if inconsistent != 0 {
		t.Errorf("scallop has %d inconsistently-oriented edges (want 0 — orientable)", inconsistent)
	}
	if !res.IsSolid() {
		t.Error("scallop is not a solid")
	}
}

// TestCutEdgeScallopDeclinesInteriorHole: a clean interior drill (circle strictly inside both caps) is
// DrillThroughHole's job — the scallop gate must decline it, or it would steal the interior-hole path.
func TestCutEdgeScallopDeclinesInteriorHole(t *testing.T) {
	t.Parallel()
	plate, _ := SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	drill, _ := SolidCylinder(math.P3(0, 0, -1), math.V3(0, 0, 1), 2, 4) // centered, circle strictly inside
	if _, ok := CutEdgeScallop(plate, drill); ok {
		t.Error("scallop accepted a strictly-interior hole; want decline (DrillThroughHole's case)")
	}
}

// TestCutEdgeScallopDeclinesNonThrough: a drill that does not pierce two parallel caps (a blind stub) is out
// of the through-scallop scope — the gate must decline.
func TestCutEdgeScallopDeclinesNonThrough(t *testing.T) {
	t.Parallel()
	plate, _ := SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	blind, _ := SolidCylinder(math.P3(4, 0, 1), math.V3(0, 0, 1), 2, 3) // starts inside the slab, blind
	if _, ok := CutEdgeScallop(plate, blind); ok {
		t.Error("scallop accepted a non-through blind drill; want decline")
	}
}

var (
	_ = topo.Body{}
	_ = stdmath.Pi
)
