// SPDX-License-Identifier: GPL-2.0-only

package tessellate_test

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The near-pinch corpus for the chart-driven mesher (Oblikovati/Oblikovati#3518). The bodies are the
// JOIN half of kernel/ops/boolean's TestNearPinchCutJoinWatertight: two crossing cylinders whose radii
// differ by |dr|, joined, so the merged wall carries two lens windows that nearly pinch. Each join
// body has exactly one two-rim holed band; the CUT bodies have none, which is why only JOIN is here.
//
// These faces ship from the CHART mesher now: #3517 deleted the unrolled arm and the corridor gate that
// used to hold it alive, once coverShear closed the covering's own seam disagreement (#3542). The rows
// below drive the chart mesher on them directly — the same mesher the router selects — and the gate that
// says whether it may is chartRimMismatch.

// nearPinchJoinRadii and nearPinchDeltas are the corpus's two model scales and its four pinch widths.
func nearPinchJoinRadii() []float64 { return []float64{3.0, 30.0} }

// nearPinchDeltas is |dr|/R for the four crossings, scaled with R so the pinch is dimensionless.
func nearPinchDeltas() []float64 { return []float64{4e-5, 6e-5, 1.6e-4, 3.2e-4} }

// nearPinchJoinBody builds one crossing-rod union: radius r about x, radius r+dr about z.
func nearPinchJoinBody(t *testing.T, r, dr float64) *topo.Body {
	t.Helper()
	h := 4 * r
	cx, err := brep.SolidCylinder(math.P3(-2*r, 0, 0), math.V3(1, 0, 0), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder x (r=%g): %v", r, err)
	}
	cz, err := brep.SolidCylinder(math.P3(0, 0, -2*r), math.V3(0, 0, 1), r+dr, h)
	if err != nil {
		t.Fatalf("SolidCylinder z (r=%g dr=%g): %v", r, dr, err)
	}
	body, err := ops.Boolean(ops.Join, cx, cz)
	if err != nil {
		t.Fatalf("ops.Boolean(Join, r=%g, dr=%g): %v", r, dr, err)
	}
	return body
}

// TestTheChartMesherBoundsEveryNearPinchBandByItsOwnRim is the regression the boundary-side
// classification exists for (#3518). Before it, the chart mesher kept triangles in the band between
// its chart's contour and the finer chord polygon its shared edges gave it — a skin inside the face's
// own lens — and 1777 rim segments over these sixteen rows came back bounded by TWO triangles instead
// of one. Not one may now.
//
// Only the "does not bound" half is asserted. The other half — unpaired edges that are no rim segment
// — is still non-zero on six of the sixteen for a different reason, at the covering's seam
// (chart_face_replica.go), and TestSixNearPinchRowsStillCarrySeamEdges pins that separately so this
// row cannot quietly absorb it.
func TestTheChartMesherBoundsEveryNearPinchBandByItsOwnRim(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~25s): `make test-corpus`")
	}
	t.Parallel()
	forEachNearPinchBand(t, func(name string, extra, missing int, meshed bool, why string) {
		if !meshed {
			t.Errorf("%s: the chart mesher gave up a near-pinch band it owns — %s", name, why)
			return
		}
		if missing != 0 {
			t.Errorf("%s: %d rim segment(s) the mesh does not bound (%d unpaired edges are no rim segment); "+
				"the region kept triangles inside the face's own lens", name, missing, extra)
		}
	})
}

// TestNoNearPinchRowCarriesASeamEdge was TestSixNearPinchRowsStillCarrySeamEdges, and the six are
// GONE (#3542, closed by covering_vertices.go's coverShear). A covering's premise is that its two
// branch-window ends are the same triangulation; near-cocircular quads at the seam flipped their
// diagonal differently at the two ends, and shearing the triangulator's own frame so a lattice cell is
// not concyclic removes the tie that made them near-cocircular. Measured over the whole sixteen-row
// corpus: 6 rows and 38 seam edges before, 0 and 0 after, across a shear plateau four decades wide.
//
// It still fails in both directions. A rise means the seam regressed. It can no longer fall.
func TestNoNearPinchRowCarriesASeamEdge(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~25s): `make test-corpus`")
	}
	t.Parallel()
	rows, total := 0, 0
	forEachNearPinchBand(t, func(_ string, extra, _ int, meshed bool, _ string) {
		if meshed && extra > 0 {
			rows++
			total += extra
		}
	})
	if rows != 0 || total != 0 {
		t.Errorf("%d of the sixteen near-pinch rows carry seam edges, %d in all; the measurement is 0 and 0 "+
			"since #3542 closed. The covering's two branch-window ends have stopped agreeing — see "+
			"coverShear", rows, total)
	}
}

// forEachNearPinchBand runs visit on every two-rim holed band of the near-pinch join corpus, at both
// facetings — sixteen rows, and the count is asserted so a corpus that stopped presenting them fails.
func forEachNearPinchBand(t *testing.T, visit func(name string, extra, missing int, meshed bool, why string)) {
	t.Helper()
	seen := 0
	for _, gq := range gateQualities() {
		for _, r := range nearPinchJoinRadii() {
			for _, k := range nearPinchDeltas() {
				dr := k * (r / 3.0)
				seen += visitNearPinchBands(t, gq, r, dr, visit)
			}
		}
	}
	if seen != 16 {
		t.Fatalf("the near-pinch corpus presented %d two-rim holed bands; want 16 (8 join bodies × 2 facetings)", seen)
	}
}

// visitNearPinchBands runs visit on each two-rim holed band of one body at one faceting, and returns
// how many it found.
func visitNearPinchBands(t *testing.T, gq gateQuality, r, dr float64, visit func(string, int, int, bool, string)) int {
	t.Helper()
	seen := 0
	for i, f := range nearPinchJoinBody(t, r, dr).Faces() {
		if !tessellate.IsTwoRimHoledBandShape(f, gq.q) {
			continue
		}
		seen++
		extra, missing, meshed, why := tessellate.ChartFaceRimMismatch(f, gq.q)
		visit(fmt.Sprintf("%s R=%g dr=%g face %d", gq.name, r, dr, i), extra, missing, meshed, why)
	}
	return seen
}
