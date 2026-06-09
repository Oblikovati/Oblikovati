// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// hexPrismBody builds a regular hexagonal prism (circumradius r, spanning z0..z1) as a
// flat-faced solid — the OpenSCAD `cylinder(R, $fn = 6)` idiom used for screw sockets.
func hexPrismBody(r, z0, z1 float64) *topo.Body {
	var v []math.Point3
	for k := 0; k < 6; k++ {
		a := float64(k) * stdmath.Pi / 3
		v = append(v, math.P3(r*stdmath.Cos(a), r*stdmath.Sin(a), z0))
	}
	for k := 0; k < 6; k++ {
		a := float64(k) * stdmath.Pi / 3
		v = append(v, math.P3(r*stdmath.Cos(a), r*stdmath.Sin(a), z1))
	}
	faces := [][]int{{5, 4, 3, 2, 1, 0}, {6, 7, 8, 9, 10, 11}} // bottom −Z, top +Z
	for k := 0; k < 6; k++ {
		n := (k + 1) % 6
		faces = append(faces, []int{k, n, n + 6, k + 6})
	}
	return subd.ToBody(subd.Mesh{Verts: v, Faces: faces}, "hex")
}

// TestNopCapScrewCSG re-models a socket-cap screw (NopSCADlib hs_cap) the OpenSCAD-CSG way:
// a head cylinder unioned onto a coaxial shaft cylinder, then a hex socket bored into the top
// of the head. Dimensions are an M3 cap screw (M3_cap_screw = head Ø5.5, head h 3, shaft Ø3,
// socket AF 2.5, socket depth 1.5) at length 10 mm. The socket is a regular hexagon whose
// circumradius is AF/(2·cos30°).
//
// Reference: NopSCADlib/vitamins/screw.scad (head_type hs_cap branch) + screws.scad.
func TestNopCapScrewCSG(t *testing.T) {
	const (
		headRad     = 2.75
		headHeight  = 3.0
		shaftRad    = 1.5
		length      = 10.0
		socketAF    = 2.5
		socketDepth = 1.5
	)
	socketRad := socketAF / (2 * stdmath.Cos(stdmath.Pi/6))

	head, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), headRad, headHeight)
	if err != nil {
		t.Fatalf("head SolidCylinder: %v", err)
	}
	// Shaft overshoots the head/shaft junction by a hair so the union welds cleanly (the
	// OpenSCAD epsilon trick) rather than meeting on an exactly coplanar disk.
	shaft, err := brep.SolidCylinder(math.P3(0, 0, -length), math.V3(0, 0, 1), shaftRad, length+0.01)
	if err != nil {
		t.Fatalf("shaft SolidCylinder: %v", err)
	}
	body, err := ops.Boolean(ops.Join, head, shaft)
	if err != nil {
		t.Fatalf("Boolean(Join head+shaft): %v", err)
	}

	// Hex socket bored into the top socketDepth of the head; overshoot the top face.
	socket := hexPrismBody(socketRad, headHeight-socketDepth, headHeight+0.01)
	body, err = ops.Boolean(ops.Cut, body, socket)
	if err != nil {
		t.Fatalf("Boolean(Cut socket): %v", err)
	}

	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("cap screw not a valid solid: %+v", r)
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Fatalf("cap screw has %d boundary edges, want 0 (watertight)", len(open))
	}

	got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	wantExact := stdmath.Pi*headRad*headRad*headHeight +
		stdmath.Pi*shaftRad*shaftRad*length -
		3*stdmath.Sqrt(3)/2*socketRad*socketRad*socketDepth
	if rel := stdmath.Abs(got-wantExact) / wantExact; rel > 0.03 {
		t.Errorf("cap screw volume = %.5f, want analytic %.5f (rel %.4f > 3%% faceting band)", got, wantExact, rel)
	}
}
