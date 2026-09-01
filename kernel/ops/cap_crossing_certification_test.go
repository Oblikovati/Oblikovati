// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cap-crossing slice-1 certification (EPIC Oblikovati/Oblikovati#1724, ADR-0046). The interior-exit
// cap-crossing cut (brep.CapCrossingCutGeneral) is the narrowest sub-family of the deferred cap-crossing
// gate: an oblique cylinder tool that enters the target wall once and EXITS one planar cap through an
// ellipse strictly inside the rim. A partial curved-boolean implementation ships manifold-but-wrong-shape
// solids, so this slice is certified NOT by internal manifold/volume nets alone but against an INDEPENDENT
// moment set from OpenCASCADE (BRepAlgoAPI_Cut + BRepGProp, experiments/occ-boolean-oracle): the 0th moment
// (volume), the surface area, the 1st moment (centroid), the face count, AND a point-membership audit vs
// the analytic CSG predicate — the strong oracle that catches a right-volume-wrong-shape build.

// OCCT ground truth for the slice-1 fixture (experiments/occ-boolean-oracle/cap_crossing_oracle.cpp,
// BRepGProp exact analytic moments). The tessellated B-rep sits a hair UNDER these by the SSI-imprint
// polyline deficit — the same ~0.4% every curved boolean carries (identical to the shipped crossing-cut),
// not a shape error: the analytic construction matches to 0.01% (see the membership audit below).
const (
	occCapVol   = 266.6720995
	occCapArea  = 273.2430771
	occCapCx    = 0.0414463
	occCapCy    = 0.0
	occCapCz    = 4.8359787
	occCapFaces = 4 // wall (holed) + top cap (elliptical hole) + bottom cap + tunnel
)

// capCrossFixture builds the certified slice-1 pair: r=3 h=10 target, oblique r=0.9 tool at 45° that exits
// the top cap. Kept in lockstep with experiments/occ-boolean-oracle and capCrossingCutBodies.
func capCrossFixture(t *testing.T) (target, tool *topo.Body) {
	t.Helper()
	s := 1 / stdmath.Sqrt2
	tg, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target SolidCylinder: %v", err)
	}
	tl, err := brep.SolidCylinder(math.P3(-6.5, 0, 2), math.V3(math.Scalar(s), 0, math.Scalar(s)), 0.9, 16)
	if err != nil {
		t.Fatalf("tool SolidCylinder: %v", err)
	}
	return tg, tl
}

// capCertQuality is a fine-enough tessellation that the residual error vs OCC is the SSI-imprint polyline
// deficit (~0.4%), not coarse chording — the regime the moment tolerances below are calibrated for.
func capCertQuality() Quality {
	return Quality{ChordTolerance: 0.005, AngleTolerance: 2 * stdmath.Pi / 180}
}

func TestCapCrossingCutIsWatertightAndFoldFree(t *testing.T) {
	t.Parallel()
	target, tool := capCrossFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	for _, gq := range certGateQualities() {
		mesh, _ := TessellateBody(res, gq.q)
		if free := freeEdgeCount(mesh); free != 0 {
			t.Errorf("%s quality: cap-crossing cut tessellated with %d free edges; want 0 — a cross-face T-junction crack "+
				"(regression of the by-value ellipse imprint fix, #1724)", gq.name, free)
		}
		if folds := FoldEdgeCount(mesh); folds != 0 {
			t.Errorf("%s quality: cap-crossing cut mesh has %d fold edges; want 0 (no self-overlap)", gq.name, folds)
		}
	}
}

func TestCapCrossingCutMomentsMatchOCC(t *testing.T) {
	t.Parallel()
	target, tool := capCrossFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if n := len(res.Faces()); n != occCapFaces {
		t.Errorf("cap-crossing cut has %d faces; want %d (matching OCC topology: holed wall + holed top cap "+
			"+ bottom cap + tunnel)", n, occCapFaces)
	}
	gp := BodyGeometryProperties(res, capCertQuality())
	if rel := stdmath.Abs(gp.Volume-occCapVol) / occCapVol; rel > 0.006 {
		t.Errorf("volume %.4f vs OCC %.4f (rel %.4f > 0.006) — beyond the SSI-imprint facet deficit", gp.Volume, occCapVol, rel)
	}
	if rel := stdmath.Abs(gp.Area-occCapArea) / occCapArea; rel > 0.004 {
		t.Errorf("area %.4f vs OCC %.4f (rel %.4f > 0.004)", gp.Area, occCapArea, rel)
	}
	occCentroid := math.P3(occCapCx, occCapCy, occCapCz)
	if d := float64(gp.Centroid.DistanceTo(occCentroid)); d > 0.05 {
		t.Errorf("centroid (%.4f,%.4f,%.4f) vs OCC (%.4f,%.4f,%.4f): distance %.4f > 0.05",
			gp.Centroid.X, gp.Centroid.Y, gp.Centroid.Z, occCapCx, occCapCy, occCapCz, d)
	}
}

// TestCapCrossingCutMembershipMatchesCSG is the strong shape oracle: it samples an interior grid and asserts
// the built solid's inside/outside agrees with the analytic CSG predicate target\tool everywhere away from
// the boundary — so a right-volume-but-wrong-shape build (correct moments, displaced material) still fails.
// Points within a shell of the boundary are skipped: the inscribed facets there disagree by design, and that
// noise is what the moment tolerances already bound.
func TestCapCrossingCutMembershipMatchesCSG(t *testing.T) {
	t.Parallel()
	target, tool := capCrossFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	inTarget := func(p math.Point3) bool {
		return stdmath.Hypot(float64(p.X), float64(p.Y)) <= 3 && p.Z >= 0 && p.Z <= 10
	}
	tb, td := math.P3(-6.5, 0, 2), math.V3(math.Scalar(1/stdmath.Sqrt2), 0, math.Scalar(1/stdmath.Sqrt2))
	inTool := func(p math.Point3) bool {
		w := tb.VectorTo(p)
		ax := float64(w.Dot(td))
		if ax < 0 || ax > 16 {
			return false
		}
		return float64(w.Sub(td.Scale(math.Scalar(ax))).Length()) <= 0.9
	}
	const shell = 0.15 // skip points within this distance of either surface (facet noise)
	nearSurface := func(p math.Point3) bool {
		rWall := stdmath.Abs(stdmath.Hypot(float64(p.X), float64(p.Y)) - 3)
		w := tb.VectorTo(p)
		ax := float64(w.Dot(td))
		rTool := stdmath.Abs(float64(w.Sub(td.Scale(math.Scalar(ax))).Length()) - 0.9)
		zCap := stdmath.Min(stdmath.Abs(float64(p.Z)), stdmath.Abs(float64(p.Z)-10))
		return rWall < shell || rTool < shell || zCap < shell
	}
	mesh, _ := TessellateBody(res, DefaultQuality()) // tessellate ONCE; the O(n³) grid would re-mesh per point via PointInsideBody
	mismatches := 0
	const n = 60
	for i := range n {
		for j := range n {
			for k := range n {
				p := math.P3(math.Scalar(-3+6*(float64(i)+0.5)/n), math.Scalar(-3+6*(float64(j)+0.5)/n), math.Scalar(10*(float64(k)+0.5)/n))
				if nearSurface(p) {
					continue
				}
				if pointInMesh(mesh, p) != (inTarget(p) && !inTool(p)) {
					mismatches++
				}
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("membership audit: %d interior points disagree with the analytic CSG predicate target\\tool "+
			"— a right-volume-wrong-shape build (#1724 slice 1)", mismatches)
	}
}
