// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Saddle-band loft tessellation (M2 Phase 2, Oblikovati/Oblikovati#1335). A cylinder's wall inside a
// crossing cylinder is a full-period band with non-circular (saddle) rims. These tests build that band as
// a real face and check the lofted mesh against the analytic lateral area — the rod-of-radius-r wall
// inside the fat-cylinder-of-radius-R, whose width at angle α is 2√(R²−r²cos²α).

const bandR, bandr = 3.0, 1.5

// analyticBandArea numerically integrates the rod-wall lateral area ∫ r·2√(R²−r²cos²α) dα over [0,2π].
func analyticBandArea(rRod, rFat float64) float64 {
	const n = 20000
	sum := 0.0
	for i := range n {
		a := 2 * stdmath.Pi * (float64(i) + 0.5) / n
		sum += rRod * 2 * stdmath.Sqrt(rFat*rFat-rRod*rRod*stdmath.Cos(a)*stdmath.Cos(a))
	}
	return sum * 2 * stdmath.Pi / n
}

// rodInsideFatBand builds the periodic cylinder-side face of a rod (radius r, axis x) trimmed to the band
// inside a fat cylinder (radius R, axis z): its two rims are the saddle curves x = ±√(R²−r²cos²α).
func rodInsideFatBand(t *testing.T, samples int) *topo.Face {
	t.Helper()
	rod, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), bandr)
	loPts, hiPts := saddleRimPoints(samples)
	loPoly, _ := geom.NewPolyline(loPts)
	hiPoly, _ := geom.NewPolyline(hiPts)
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("band", "body", 0)))
	vLo := bld.AddVertex(loPts[0], topo.NewLineage(topo.Tok("band", "vlo", 0)))
	vHi := bld.AddVertex(hiPts[0], topo.NewLineage(topo.Tok("band", "vhi", 0)))
	eLo := bld.AddEdge(loPoly, vLo, vLo, topo.NewLineage(topo.Tok("band", "elo", 0)))
	eHi := bld.AddEdge(hiPoly, vHi, vHi, topo.NewLineage(topo.Tok("band", "ehi", 0)))
	eSeam := bld.AddEdge(geom.NewLineSegment(loPts[0], hiPts[0]), vLo, vHi, topo.NewLineage(topo.Tok("band", "seam", 0)))
	bld.AddFace(rod, topo.NewLineage(topo.Tok("band", "face", 0)),
		topo.OuterLoop(topo.Fwd(eSeam), topo.Rev(eHi), topo.Rev(eSeam), topo.Fwd(eLo)))
	return bld.Build().Faces()[0]
}

// saddleRimPoints returns the lo (x<0) and hi (x>0) rim polylines, closed (last point repeats the first).
func saddleRimPoints(samples int) (lo, hi []math.Point3) {
	for i := 0; i <= samples; i++ {
		a := 2 * stdmath.Pi * float64(i%samples) / float64(samples)
		x := stdmath.Sqrt(bandR*bandR - bandr*bandr*stdmath.Cos(a)*stdmath.Cos(a))
		y, z := bandr*stdmath.Cos(a), bandr*stdmath.Sin(a)
		lo = append(lo, math.P3(math.Scalar(-x), math.Scalar(y), math.Scalar(z)))
		hi = append(hi, math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)))
	}
	return lo, hi
}

