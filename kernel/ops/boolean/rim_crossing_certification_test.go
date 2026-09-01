// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cap-crossing slice-2 certification (EPIC Oblikovati/Oblikovati#1724, ADR-0046). The RIM-CROSSING cut
// (brep.RimCrossingCutGeneral) is the next cap-crossing sub-family after the interior-exit slice: an oblique
// cylinder tool whose exit ellipse CROSSES the cap rim at two corners, so the tool exits partly through the
// cap and partly through the wall — the wall gains a top-rim NOTCH plus its entry hole, the cap a mixed
// rim-arc + ellipse-arc bite, and the tunnel an [entry ⊕ ellipse-arc] mouth. As with slice 1 this is certified
// against an INDEPENDENT OpenCASCADE moment set (gmsh occ.cut + getMass), NOT internal manifold/volume nets
// alone, plus a point-membership audit vs the analytic CSG predicate — the strong right-volume-wrong-shape gate.

// OCC ground truth for the slice-2 fixture (base -5.6), from gmsh OCC occ.cut + getMass. The tessellated
// B-rep sits ~0.3% UNDER by the SSI-imprint polyline deficit — the same fraction every curved boolean carries,
// not a shape error (the analytic construction matches to <0.1%, per the membership audit below).
const (
	occRimVol   = 263.6617744
	occRimArea  = 279.0942805
	occRimCx    = 0.0207324
	occRimCy    = 0.0
	occRimCz    = 4.8372578
	occRimFaces = 4 // notched wall (holed) + top cap (mixed-arc bite) + bottom cap + tunnel
)

// rimCrossFixture builds the certified slice-2 pair: r=3 h=10 target, oblique r=0.9 tool at 45° whose exit
// ellipse CROSSES the top rim (base -5.6 vs slice 1's -6.5, which exits strictly inside the rim).
func rimCrossFixture(t *testing.T) (target, tool *topo.Body) {
	t.Helper()
	s := 1 / stdmath.Sqrt2
	tg, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target SolidCylinder: %v", err)
	}
	tl, err := brep.SolidCylinder(math.P3(-5.6, 0, 2), math.V3(math.Scalar(s), 0, math.Scalar(s)), 0.9, 16)
	if err != nil {
		t.Fatalf("tool SolidCylinder: %v", err)
	}
	return tg, tl
}

func TestRimCrossingCutIsWatertightAndFoldFree(t *testing.T) {
	t.Parallel()
	target, tool := rimCrossFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if r := Validate(res); !r.Valid {
		t.Errorf("rim-crossing cut is not a valid solid: manifold=%v closed=%v orient=%v euler=%v issues=%v",
			r.Manifold, r.Closed, r.OrientationOK, r.EulerConsistent, r.Issues)
	}
	for _, gq := range certGateQualities() {
		mesh, _ := tessellate.TessellateBody(res, gq.q)
		if free := freeEdgeCount(mesh); free != 0 {
			t.Errorf("%s quality: rim-crossing cut tessellated with %d free edges; want 0 — a cross-face crack (notched two-rim "+
				"wall mesher or the mixed-arc cap winding, #1724 slice 2)", gq.name, free)
		}
		if folds := validate.FoldEdgeCount(mesh); folds != 0 {
			t.Errorf("%s quality: rim-crossing cut mesh has %d fold edges; want 0 (no self-overlap)", gq.name, folds)
		}
	}
}

func TestRimCrossingCutMomentsMatchOCC(t *testing.T) {
	t.Parallel()
	target, tool := rimCrossFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if n := len(res.Faces()); n != occRimFaces {
		t.Errorf("rim-crossing cut has %d faces; want %d (notched holed wall + mixed-arc top cap + bottom cap "+
			"+ tunnel)", n, occRimFaces)
	}
	gp := query.BodyGeometryProperties(res, capCertQuality())
	if rel := stdmath.Abs(gp.Volume-occRimVol) / occRimVol; rel > 0.006 {
		t.Errorf("volume %.4f vs OCC %.4f (rel %.4f > 0.006) — beyond the SSI-imprint facet deficit", gp.Volume, occRimVol, rel)
	}
	if rel := stdmath.Abs(gp.Area-occRimArea) / occRimArea; rel > 0.004 {
		t.Errorf("area %.4f vs OCC %.4f (rel %.4f > 0.004)", gp.Area, occRimArea, rel)
	}
	occCentroid := math.P3(occRimCx, occRimCy, occRimCz)
	if d := float64(gp.Centroid.DistanceTo(occCentroid)); d > 0.05 {
		t.Errorf("centroid (%.4f,%.4f,%.4f) vs OCC (%.4f,%.4f,%.4f): distance %.4f > 0.05",
			gp.Centroid.X, gp.Centroid.Y, gp.Centroid.Z, occRimCx, occRimCy, occRimCz, d)
	}
}

// TestRimCrossingCutMembershipMatchesCSG samples an interior grid and asserts the built solid's inside/outside
// agrees with the analytic CSG predicate target\tool everywhere away from the boundary — so a right-volume-
// wrong-shape build (correct moments, displaced material at the notch or the tunnel) still fails.
func TestRimCrossingCutMembershipMatchesCSG(t *testing.T) {
	t.Parallel()
	target, tool := rimCrossFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	inTarget := func(p math.Point3) bool {
		return stdmath.Hypot(float64(p.X), float64(p.Y)) <= 3 && p.Z >= 0 && p.Z <= 10
	}
	tb, td := math.P3(-5.6, 0, 2), math.V3(math.Scalar(1/stdmath.Sqrt2), 0, math.Scalar(1/stdmath.Sqrt2))
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
	mesh, _ := tessellate.TessellateBody(res, DefaultQuality())
	mismatches := 0
	const n = 60
	for i := range n {
		for j := range n {
			for k := range n {
				p := math.P3(math.Scalar(-3+6*(float64(i)+0.5)/n), math.Scalar(-3+6*(float64(j)+0.5)/n), math.Scalar(10*(float64(k)+0.5)/n))
				if nearSurface(p) {
					continue
				}
				if query.PointInMesh(mesh, p) != (inTarget(p) && !inTool(p)) {
					mismatches++
				}
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("membership audit: %d interior points disagree with the analytic CSG predicate target\\tool "+
			"— a right-volume-wrong-shape build (#1724 slice 2)", mismatches)
	}
}
