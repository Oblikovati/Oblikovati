// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Corner-junction partial-rim certification (EPIC Oblikovati/Oblikovati#1738, ADR-0048). The SECOND cut's rod
// imprint CROSSES the first-cut notch boundary: the target cylinder, the rod surface and the notch plane meet
// at two exact triple points, shared by the target wall, the rod tunnel and the bitten notch cap. Certified
// against an INDEPENDENT two-cut moment set from OpenCASCADE — occ.cut(occ.cut(cyl, halfspace), rod) with the
// rod raised to z=7 so its front loop crosses the notch (experiments/occ-boolean-oracle/
// partial_rim_corner_junction_oracle.py) — PLUS a 60³ point-membership audit vs the analytic predicate
// ((target \ notch) \ rod), the strong right-volume-wrong-shape oracle.

// OCC ground truth for the corner-junction fixture (two chained occ.cut; exact BRepGProp moments).
const (
	occCornerVol   = 239.9263179
	occCornerArea  = 255.0365956
	occCornerCx    = -0.1529414
	occCornerCy    = 0.0
	occCornerCz    = 4.4347653
	occCornerFaces = 5 // holed wall + bottom cap + partial top cap + bitten notch cap + rod tunnel
)

// cornerRod is the second-cut tool: a rod r=1 axis +x at z=7, drilled so its front loop CROSSES the notch
// (the notch floor is z=6.5 at the front rim, so the front bite straddles it). Kept in lockstep with
// partial_rim_corner_junction_oracle.py.
func cornerRod(t *testing.T) *topo.Body {
	t.Helper()
	rod, err := brep.SolidCylinder(math.P3(-6, 0, 7), math.V3(1, 0, 0), 1, 12)
	if err != nil {
		t.Fatalf("corner rod SolidCylinder: %v", err)
	}
	return rod
}

func TestPartialRimCornerCutMomentsMatchOCC(t *testing.T) {
	res, err := Boolean(Cut, notchedTarget(t), cornerRod(t))
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if n := len(res.Faces()); n != occCornerFaces {
		t.Errorf("corner-junction cut has %d faces; want %d (holed wall + 3 caps + tunnel)", n, occCornerFaces)
	}
	gp := BodyGeometryProperties(res, capCertQuality())
	if rel := stdmath.Abs(gp.Volume-occCornerVol) / occCornerVol; rel > 0.006 {
		t.Errorf("volume %.4f vs OCC %.4f (rel %.4f > 0.006) — beyond the SSI-imprint facet deficit", gp.Volume, occCornerVol, rel)
	}
	if rel := stdmath.Abs(gp.Area-occCornerArea) / occCornerArea; rel > 0.004 {
		t.Errorf("area %.4f vs OCC %.4f (rel %.4f > 0.004)", gp.Area, occCornerArea, rel)
	}
	occCentroid := math.P3(occCornerCx, occCornerCy, occCornerCz)
	if d := float64(gp.Centroid.DistanceTo(occCentroid)); d > 0.05 {
		t.Errorf("centroid (%.4f,%.4f,%.4f) vs OCC (%.4f,%.4f,%.4f): distance %.4f > 0.05",
			gp.Centroid.X, gp.Centroid.Y, gp.Centroid.Z, occCornerCx, occCornerCy, occCornerCz, d)
	}
}

// cornerMeshFreeEdges is the corner-certification watertightness metric, delegating to the production
// FreeEdgeCount so it welds at the MODEL's own resolution rather than a fixed 1e-6 grid — a fixed grid
// over-merges any model finer than itself and reports the over-merge as a free edge.
func cornerMeshFreeEdges(m *Mesh) int {
	return FreeEdgeCount(m)
}

// TestPartialRimCornerCutTessellationIsWatertight is the top-priority tessellation gate (CLAUDE.md: the user
// only ever sees the mesh). The corner-junction's coupled wall/tunnel/bitten-cap tessellation must weld into a
// crack-free surface — every triangle edge shared by exactly two triangles — so the rendered frame shows no
// tear at the triple points. This is the reproducible headless equivalent of the live render check.
func TestPartialRimCornerCutTessellationIsWatertight(t *testing.T) {
	res, err := Boolean(Cut, notchedTarget(t), cornerRod(t))
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	for _, gq := range gateQualities() {
		mesh, _ := TessellateBody(res, gq.q)
		if free := cornerMeshFreeEdges(mesh); free != 0 {
			t.Errorf("%s quality: corner-junction tessellation has %d free edges (want 0) — a visible crack in the "+
				"rendered mesh", gq.name, free)
		}
	}
}

// TestPartialRimCornerCutMembershipMatchesCSG is the strong shape oracle: the built solid's inside/outside
// must agree with ((target \ notch) \ rod) everywhere away from the boundary — so a right-volume-but-wrong-
// shape build (correct moments, displaced material) still fails. The rod is at z=7 (crossing the notch).
func TestPartialRimCornerCutMembershipMatchesCSG(t *testing.T) {
	res, err := Boolean(Cut, notchedTarget(t), cornerRod(t))
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	inTarget := func(p math.Point3) bool {
		return stdmath.Hypot(float64(p.X), float64(p.Y)) <= 3 && p.Z >= 0 && p.Z <= 10
	}
	aboveNotch := func(p math.Point3) bool { return float64(p.X)+float64(p.Z) > 9.5 }
	inRod := func(p math.Point3) bool { return stdmath.Hypot(float64(p.Y), float64(p.Z)-7) <= 1 }
	const shell = 0.15
	nearSurface := func(p math.Point3) bool {
		rWall := stdmath.Abs(stdmath.Hypot(float64(p.X), float64(p.Y)) - 3)
		rNotch := stdmath.Abs(float64(p.X)+float64(p.Z)-9.5) / stdmath.Sqrt2
		rRod := stdmath.Abs(stdmath.Hypot(float64(p.Y), float64(p.Z)-7) - 1)
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
		t.Errorf("membership audit: %d interior points disagree with (target\\notch)\\rod (#1738 corner-junction)", mismatches)
	}
}
