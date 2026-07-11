// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"strings"
	"testing"

	m "oblikovati.org/math"
)

// TestConcaveFilletOverRadiusRejected is the #1800 concave gap (rampam item 4): the max-radius guard
// only bounded convex edges — a concave (pocket-fill) fillet whose radius overruns the finite walls
// passed Validate and shipped self-intersecting geometry. This L-prism has a 2-unit-deep pocket at
// the reflex edge (2,2) with a 90° dihedral (k=1), so r_max ≈ 2: a small radius fits, an over-large
// one is rejected with an honest max-radius message (not silently accepted).
func TestConcaveFilletOverRadiusRejected(t *testing.T) {
	b := zPrism([]m.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 2}, {X: 2, Y: 2}, {X: 2, Y: 4}, {X: 0, Y: 4}}, 0, 4, "L")
	var key []byte
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if a.X == c.X && a.Y == c.Y && ClassifyEdgeConvexity(e) == EdgeConcave {
			key = e.ReferenceKey()
			break
		}
	}
	if key == nil {
		t.Fatal("no concave vertical edge on the L-prism")
	}

	for _, r := range []float64{1.0, 1.9} {
		if _, err := FilletEdges(b, [][]byte{key}, r); err != nil {
			t.Errorf("concave fillet r=%.1f should fit the 2-deep pocket, got: %v", r, err)
		}
	}
	for _, r := range []float64{3.0, 10.0, 20.0} {
		_, err := FilletEdges(b, [][]byte{key}, r)
		if err == nil {
			t.Errorf("concave fillet r=%.0f overruns the 2-deep pocket but was accepted (self-intersects)", r)
			continue
		}
		if !strings.Contains(err.Error(), "geometric maximum") {
			t.Errorf("concave over-radius should fail with the max-radius message, got: %v", err)
		}
	}
}
