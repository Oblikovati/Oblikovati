// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// sphereFaces tallies the analytic geom.Sphere faces of a body.
func sphereFaces(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Sphere); ok {
			n++
		}
	}
	return n
}

// hemisphereCappedMeridian is the (r,z) meridian of a cylinder (radius r, height h) closed at the top
// by a hemispherical cap: a bottom disk, a cylinder wall, then a 90° arc from the rim (r,h) to the pole
// (0, h+r) about the on-axis centre (0,h). Revolved, that arc is a spherical cap — the canonical on-apex
// domed end (a tapered roller's big end).
func hemisphereCappedMeridian(r, h float64) []brep.RevolveVertex {
	c := math.P2(0, h)
	return []brep.RevolveVertex{
		{P: math.P2(0, 0)},
		{P: math.P2(r, 0)},
		{P: math.P2(r, h)},
		{P: math.P2(0, h+r), ArcCenter: &c}, // the arc edge (r,h)→(0,h+r) about the on-axis centre (0,h)
	}
}

// TestSolidOfRevolutionSphereCap is the #129 curved-meridian follow-up (sphere case, the tapered-roller
// domed end): an on-axis arc closing at the pole revolves to ONE analytic geom.Sphere cap face — not the
// hundreds of cone slivers the faceted swept solid leaves — so the mesh carries true curvature and the
// per-frame hover-pick stays cheap. The body is a cylinder (r=2,h=5) topped by a hemisphere.
func TestSolidOfRevolutionSphereCap(t *testing.T) {
	r, h := 2.0, 5.0
	body, err := brep.SolidOfRevolutionMeridian(math.P3(0, 0, 0), math.V3(0, 0, 1), hemisphereCappedMeridian(r, h), "capped")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolutionMeridian(hemisphere-capped cylinder) = (%v, %v), want a body", body, err)
	}
	if res := ops.Validate(body); !res.Valid || !body.IsSolid() {
		t.Fatalf("hemisphere-capped cylinder is not a valid solid: %+v", res.Issues)
	}
	if got := sphereFaces(body); got != 1 {
		t.Fatalf("hemisphere-capped cylinder has %d sphere faces, want exactly 1 (got %d total faces)", got, len(body.Faces()))
	}
	if got := len(body.Faces()); got != 3 {
		t.Fatalf("hemisphere-capped cylinder has %d faces, want 3 (bottom disk + wall + cap)", got)
	}
	want := stdmath.Pi*r*r*h + 2.0/3.0*stdmath.Pi*r*r*r // cylinder + hemisphere
	if got := vol(body); relErrF(got, want) > 0.02 {
		t.Errorf("hemisphere-capped cylinder volume = %g, want ≈%g (πr²h + ⅔πr³)", got, want)
	}
}

// TestSolidOfRevolutionSphereCapAutoOrients feeds the SAME meridian wound CLOCKWISE (negative area): the
// builder must re-wind it to CCW AND re-key the arc centre onto the reversed edge's new end vertex
// (ccwMeridianVerts), so a caller need not know the projected winding. The result is the identical valid
// hemisphere-capped solid.
func TestSolidOfRevolutionSphereCapAutoOrients(t *testing.T) {
	r, h := 2.0, 5.0
	c := math.P2(0, h)
	// Clockwise traversal: (0,0) → pole (0,h+r) → rim (r,h) → (r,0). The arc edge is (0,h+r)→(r,h)
	// about (0,h), so its centre rides the rim vertex (r,h) — the END of that CW edge.
	cw := []brep.RevolveVertex{
		{P: math.P2(0, 0)},
		{P: math.P2(0, h+r)},
		{P: math.P2(r, h), ArcCenter: &c},
		{P: math.P2(r, 0)},
	}
	body, err := brep.SolidOfRevolutionMeridian(math.P3(0, 0, 0), math.V3(0, 0, 1), cw, "capped-cw")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolutionMeridian(CW hemisphere) = (%v, %v), want a body", body, err)
	}
	if res := ops.Validate(body); !res.Valid || !body.IsSolid() {
		t.Fatalf("CW hemisphere-capped cylinder is not a valid solid: %+v", res.Issues)
	}
	if got := sphereFaces(body); got != 1 {
		t.Fatalf("CW hemisphere cap has %d sphere faces, want 1", got)
	}
	want := stdmath.Pi*r*r*h + 2.0/3.0*stdmath.Pi*r*r*r
	if got := vol(body); relErrF(got, want) > 0.02 {
		t.Errorf("CW hemisphere-capped volume = %g, want ≈%g", got, want)
	}
}

// TestSolidOfRevolutionSphereZoneFallsBack pins the analytic boundary: an on-axis arc whose BOTH
// endpoints are off the axis revolves to a sphere ZONE (a barrel), which needs a framed sphere
// parameterization not yet built, so the builder returns (nil,nil) — the signal for the caller to keep
// the faceted revolve. r0=2, H=4, centre (0,2) radius √8.
func TestSolidOfRevolutionSphereZoneFallsBack(t *testing.T) {
	c := math.P2(0, 2) // on the axis, but both arc endpoints are at r=2 (off-axis) → a zone, not a cap
	zone := []brep.RevolveVertex{
		{P: math.P2(0, 0)},
		{P: math.P2(2, 0)},
		{P: math.P2(2, 4), ArcCenter: &c},
		{P: math.P2(0, 4)},
	}
	body, err := brep.SolidOfRevolutionMeridian(math.P3(0, 0, 0), math.V3(0, 0, 1), zone, "zone")
	if err != nil {
		t.Fatalf("SolidOfRevolutionMeridian(zone) error = %v, want nil (a clean fallback signal)", err)
	}
	if body != nil {
		t.Fatalf("SolidOfRevolutionMeridian(sphere zone) built %d faces, want nil (fall back to faceted)", len(body.Faces()))
	}
}

// relErrF is the relative error between got and want (want ≠ 0).
func relErrF(got, want float64) float64 { return stdmath.Abs(got-want) / stdmath.Abs(want) }
