// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// planarPatch builds a one-face surface (non-solid) body: the rectangle [0,w]×[0,h] in the
// z=0 plane (normal +Z).
func planarPatch(t *testing.T, w, h float64) *topo.Body {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "patch", 0))
	bld := topo.NewBuilder(false, lin)
	p := []math.Point3{{X: 0, Y: 0}, {X: w, Y: 0}, {X: w, Y: h}, {X: 0, Y: h}}
	v := make([]*topo.Vertex, 4)
	for i, q := range p {
		v[i] = bld.AddVertex(q, lin)
	}
	uses := make([]topo.Use, 4)
	for i := range p {
		e := bld.AddEdge(geom.NewLineSegment(p[i], p[(i+1)%4]), v[i], v[(i+1)%4], lin)
		uses[i] = topo.Use{Edge: e}
	}
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, lin, topo.OuterLoop(uses...))
	return bld.Build()
}

// TestThickenPatchToSlab thickens a 2×3 planar patch by 0.5 into a slab solid of volume
// 2·3·0.5 = 3, validated.
func TestThickenPatchToSlab(t *testing.T) {
	patch := planarPatch(t, 2, 3)
	if patch.IsSolid() {
		t.Fatal("patch should be a surface body, not a solid")
	}
	slab, err := ops.Thicken(patch, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(slab); !r.Valid || !slab.IsSolid() {
		t.Fatalf("thickened patch not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(slab, ops.DefaultQuality()).Volume; stdmath.Abs(got-3) > 1e-6 {
		t.Errorf("slab volume = %g, want 3", got)
	}
}

// TestThickenThicknessMustBePositive guards the thickness.
func TestThickenThicknessMustBePositive(t *testing.T) {
	if _, err := ops.Thicken(planarPatch(t, 1, 1), 0); err == nil {
		t.Error("zero thickness should error")
	}
}
