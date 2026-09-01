// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// complex/D8's MULTI-FACE far-end trim: the atomicity guard, and the closed forms it buys.
//
// WHAT THE BAND ACTUALLY STOPS ON. The r=30 band's terminal section does not stop on one wall. It stops on
// the tangent-continuous CHAIN of the radius-24 corner round and the flat wall tangent to that round, and
// it crosses from one to the other at the triple point band ∩ round ∩ wall — analytically
// (223.39418029785, 35.093784332275, −20+√864). Every station past that crossing used to be slid onto the
// round's IMPLICIT cylinder instead, 6.064 of developed length into the flat wall's territory
// (selfcross-trim-report.md §3), which is what made the round's own developed boundary self-cross.
//
// WHY THE GUARD IS ATOMICITY AND NOT AN AREA. The band's cap and the four host boundaries are the SAME
// curves. Adopting the split on one side only opens the shell along the difference — measured on simple/Y2,
// where routing the host plane alone took it 8475 → 8450 while its band still claimed the old span
// (chain-retrim-report.md §5.2). So the guard states the junction as a shared VERTEX of all three faces
// that meet there, which no half-application can satisfy, and then pins each face on its own closed form.
//
// D8 STAYS FAIL(area), AND THAT IS CORRECT. DRAWEXE propagates the blend along the whole tangent-continuous
// top-edge loop (8 edges, 18 faces); we fillet the one picked edge (11 faces). The single-edge body's own
// closed-form total is 348183.68 against DRAWEXE's 336159 — +3.58%, unreachable at deps 0.01. Only
// tangent-chain propagation can close that, so nothing here is gated on the whole-body number.

const (
	d8RoundAxisX = 223.39418029785 // the corner rounds' axis x
	d8BandAxisX  = 217.39418029785 // the fillet band's axis x
	d8BandAxisZ  = -20.0           // the fillet band's axis z
	d8RoundR     = 24.0            // the corner rounds' radius
	d8BandR      = 30.0            // the fillet radius
	d8PlateBotZ  = -90.0           // the plate's underside
	d8PlateTopZ  = 10.0            // the plate's top face (also the filleted edge's own z)
	d8LeftWallX  = 36.478778839111 // the far x wall
	d8FlatYLo    = 35.093784332275 // the flat wall the near corner rounds hand off to
	d8FlatYHi    = 545.93713378906 // its mirror
	d8CornerX    = 60.478778839111 // the far corner rounds' axis x
)

// TestD8FarEndSplitIsAtomicAndHitsItsClosedForms is the routing invariant's guard: the split is applied to
// the band AND to every host it touches, or to none of them.
func TestD8FarEndSplitIsAtomicAndHitsItsClosedForms(t *testing.T) {
	t.Parallel()
	body := gridCaseBody(t, corpusRecord(t, "complex", "D8"))
	assertD8SurfacesMatchTheConstants(t, body)
	assertD8JunctionIsSharedByAllThreeFaces(t, body)
	for _, tc := range []struct {
		lineage string
		want    float64
	}{
		{"import:step#16:face#1", d8CornerRoundArea()},
		{"import:step#16:face#2", d8CornerRoundArea()},
		{"import:step#16:face#7", d8FlatWallArea()},
		{"import:step#16:face#3", d8FlatWallArea()},
		{"import:step#16:face#9", d8TopPlaneArea()},
	} {
		got := ops.MeshArea(shippedFaceMesh(t, body, faceByLineage(t, body, tc.lineage)))
		if stdmath.Abs(got-tc.want)/tc.want > d8PerFaceTol {
			t.Errorf("complex/D8 %s meshes %.7g, closed form %.7g (rel %+.5f%%, budget %.1g)",
				tc.lineage, got, tc.want, (got-tc.want)/tc.want*100, d8PerFaceTol)
		}
	}
	if bad := blend.SelfCrossingFaceLoops(body, ops.PropertyQuality()); len(bad) != 0 {
		t.Errorf("complex/D8 self-crosses on %d face loop(s): %s", len(bad), describeSelfCrossing(bad))
	}
	assertD8MeshIsWatertightAtEveryQuality(t, body)
}

// assertD8MeshIsWatertightAtEveryQuality is what the self-crossing boundary actually cost downstream, and
// the measurement no previous slice took: with the trim curve bounding two faces at once, the corner
// round's triangulation did not tile the same boundary its neighbours did, and D8's welded body mesh
// LEAKED — 36 free edges at property quality, 8 at default, on a body the scoreboard called healthy. The
// split closes it to 0 at both. Fold-freeness held (0) throughout and must stay so.
//
// It measures through shippedMeshFreeEdges — the corpus-wide ratchet's own measurement
// (welded_mesh_leak_test.go) — rather than the hand-rolled welder it used to carry, so D8's zero and the
// ratchet's ceilings are read by ONE ruler, the production model-relative tessellate.FreeEdgeCount. D8 is
// deliberately absent from knownMeshLeaks: its ceiling is this zero.
func assertD8MeshIsWatertightAtEveryQuality(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, gq := range gateQualities() {
		folds := 0
		for _, m := range tessellate.CalculateBodyFacets(body, gq.q).FaceMeshes {
			folds += ops.FoldEdgeCount(m)
		}
		free := shippedMeshFreeEdges(body, gq.q)
		if folds != 0 || free != 0 {
			t.Errorf("complex/D8 body mesh at %s quality: %d fold edges, %d free edges; want 0 and 0",
				gq.name, folds, free)
		}
	}
}

