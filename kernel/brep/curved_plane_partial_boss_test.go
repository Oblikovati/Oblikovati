// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestJoinPartialBossStraddlesEdge: a boss whose base circle CLIPS the plate's top-face edge (centre (4,0),
// r2, on a plate spanning x∈[-5,5]) unions to a watertight solid of volume 200 + 12π (the whole cantilevered
// boss added, no volume overlap since the boss sits at z≥2). Exercises planeUV end-to-end (#1591 Slice B).
func TestJoinPartialBossStraddlesEdge(t *testing.T) {
	plate, _ := SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	boss, _ := SolidCylinder(math.P3(4, 0, 2), math.V3(0, 0, 1), 2, 3)
	res, ok := JoinPartialBoss(plate, boss)
	if !ok {
		t.Fatal("partial boss union declined; want plate + straddling boss")
	}
	freeEdges := 0
	for _, e := range res.Edges() {
		if len(e.Uses()) != 2 {
			freeEdges++
		}
	}
	t.Logf("faces=%d edges=%d freeEdges=%d solid=%v shells=%d", len(res.Faces()), len(res.Edges()), freeEdges, res.IsSolid(), len(res.Shells()))
	if freeEdges != 0 {
		t.Errorf("partial boss has %d free edges (want 0 — a watertight manifold)", freeEdges)
	}
}

var (
	_ = topo.Body{}
	_ = stdmath.Pi
)
