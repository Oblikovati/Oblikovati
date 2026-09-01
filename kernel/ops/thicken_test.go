// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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
	t.Parallel()
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
	t.Parallel()
	if _, err := ops.Thicken(planarPatch(t, 1, 1), 0); err == nil {
		t.Error("zero thickness should error")
	}
}

// twoFaceSheet builds a flat two-face surface: rectangles [0,w]×[0,h] and [w,2w]×[0,h] in z=0,
// sharing the x=w edge, each with its own lineage so the faces have distinct reference keys.
func twoFaceSheet(t *testing.T, w, h float64) (*topo.Body, [][]byte) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "sheet", 0))
	bld := topo.NewBuilder(false, lin)
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	var keys [][]byte
	quad := func(x0 float64, tag int) {
		p := []math.Point3{{X: x0, Y: 0}, {X: x0 + w, Y: 0}, {X: x0 + w, Y: h}, {X: x0, Y: h}}
		fl := topo.NewLineage(topo.Tok("test", "face", tag))
		v := make([]*topo.Vertex, 4)
		for i, q := range p {
			v[i] = bld.AddVertex(q, fl)
		}
		uses := make([]topo.Use, 4)
		for i := range p {
			e := bld.AddEdge(geom.NewLineSegment(p[i], p[(i+1)%4]), v[i], v[(i+1)%4], fl)
			uses[i] = topo.Use{Edge: e}
		}
		f := bld.AddFace(plane, fl, topo.OuterLoop(uses...))
		keys = append(keys, f.ReferenceKey())
	}
	quad(0, 1)
	quad(w, 2)
	return bld.Build(), keys
}

// TestThickenDirectionPlacesSlab pins the #1876 direction: a 2×2 patch thickened 0.5 lands on the
// +normal side (centroid z +t/2), the −normal side, or centred (symmetric) — all volume 2.
func TestThickenDirectionPlacesSlab(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		dir    ops.ThickenDirection
		wantCZ float64
	}{
		{"positive", ops.ThickenPositive, 0.25},
		{"negative", ops.ThickenNegative, -0.25},
		{"symmetric", ops.ThickenSymmetric, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			slab, err := ops.ThickenSolid(planarPatch(t, 2, 2), 0.5, tc.dir, nil, true)
			if err != nil {
				t.Fatal(err)
			}
			p := ops.BodyGeometryProperties(slab, ops.DefaultQuality())
			if stdmath.Abs(p.Volume-2) > 1e-6 {
				t.Errorf("volume = %g, want 2", p.Volume)
			}
			if stdmath.Abs(float64(p.Centroid.Z)-tc.wantCZ) > 1e-6 {
				t.Errorf("centroid z = %g, want %g", p.Centroid.Z, tc.wantCZ)
			}
		})
	}
}

// TestThickenFaceSubset thickens only one face of a two-face sheet: the result is a valid solid of
// just that face's volume (1·2·0.5 = 1), the shared edge closed by a vertical surface (#1876).
func TestThickenFaceSubset(t *testing.T) {
	t.Parallel()
	sheet, keys := twoFaceSheet(t, 1, 2)
	solid, err := ops.ThickenSolid(sheet, 0.5, ops.ThickenPositive, [][]byte{keys[0]}, true)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(solid); !r.Valid || !solid.IsSolid() {
		t.Fatalf("subset thicken not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(solid, ops.DefaultQuality()).Volume; stdmath.Abs(got-1) > 1e-6 {
		t.Errorf("subset volume = %g, want 1 (one 1×2 face × 0.5)", got)
	}
}