// TestSaddleBandLoftAreaMatchesAnalytic builds the rod-inside-fat band face and checks the lofted mesh's
// area against the analytic lateral area — confirming the loft follows the saddle rims, not a flat or
// full-domain fallback.
func TestSaddleBandLoftAreaMatchesAnalytic(t *testing.T) {
	t.Parallel()
	face := rodInsideFatBand(t, 64)
	mesh := tessellateCurvedFace(face, DefaultQuality())
	if mesh == nil || len(mesh.Indices) == 0 {
		t.Fatal("saddle band produced no mesh")
	}
	got := meshArea(mesh)
	want := analyticBandArea(bandr, bandR)
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("saddle band area %.4f, want %.4f (analytic) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestSaddleBandLoftMeshIsRouted confirms the dispatch reaches saddleBandLoftMesh for this band (a
// non-circular-rim periodic face) rather than a fallback: the helper returns ok.
func TestSaddleBandLoftMeshIsRouted(t *testing.T) {
	t.Parallel()
	face := rodInsideFatBand(t, 48)
	if _, ok := saddleBandLoftMesh(face, face.Geometry(), DefaultQuality()); !ok {
		t.Error("saddleBandLoftMesh declined the rod-inside-fat band; the dispatch would fall back")
	}
}

// TestSaddleBandLoftDeclinesNonBand: a planar face is not a singly-periodic ruled band, so the loft must
// decline (ok=false) and leave the dispatch to its other meshers.
func TestSaddleBandLoftDeclinesNonBand(t *testing.T) {
	t.Parallel()
	block, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "b")
	face := block.Faces()[0] // a planar face: both parameter directions are non-periodic
	if _, ok := saddleBandLoftMesh(face, face.Geometry(), DefaultQuality()); ok {
		t.Error("a planar face is not a periodic band; saddleBandLoftMesh should decline")
	}
}

// TestBandWrapRingsReadsTheSharedEdgeDiscretization is the regression gate for the ring-read bypass
// that leaked simple/U4's 44 free edges: bandWrapRings must read each rim through discretizeEdge — the
// ONE polyline every other face of that edge already uses (edge_discretize.go's package doc) — not
// through the raw TessellateEdge curve sampler.
//
// The two differ on exactly two inputs, and both are inputs a band rim really carries: a HEALED edge
// (M25, whose stored on-surface polyline discretizeEdge returns verbatim) and a straight edge under the
// #2009 starved-rail densification. This test drives the healed case, because it is hermetic — one
// SetSnappedCurve on a fixture rim — while the densification case needs a whole high-aspect B-spline
// neighbour; U4 itself gates that one end to end (TestU4DualHostMeshIsClosedAtEveryQuality).
//
// Falsification: restore `TessellateEdge(e, q)` in bandWrapRings and this fails, because the raw
// sampler re-samples the rim's own curve and never sees the stored polyline.
// densePinchRing builds a smooth, genuinely single-valued v(u) ring on a radius-R cylinder — N points
// evenly spaced in angle around the full 2π, on the continuous curve v = R + amplitude·cos(u) — PLUS one
// extra sample spliced in at an infinitesimal angular offset from an existing station. That extra sample
// mimics what curvature-driven chord/sagitta refinement does near a near-pinch crossing (#2009's whole
// argument): it packs extra points in tight where the curve bends hardest, so two ADJACENT sorted
// samples can land a sliver of a radian apart even on an ordinary, nowhere-vertical curve.
func densePinchRing(cyl geom.Cylinder, n int, tinyGap float64) []math.Point3 {
	v := func(u float64) float64 { return cyl.Radius + 0.02*cyl.Radius*stdmath.Cos(u) }
	pts := make([]math.Point3, 0, n+1)
	for i := range n {
		u := 2 * stdmath.Pi * float64(i) / float64(n)
		pts = append(pts, cyl.PointAt(u, v(u)))
	}
	uExtra := 2*stdmath.Pi*float64(n/2)/float64(n) + tinyGap // one sliver-close neighbour of an existing station
	pts = append(pts, cyl.PointAt(uExtra, v(uExtra)))
	return pts
}

// TestRingSingleValuedInAngleAcceptsDenseNearPinchSampling is the regression gate for the near-pinch
// tessellation defect (Oblikovati/Oblikovati, 2026-07-31): ringSingleValuedInAngle used to compare the
// raw angle gap between adjacent sorted samples against a BARE radians constant (seamAngularTol),
// which is not model-relative (ADR-0042) — it ignores the ring's local radius, so "how close is too
// close" silently changed meaning with scale and with sampling density. A near-pinch crossing's saddle
// rim is sampled very densely right where curvature peaks, so two adjacent sorted samples land under
// that fixed 1e-6 rad constant while still moving ~300x the model's own weld tolerance once the local
// radius is folded in — the guard declined a perfectly good v(u) rim, saddleBandLoftMesh fell through to
// the general trimmed-CDT path, and TestNearPinchCutJoinWatertight regressed to 109-244 free edges.
//
// Falsification: reintroduce the bare-epsilon comparison (`angles[i]-angles[i-1] < seamAngularTol`) and
// this fails, because densePinchRing's spliced-in sample sits at a sub-radian gap that trips it even
// though the ring is genuinely single-valued.
func TestRingSingleValuedInAngleAcceptsDenseNearPinchSampling(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3.0)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	ring := orderedRing(cyl, densePinchRing(cyl, 400, 8e-7))
	if !ringSingleValuedInAngle(cyl, ring) {
		t.Error("ringSingleValuedInAngle declined a densely-sampled but genuinely single-valued ring — " +
			"the near-pinch regression (spurious vertical-step flag on curvature-dense sampling)")
	}
}

