// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// simple/U4's DUAL-HOST body must tessellate into a CLOSED surface at every gate quality — the hard
// zero that replaces its retired knownMeshLeaks entry (44 free edges at BOTH default and property).
//
// WHAT LEAKED, MEASURED. Swept by provenance on the SHIPPED body with
// tessellate.FreeEdgeCount(tessellate.CalculateBodyFacets(body, q).Mesh), instrumented per free edge with its owning
// face: 21 edges on face 454 (geom.BSplineSurface, the low-z coons4 sliver panel), 21 on face 462 (the
// high-z sliver), and 2 on face 441 (geom.EllipticalCylinder, the oblique boss's wall) — 44, identical
// at both qualities.
//
// ★ THE ATTRIBUTION IT ARRIVED WITH WAS WRONG, and the measurement is what says so. The debt table
// blamed "a fitted patch rail and its host tiling the same seam from DIFFERENT curves", citing the two
// sides' chord lengths differing in the 5th digit (0.0220470 vs 0.0220538). Those two numbers are not
// two sides of one seam — they are the two MIRROR slivers' own chords, on opposite ends of the body.
// The real pairing is this: face 454 tiled edge 440 with 21 chords of 0.0220470 while face 441 tiled
// the SAME topological edge 440 with ONE chord of 0.4629860 = 21 x 0.0220470, between the SAME two
// vertices. The curve was never in doubt (edge 440 is a geom.LineSegment both faces share, and the
// polyline is exactly collinear); only the station COUNT differed. Nothing here is a surface-fidelity
// defect, so no change of the panels' fill surface could have closed it.
//
// THE ROOT. The boss wall is a notched two-rim band, so it meshes through notchedRimBandMesh ->
// saddleBandLoftMesh, whose bandWrapRings read each rim with the RAW TessellateEdge sampler instead of
// discretizeEdge. discretizeEdge is the one function every face of an edge shares, and it is where the
// #2009 starved-rail densification lives — deliberately, because that densification's whole
// correctness argument is that it is caller-INDEPENDENT (densifyStarvedRail's own doc). The sliver
// panel is a high-aspect B-spline (A = 15.19, over aspectDensifyThreshold), so its CDT got the
// straight rail at 21 stations; the band loft, reading raw, got it at 2. Reading the ring through
// discretizeEdge closes it: 44 -> 0 at BOTH qualities, with every one of U4's sixteen per-face areas
// unchanged to 2e-6 and the whole-body area unchanged at 6583.287851 (property quality).
func TestU4DualHostMeshIsClosedAtEveryQuality(t *testing.T) {
	t.Parallel()
	body, ok := shippedCaseBody(caseRecord(t, "simple", "U4"), CorpusFixtureDir())
	if !ok {
		t.Fatal("simple/U4 ships no healthy body")
	}
	for _, gq := range gateQualities() {
		if free := shippedMeshFreeEdges(body, gq.q); free != 0 {
			t.Errorf("simple/U4: welded body mesh leaks %d free edge(s) at %s quality, want 0 — the "+
				"dual-host panels and the boss wall are tiling their shared rim at different station counts",
				free, gq.name)
		}
	}
}
