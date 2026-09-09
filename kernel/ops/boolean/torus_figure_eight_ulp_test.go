// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
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
// parameter — at which every decision holds on every platform. It is {0, +1}, and it is short
// because of what was measured, not because less was tried.
//
// The number below is the intersect piece's torus-face mesh area at the property faceting. Its own
// share is 111.68 mm²; 394.75 is the WHOLE torus, i.e. the tangency classification has taken the
// entire tube for the cap and the piece is wrong:
//
//	k (ulps of 5)   −3       −2       −1        0       +1       +2
//	amd64          394.75   394.75   111.67   111.67   111.67   111.67
//	arm64          394.75   111.67   394.75   111.67   111.67   394.75
//
// amd64 has a boundary at −2 and arm64 has none: the verdict alternates. So within a few ulps of
// the tangency this decision is not a function of the geometry, it is a function of the rounding —
// and ADR-0064 cannot fix it, because it lives in kernel/brep and kernel/ops, which still contract.
// Both rows are pre-existing: amd64 reproduces bit-for-bit at this wave's base 6590a9ba in a clean
// worktree, and arm64 at the base is wrong at −2 where amd64 is wrong at −2 and −3.
var figureEightOffsetUlps = []int{0, 1}

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

// figureEightMisreadOffsets are the tangency offsets just below the exact tangency where the
// classification is known to fail. It fails at a DIFFERENT one on each platform (see the table on
// figureEightOffsetUlps), so the pin below asserts that at least one of them still fails rather
// than naming which — the defect is that any of them does.
var figureEightMisreadOffsets = []int{-1, -2, -3}

// TestTheFigureEightMisreadsAnOffsetJustBelowTheTangency pins the defect the row above stops short
// of, so the band cannot stay narrow by inattention.
//
// Within three ulps BELOW the exact tangency the intersect piece's torus face meshes the whole
// torus — 394.75 mm² where its share is 111.68 — while meshing its share at the default faceting,
// so the face's own region is read differently at two facetings. Which offsets do it depends on the
// platform, which is the point: nothing about the geometry changes over 2.7e-15 of a 5 mm radius.
// It is pre-existing (it reproduces at this wave's base 6590a9ba in a clean worktree) and outside
// ADR-0064's scope, which converts the arithmetic floor and not kernel/brep or kernel/ops.
//
// When this test fails, the classification is fixed: widen figureEightOffsetUlps and delete it.
func TestTheFigureEightMisreadsAnOffsetJustBelowTheTangency(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~10s): `make test-corpus`")
	}
	t.Parallel()
	for _, k := range figureEightMisreadOffsets {
		if figureEightIntersectIsMisread(t, k) {
			return
		}
	}
	t.Fatalf("the intersect piece is now correct at every offset in %v ulps below the tangency — "+
		"the classification survives the perturbation. Widen figureEightOffsetUlps and delete this "+
		"test (ADR-0064, #3528)", figureEightMisreadOffsets)
}

// figureEightIntersectIsMisread reports whether the intersect piece at k ulps below the tangency
// comes back carrying the whole torus (or is refused outright, which is also "not the right body").
func figureEightIntersectIsMisread(t *testing.T, k int) bool {
	t.Helper()
	target, tool := perturbedFigureEight(t, ulpPerturbation{offset: k})
	res, err := ops.Boolean(ops.Intersect, target, tool)
	if err != nil {
		return true
	}
	return torusMeshArea(t, res) > figureEightTorusArea*0.9
}

// torusMeshArea is the total mesh area of a body's torus faces at the PROPERTY faceting — the one
// the misread offsets misread.
func torusMeshArea(t *testing.T, b *topo.Body) float64 {
	t.Helper()
	sum := 0.0
	for _, f := range b.Faces() {
		if _, isTorus := f.Geometry().(geom.Torus); isTorus {
			sum += tessellate.MeshGeometryProperties(tessellate.TessellateFace(f, ops.PropertyQuality())).Area
		}
	}
	return sum
}