// TestRingSingleValuedInAngleDeclinesGenuineVerticalStep is the mutation-proof companion to the test
// above: it pins the guard's ORIGINAL purpose (commit e729cfe5, "closed seam-arc reversal keeps its own
// parameters; saddle-loft declines a vertical-step rim") by building a rim that carries a REAL vertical
// segment — two points at the SAME angle (to float noise) but a macroscopic axial (v) jump, the pattern
// a miter rail / seam-line remnant / bridge riser leaves in a rim. The scale-aware rewrite must still
// decline this, or the near-pinch fix would just be reverting the guard rather than sharpening it.
func TestRingSingleValuedInAngleDeclinesGenuineVerticalStep(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3.0)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	pts := densePinchRing(cyl, 400, 8e-7)
	const uStep, vJump = stdmath.Pi, 1.5 // a real vertical run: same angle, half-radius axial jump
	pts = append(pts, cyl.PointAt(uStep, cyl.Radius), cyl.PointAt(uStep, cyl.Radius+vJump))
	ring := orderedRing(cyl, pts)
	if ringSingleValuedInAngle(cyl, ring) {
		t.Error("ringSingleValuedInAngle accepted a rim carrying a genuine vertical step — the scale-aware " +
			"rewrite must still decline this (it is what e729cfe5 was protecting against)")
	}
}

func TestBandWrapRingsReadsTheSharedEdgeDiscretization(t *testing.T) {
	t.Parallel()
	face := rodInsideFatBand(t, 48)
	var healed *topo.Edge
	for _, e := range face.Edges() {
		if e.StartVertex() == e.EndVertex() {
			healed = e
			break
		}
	}
	if healed == nil {
		t.Fatal("fixture precondition: the band has no closed rim edge to heal")
	}
	snapped := []math.Point3{}
	for i := 0; i <= 7; i++ {
		snapped = append(snapped, healed.Geometry().PointAt(float64(i)/7))
	}
	healed.SetSnappedCurve(snapped, 0)
	want := dropClosingDup(discretizeEdge(healed, DefaultQuality()))
	for _, ring := range bandWrapRings(face, DefaultQuality()) {
		if len(ring) != len(want) {
			continue
		}
		for i := range ring {
			if ring[i] != want[i] {
				t.Fatalf("ring point %d = %v, want the shared discretizeEdge point %v — bandWrapRings is "+
					"not reading the edge's shared polyline", i, ring[i], want[i])
			}
		}
		return
	}
	t.Fatalf("no bandWrapRings ring matched the healed rim's %d-point shared discretization — the loft "+
		"re-sampled the rim instead of reading discretizeEdge", len(want))
}
