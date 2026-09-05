// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The gate for ADR-0062: a half-space cut is a difference against the plane's positive side, bounded to
// the target's box — what OCCT's BRepPrimAPI_MakeHalfSpace hands to the ordinary BOP, and what
// solvespace's symmetric CopySurfacesTrimAgainst assumes. When every row below is `equal`, the whole
// parallel half-space pipeline is deleted and HalfSpaceCut becomes the prism plus Boolean.
//
// It is a RATCHET. A row moving from `differs` to `equal` is a stage of the retirement landing and the
// expectation comes with it; no row may move the other way.

// closedSolidFaces reports a body's face count and whether it is a closed solid; (0, false) for a body
// the path declined to build.
func closedSolidFaces(b *topo.Body, err error) (int, bool) {
	if err != nil || b == nil {
		return 0, false
	}
	for _, e := range b.Edges() {
		if len(e.Uses()) != 2 {
			return len(b.Faces()), false
		}
	}
	return len(b.Faces()), b.IsSolid()
}

func TestHalfSpaceCutEqualsABoundedDifference(t *testing.T) {
	t.Parallel()
	pl := func(t *testing.T, ox, oy, oz float64, d math.Vector3) geom.Plane {
		t.Helper()
		p, err := geom.NewPlane(math.P3(math.Scalar(ox), math.Scalar(oy), math.Scalar(oz)), d)
		if err != nil {
			t.Fatalf("plane: %v", err)
		}
		return p
	}
	cyl := func() *topo.Body { c, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10); return c }
	cone := func() *topo.Body {
		c, _ := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 4, 1, "cone")
		return c
	}
	apex := func() *topo.Body {
		c, _ := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 0, "apexcone")
		return c
	}
	sph := func() *topo.Body { s, _ := SolidSphere(math.P3(0, 0, 0), 5, "s"); return s }
	tor := func() *topo.Body { s, _ := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "t"); return s }
	for _, tc := range []struct {
		name  string
		body  func() *topo.Body
		plane func(*testing.T) geom.Plane
		// equal is the ratchet: true once the difference reproduces the cut. Flip a row to true in the
		// commit that makes it so, and say which stage of ADR-0061 did it.
		equal bool
		gate  string
	}{
		{"cylinder/axis-parallel off-centre", cyl, func(t *testing.T) geom.Plane { return pl(t, 1.5, 0, 0, math.V3(1, 0, 0)) }, true, ""},
		{"cylinder/oblique", cyl, func(t *testing.T) geom.Plane { return pl(t, 0, 0, 5, math.V3(0.3, 0, 1)) }, true, ""},
		{"cylinder/perpendicular", cyl, func(t *testing.T) geom.Plane { return pl(t, 0, 0, 6, math.V3(0, 0, 1)) }, true, ""},
		{"cone/oblique ellipse", cone, func(t *testing.T) geom.Plane { return pl(t, 0, 0, 5, math.V3(0.2, 0, 1)) }, true, ""},
		{"cone/through the apex", cone, func(t *testing.T) geom.Plane { return pl(t, 0, 0, 9.5, math.V3(0.4, 0, 1)) }, true, ""},
		{"cone/axis-parallel hyperbola", cone, func(t *testing.T) geom.Plane { return pl(t, 1.2, 0, 0, math.V3(1, 0, 0)) }, true, ""},
		{"cone-apex/perpendicular, base kept", apex, func(t *testing.T) geom.Plane { return pl(t, 0, 0, 5, math.V3(0, 0, 1)) }, true, ""},
		{"cone-apex/perpendicular, tip kept", apex, func(t *testing.T) geom.Plane { return pl(t, 0, 0, 5, math.V3(0, 0, -1)) }, true, ""},
		{"cone-apex/oblique below the apex", apex, func(t *testing.T) geom.Plane { return pl(t, 0, 0, 6, math.V3(0.3, 0, 1)) }, true, ""},
		{"cone-apex/axis-parallel hyperbola", apex, func(t *testing.T) geom.Plane { return pl(t, 1, 0, 0, math.V3(1, 0, 0)) }, true, ""},
		{"sphere/cap", sph, func(t *testing.T) geom.Plane { return pl(t, 0, 0, 1, math.V3(0, 0, 1)) }, true, ""},
		{"sphere/oblique cap", sph, func(t *testing.T) geom.Plane { return pl(t, 1, 1, 1, math.V3(1, 1, 1)) }, true, ""},
		{"torus/perpendicular", tor, func(t *testing.T) geom.Plane { return pl(t, 0, 0, 0.5, math.V3(0, 0, 1)) }, true, ""},
		{"torus/spiric single oval cap", tor, func(t *testing.T) geom.Plane { return pl(t, 0, 6, 0, math.V3(0, -1, 0)) }, true, ""},
		{"torus/spiric oval complement", tor, func(t *testing.T) geom.Plane { return pl(t, 0, 6, 0, math.V3(0, 1, 0)) }, true, ""},
		{"torus/spiric axis-parallel", tor, func(t *testing.T) geom.Plane { return pl(t, 1, 0, 0, math.V3(1, 0, 0)) }, true, ""},
	} {
		plane := tc.plane(t)
		cutFaces, cutOK := closedSolidFaces(HalfSpaceCut(tc.body(), plane))
		if !cutOK {
			t.Errorf("%s: the half-space cut itself does not produce a closed solid — the row's premise is gone", tc.name)
			continue
		}
		body := tc.body()
		boolFaces, boolOK := closedSolidFaces(Boolean(Difference, body, BoundedHalfSpace(plane, body.RangeBox())))
		equal := boolOK && boolFaces == cutFaces
		switch {
		case equal && !tc.equal:
			t.Errorf("%s: the difference now reproduces the cut (%d faces) — a stage landed; set equal:true and say which", tc.name, boolFaces)
		case !equal && tc.equal:
			t.Errorf("%s: the difference no longer reproduces the cut (cut %d faces, difference %d faces closed=%v)",
				tc.name, cutFaces, boolFaces, boolOK)
		case !equal:
			t.Logf("%s: still differs — %s", tc.name, tc.gate)
		}
	}
}
