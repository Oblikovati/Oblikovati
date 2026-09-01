// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone-tool cap-crossing certification (EPIC Oblikovati/Oblikovati#1724, ADR-0046). Slice 1 certified an
// oblique CYLINDER tool that enters the target wall once and exits ONE cap through an ellipse strictly inside
// the rim. A conical tool (a countersink / tapered pin) makes the SAME arrangement; only the analytics differ
// — the wall entry is the cone∩cylinder imprint and the exit section is the cone∩plane ellipse — so the whole
// operand-generic builder (buildCapCrossCut) is reused. Certified NOT by internal manifold/volume nets alone
// but against an INDEPENDENT moment set from OpenCASCADE (gmsh occ.cut + occ.getMass exact B-rep moments,
// experiments/occ-boolean-oracle/cone_cap_crossing_oracle.py): volume, area, centroid, face count, PLUS a
// point-membership audit vs the analytic CSG predicate — the strong oracle that catches a right-volume-
// wrong-shape build.

// OCC ground truth for the cone-tool fixture, from experiments/occ-boolean-oracle/cone_cap_crossing_oracle.py
// (occ.cut of the target cylinder and the frustum tool; occ.getMass / occ.getCenterOfMass exact moments).
const (
	occConeCapVol   = 271.6205285
	occConeCapArea  = 269.5473548
	occConeCapCx    = 0.0340767
	occConeCapCy    = 0.0
	occConeCapCz    = 4.8934008
	occConeCapFaces = 4 // wall (holed) + top cap (elliptical hole) + bottom cap + cone tunnel
)

// coneCapFixture builds the certified cone-tool pair: r=3 h=10 target and an oblique frustum (rBase 0.9 →
// rTop 0.6) at 45° whose slender wall enters the wall once and exits the top cap. Kept in lockstep with
// experiments/occ-boolean-oracle/cone_cap_crossing_oracle.py and coneCapCrossingCutBodies.
func coneCapFixture(t *testing.T) (target, tool *topo.Body) {
	t.Helper()
	s := 1 / stdmath.Sqrt2
	tg, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target SolidCylinder: %v", err)
	}
	top := math.P3(math.Scalar(-6.5+16*s), 0, math.Scalar(2+16*s))
	tl, err := brep.SolidCylinderCone(math.P3(-6.5, 0, 2), top, 0.9, 0.6, "cone")
	if err != nil {
		t.Fatalf("tool SolidCylinderCone: %v", err)
	}
	return tg, tl
}

func TestConeCapCrossingCutIsWatertightAndValid(t *testing.T) {
	t.Parallel()
	target, tool := coneCapFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if r := Validate(res); !r.Valid {
		t.Errorf("cone-cap cut is not a valid solid: manifold=%v closed=%v orient=%v euler=%v issues=%v",
			r.Manifold, r.Closed, r.OrientationOK, r.EulerConsistent, r.Issues)
	}
	for _, gq := range certGateQualities() {
		mesh, _ := tessellate.TessellateBody(res, gq.q)
		if free := freeEdgeCount(mesh); free != 0 {
			t.Errorf("%s quality: cone-cap cut tessellated with %d free edges; want 0 — a cross-face crack at the cap ellipse", gq.name, free)
		}
		if folds := validate.FoldEdgeCount(mesh); folds != 0 {
			t.Errorf("%s quality: cone-cap cut mesh has %d fold edges; want 0", gq.name, folds)
		}
	}
}

func TestConeCapCrossingCutMomentsMatchOCC(t *testing.T) {
	t.Parallel()
	target, tool := coneCapFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if n := len(res.Faces()); n != occConeCapFaces {
		t.Errorf("cone-cap cut has %d faces; want %d (holed wall + holed top cap + bottom cap + tunnel)", n, occConeCapFaces)
	}
	gp := BodyGeometryProperties(res, capCertQuality())
	if rel := stdmath.Abs(gp.Volume-occConeCapVol) / occConeCapVol; rel > 0.006 {
		t.Errorf("volume %.4f vs OCC %.4f (rel %.4f > 0.006) — beyond the SSI-imprint facet deficit", gp.Volume, occConeCapVol, rel)
	}
	if rel := stdmath.Abs(gp.Area-occConeCapArea) / occConeCapArea; rel > 0.004 {
		t.Errorf("area %.4f vs OCC %.4f (rel %.4f > 0.004)", gp.Area, occConeCapArea, rel)
	}
	occCentroid := math.P3(occConeCapCx, occConeCapCy, occConeCapCz)
	if d := float64(gp.Centroid.DistanceTo(occCentroid)); d > 0.05 {
		t.Errorf("centroid (%.4f,%.4f,%.4f) vs OCC (%.4f,%.4f,%.4f): distance %.4f > 0.05",
			gp.Centroid.X, gp.Centroid.Y, gp.Centroid.Z, occConeCapCx, occConeCapCy, occConeCapCz, d)
	}
}

// TestConeCapCrossingCutMembershipMatchesCSG is the strong shape oracle: it samples an interior grid and
// asserts the built solid's inside/outside agrees with the analytic CSG predicate target\tool everywhere
// away from the boundary — so a right-volume-but-wrong-shape build (correct moments, displaced material)
// still fails. The tool is a FRUSTUM: its radius tapers linearly along the axis, so the membership predicate
// interpolates rBase→rTop by the point's fractional axial position.
func TestConeCapCrossingCutMembershipMatchesCSG(t *testing.T) {
	t.Parallel()
	target, tool := coneCapFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	s := 1 / stdmath.Sqrt2
	base := math.P3(-6.5, 0, 2)
	dir := math.V3(math.Scalar(s), 0, math.Scalar(s))
	const rBase, rTop, coneLen = 0.9, 0.6, 16.0
	inTarget := func(p math.Point3) bool {
		return stdmath.Hypot(float64(p.X), float64(p.Y)) <= 3 && p.Z >= 0 && p.Z <= 10
	}
	axialOf := func(p math.Point3) float64 { return float64(base.VectorTo(p).Dot(dir)) }
	radiusAt := func(ax float64) float64 { return rBase + (rTop-rBase)*ax/coneLen }
	radialOf := func(p math.Point3, ax float64) float64 {
		return float64(base.VectorTo(p).Sub(dir.Scale(math.Scalar(ax))).Length())
	}
	inTool := func(p math.Point3) bool {
		ax := axialOf(p)
		if ax < 0 || ax > coneLen {
			return false
		}
		return radialOf(p, ax) <= radiusAt(ax)
	}
	const shell = 0.15
	nearSurface := func(p math.Point3) bool {
		rWall := stdmath.Abs(stdmath.Hypot(float64(p.X), float64(p.Y)) - 3)
		ax := axialOf(p)
		rCone := stdmath.Abs(radialOf(p, ax) - radiusAt(ax))
		zCap := stdmath.Min(stdmath.Abs(float64(p.Z)), stdmath.Abs(float64(p.Z)-10))
		return rWall < shell || rCone < shell || zCap < shell
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
				if pointInMesh(mesh, p) != (inTarget(p) && !inTool(p)) {
					mismatches++
				}
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("membership audit: %d interior points disagree with target\\tool (#1724 cone-cap)", mismatches)
	}
}
