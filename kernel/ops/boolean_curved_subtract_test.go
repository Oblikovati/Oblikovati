// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// Cylinder − box subtract (M2 Phase 1, Oblikovati/Oblikovati#1334). A box tunnelling along the cylinder
// axis, inside the radius, must Cut to an exact cylinder with a prismatic hole — the side preserved, each
// cap an annulus-like planar face the tunnel passes through — not triangle-soup CSG.

// TestBooleanCutCylinderAxialSquareHole drills a 2×2 square hole down the axis of a radius-3 cylinder.
// Volume = cylinder − the square prism; the cylinder side survives as one exact face.
func TestBooleanCutCylinderAxialSquareHole(t *testing.T) {
	t.Parallel()
	const r, h, a = 3.0, 10.0, 1.0
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	box, _ := brep.SolidBlock(math.P3(-a, -a, -5), math.P3(a, a, 15), "box") // spans beyond both caps

	res, err := ops.Boolean(ops.Cut, cyl, box)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("cylinder−box is not a valid closed manifold solid: %+v", v)
	}
	cylFaces := 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			cylFaces++
		case geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic (the exact path must be taken, not CSG)", f.Geometry())
		}
	}
	if cylFaces != 1 {
		t.Errorf("result has %d cylinder faces, want 1 (the side survived the axial tunnel)", cylFaces)
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*r*r*h - (2*a)*(2*a)*h
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("cylinder−box volume %.4f, want %.4f (analytic) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestBooleanCutCylinderBlindPocketDefers: a box that does NOT pass through both caps is a blind pocket,
// not an axial tunnel; the exact path declines it so the CSG fallback still runs (no wrong result).
func TestBooleanCutCylinderBlindPocketDefers(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	box, _ := brep.SolidBlock(math.P3(-1, -1, 4), math.P3(1, 1, 6), "box") // wholly inside, spans nothing
	res, err := ops.Boolean(ops.Cut, cyl, box)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !res.IsSolid() {
		t.Fatalf("blind-pocket cut is not a valid solid: %+v", v)
	}
}
