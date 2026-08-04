// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// TestV3FilletClosesToSolid is the behavioural crux of the n-valent runout feature: V3's picked
// edge terminates at a valence-5 vertex, and before the spread is wired the rebuild drops the
// far-face cap into 4 open edges (the cylinder end arc ta→tb is used once while the three far-face
// pieces ta→split0, split0→split1, split1→tb are each used once). After wiring, every edge must be
// used exactly twice and the body must be a solid. The pick edge is located coordinate-robustly by
// its two end vertices (the real v5 sits at y=93.969, so a rounded literal would miss it).
func TestV3FilletClosesToSolid(t *testing.T) {
	b := importCorpusSolid(t, "simple/V3")
	v5 := vertexNear(t, b, math.P3(34.2, 94, 50))
	v3 := vertexNear(t, b, math.P3(-0.612, 86, 59.7))
	e := edgeBetween(t, b, v5, v3)
	res, err := FilletEdges(b, [][]byte{e.ReferenceKey()}, 5)
	if err != nil {
		t.Fatalf("V3 fillet errored: %v", err)
	}
	open := 0
	for _, ed := range res.Edges() {
		if len(ed.Uses()) != 2 {
			open++
		}
	}
	if open != 0 {
		t.Fatalf("V3 fillet left %d open edges — the runout still does not close", open)
	}
	if !res.IsSolid() {
		t.Fatal("V3 fillet result is not marked solid")
	}
}
