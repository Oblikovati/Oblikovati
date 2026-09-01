// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone half-space cut (M2 Phase 1, Oblikovati/Oblikovati#1334). A plane perpendicular to the axis trims a
// cone to an exact cone or frustum (geom.Cone preserved). The oracle is the analytic frustum volume
// (πh/3)(R²+Rr+r²); the tessellated cone runs ~0.6% under from chord inscription, so 2% tolerance.

// frustumVolume is the analytic volume of a frustum of height h with end radii r0 and r1.
func frustumVolume(h, r0, r1 float64) float64 {
	return stdmath.Pi * h / 3 * (r0*r0 + r0*r1 + r1*r1)
}

// coneAndPlaneFaces checks the exact path kept analytic surfaces (a cone side + planar caps), not soup.
func coneAndPlaneFaces(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone, geom.Plane:
		default:
			t.Errorf("result face surface %T is not analytic (cone cut must keep exact surfaces)", f.Geometry())
		}
	}
}

// TestHalfSpaceCutConeBaseSideIsFrustum keeps the base side of a cone (apex at z=10, base radius 3 at
// z=0) below z=5: an exact frustum of end radii 3 and 1.5.
func TestHalfSpaceCutConeBaseSideIsFrustum(t *testing.T) {
	t.Parallel()
	cone, _ := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 0, "cone")
	plane, _ := geom.NewPlane(math.P3(0, 0, 5), math.V3(0, 0, 1)) // keep z ≤ 5 (the wide base side)
	res, err := brep.HalfSpaceCut(cone, plane)
	if err != nil {
		t.Fatalf("cone ⟂ cut: %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("frustum is not a valid closed manifold solid: %+v", v)
	}
	coneAndPlaneFaces(t, res)
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := frustumVolume(5, 3, 1.5)
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("frustum volume %.4f, want %.4f (analytic) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestHalfSpaceCutConeApexSideIsCone keeps the apex side above z=5: a smaller exact cone (radius 1.5).
func TestHalfSpaceCutConeApexSideIsCone(t *testing.T) {
	t.Parallel()
	cone, _ := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 0, "cone")
	plane, _ := geom.NewPlane(math.P3(0, 0, 5), math.V3(0, 0, -1)) // keep z ≥ 5 (the apex tip)
	res, err := brep.HalfSpaceCut(cone, plane)
	if err != nil {
		t.Fatalf("cone ⟂ cut: %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("tip cone is not a valid closed manifold solid: %+v", v)
	}
	coneAndPlaneFaces(t, res)
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := frustumVolume(5, 1.5, 0) // a cone is a frustum with one radius zero
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("tip cone volume %.4f, want %.4f (analytic) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestHalfSpaceCutConeClearsKeepsWhole: a perpendicular plane beyond the base keeps the cone whole.
func TestHalfSpaceCutConeClearsKeepsWhole(t *testing.T) {
	t.Parallel()
	cone, _ := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 0, "cone")
	full := query.BodyGeometryProperties(cone, ops.DefaultQuality()).Volume
	plane, _ := geom.NewPlane(math.P3(0, 0, 12), math.V3(0, 0, 1)) // z ≤ 12 ⊇ the whole z∈[0,10] cone
	res, err := brep.HalfSpaceCut(cone, plane)
	if err != nil {
		t.Fatalf("cone ⟂ cut: %v", err)
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	if stdmath.Abs(got-full) > 1e-6 {
		t.Errorf("cleared cone volume %.6f, want %.6f (whole)", got, full)
	}
}
