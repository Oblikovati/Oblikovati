// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The mixed boolean's WALL route is the ruled route (ADR-0058): a cylinder is the degenerate cone,
// so both take the same band description, the same imprint and the same (u,v) split. Before this a
// cone side fell to the pass-through bucket, which can only carry a face provably CLEAR of the other
// operand — so every cone that actually met something declined the whole boolean.

// TestRuledSideBandTakesACylinder: the band carries equal radii and the full height.
func TestRuledSideBandTakesACylinder(t *testing.T) {
	t.Parallel()
	body, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 7)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	rs, ok := ruledSideBandOf(sideFaceOf(t, body))
	if !ok {
		t.Fatal("a full cylinder side band was not recognised as a ruled wall")
	}
	if _, isCyl := rs.surface.(geom.Cylinder); !isCyl {
		t.Errorf("surface is %T, want geom.Cylinder", rs.surface)
	}
	if rs.band.rBot != 3 || rs.band.rTop != 3 {
		t.Errorf("band radii = (%g, %g), want (3, 3)", rs.band.rBot, rs.band.rTop)
	}
	if got := rs.band.vMax - rs.band.vMin; stdmath.Abs(got-7) > 1e-12 {
		t.Errorf("band height = %g, want 7", got)
	}
}

// TestRuledSideBandTakesACone: the same description with DIFFERENT radii — the case that used to
// fall through to pass-through.
func TestRuledSideBandTakesACone(t *testing.T) {
	t.Parallel()
	body, err := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 6), 4, 1, "cone")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	rs, ok := ruledSideBandOf(sideFaceOf(t, body))
	if !ok {
		t.Fatal("a full cone side band was not recognised as a ruled wall")
	}
	if _, isCone := rs.surface.(geom.Cone); !isCone {
		t.Errorf("surface is %T, want geom.Cone", rs.surface)
	}
	// The band is ordered along the CONE's own axis, which points toward the apex — not along world
	// +z — so the two rim radii are asserted as a set, with the narrow end at the far (vMax) rim.
	if stdmath.Abs(rs.band.rBot-1) > 1e-9 || stdmath.Abs(rs.band.rTop-4) > 1e-9 {
		t.Errorf("band radii = (%g, %g), want the rims ordered along the cone axis: (1, 4)", rs.band.rBot, rs.band.rTop)
	}
	if a := stdmath.Abs(float64(rs.axis.Dot(math.V3(0, 0, 1)))); stdmath.Abs(a-1) > 1e-12 {
		t.Errorf("band axis %v is not parallel to the cone's z axis", rs.axis)
	}
	// The band's own frame must close: walking vMax along the axis from the bottom rim centre lands
	// on the top rim centre.
	walked := rs.band.bottom.TranslateBy(rs.axis.Scale(math.Scalar(rs.band.vMax - rs.band.vMin)))
	if d := float64(walked.DistanceTo(rs.band.top)); d > 1e-9 {
		t.Errorf("walking the band height along its axis misses the top rim by %g", d)
	}
}

// TestRuledSideBandDeclinesAPlane: only a full periodic side band is a wall; a cap is not.
func TestRuledSideBandDeclinesAPlane(t *testing.T) {
	t.Parallel()
	body, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 7)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	for _, cf := range facesOfAny(body) {
		if _, isPlane := cf.surface.(geom.Plane); !isPlane {
			continue
		}
		if _, ok := ruledSideBandOf(cf); ok {
			t.Error("a planar cap was accepted as a ruled wall")
		}
	}
}

// TestRuledSideSizeSpansTheWiderRim: the characteristic length must use the LARGER radius, or a cone
// would take its intersections at a resolution scaled to its narrow end.
func TestRuledSideSizeSpansTheWiderRim(t *testing.T) {
	t.Parallel()
	rs := ruledSide{band: coneSideBand_{rBot: 4, rTop: 1, vMin: 0, vMax: 6}}
	if got := rs.size(); stdmath.Abs(got-14) > 1e-12 {
		t.Errorf("size = %g, want 2*4 + 6 = 14", got)
	}
}

// sideFaceOf returns the body's single non-planar face.
func sideFaceOf(t *testing.T, b *topo.Body) curvedFace {
	t.Helper()
	for _, cf := range facesOfAny(b) {
		if _, isPlane := cf.surface.(geom.Plane); !isPlane {
			return cf
		}
	}
	t.Fatal("body has no curved side face")
	return curvedFace{}
}
