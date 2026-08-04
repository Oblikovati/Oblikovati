// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"
)

// Per-face DRAWEXE reconciliation for the EllipticalCylinder∧Cone pinched canal
// (kernel/ops/fillet_elliptic_cone_*.go — cluster W-F): a circle extruded obliquely (the
// elementarised right elliptical cylinder) glued to a cone whose slope EQUALS the extrusion tilt,
// so the wall and cone are tangent along one ruling and the constant-radius fillet's cross-section
// COLLAPSES TO A POINT there (a teardrop canal, geom.LoftPinchedCanalStations).
//
// Three of the five tolblend_simple cases in this family verify: B4/B8 (the OPEN-arc runout, r=75,
// tolBlend exact/near-tangent siblings) and C3 (the CLOSED rim whose pinch sits exactly at the
// host seam azimuth, r=25 — no rail split needed). B7/C2 (the CLOSED rim whose pinch sits OFF the
// host seam, r=75) are a MEASURED decline — see closedEllipticConeSpan's doc comment in
// fillet_elliptic_cone_stations.go — not covered here.
//
// DRAWEXE 8.0.0 receipts captured live (tolblend_simple/{B4,B8,C3}; drawenv.sh +
// CSF_TestDataPath=test-utilities/occt-blend/data; `dset SCALE 1000; dset SCALE1 5; tolblend 0.01
// 1e-04 1e-05 1e-03` then the fixture's own script; `nexplode result f` + `sprops result_i 1.e-12`;
// `checkshape result` reports valid on all three).

// ellipticConePerFaceTol is the per-face RELATIVE budget. B4/B8's smallest face (the imprinted
// runout LENS on the side plane, the newly-added sliver between the old corner and the arc) reads
// ~0.58% off DRAWEXE (14047.83 vs 13966.8) — larger than the cyl∧cyl seam family's 2e-3 because
// this face sits at the degenerate teardrop LIMIT (the arc's far end IS the host-tangency pinch,
// where the section's own closed form is most sensitive to the station density the envelope-error
// loop converges to), not because of a compensating-error cancellation: every OTHER face on all
// three cases (11 of 12 rank pairs) is within 0.04%, and C3 — the SAME closed-form spine, no
// runout — is within 0.09% throughout. 1e-2 leaves ~1.7x headroom over the measured worst while
// staying two decades under the +40%-class compensating error this gate exists to catch.
const ellipticConePerFaceTol = 1e-2

// ellipticConeCanalCases is the pinned population with DRAWEXE 8.0.0 per-face receipts, sorted
// DESCENDING (rank-paired — sound here since the closest non-symmetry pair in any of the three
// differs by >13%).
func ellipticConeCanalCases() []cylCylSeamCase {
	return []cylCylSeamCase{
		{grid: "tolblend_simple", drawexeFaceCase: drawexeFaceCase{name: "B4", totalArea: 547392, perFaceTol: ellipticConePerFaceTol,
			drawexe: []float64{157258, 98174.8, 96992.6, 78329.5, 78125, 24543.7, 13966.8}}},
		{grid: "tolblend_simple", drawexeFaceCase: drawexeFaceCase{name: "B8", totalArea: 547392, perFaceTol: ellipticConePerFaceTol,
			drawexe: []float64{157258, 98174.8, 96992.6, 78329.5, 78125, 24543.7, 13966.8}}},
		{grid: "tolblend_simple", drawexeFaceCase: drawexeFaceCase{name: "C3", totalArea: 782458, perFaceTol: ellipticConePerFaceTol,
			drawexe: []float64{324462, 203353, 196350, 49087.4, 9171.8}}},
	}
}

// TestEllipticConeCanalPerFaceParity is the oracle gate for the EllipticalCylinder∧Cone pinched
// canal family.
func TestEllipticConeCanalPerFaceParity(t *testing.T) {
	t.Parallel()
	for _, tc := range ellipticConeCanalCases() {
		t.Run(tc.grid+"/"+tc.name, func(t *testing.T) {
			body := cylCylSeamResultBody(t, tc.grid, tc.name)
			assertPerFaceAgainstDrawexe(t, tc.drawexeFaceCase, body)
			assertLoopSegmentsOnFaces(t, Record{Grid: tc.grid, Case: tc.name}, body, ellipticConeOnFaceTol)
		})
	}
}

// ellipticConeOnFaceTol is the "every boundary edge lies on the face it bounds" budget relative to
// the bounding diagonal — the corpus-wide 1e-6 ratchet bar, held with NO debt entry (the rails and
// the runout section arc are the loft's own iso-curves, so band boundary and band surface agree by
// construction).
const ellipticConeOnFaceTol = 1e-6

// TestEllipticConeCanalWatertight pins the results as watertight manifold solids at the DRAWEXE
// face-count join: every edge exactly 2-incident, 0 free edges, no retracing/self-crossing loops
// at BOTH gate qualities, and every face fold-free at both.
func TestEllipticConeCanalWatertight(t *testing.T) {
	t.Parallel()
	for _, tc := range ellipticConeCanalCases() {
		t.Run(tc.grid+"/"+tc.name, func(t *testing.T) {
			body := cylCylSeamResultBody(t, tc.grid, tc.name)
			assertWatertight(t, tc.name, body, len(tc.drawexe))
			assertBodyIsWatertightAndSimple(t, body, tc.grid+"/"+tc.name)
			for _, f := range body.Faces() {
				assertFaceFoldFreeAtEveryQuality(t, tc.grid+"/"+tc.name, f, nil)
			}
		})
	}
}

// TestEllipticConeCanalOffSeamDeclines is the MUTATION-PROOF for the B7/C2 decline: it asserts the
// off-seam closed-pinch sub-case still refuses (do-no-harm), and that reverting
// closedEllipticConeSpan's guard to the pre-decline behaviour (accepting the off-seam pinch) is what
// this test is FOR — see the git history / wave-report-F.md for the mutation receipt (reverting the
// guard flips B7/C2 to filletOK=true with a ~3.9x oversized band, exactly the defect this refusal
// prevents).
func TestEllipticConeCanalOffSeamDeclines(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"B7", "C2"} {
		t.Run(name, func(t *testing.T) {
			var rec Record
			for _, r := range Corpus() {
				if r.Grid == "tolblend_simple" && r.Case == name {
					rec = r
				}
			}
			body, err := importInput(CorpusFixtureDir() + "/" + rec.InputStep)
			if err != nil {
				t.Fatalf("%s import-divergence: %v", name, err)
			}
			sets, ok := scoreLocate(rec, body)
			if !ok {
				t.Fatalf("%s picks could not be located on the imported body", name)
			}
			_, filletOK, reason := runFillet(body, sets)
			if filletOK {
				t.Fatalf("%s: expected an honest decline (off-seam closed pinch, measured ~3.9x oversized "+
					"band — see closedEllipticConeSpan), got filletOK=true", name)
			}
			if reason == "" {
				t.Fatalf("%s: decline carries no reason", name)
			}
		})
	}
}
