// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/internal/testcage"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// triBox builds a box [0,sx]×[0,sy]×[0,sz] whose six faces are each split into two outward-wound
// triangles — the shattered, one-face-per-triangle shape ops.Facet leaves, but built directly so
// UnifyCoplanarFaces can be exercised in isolation.
func triBox(sx, sy, sz float64) *topo.Body {
	v := []math.Point3{
		math.P3(0, 0, 0), math.P3(sx, 0, 0), math.P3(sx, sy, 0), math.P3(0, sy, 0),
		math.P3(0, 0, sz), math.P3(sx, 0, sz), math.P3(sx, sy, sz), math.P3(0, sy, sz),
	}
	quads := [][4]int{
		{0, 3, 2, 1}, // bottom (−Z)
		{4, 5, 6, 7}, // top (+Z)
		{0, 1, 5, 4}, // −Y
		{2, 3, 7, 6}, // +Y
		{1, 2, 6, 5}, // +X
		{0, 4, 7, 3}, // −X
	}
	var faces [][]int
	for _, q := range quads {
		faces = append(faces, []int{q[0], q[1], q[2]}, []int{q[0], q[2], q[3]})
	}
	return subd.ToBody(subd.Mesh{Verts: v, Faces: faces}, "tribox")
}

// TestUnifyCoplanarMergesTriBox merges a triangulated box's 12 triangle faces back into its 6
// planar faces, staying a valid solid with χ=2 and volume preserved.
func TestUnifyCoplanarMergesTriBox(t *testing.T) {
	t.Parallel()
	tri := triBox(2, 3, 4)
	if len(tri.Faces()) != 12 {
		t.Fatalf("triBox has %d faces, want 12", len(tri.Faces()))
	}
	u := brep.UnifyCoplanarFaces(tri, "unify")
	if u == nil {
		t.Fatal("UnifyCoplanarFaces returned nil")
	}
	if len(u.Faces()) != 6 {
		t.Errorf("unified box has %d faces, want 6", len(u.Faces()))
	}
	r := ops.Validate(u)
	if !r.Valid || !u.IsSolid() || r.EulerCharacteristic != 2 {
		t.Fatalf("unified box invalid: valid=%v solid=%v χ=%d issues=%v", r.Valid, u.IsSolid(), r.EulerCharacteristic, r.Issues)
	}
	if got := query.BodyGeometryProperties(u, ops.DefaultQuality()).Volume; stdmath.Abs(got-24) > 1e-6 {
		t.Errorf("unified box volume = %g, want 24", got)
	}
}

// TestUnifyCoplanarKeepsHoleLoop a coplanar region with a hole (a plate faceted into triangles
// around a square opening) must unify into ONE face carrying an outer loop AND the hole loop, not
// bridge the hole shut. Built by faceting a genuine holed prism, then unifying.
func TestUnifyCoplanarKeepsHoleLoop(t *testing.T) {
	t.Parallel()
	outer, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(4, 4, 1), "outer")
	inner, _ := brep.SolidBlock(math.P3(1.5, 1.5, -1), math.P3(2.5, 2.5, 2), "inner")
	holed, err := brep.Boolean(brep.Difference, outer, inner)
	if err != nil {
		t.Fatalf("difference: %v", err)
	}
	u := brep.UnifyCoplanarFaces(testcage.Body(holed, "facet"), "unify")
	r := ops.Validate(u)
	if !r.Valid || !u.IsSolid() {
		t.Fatalf("unified holed plate invalid: %+v", r)
	}
	// The +Z top plate (z=1) is an annulus: exactly one face there, with an outer loop and a hole.
	top := 0
	holeLoops := 0
	for _, f := range u.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || pl.Normal().Z < 0.99 || pl.Origin.Z < 1-1e-4 {
			continue
		}
		top++
		holeLoops += len(f.Loops()) - 1
	}
	if top != 1 {
		t.Errorf("top plate split into %d faces, want 1", top)
	}
	if holeLoops != 1 {
		t.Errorf("top plate carries %d hole loops, want 1 (the square opening)", holeLoops)
	}
}
