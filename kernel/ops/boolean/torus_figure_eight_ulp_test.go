// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The ulp-perturbation row on the figure-eight fixture (ADR-0064, Oblikovati#3528).
//
// ADR-0064 takes the compiler's fused multiply-add out of the arithmetic floor so intermediate
// values stop depending on the platform. That is worth nothing on its own: a pipeline whose
// classification flips when a coordinate moves in its last bit is fragile whatever produced the
// bit. So this row moves the fixture's inputs DELIBERATELY, by whole ulps of the model's own
// scale, and asserts that every DECISION downstream is unchanged — the boolean resolves, the piece
// inventory is the same, and the two torus faces still partition the torus at BOTH facetings.
//
// The figure eight is the right fixture because it is maximally sensitive by construction: the box
// face sits EXACTLY on the torus's inner equator (offset R−r = 3), so the section is one pinched
// curve. ADR-0061 records an fmahash bisect showing many independent contraction sites reach it.
const figureEightUlp = 8.881784197001252e-16 // ulp(5): the last bit of this model's major radius

// figureEightOffsetUlps are the perturbations of the TANGENCY OFFSET — the fixture's bifurcation
// parameter — at which every decision holds on every platform.
//
// It was {0, +1}, because the intersect piece's torus face used to mesh the WHOLE torus (394.75 mm²
// against its own share of 111.68) at offsets just below the tangency, on a pattern that differed
// between platforms:
//
//	k (ulps of 5)   −3       −2       −1        0       +1       +2
//	amd64          394.75   394.75   111.67   111.67   111.67   111.67
//	arm64          394.75   111.67   394.75   111.67   111.67   394.75
//
// That was never a rounding defect in the arithmetic, which is why ADR-0064 could not reach it. It
// was #3551: the covering laid THREE vertices at the pinch — the loop passes it twice and the shift
// that carries the far pass back lands a third copy there — and a constrained triangulation cannot
// recover a constraint incident to a vertex another vertex sits on. Whether recovery happened to
// succeed depended on the neighbourhood, which is why the verdict alternated with the last bit and
// with the platform. The covering merges coincident locations now (coverVertices.
// MergeCoincidentLocations) and the misread is gone.
//
// RE-MEASURED over every offset from −8 to +8: the intersect piece meshes 111.67488 mm² and the cut
// piece 283.07521 at EVERY offset from −8 to +4 (111.67492 / 283.07521 at −8 and −7). At +5 and
// beyond the BOOLEAN refuses the fixture outright — "intersect of a 1-face target and a 6-face tool:
// the exact result failed its own acceptance gate" — because the plane has stopped touching and the
// section is two ovals; that is the tangential-contact gap (#3552), not this row's.
//
// The set below spans the three offsets that used to be misread and the widest that still build.
var figureEightOffsetUlps = []int{-8, -3, -2, -1, 0, 1, 2, 4}

// figureEightCentreUlps perturbs the torus CENTRE in x and z, which is NOT a bifurcation parameter:
// the plane y=3 is tangent to the inner equator whatever they are, so no decision may move at any
// magnitude. Eight ulps says so with room, on both platforms.
var figureEightCentreUlps = []int{-8, 8}

// ulpPerturbation is one row of the grid: how far to move each knob, in ulps of the model scale.
type ulpPerturbation struct {
	name   string
	offset int // the box face at y = 3, the tangency offset
	centre int // the torus centre in x and z
}

// TestTheFigureEightHoldsUnderAnUlpPerturbation is the corpus row.
func TestTheFigureEightHoldsUnderAnUlpPerturbation(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~25s): `make test-corpus`")
	}
	t.Parallel()
	for _, row := range figureEightUlpRows() {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			assertFigureEightPartitionsTheTorus(t, row)
		})
	}
}

// figureEightUlpRows is the perturbation grid.
func figureEightUlpRows() []ulpPerturbation {
	var rows []ulpPerturbation
	for _, k := range figureEightOffsetUlps {
		rows = append(rows, ulpPerturbation{fmt.Sprintf("tangency offset %+dulp", k), k, 0})
	}
	for _, k := range figureEightCentreUlps {
		rows = append(rows, ulpPerturbation{fmt.Sprintf("centre %+dulp", k), 0, k})
	}
	return rows
}

// perturbedFigureEight builds the fixture with each knob moved by its whole number of ulps.
func perturbedFigureEight(t *testing.T, p ulpPerturbation) (*topo.Body, *topo.Body) {
	t.Helper()
	c := float64(p.centre) * figureEightUlp
	tor, err := brep.SolidTorus(math.P3(c, 0, c), math.V3(0, 0, 1), 5, 2, "torus")
	if err != nil {
		t.Fatalf("SolidTorus at centre %+d ulp: %v", p.centre, err)
	}
	y := 3 + float64(p.offset)*figureEightUlp
	blk, err := brep.SolidBlock(math.P3(-20, y, -20), math.P3(20, 20, 20), "block")
	if err != nil {
		t.Fatalf("SolidBlock at y=%.17g: %v", y, err)
	}
	return tor, blk
}

// assertFigureEightPartitionsTheTorus is the whole decision set for one perturbed fixture: both
// booleans resolve, each piece carries the same inventory, and the two torus faces still partition
// the torus. It is the unperturbed row's gate, re-run off the model's last bit.
func assertFigureEightPartitionsTheTorus(t *testing.T, p ulpPerturbation) {
	t.Helper()
	cut := perturbedFigureEightPiece(t, p, ops.Cut)
	intersect := perturbedFigureEightPiece(t, p, ops.Intersect)
	// BOTH facetings, and that is load-bearing rather than thorough: a misread piece meshes its own
	// share at the default quality and the WHOLE torus at the property one, so a single faceting
	// would have called the failing rows green.
	for _, gq := range figureEightQualities() {
		below := torusFaceArea(t, cut, gq.q)
		above := torusFaceArea(t, intersect, gq.q)
		assertChordDeficit(t, p.name+" "+gq.name+" below y=3", below, figureEightBelowArea)
		assertChordDeficit(t, p.name+" "+gq.name+" above y=3", above, figureEightAboveArea)
		assertChordDeficit(t, p.name+" "+gq.name+" sum", below+above, figureEightTorusArea)
	}
}

// perturbedFigureEightPiece runs one boolean and returns the result, asserting the inventory the
// area gate is only meaningful against.
func perturbedFigureEightPiece(t *testing.T, p ulpPerturbation, op ops.PartFeatureOperation) *topo.Body {
	t.Helper()
	target, tool := perturbedFigureEight(t, p)
	res, err := ops.Boolean(op, target, tool)
	if err != nil {
		t.Fatalf("%s: Boolean(%v) refused a fixture %d/%d ulps off the tangency: %v",
			p.name, op, p.offset, p.centre, err)
	}
	if n := countTorusFaces(res); n != 1 {
		t.Fatalf("%s: the %v piece has %d torus faces, want 1", p.name, op, n)
	}
	if n := len(res.Faces()); n != 3 {
		t.Fatalf("%s: the %v piece has %d faces, want 3 (torus band + two lids)", p.name, op, n)
	}
	return res
}
