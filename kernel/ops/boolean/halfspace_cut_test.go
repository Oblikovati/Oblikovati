// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

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

// Acceptance for the curved half-space cut (M2 Phase 1, Oblikovati/Oblikovati#1334): cutting an
// analytic curved solid by ONE plane yields an EXACT curved B-rep — the kept faces preserve their
// analytic surface (a geom.Sphere, not tessellated soup) and the body's volume matches the analytic
// formula. brep.HalfSpaceCut keeps the part on the plane's negative side, capped by a planar lid.

// capVolume is the analytic spherical-cap volume π·h²(3R−h)/3, the oracle for the kept −n cap.
func capVolume(radius, height float64) float64 {
	return stdmath.Pi * height * height * (3*radius - height) / 3
}

// TestHalfSpaceCutSphereIsExactCap cuts a sphere by several planes and checks the result is a valid
// closed solid whose two faces are an exact sphere + plane, with the analytic spherical-cap volume.
func TestHalfSpaceCutSphereIsExactCap(t *testing.T) {
	t.Parallel()
	const R = 5.0
	cases := []struct {
		name        string
		planePoint  math.Point3
		planeNormal math.Vector3
		height      float64 // kept (−normal) cap height h = R + d, d = (planePoint − centre)·normal
	}{
		{"hemisphere", math.P3(0, 0, 0), math.V3(0, 0, 1), R},
		{"shallow cap", math.P3(0, 0, 3), math.V3(0, 0, 1), R + 3},     // keep −z side, h = 8
		{"deep keep", math.P3(0, 0, -3), math.V3(0, 0, 1), R - 3},      // keep −z side, h = 2
		{"oblique", math.P3(0, 0, 0), math.V3(1, 1, 1), R},             // hemisphere, tilted cut
		{"oblique offset", math.P3(-2, 0, 0), math.V3(1, 0, 0), R - 2}, // keep −x side, h = 3
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sphere, err := brep.SolidSphere(math.P3(0, 0, 0), R, "s")
			if err != nil {
				t.Fatalf("SolidSphere: %v", err)
			}
			plane, err := geom.NewPlane(tc.planePoint, tc.planeNormal)
			if err != nil {
				t.Fatalf("NewPlane: %v", err)
			}
			cap, err := brep.HalfSpaceCut(sphere, plane)
			if err != nil {
				t.Fatalf("HalfSpaceCut: %v", err)
			}
			if r := ops.Validate(cap); !r.Valid || !r.Closed || !r.Manifold || !cap.IsSolid() {
				t.Fatalf("cap not a valid closed manifold solid: %+v", r)
			}
			assertSphereAndPlaneFaces(t, cap)
			got := query.BodyGeometryProperties(cap, ops.DefaultQuality()).Volume
			want := capVolume(R, tc.height)
			if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
				t.Errorf("cap volume %.4f, want %.4f (rel err %.4f > 2%%)", got, want, rel)
			}
		})
	}
}

// assertSphereAndPlaneFaces verifies the result preserves the EXACT analytic surfaces: one
// geom.Sphere face (the cap) and one geom.Plane face (the lid) — not a tessellated approximation.
func assertSphereAndPlaneFaces(t *testing.T, body *topo.Body) {
	t.Helper()
	spheres, planes := 0, 0
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Sphere:
			spheres++
		case geom.Plane:
			planes++
		default:
			t.Errorf("unexpected result face surface %T (curved boolean must keep exact surfaces)", f.Geometry())
		}
	}
	if spheres != 1 || planes != 1 {
		t.Errorf("cap has %d sphere + %d plane faces, want 1 + 1", spheres, planes)
	}
}

// TestHalfSpaceCutCylinderPerpendicularIsFrustum cuts a cylinder by a plane ⟂ its axis: the kept
// band is a shorter cylinder (exact side + caps). The oracle is the tessellated volume of an
// independently-built SolidCylinder of the kept height — identical faceting, so they must agree
// exactly (comparing to the analytic π·r²·h' would only confirm the cylinder's own ~0.6% chord
// inscription, not the cut).
func TestHalfSpaceCutCylinderPerpendicularIsFrustum(t *testing.T) {
	t.Parallel()
	const r, h = 3.0, 4.0
	facetedVol := func(height float64) float64 {
		c, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, height)
		return query.BodyGeometryProperties(c, ops.DefaultQuality()).Volume
	}
	cases := []struct {
		name        string
		planePoint  math.Point3
		planeNormal math.Vector3
		keptHeight  float64 // 0 ⇒ empty, h ⇒ whole
	}{
		{"keep lower band", math.P3(0, 0, 2.5), math.V3(0, 0, 1), 2.5},
		{"keep upper band", math.P3(0, 0, 1.5), math.V3(0, 0, -1), 2.5},
		{"plane clears (whole)", math.P3(0, 0, 10), math.V3(0, 0, 1), h},
		{"plane clears (empty)", math.P3(0, 0, -1), math.V3(0, 0, 1), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
			if err != nil {
				t.Fatalf("SolidCylinder: %v", err)
			}
			plane, _ := geom.NewPlane(tc.planePoint, tc.planeNormal)
			res, err := brep.HalfSpaceCut(cyl, plane)
			if err != nil {
				t.Fatalf("HalfSpaceCut: %v", err)
			}
			if tc.keptHeight == 0 {
				if res.IsSolid() && len(res.Faces()) > 0 {
					t.Fatalf("expected empty result, got %d faces", len(res.Faces()))
				}
				return
			}
			if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
				t.Fatalf("frustum not a valid closed manifold solid: %+v", r)
			}
			assertCylinderAndPlaneFaces(t, res)
			got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
			want := facetedVol(tc.keptHeight)
			if rel := stdmath.Abs(got-want) / want; rel > 1e-6 {
				t.Errorf("kept volume %.6f, want %.6f (rel %.2e)", got, want, rel)
			}
		})
	}
}