// d8PerFaceTol is mesh quantization only — the five faces measure −0.0016 %, −0.0016 %, −1.5e-5 %,
// −1.5e-5 % and −2.5e-5 % of their closed forms, decades inside it. The defect it gates against is
// +0.77 % / +2.58 % on the two rounds and +1.5144 per corner on the top plane.
const d8PerFaceTol = 2e-4

// d8CornerRoundArea is the surviving area of ONE radius-24 corner round after the band bites it, in closed
// form. Parameterise the round by θ ∈ [0, π/2], θ=0 at the ruling it shares with the flat wall and θ=π/2 at
// the ruling it shares with the filleted wall, so x = axis + R sin θ. The band's own cylinder puts the trim
// at z(θ) = zBand + √(Rb² − (Δx + R sin θ)²), Δx being the two axes' x offset, and the face runs from the
// plate's underside up to it: R·∫₀^{π/2} (z(θ) − zBot) dθ = 3307.1167946.
func d8CornerRoundArea() float64 {
	dx := d8RoundAxisX - d8BandAxisX
	return d8RoundR * simpson(func(th float64) float64 {
		off := dx + d8RoundR*stdmath.Sin(th)
		return d8BandAxisZ + stdmath.Sqrt(stdmath.Max(d8BandR*d8BandR-off*off, 0)) - d8PlateBotZ
	}, 0, stdmath.Pi/2)
}

// d8FlatWallArea is the flat wall the round hands the trim off to, less the bite the band now takes out of
// its top corner: the full rectangle minus ∫₀^{Δx} (Rb − √(Rb² − t²)) dt = 16291.5401 − 1.2073.
func d8FlatWallArea() float64 {
	dx := d8RoundAxisX - d8BandAxisX
	bite := simpson(func(t float64) float64 {
		return d8BandR - stdmath.Sqrt(stdmath.Max(d8BandR*d8BandR-t*t, 0))
	}, 0, dx)
	return (d8RoundAxisX-d8CornerX)*(d8PlateTopZ-d8PlateBotZ) - bite
}

// d8TopPlaneArea is the plate's top face after the fillet: the rectangle from the far wall out to the
// band's own tangent line, less the two far corner rounds' quarter-disc notches. It carries NO detour
// around the near rounds any more — the trim ends on this face's own y = 35.09378 boundary edge, not
// 0.762 inside it, so the 1.5144 per corner the old boundary enclosed (territory the band had already
// removed) is gone.
func d8TopPlaneArea() float64 {
	notch := d8RoundR*d8RoundR - stdmath.Pi*d8RoundR*d8RoundR/4
	return (d8BandAxisX-d8LeftWallX)*(d8FlatYHi-d8FlatYLo) - 2*notch
}

// assertD8SurfacesMatchTheConstants keeps the closed forms above tied to the body they describe: if the
// fixture or the importer ever moves, the constants fail loud instead of silently measuring nothing.
func assertD8SurfacesMatchTheConstants(t *testing.T, body *topo.Body) {
	t.Helper()
	round := faceByLineage(t, body, "import:step#16:face#1").Geometry().(geom.Cylinder)
	band := faceByLineage(t, body, "import:step#16:edge#1/fillet:cyl#0").Geometry().(geom.Cylinder)
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"round radius", round.Radius, d8RoundR},
		{"round axis x", float64(round.Origin.X), d8RoundAxisX},
		{"band radius", band.Radius, d8BandR},
		{"band axis x", float64(band.Origin.X), d8BandAxisX},
		{"band axis z", float64(band.Origin.Z), d8BandAxisZ},
	} {
		if stdmath.Abs(c.got-c.want) > 1e-9 {
			t.Fatalf("complex/D8 %s is %.12g, the closed forms assume %.12g", c.name, c.got, c.want)
		}
	}
}

// assertD8JunctionIsSharedByAllThreeFaces is the ATOMICITY statement. The trim crosses from the round to
// the flat wall at the triple point band ∩ round ∩ flat wall — (223.39418029785, 35.093784332275,
// −20+√864), where √864 = √(Rb²−Δx²). All three faces must carry it as a boundary vertex; a split adopted
// by the hosts but not by the band (or the reverse) leaves the band's cap running somewhere else entirely,
// and the point is on two faces or none.
func assertD8JunctionIsSharedByAllThreeFaces(t *testing.T, body *topo.Body) {
	t.Helper()
	dx := d8RoundAxisX - d8BandAxisX
	j := math.P3(d8RoundAxisX, d8FlatYLo, math.Scalar(d8BandAxisZ+stdmath.Sqrt(d8BandR*d8BandR-dx*dx)))
	for _, lineage := range []string{
		"import:step#16:face#1", "import:step#16:face#7", "import:step#16:edge#1/fillet:cyl#0",
	} {
		if !faceHasVertexAt(faceByLineage(t, body, lineage), j) {
			t.Errorf("complex/D8: face %s does not meet the round/flat-wall hand-off at %v — the far-end "+
				"split was applied to some of the faces sharing that trim but not to all of them", lineage, j)
		}
	}
}

// faceHasVertexAt reports whether any boundary vertex of f sits on p (1e-6 absolute, four decades under
// the 0.762 the old tangent point missed the flat wall by).
func faceHasVertexAt(f *topo.Face, p math.Point3) bool {
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			if float64(useFromVertexPoint(u).DistanceTo(p)) <= 1e-6 {
				return true
			}
		}
	}
	return false
}

// useFromVertexPoint is the position of an edge use's start vertex, honouring reversal.
func useFromVertexPoint(u *topo.EdgeUse) math.Point3 {
	if u.Reversed() {
		return u.Edge().EndVertex().Point()
	}
	return u.Edge().StartVertex().Point()
}
