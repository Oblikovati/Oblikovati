// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
)

// p2TorusBandCases are the B1 cases whose surviving torus rim-fillet band formerly fell through
// to fullDomainGridMesh — meshing the ENTIRE torus [0,2π]² (area inflated +20…+62%, 262144 tris)
// — because bandRingsAndSeam recognized rings only as geom.Circle and missed the full-sweep
// geom.Arc3d rims. With the rims classified, the existing closedBandLoftMesh meshes the true band.
var p2TorusBandCases = map[string]bool{"S9": true, "T1": true, "T3": true, "T4": true}

// TestP2TorusBandNotFullDomain guards the loft fix: each torus-band case must mesh to an area far
// below the old full-domain blowup. A generous 5% bound cleanly separates "loft meshes the band"
// (now 0.7–3.6%) from the pre-fix gross fallback (>20%); the residual 1–3% is the shared curved-
// fillet accuracy gap (P1), asserted elsewhere, not here.
func TestP2TorusBandNotFullDomain(t *testing.T) {
	t.Parallel()
	fixtureDir := CorpusFixtureDir()
	const grossFallbackBound = 0.05
	seen := 0
	for _, r := range Corpus() {
		if r.Grid != "simple" || !p2TorusBandCases[r.Case] {
			continue
		}
		seen++
		body, err := importInput(filepath.Join(fixtureDir, r.InputStep))
		if err != nil {
			t.Fatalf("%s: import failed: %v", r.Case, err)
		}
		sets, ok := scoreLocate(r, body)
		if !ok {
			t.Fatalf("%s: could not locate picked edges", r.Case)
		}
		res, filletOK, reason := runFillet(body, sets)
		if !filletOK || len(res) != 1 {
			t.Fatalf("%s: fillet not a valid solid: %s", r.Case, reason)
		}
		got := ops.BodyGeometryProperties(res[0], ops.PropertyQuality()).Area
		rel := (got - r.ExpectedArea) / r.ExpectedArea
		if rel < 0 {
			rel = -rel
		}
		if rel > grossFallbackBound {
			t.Errorf("%s: area %.3f vs OCCT %.3f (rel %.2f%%) — torus band fell back to full-domain mesh",
				r.Case, got, r.ExpectedArea, rel*100)
		}
	}
	if seen != len(p2TorusBandCases) {
		t.Fatalf("expected %d torus-band cases, ran %d", len(p2TorusBandCases), seen)
	}
}
