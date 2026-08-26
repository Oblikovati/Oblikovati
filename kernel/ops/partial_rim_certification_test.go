// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Disjoint partial-rim certification (EPIC Oblikovati/Oblikovati#1724, #1732, ADR-0046). A SECOND curved cut
// on an already-notched cylinder side, the new rod imprint lying entirely in the still-full lower band (clear
// of the prior notch). This is the positive path the coupled-arrangement overlay ships; the corner-junction
// (imprint crossing the prior arc) is out of scope and keeps declining OBSERVABLY
// (TestPartialRimInteractingCutFallsBackObservably). Certified NOT by the internal manifold/volume nets alone
// but against an INDEPENDENT two-cut moment set from OpenCASCADE — occ.cut(occ.cut(cyl, halfspace), rod) with
// occ.getMass exact B-rep moments, experiments/occ-boolean-oracle/partial_rim_disjoint_oracle.py — PLUS a
// point-membership audit vs the analytic predicate ((target \ notch) \ rod), the strong right-volume-wrong-
// shape oracle.

// OCC ground truth for the disjoint partial-rim fixture, from
// experiments/occ-boolean-oracle/partial_rim_disjoint_oracle.py (two chained occ.cut; exact BRepGProp moments).
const (
	occPartialRimVol   = 238.3426419
	occPartialRimArea  = 259.4287354
	occPartialRimCx    = -0.1706961
	occPartialRimCy    = 0.0
	occPartialRimCz    = 4.7267794
	occPartialRimFaces = 5 // holed wall + bottom cap + notch cap + rod tunnel (split by the seam)
)

// disjointRod is the second-cut tool: a rod r=1 axis +x at z=3, drilled through the still-full lower band of
// notchedTarget — well below the notch floor (z=6.5 at the front rim) so its imprint is disjoint from the
// prior section arc. Kept in lockstep with partial_rim_disjoint_oracle.py.
func disjointRod(t *testing.T) *topo.Body {
	t.Helper()
	rod, err := brep.SolidCylinder(math.P3(-6, 0, 3), math.V3(1, 0, 0), 1, 12)
	if err != nil {
		t.Fatalf("disjoint rod SolidCylinder: %v", err)
	}
	return rod
}

func TestPartialRimDisjointCutMomentsMatchOCC(t *testing.T) {
	res, err := Boolean(Cut, notchedTarget(t), disjointRod(t))
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if n := len(res.Faces()); n != occPartialRimFaces {
		t.Errorf("partial-rim cut has %d faces; want %d (holed wall + bottom cap + notch cap + tunnel)", n, occPartialRimFaces)
	}
	gp := BodyGeometryProperties(res, capCertQuality())
	if rel := stdmath.Abs(gp.Volume-occPartialRimVol) / occPartialRimVol; rel > 0.006 {
		t.Errorf("volume %.4f vs OCC %.4f (rel %.4f > 0.006) — beyond the SSI-imprint facet deficit", gp.Volume, occPartialRimVol, rel)
	}
	if rel := stdmath.Abs(gp.Area-occPartialRimArea) / occPartialRimArea; rel > 0.004 {
		t.Errorf("area %.4f vs OCC %.4f (rel %.4f > 0.004)", gp.Area, occPartialRimArea, rel)
	}
	occCentroid := math.P3(occPartialRimCx, occPartialRimCy, occPartialRimCz)
	if d := float64(gp.Centroid.DistanceTo(occCentroid)); d > 0.05 {
		t.Errorf("centroid (%.4f,%.4f,%.4f) vs OCC (%.4f,%.4f,%.4f): distance %.4f > 0.05",
			gp.Centroid.X, gp.Centroid.Y, gp.Centroid.Z, occPartialRimCx, occPartialRimCy, occPartialRimCz, d)
	}
}

// TestPartialRimDisjointCutMembershipMatchesCSG is the strong shape oracle: it samples an interior grid and
// asserts the built solid's inside/outside agrees with the analytic predicate ((target \ notch) \ rod)
// everywhere away from the boundary — so a right-volume-but-wrong-shape build (correct moments, displaced
// material) still fails. notch = the half-space x+z>9.5 removed by the first cut; rod = r1 cylinder about the
// x-axis through (·,0,3).
func TestPartialRimDisjointCutMembershipMatchesCSG(t *testing.T) {
	res, err := Boolean(Cut, notchedTarget(t), disjointRod(t))
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	inTarget := func(p math.Point3) bool {
		return stdmath.Hypot(float64(p.X), float64(p.Y)) <= 3 && p.Z >= 0 && p.Z <= 10
	}
	aboveNotch := func(p math.Point3) bool { return float64(p.X)+float64(p.Z) > 9.5 }
	inRod := func(p math.Point3) bool { return stdmath.Hypot(float64(p.Y), float64(p.Z)-3) <= 1 }
	const shell = 0.15
	nearSurface := func(p math.Point3) bool {
		rWall := stdmath.Abs(stdmath.Hypot(float64(p.X), float64(p.Y)) - 3)
		rNotch := stdmath.Abs(float64(p.X)+float64(p.Z)-9.5) / stdmath.Sqrt2
		rRod := stdmath.Abs(stdmath.Hypot(float64(p.Y), float64(p.Z)-3) - 1)
		zCap := stdmath.Min(stdmath.Abs(float64(p.Z)), stdmath.Abs(float64(p.Z)-10))
		return rWall < shell || rNotch < shell || rRod < shell || zCap < shell
	}
	mesh, _ := TessellateBody(res, DefaultQuality())
	mismatches := 0
	const n = 60
	for i := range n {
		for j := range n {
			for k := range n {
				p := math.P3(math.Scalar(-3+6*(float64(i)+0.5)/n), math.Scalar(-3+6*(float64(j)+0.5)/n), math.Scalar(10*(float64(k)+0.5)/n))
				if nearSurface(p) {
					continue
				}
				if pointInMesh(mesh, p) != (inTarget(p) && !aboveNotch(p) && !inRod(p)) {
					mismatches++
				}
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("membership audit: %d interior points disagree with (target\\notch)\\rod (#1732 disjoint)", mismatches)
	}
}
