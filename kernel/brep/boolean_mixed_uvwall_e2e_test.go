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

// The uv×wall conic imprint end-to-end (#2247, #3460). A cylinder (and a cone) run clean through a plate:
// the wall's section with each plate face is a CIRCLE, which the polygonal bucket cannot receive. The
// receiving faces are promoted to the exact-frame bucket and both sides imprint the SAME circle, so the
// union comes out ANALYTIC — the wall survives as true cylinder/cone geometry (not a faceted prism) and
// the volume matches the closed form to the analytic integrator's precision. Before the promotion both
// booleans declined to the faceted fallback.

// analyticUnionVolume validates a boolean result as a watertight solid and returns its ANALYTIC volume,
// failing when a face is not analytically integrable (which is exactly what a faceted fallback produces).
func analyticUnionVolume(t *testing.T, b *topo.Body) float64 {
	t.Helper()
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("union is not a valid solid: %+v", r.Issues)
	}
	if open := ops.BoundaryEdges(b); len(open) != 0 {
		t.Fatalf("union has %d boundary edges, want 0 (watertight)", len(open))
	}
	props, ok := ops.AnalyticGeometryProperties(b)
	if !ok {
		t.Fatal("union is not analytically integrable: the boolean fell back to faceted geometry")
	}
	return props.Volume
}

// curvedFaceCount counts the result's faces whose surface is not a plane — the wall bands that prove the
// curved geometry survived the boolean.
func curvedFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, isPlane := f.Geometry().(geom.Plane); !isPlane {
			n++
		}
	}
	return n
}

// TestUnionCylinderThroughPlateKeepsCylindricalWall pins the headline case: a Ø10×4 cylinder standing
// through a 16×16×2 plate. The union must keep the cylinder wall as two true cylindrical bands (below and
// above the plate) and have the closed-form volume block + cylinder − overlap.
func TestUnionCylinderThroughPlateKeepsCylindricalWall(t *testing.T) {
	const r, h, side, thick = 5.0, 4.0, 16.0, 2.0
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatal(err)
	}
	plate, err := brep.SolidBlock(math.P3(-side/2, -side/2, 1), math.P3(side/2, side/2, 1+thick), "plate")
	if err != nil {
		t.Fatal(err)
	}
	res, err := brep.Boolean(brep.Union, cyl, plate)
	if err != nil {
		t.Fatalf("Boolean(Union, cylinder, plate) = %v", err)
	}
	if got := curvedFaceCount(res); got != 2 {
		t.Errorf("curved faces = %d, want 2 (the wall band under and over the plate)", got)
	}
	want := side*side*thick + stdmath.Pi*r*r*h - stdmath.Pi*r*r*thick
	// The budget is the analytic integrator's own precision, not a faceting budget: a faceted fallback
	// on a Ø10 wall loses ~1e-3 here, three orders of magnitude wider.
	if got := analyticUnionVolume(t, res); stdmath.Abs(got-want) > 1e-8 {
		t.Errorf("union volume = %.9f, want %.9f", got, want)
	}
}

// TestUnionConeThroughPlateKeepsConicalWall is the same contact with a CONE side band: the section circles
// have DIFFERENT radii on the two plate faces, so it also pins that the promotion is per receiving face.
func TestUnionConeThroughPlateKeepsConicalWall(t *testing.T) {
	const rBot, rTop, h, side = 4.0, 1.0, 6.0, 16.0
	cone, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, h), rBot, rTop, "cone")
	if err != nil {
		t.Fatal(err)
	}
	plate, err := brep.SolidBlock(math.P3(-side/2, -side/2, 1), math.P3(side/2, side/2, 3), "plate")
	if err != nil {
		t.Fatal(err)
	}
	res, err := brep.Boolean(brep.Union, cone, plate)
	if err != nil {
		t.Fatalf("Boolean(Union, cone, plate) = %v", err)
	}
	if got := curvedFaceCount(res); got != 2 {
		t.Errorf("curved faces = %d, want 2 (the cone band under and over the plate)", got)
	}
	want := side*side*2 + frustumVolume(rBot, rTop, h) - frustumVolume(3.5, 2.5, 2)
	if got := analyticUnionVolume(t, res); stdmath.Abs(got-want) > 1e-8 {
		t.Errorf("union volume = %.9f, want %.9f", got, want)
	}
}

// frustumVolume is the closed-form volume of a circular frustum of radii r0/r1 and height h.
func frustumVolume(r0, r1, h float64) float64 {
	return stdmath.Pi * h / 3 * (r0*r0 + r0*r1 + r1*r1)
}

// TestDifferencePlateMinusCylinderKeepsBoreWall is the CUT half of the same contact: the plate keeps the
// cylinder's wall as its bore. It pins that the promoted receiver and the wall agree under the keep table's
// reversal too — a kept Difference tool wall is flipped into the cavity, and the shared circle must still
// weld.
func TestDifferencePlateMinusCylinderKeepsBoreWall(t *testing.T) {
	const r, side, thick = 5.0, 16.0, 2.0
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, 4)
	if err != nil {
		t.Fatal(err)
	}
	plate, err := brep.SolidBlock(math.P3(-side/2, -side/2, 1), math.P3(side/2, side/2, 1+thick), "plate")
	if err != nil {
		t.Fatal(err)
	}
	res, err := brep.Boolean(brep.Difference, plate, cyl)
	if err != nil {
		t.Fatalf("Boolean(Difference, plate, cylinder) = %v", err)
	}
	if got := curvedFaceCount(res); got != 1 {
		t.Errorf("curved faces = %d, want 1 (the bore wall)", got)
	}
	want := side*side*thick - stdmath.Pi*r*r*thick
	if got := analyticUnionVolume(t, res); stdmath.Abs(got-want) > 1e-8 {
		t.Errorf("bored-plate volume = %.9f, want %.9f", got, want)
	}
}
