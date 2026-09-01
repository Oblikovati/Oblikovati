// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// bandCylinder is the R=3, z∈[0,h] test cylinder for the two-rim holed-band mesher.
func bandCylinder(h float64) geom.Cylinder {
	c, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	_ = h
	return c
}

// rimLoop samples a full-circle rim at axial v as a 3D point loop on the cylinder.
func rimLoop(s geom.Surface, v float64, n int) []math.Point3 {
	out := make([]math.Point3, 0, n)
	for k := range n {
		out = append(out, s.PointAt(2*stdmath.Pi*float64(k)/float64(n), v))
	}
	return out
}

// lensLoop samples a small closed lens hole centred at (theta0,v0) in the wall's (θ,v) chart.
func lensLoop(s geom.Surface, theta0, v0, rTheta, rV float64, n int) []math.Point3 {
	out := make([]math.Point3, 0, n)
	for k := range n {
		a := 2 * stdmath.Pi * float64(k) / float64(n)
		out = append(out, s.PointAt(theta0+rTheta*stdmath.Cos(a), v0+rV*stdmath.Sin(a)))
	}
	return out
}

// TestTwoRimHoledBandMeshesFullBand: a full cylinder band (two circular rims) carrying one lens hole meshes
// through the seam bridge to the unrolled CDT with the correct curved area (≈ 2πRh, minus the small lens) —
// NOT the flat best-fit-plane patch the generic trim path would give (which would grossly under-report area).
func TestTwoRimHoledBandMeshesFullBand(t *testing.T) {
	t.Parallel()
	s := bandCylinder(10)
	top := rimLoop(s, 10, 96)
	bot := rimLoop(s, 0, 96)
	lens := lensLoop(s, 0, 5, 0.12, 0.35, 40)
	q := Quality{ChordTolerance: 0.005, AngleTolerance: 2 * stdmath.Pi / 180}
	m, ok := twoRimHoledBandMesh(s, top, [][]math.Point3{bot, lens}, q)
	if !ok {
		t.Fatal("twoRimHoledBandMesh declined a full two-rim band with a lens hole")
	}
	full := 2 * stdmath.Pi * 3 * 10 // 188.50
	area := meshArea(m)
	if area < full-3 || area > full+0.5 {
		t.Errorf("band area %.3f; want ≈ %.3f (full band minus a small lens), got a flat/torn patch?", area, full)
	}
	if len(m.Indices) == 0 {
		t.Fatal("empty mesh")
	}
}

// TestTwoRimHoledBandDeclinesWithoutWrappingRim: with no second full-wrap rim among the holes (only a lens),
// the face is the ordinary drilled-wall case holedConicWallMesh owns, so this mesher must defer.
func TestTwoRimHoledBandDeclinesWithoutWrappingRim(t *testing.T) {
	t.Parallel()
	s := bandCylinder(10)
	seamWrap := rimLoop(s, 10, 64) // a single seam-wrapping outer (stand-in) — no rim among the holes
	lens := lensLoop(s, 0, 5, 0.12, 0.35, 40)
	if _, ok := twoRimHoledBandMesh(s, seamWrap, [][]math.Point3{lens}, DefaultQuality()); ok {
		t.Error("meshed a face with zero wrapping rim-holes; want decline (holedConicWallMesh's case)")
	}
}

// TestSplitWrappingHolesPartitions: a full-circle rim is classed as a wrapping rim, a small lens as a lens.
func TestSplitWrappingHolesPartitions(t *testing.T) {
	t.Parallel()
	s := bandCylinder(10)
	rim := rimLoop(s, 0, 64)
	lens := lensLoop(s, stdmath.Pi, 5, 0.1, 0.3, 32)
	rims, lenses := splitWrappingHoles(s, [][]math.Point3{rim, lens})
	if len(rims) != 1 || len(lenses) != 1 {
		t.Fatalf("splitWrappingHoles got rims=%d lenses=%d; want 1 and 1", len(rims), len(lenses))
	}
	if !holeWrapsPeriod(s, rim) || holeWrapsPeriod(s, lens) {
		t.Error("holeWrapsPeriod misclassified the rim or the lens")
	}
}
