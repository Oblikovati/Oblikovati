// SPDX-License-Identifier: GPL-2.0-only

package tessellate_test

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The rim-only ear, at the corpus (ADR-0061, final fix wave, finding 1).
//
// A triangle whose three vertices all lie on a face's boundary carries no point of the surface, and on
// the figure-eight torus band it lay in the LID's plane, doubling the lid's tip triangle with the
// opposite normal — two free edges of degree four at DefaultQuality. The mesher now splits every such
// ear at the surface point under its centroid, and this is the invariant that says it did: over every
// charted face of the classification corpus, at both facetings, the chart mesh emits no rim-only
// triangle. The figure-eight pieces are in the corpus so the case that found it is measured every run.

// TestNoChartedFaceEmitsARimOnlyTriangle is the mesher-level invariant over the chart corpus.
func TestNoChartedFaceEmitsARimOnlyTriangle(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3 min on CI): `make test-corpus`")
	}
	t.Parallel()
	coarse, fine := refinementQualities()
	seen := 0
	forEachCurvedCorpusFace(t, func(body string, i int, f *topo.Face) {
		for _, q := range []ops.Quality{coarse, fine} {
			n, ok := tessellate.ChartRimOnlyTriangles(f, q)
			if !ok {
				continue
			}
			seen++
			if n != 0 {
				t.Errorf("%s face %d (%T) at chord %g: the chart mesh emits %d rim-only triangle(s); a "+
					"triangle with no surface point of its own lies in whatever plane its rim points do",
					body, i, f.Geometry(), q.ChordTolerance, n)
			}
		}
	})
	if seen == 0 {
		t.Error("no corpus face reached the chart-driven mesher — the rim-only invariant covers nothing")
	}
}

// figureEightPiece is one side of the torus R=5 r=2 split by the plane y=3, which touches its inner
// equator: the section is a figure eight and the material corner at the pinch is ~102° wide against
// boundary chords of 1.23 mm at DefaultQuality — the geometry that produced the ear.
func figureEightPiece(t *testing.T, op ops.PartFeatureOperation) *topo.Body {
	t.Helper()
	torus, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "ring")
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	body, err := ops.Boolean(op, torus, mustBlock(t, math.P3(-20, 3, -20), math.P3(20, 20, 20)))
	if err != nil {
		t.Fatalf("figure-eight %v: %v", op, err)
	}
	return body
}
