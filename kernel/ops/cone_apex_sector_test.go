// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// coneApexSectorFace builds an apex-collapsed cone SECTOR face: a cone (apex on axis, half angle)
// whose loop is a base arc of the given sweep at axial distance baseV plus two meridian rulings back
// to the apex — the shape of a partial revolve's side wall (OCCT blend/simple H6's two host cones).
// It mirrors the real fixture so both axis orientations can be tested for orientation-independence.
func coneApexSectorFace(t *testing.T, apex math.Point3, axis math.Vector3, half, baseV, sweep float64) *topo.Face {
	t.Helper()
	cone, err := geom.NewConeWithRef(apex, axis, math.V3(1, 0, 0), half)
	if err != nil {
		t.Fatal(err)
	}
	radius := baseV * stdmath.Tan(half)
	baseCenter := apex.TranslateBy(cone.AxisDir.AsVector().Scale(baseV))
	arc, err := geom.NewArc3d(baseCenter, cone.AxisDir.AsVector(), cone.Ref.AsVector(), radius, 0, sweep)
	if err != nil {
		t.Fatal(err)
	}
	p0, p1 := cone.PointAt(0, baseV), cone.PointAt(sweep, baseV)
	lin := topo.NewLineage(topo.Tok("test", "conesector", 0))
	bld := topo.NewBuilder(false, lin)
	vApex := bld.AddVertex(apex, lin)
	v0, v1 := bld.AddVertex(p0, lin), bld.AddVertex(p1, lin)
	eMer0 := bld.AddEdge(geom.NewLineSegment(apex, p0), vApex, v0, lin)
	eArc := bld.AddEdge(arc, v0, v1, lin)
	eMer1 := bld.AddEdge(geom.NewLineSegment(p1, apex), v1, vApex, lin)
	bld.AddFace(cone, lin, topo.OuterLoop(topo.Fwd(eMer0), topo.Fwd(eArc), topo.Fwd(eMer1)))
	return bld.Build().Faces()[0]
}

// TestConeApexSectorAreaOrientationIndependent is the ROOT-1 regression: two GEOMETRICALLY CONGRUENT
// 270° cone sectors in MIRROR orientations (apex up/axis down vs apex down/axis up — H6's two host
// cones, both 133286 in OCCT) must tessellate to the SAME area, equal to the analytic sector area.
// Before the coneApexSectorMesh fix the apex-collapsed sector routed through the (u,v)/seam path,
// which read one orientation's loop as seam-crossing and over-covered it as a full 2π cone (×1.26,
// 167927) while the sibling meshed correctly — an orientation-dependent tessellation defect.
func TestConeApexSectorAreaOrientationIndependent(t *testing.T) {
	t.Parallel()
	const half, baseV, sweep = stdmath.Pi / 4, 200.0, 3 * stdmath.Pi / 2
	radius := baseV * stdmath.Tan(half)
	slant := stdmath.Sqrt(baseV*baseV + radius*radius)
	analytic := sweep / 2 * radius * slant // (θ/2)·R·slant, the exact cone-sector lateral area
	fUp := coneApexSectorFace(t, math.P3(0, 0, 250), math.V3(0, 0, -1), half, baseV, sweep)
	fDown := coneApexSectorFace(t, math.P3(0, 0, -250), math.V3(0, 0, 1), half, baseV, sweep)
	areaUp := validate.MeshArea(tessellate.TessellateFace(fUp, PropertyQuality()))
	areaDown := validate.MeshArea(tessellate.TessellateFace(fDown, PropertyQuality()))
	if rel := stdmath.Abs(areaUp-areaDown) / analytic; rel > 1e-6 {
		t.Fatalf("cone-sector area is orientation-dependent: up=%.4f down=%.4f (rel %.2g)", areaUp, areaDown, rel)
	}
	for _, tc := range []struct {
		name string
		area float64
	}{{"apex-up", areaUp}, {"apex-down", areaDown}} {
		if rel := stdmath.Abs(tc.area-analytic) / analytic; rel > 1e-3 {
			t.Errorf("%s cone-sector area %.4f != analytic %.4f (rel %.2g, want the OCCT 133286 oracle)",
				tc.name, tc.area, analytic, rel)
		}
	}
}
