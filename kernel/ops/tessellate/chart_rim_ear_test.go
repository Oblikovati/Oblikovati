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

// TestNoChartedFaceEmitsARimOnlyTriangle is the mesher-level invariant over the chart corpus, and the
// row that says every face the classification SENDS to this mesher is one the mesher accepts.
//
// The acceptance half is not decoration. Spending the ear-splitting rounds ends in a DECLINE, so a row
// that skips declined faces reads green for exactly the input the invariant is about — which is what
// this row did until #3520. A decline on a face the classification selected the chart mesher for is
// therefore a failure, not a skip.
//
// The two answers must stay apart, though: the mesher also refuses faces it was never given (no chart,
// or an aperiodic surface), and it is driven here DIRECTLY, past the classification, on faces another
// arm meshes. Measured on this corpus at chord 0.001, three such faces exist — the near-pinch body's
// two ruled-band-loft walls and its two-rim-holed-band wall, whose chart mesh is not bounded by its own
// rim; that the chart cannot serve them is the documented reason those arms exist
// (TestTheTwoRimArmKeepsOnlyWhatTheChartCannotServe). Asserting acceptance on a mesher the face never
// routes to would assert something the pipeline does not claim, so acceptance is asserted for the faces
// whose selected mesher this IS, and the rim-only count still covers every face it accepts.
func TestNoChartedFaceEmitsARimOnlyTriangle(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3 min on CI): `make test-corpus`")
	}
	t.Parallel()
	coarse, fine := refinementQualities()
	seen, accepted := 0, 0
	forEachCurvedCorpusFace(t, func(body string, i int, f *topo.Face) {
		for _, q := range []ops.Quality{coarse, fine} {
			n, meshed, declined := tessellate.ChartRimOnlyTriangles(f, q)
			selected := tessellate.ClassifyCurvedTrimName(f, q) == tessellate.ChartedTrimKindName()
			assertChartMesherAccepted(t, body, i, f, q, declined, selected)
			if selected && meshed {
				accepted++
			}
			if !meshed {
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
	if accepted == 0 {
		t.Error("the classification sent no corpus face to the chart-driven mesher — the acceptance " +
			"assertion covers nothing")
	}
}

// assertChartMesherAccepted fails the row when the mesher the classification SELECTED for this face
// gave it up. A decline on a face routed to another arm is not this invariant's business (see the test's
// own comment), and "the mesher never owned it" is no decline at all.
func assertChartMesherAccepted(t *testing.T, body string, i int, f *topo.Face, q ops.Quality,
	declined string, selected bool) {
	t.Helper()
	if declined == "" || !selected {
		return
	}
	t.Errorf("%s face %d (%T) at chord %g: the classification selected the chart-driven mesher for this "+
		"face and the mesher gave it up — %s; the face then ships from a covering its chart never "+
		"certified", body, i, f.Geometry(), q.ChordTolerance, declined)
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
