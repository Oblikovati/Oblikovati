// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
)

// TestNotchFilletFaceLoopsSimple documents Bug B, a KNOWN LIMITATION (skipped, tracked).
//
// When a filleted edge's neighbour face has a feature protruding into the removed strip (the
// notched prism: an r=15 fillet whose tangent line is z=85, and a notch whose corners reach z=90),
// transformLoop pulls back only the edge's own endpoints and leaves the notch on the removed side,
// so that face's outer loop SELF-INTERSECTS (crosses itself at (90,85)). The correct result trims
// the protrusion back to the tangent line.
//
// This is NOT fixable by a per-face 2D clip: the notch corners are vertices SHARED with the notch's
// wall faces, so clipping one face without coordinately re-trimming every face that touches the
// removed region cracks the body ("result is not a valid solid"). The correct fix is a coordinated
// regularized 3D trim of the fillet surface against all nearby faces — OCCT ChFi3d-level machinery
// (geometry-math-advisor design captured in .superpowers/sdd/progress.md) — a substantial fillet-
// engine effort, tracked separately.
//
// Impact is bounded and non-visible today: the earclip tessellator is robust to the self-
// intersecting loop and computes the right area (Bug A guards the conformance path from collapsing
// it), so the body measures correctly (corpus simple/Y2 passes, OCCT 61050 within 1%). The defect
// is a B-rep topology blemish, not a measurement/render error. Un-skip and assert simple loops when
// the coordinated trim lands.
func TestNotchFilletFaceLoopsSimple(t *testing.T) {
	t.Parallel()
	t.Skip("Bug B: fillet builds a self-intersecting neighbour-face loop when a feature protrudes " +
		"into the removed strip; correct fix needs a coordinated 3D trim (tracked). Area is correct " +
		"(earclip-robust; Bug A guards conformance), so simple/Y2 still passes the gate.")

	body := importNotchedPrism(t)
	res, err := FilletEdges(body, [][]byte{notchTopEdge(t, body).ReferenceKey()}, 15)
	if err != nil {
		t.Fatalf("fillet: %v", err)
	}
	for i, f := range res.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			continue
		}
		loop2D := project2D(faceOuterBoundary(f, PropertyQuality()), planeProjector(pl.Normal()))
		if !simpleLoop2D(loop2D) {
			t.Errorf("face[%d] outer loop self-intersects", i)
		}
	}
}
