// SPDX-License-Identifier: GPL-2.0-only

package surface_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops/surface"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// keyedPatch builds the [0,w]×[0,h] z=0 patch with each of its four boundary edges carrying its own
// lineage (so FindEdgeByKey is unambiguous). Returns the body and the edge keys in order
// bottom, right, top, left.
func keyedPatch(t *testing.T, w, h float64) (*topo.Body, [][]byte) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "patch", 0))
	bld := topo.NewBuilder(false, lin)
	p := []math.Point3{{X: 0, Y: 0}, {X: w, Y: 0}, {X: w, Y: h}, {X: 0, Y: h}}
	v := make([]*topo.Vertex, 4)
	for i, q := range p {
		v[i] = bld.AddVertex(q, lin)
	}
	uses := make([]topo.Use, 4)
	keys := make([][]byte, 4)
	for i := range p {
		e := bld.AddEdge(geom.NewLineSegment(p[i], p[(i+1)%4]), v[i], v[(i+1)%4], topo.NewLineage(topo.Tok("test", "edge", i)))
		uses[i] = topo.Use{Edge: e}
		keys[i] = e.ReferenceKey()
	}
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, lin, topo.OuterLoop(uses...))
	return bld.Build(), keys
}

// TestExtendEdgesByDistanceGrows extends the right edge of a 2×3 patch (area 6) outward by 1: the
// patch grows to 3×3 (area 9) (#1878).
func TestExtendEdgesByDistanceGrows(t *testing.T) {
	t.Parallel()
	patch, keys := keyedPatch(t, 2, 3)
	out, err := surface.ExtendEdgesByDistance(patch, [][]byte{keys[1]}, 1, "ext")
	if err != nil {
		t.Fatal(err)
	}
	if got := query.BodyGeometryProperties(out, ops.DefaultQuality()).Area; stdmath.Abs(got-9) > 1e-6 {
		t.Errorf("area = %g, want 9 (2×3 grown to 3×3)", got)
	}
}

// TestExtendEdgesMultiGrows extends both the left and right edges of a 2×3 patch by 1 each: width
// grows to 4, area to 12 (#1878).
func TestExtendEdgesMultiGrows(t *testing.T) {
	t.Parallel()
	patch, keys := keyedPatch(t, 2, 3)
	out, err := surface.ExtendEdgesByDistance(patch, [][]byte{keys[1], keys[3]}, 1, "ext")
	if err != nil {
		t.Fatal(err)
	}
	if got := query.BodyGeometryProperties(out, ops.DefaultQuality()).Area; stdmath.Abs(got-12) > 1e-6 {
		t.Errorf("area = %g, want 12 (both sides +1 ⇒ 4×3)", got)
	}
}

// TestExtendEdgesToPlane extends the right edge of a 2×3 patch until it reaches the plane x=5: the
// patch grows to 5×3 (area 15) (#1878).
func TestExtendEdgesToPlane(t *testing.T) {
	t.Parallel()
	patch, keys := keyedPatch(t, 2, 3)
	target, _ := geom.NewPlane(math.P3(5, 0, 0), math.V3(1, 0, 0))
	out, err := surface.ExtendEdgesToPlane(patch, [][]byte{keys[1]}, target, "ext")
	if err != nil {
		t.Fatal(err)
	}
	if got := query.BodyGeometryProperties(out, ops.DefaultQuality()).Area; stdmath.Abs(got-15) > 1e-6 {
		t.Errorf("area = %g, want 15 (right edge reached x=5 ⇒ 5×3)", got)
	}
}

// TestExtendEdgesLostKeyErrors reports a vanished edge so the feature can go Sick.
func TestExtendEdgesLostKeyErrors(t *testing.T) {
	t.Parallel()
	patch, _ := keyedPatch(t, 2, 3)
	if _, err := surface.ExtendEdgesByDistance(patch, [][]byte{[]byte("ghost")}, 1, "ext"); err == nil {
		t.Error("extend with a lost edge key should error")
	}
	if _, err := surface.ExtendEdgesByDistance(patch, nil, 1, "ext"); err == nil {
		t.Error("extend with no edges should error")
	}
}
