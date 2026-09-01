// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"slices"
	"strings"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestFilletRejectsOverLargeRadius is the #1800 regression: a radius far larger than the feature
// (r=20 on a 2×2×2 box, whose faces admit at most r=2) must fail honestly — surfacing the
// offending radius and the computed geometric maximum — instead of shipping self-intersecting
// geometry that only passes topological validation. A radius within the bound still succeeds.
func TestFilletRejectsOverLargeRadius(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 2, 2, 2)
	edge := verticalEdgeKey(t, box)

	_, err := ops.FilletEdges(box, [][]byte{edge}, 20)
	if err == nil {
		t.Fatal("fillet with r=20 on a 2×2×2 box must be rejected, not shipped")
	}
	if !strings.Contains(err.Error(), "geometric maximum") {
		t.Errorf("error should report the geometric maximum, got: %v", err)
	}

	if _, err := ops.FilletEdges(box, [][]byte{edge}, 0.5); err != nil {
		t.Fatalf("a radius within the bound (0.5) must still succeed: %v", err)
	}
}

// TestFilletRejectsCollidingNeighbours exercises constraint (b): two adjacent vertical edges
// sharing a 2-wide side face, each filleted, whose bands recede toward each other. r=1.5 each
// overshoots (1.5+1.5 > 2) and is rejected; r=0.8 each (1.6 < 2) fits.
func TestFilletRejectsCollidingNeighbours(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 2, 2, 2)
	a, b := adjacentVerticalEdges(t, box)

	if _, err := ops.FilletEdges(box, [][]byte{a, b}, 1.5); err == nil {
		t.Error("two r=1.5 fillets on a shared 2-wide face collide and must be rejected")
	}
	if _, err := ops.FilletEdges(box, [][]byte{a, b}, 0.8); err != nil {
		t.Errorf("two r=0.8 fillets fit within the shared face and must succeed: %v", err)
	}
}

// adjacentVerticalEdges returns the keys of two vertical box edges that share a side face.
func adjacentVerticalEdges(t *testing.T, b *topo.Body) ([]byte, []byte) {
	t.Helper()
	var verts []*topo.Edge
	for _, e := range b.Edges() {
		if p, q := e.StartVertex().Point(), e.EndVertex().Point(); p.X == q.X && p.Y == q.Y {
			verts = append(verts, e)
		}
	}
	for i := range verts {
		for j := i + 1; j < len(verts); j++ {
			if sharesFace(verts[i], verts[j]) {
				return verts[i].ReferenceKey(), verts[j].ReferenceKey()
			}
		}
	}
	t.Fatal("no two vertical edges sharing a face")
	return nil, nil
}

// sharesFace reports whether two edges bound a common face.
func sharesFace(a, b *topo.Edge) bool {
	for _, fa := range a.Faces() {
		if slices.Contains(b.Faces(), fa) {
			return true
		}
	}
	return false
}
