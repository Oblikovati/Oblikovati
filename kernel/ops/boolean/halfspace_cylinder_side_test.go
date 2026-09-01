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

// Cylinder ∩ box acceptance (M2 Phase 1, Oblikovati/Oblikovati#1334). A box wall parallel to the
// cylinder axis cuts a chord off the circular cross-section: the kept band is an exact arc-band cylinder
// face capped by the box wall (a planar lid) and the trimmed circular caps. These tests drive that
// through HalfSpaceCut — one wall (symmetry + analytic segment) and two opposite walls (a slab, which
// re-cuts the seam-free arc band through the general looped split).

const cylR, cylH = 3.0, 10.0

// segmentArea returns the area of the smaller circular segment cut off a disk of radius r by a chord at
// distance m from the centre (m ≤ r): r²·acos(m/r) − m·√(r²−m²).
func segmentArea(r, m float64) float64 {
	return r*r*stdmath.Acos(m/r) - m*stdmath.Sqrt(r*r-m*m)
}

// TestHalfSpaceCutCylinderHalvesBySymmetry cuts an axis-aligned cylinder by a plane of symmetry through
// the axis (x ≤ 0): the kept band must be a valid analytic solid of exactly half the cylinder's volume.
func TestHalfSpaceCutCylinderHalvesBySymmetry(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), cylR, cylH)
	full := query.BodyGeometryProperties(cyl, ops.DefaultQuality()).Volume

	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(1, 0, 0)) // keep x ≤ 0
	half, err := brep.HalfSpaceCut(cyl, plane)
	if err != nil {
		t.Fatalf("axis-parallel cut: %v", err)
	}
	if r := ops.Validate(half); !r.Valid || !r.Closed || !r.Manifold || !half.IsSolid() {
		t.Fatalf("half cylinder is not a valid closed manifold solid: %+v", r)
	}
	assertOnlyCylinderAndPlaneFaces(t, half)
	got := query.BodyGeometryProperties(half, ops.DefaultQuality()).Volume
	if rel := stdmath.Abs(got-full/2) / (full / 2); rel > 0.02 {
		t.Errorf("symmetric half volume %.4f, want %.4f (full/2) — rel %.4f > 2%%", got, full/2, rel)
	}
}

// TestHalfSpaceCutCylinderSegmentVsAnalytic clips the cylinder by an off-centre axis-parallel plane
// (keep x ≤ 1.5): the kept cross-section is the disk minus the minor segment beyond the chord, so the
// volume is that area times the height — an analytic oracle the exact arc-band face must match.
func TestHalfSpaceCutCylinderSegmentVsAnalytic(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), cylR, cylH)
	plane, _ := geom.NewPlane(math.P3(1.5, 0, 0), math.V3(1, 0, 0)) // keep x ≤ 1.5
	clipped, err := brep.HalfSpaceCut(cyl, plane)
	if err != nil {
		t.Fatalf("axis-parallel cut: %v", err)
	}
	if r := ops.Validate(clipped); !r.Valid || !r.Closed || !r.Manifold || !clipped.IsSolid() {
		t.Fatalf("clipped cylinder is not a valid closed manifold solid: %+v", r)
	}
	assertOnlyCylinderAndPlaneFaces(t, clipped)
	got := query.BodyGeometryProperties(clipped, ops.DefaultQuality()).Volume
	want := (stdmath.Pi*cylR*cylR - segmentArea(cylR, 1.5)) * cylH // major part of the disk × height
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("clipped cylinder volume %.4f, want %.4f (analytic) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestHalfSpaceCutCylinderSlab keeps the |x| ≤ 1.5 slab by composing two opposite axis-parallel walls.
// The second wall re-cuts the seam-free arc band the first produced, exercising the general looped split
// bridging across an axis-parallel line pair. The kept area is the disk minus two minor segments.
func TestHalfSpaceCutCylinderSlab(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), cylR, cylH)
	pHi, _ := geom.NewPlane(math.P3(1.5, 0, 0), math.V3(1, 0, 0))   // keep x ≤ 1.5
	pLo, _ := geom.NewPlane(math.P3(-1.5, 0, 0), math.V3(-1, 0, 0)) // keep x ≥ −1.5

	band, err := brep.HalfSpaceCut(cyl, pHi)
	if err != nil {
		t.Fatalf("first wall: %v", err)
	}
	slab, err := brep.HalfSpaceCut(band, pLo)
	if err != nil {
		t.Fatalf("second wall: %v", err)
	}
	if r := ops.Validate(slab); !r.Valid || !r.Closed || !r.Manifold || !slab.IsSolid() {
		t.Fatalf("slab is not a valid closed manifold solid: %+v", r)
	}
	assertOnlyCylinderAndPlaneFaces(t, slab)
	got := query.BodyGeometryProperties(slab, ops.DefaultQuality()).Volume
	want := (stdmath.Pi*cylR*cylR - 2*segmentArea(cylR, 1.5)) * cylH
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("slab volume %.4f, want %.4f (analytic) — rel %.4f > 2%%", got, want, rel)
	}
}

// assertOnlyCylinderAndPlaneFaces checks the exact path kept analytic surfaces (cylinder arc bands +
// planar caps/lids), not tessellated soup.
func assertOnlyCylinderAndPlaneFaces(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder, geom.Plane:
		default:
			t.Errorf("result face surface %T is not analytic (curved boolean must keep exact surfaces)", f.Geometry())
		}
	}
}
