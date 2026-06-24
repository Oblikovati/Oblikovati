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
	for i := 0; i < n; i++ {
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
	face := rodInsideFatBand(t, 48)
	if _, ok := saddleBandLoftMesh(face, face.Geometry(), DefaultQuality()); !ok {
		t.Error("saddleBandLoftMesh declined the rod-inside-fat band; the dispatch would fall back")
	}
}

// TestSaddleBandLoftDeclinesNonBand: a planar face is not a singly-periodic ruled band, so the loft must
// decline (ok=false) and leave the dispatch to its other meshers.
func TestSaddleBandLoftDeclinesNonBand(t *testing.T) {
	block, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "b")
	face := block.Faces()[0] // a planar face: both parameter directions are non-periodic
	if _, ok := saddleBandLoftMesh(face, face.Geometry(), DefaultQuality()); ok {
		t.Error("a planar face is not a periodic band; saddleBandLoftMesh should decline")
	}
}