// TestHalfSpaceCutCylinderObliqueExact: an oblique cut carves an ELLIPTICAL section on the cylinder side,
// the case the old line-only split deferred to CSG. The unified (u,v) ruled-side split now builds it
// exactly — a cylinder band + an elliptical planar lid + the kept bottom cap — watertight. The plane
// z=(4−x)/2 stays above the base over the whole disk, so the kept volume is the exact analytic
// ∬(4−x)/2 dA = 18π (mean height 2 × π·3²) up to the tessellation chord deficit.
func TestHalfSpaceCutCylinderObliqueExact(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 4)
	plane, _ := geom.NewPlane(math.P3(0, 0, 2), math.V3(1, 0, 2)) // tilted off the axis → within-band ellipse
	res, err := brep.HalfSpaceCut(cyl, plane)
	if err != nil {
		t.Fatalf("oblique cylinder cut should now be exact, got %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
		t.Fatalf("oblique cut is not a watertight solid: %+v", r)
	}
	if !bodyHasCylinderFace(res) || !bodyHasEllipticalEdge(res) {
		t.Error("result lacks the analytic cylinder face or the elliptical section edge — it fell back to CSG")
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := 18 * stdmath.Pi
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("kept volume %.4f, want analytic %.4f (rel %.4f > 0.02)", got, want, rel)
	}
}

// bodyHasCylinderFace reports whether the body carries an analytic cylinder face.
func bodyHasCylinderFace(b *topo.Body) bool {
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return true
		}
	}
	return false
}

// bodyHasEllipticalEdge reports whether any edge is an ellipse/elliptical arc — the oblique cut's section.
func bodyHasEllipticalEdge(b *topo.Body) bool {
	for _, f := range b.Faces() {
		for _, l := range f.Loops() {
			for _, u := range l.EdgeUses() {
				switch u.Edge().Geometry().(type) {
				case geom.EllipseFull, geom.EllipticalArc:
					return true
				}
			}
		}
	}
	return false
}

// assertCylinderAndPlaneFaces verifies an exact truncated cylinder: one geom.Cylinder side + two
// geom.Plane caps.
func assertCylinderAndPlaneFaces(t *testing.T, body *topo.Body) {
	t.Helper()
	cyls, planes := 0, 0
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			cyls++
		case geom.Plane:
			planes++
		default:
			t.Errorf("unexpected result face surface %T", f.Geometry())
		}
	}
	if cyls != 1 || planes != 2 {
		t.Errorf("frustum has %d cylinder + %d plane faces, want 1 + 2", cyls, planes)
	}
}

// TestHalfSpaceCutSpherePlaneMissKeepsOrEmpties: a plane clear of the sphere keeps the whole sphere
// (centre on the negative side) or empties it (centre on the positive side).
func TestHalfSpaceCutSpherePlaneMissKeepsOrEmpties(t *testing.T) {
	t.Parallel()
	sphere, _ := brep.SolidSphere(math.P3(0, 0, 0), 5, "s")
	full := query.BodyGeometryProperties(sphere, ops.DefaultQuality()).Volume

	below, _ := geom.NewPlane(math.P3(0, 0, 8), math.V3(0, 0, 1)) // sphere entirely on −z (kept) side
	kept, err := brep.HalfSpaceCut(sphere, below)
	if err != nil {
		t.Fatalf("HalfSpaceCut (miss, keep): %v", err)
	}
	if got := query.BodyGeometryProperties(kept, ops.DefaultQuality()).Volume; stdmath.Abs(got-full) > 1e-6 {
		t.Errorf("plane clear above sphere should keep it whole: volume %.4f, want %.4f", got, full)
	}

	above, _ := geom.NewPlane(math.P3(0, 0, -8), math.V3(0, 0, 1)) // sphere entirely on +z (dropped) side
	empty, err := brep.HalfSpaceCut(sphere, above)
	if err != nil {
		t.Fatalf("HalfSpaceCut (miss, empty): %v", err)
	}
	if empty.IsSolid() && len(empty.Faces()) > 0 {
		t.Errorf("plane clear below sphere should empty it, got %d faces", len(empty.Faces()))
	}
}
