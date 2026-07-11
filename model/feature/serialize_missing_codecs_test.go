// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// Regression for #1617: eleven creatable feature kinds (the surface-edit and direct-geometry
// families) carried no serialization codec, so a part containing any of them failed to marshal with
// "no serialization codec for feature kind …" — a silent save/undo refusal. Each must now round-trip.

// TestSurfaceEditCodecsRoundTrip builds one part holding every surface-edit kind, marshals and
// restores it, and asserts each recipe's parameters survived.
func TestSurfaceEditCodecsRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewTrimFeatures(fs).AddByPlane(math.P3(1, 2, 3), math.V3(0, 0, 1), true)
	NewExtendFeatures(fs).Add([]byte("edge-key-\x00\xff"), func() float64 { return 2.5 })
	NewSurfaceOffsetFeatures(fs).AddByDistance(func() float64 { return -1.25 })
	NewMidSurfaceFeatures(fs).AddByThickness(4)
	NewStitchFeatures(fs).Add(0.001, true)
	NewSculptFeatures(fs).Add(ops.Cut, 0.002)

	fresh := marshalRestore(t, fs)

	trim := featAt[*TrimFeature](t, fresh, 0).Definition()
	if trim.CutOrigin != math.P3(1, 2, 3) || trim.CutNormal != math.V3(0, 0, 1) || !trim.KeepPositive {
		t.Errorf("restored trim = %+v, want origin (1,2,3) normal (0,0,1) keepPositive", trim)
	}
	ext := featAt[*ExtendFeature](t, fresh, 1).Definition()
	if len(ext.EdgeKeys) != 1 || string(ext.EdgeKeys[0]) != "edge-key-\x00\xff" || ext.Distance() != 2.5 {
		t.Errorf("restored extend keys=%q dist=%g, want the original key and 2.5", ext.EdgeKeys, ext.Distance())
	}
	if off := featAt[*SurfaceOffsetFeature](t, fresh, 2).Definition(); off.Distance() != -1.25 {
		t.Errorf("restored surface-offset distance = %g, want -1.25", off.Distance())
	}
	if mid := featAt[*MidSurfaceFeature](t, fresh, 3).Definition(); mid.MaxThickness != 4 {
		t.Errorf("restored mid-surface maxThickness = %g, want 4", mid.MaxThickness)
	}
	if st := featAt[*StitchFeature](t, fresh, 4).Definition(); st.Tolerance != 0.001 || !st.MaintainAsSurface {
		t.Errorf("restored stitch = %+v, want tol 0.001 maintainAsSurface", st)
	}
	if sc := featAt[*SculptFeature](t, fresh, 5).Definition(); sc.Operation != ops.Cut || sc.Tolerance != 0.002 {
		t.Errorf("restored sculpt = %+v, want op Cut tol 0.002", sc)
	}
}

// TestDirectModelCodecsRoundTrip builds one part holding every direct-geometry kind (mesh,
// freeform, alias-freeform, hull, core-cavity) and asserts the geometry payload survived — for the
// sub-D cages, that the crease and subdivision level are not lost.
func TestDirectModelCodecsRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	g := &MeshGeometry{Vertices: []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)}, Facets: [][]int{{0, 1, 2}}}
	NewMeshFeatures(fs).Add(g)
	ff := NewFreeformFeatures(fs).AddBox(2, 2, 2, 1)
	ff.Definition().(*FreeformFeature).FreeformBody().CreaseEdges([][2]int{{0, 1}}, 0.5)
	NewAliasFreeformFeatures(fs).AddFromCage([]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)}, [][]int{{0, 1, 2, 3}}, 2)
	NewHullFeatures(fs).Add()
	NewCoreCavityFeatures(fs).AddByPartingPlane(PartingY, 3.5, 0.02)

	fresh := marshalRestore(t, fs)

	mesh := featAt[*MeshFeature](t, fresh, 0).Geometry()
	if len(mesh.Vertices) != 3 || len(mesh.Facets) != 1 || len(mesh.Facets[0]) != 3 {
		t.Errorf("restored mesh = %d verts %d facets, want 3/1", len(mesh.Vertices), len(mesh.Facets))
	}
	body := featAt[*FreeformFeature](t, fresh, 1).FreeformBody()
	if body.Level() != 1 {
		t.Errorf("restored freeform level = %d, want 1", body.Level())
	}
	if s := body.cage.Creases[[2]int{0, 1}]; s != 0.5 {
		t.Errorf("restored freeform crease(0,1) = %g, want 0.5 (crease dropped on round-trip)", s)
	}
	if alias := featAt[*AliasFreeformFeature](t, fresh, 2).FreeformBody(); alias.Level() != 2 {
		t.Errorf("restored alias-freeform level = %d, want 2", alias.Level())
	}
	if k := fresh.Item(3).Kind(); k != "hull" {
		t.Errorf("fourth feature kind = %q, want hull", k)
	}
	cc := featAt[*CoreCavityFeature](t, fresh, 4).Definition()
	if cc.Axis != PartingY || cc.Position() != 3.5 || cc.Shrinkage != 0.02 {
		t.Errorf("restored core-cavity = axis %d pos %g shrink %g, want Y/3.5/0.02", cc.Axis, cc.Position(), cc.Shrinkage)
	}
}

// marshalRestore projects a feature program to its serialized form and rebuilds it into a fresh
// engine — the save→reload path under test.
func marshalRestore(t *testing.T, fs *PartFeatures) *PartFeatures {
	t.Helper()
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	return fresh
}

// featAt returns the i-th restored feature type-asserted to T, failing the test if the kind differs.
func featAt[T Feature](t *testing.T, fs *PartFeatures, i int) T {
	t.Helper()
	f, ok := fs.Item(i).Definition().(T)
	if !ok {
		t.Fatalf("feature %d has type %T, want %T", i, fs.Item(i).Definition(), *new(T))
	}
	return f
}
