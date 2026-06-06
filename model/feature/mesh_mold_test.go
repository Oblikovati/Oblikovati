// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"strings"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/model/health"
)

// tetraSTL is a tiny ASCII STL of a tetrahedron (4 facets, 4 shared corners).
const tetraSTL = `solid tetra
facet normal 0 0 -1
 outer loop
  vertex 0 0 0
  vertex 0 1 0
  vertex 1 0 0
 endloop
endfacet
facet normal 0 -1 0
 outer loop
  vertex 0 0 0
  vertex 1 0 0
  vertex 0 0 1
 endloop
endfacet
facet normal -1 0 0
 outer loop
  vertex 0 0 0
  vertex 0 0 1
  vertex 0 1 0
 endloop
endfacet
facet normal 1 1 1
 outer loop
  vertex 1 0 0
  vertex 0 1 0
  vertex 0 0 1
 endloop
endfacet
endsolid tetra
`

func TestParseSTLWeldsSharedVertices(t *testing.T) {
	g, err := ParseSTL(strings.NewReader(tetraSTL))
	if err != nil {
		t.Fatalf("ParseSTL: %v", err)
	}
	if len(g.Facets) != 4 {
		t.Errorf("parsed %d facets, want 4", len(g.Facets))
	}
	// The 12 vertex references weld to the tetra's 4 distinct corners.
	if len(g.Vertices) != 4 {
		t.Errorf("welded to %d vertices, want 4", len(g.Vertices))
	}
}

func TestMeshFeatureExposesSelectableFacets(t *testing.T) {
	g, _ := ParseSTL(strings.NewReader(tetraSTL))
	fs := NewPartFeatures(nil, nil)
	pf := NewMeshFeatures(fs).Add(g)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("mesh feature unhealthy: %+v", pf.Health())
	}
	mf := pf.Definition().(*MeshFeature)
	if mf.Faces().Count() != 4 {
		t.Errorf("mesh has %d selectable facets, want 4", mf.Faces().Count())
	}
	if mf.Vertices().Count() != 4 {
		t.Errorf("mesh has %d vertices, want 4", mf.Vertices().Count())
	}
	if mf.Edges().Count() != 6 {
		t.Errorf("tetra mesh has %d edges, want 6", mf.Edges().Count())
	}
	// A facet handle yields a centroid (selection/measure hook).
	if c := mf.Faces().Item(0).Centroid(); c.Z != 0 {
		t.Errorf("facet 0 centroid Z = %v, want 0 (the −Z face)", c.Z)
	}
}

func TestMeshFeaturePassesSolidThrough(t *testing.T) {
	// A mesh is reference geometry: a prior solid survives the mesh feature.
	fs := NewPartFeatures(nil, nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(2), 0, ops.NewBody, func() float64 { return 3 })
	g, _ := ParseSTL(strings.NewReader(tetraSTL))
	NewMeshFeatures(fs).Add(g)
	fs.Recompute()
	if len(fs.Result()) != 1 || !fs.Result()[0].IsSolid() {
		t.Errorf("mesh feature should pass the solid through, got %d bodies", len(fs.Result()))
	}
}

func TestMeshFeatureSetGroups(t *testing.T) {
	g, _ := ParseSTL(strings.NewReader(tetraSTL))
	set := NewMeshFeatureSet("imported")
	set.Add(&MeshFeature{geom: g})
	set.Add(&MeshFeature{geom: g})
	if set.Name() != "imported" || set.Count() != 2 {
		t.Errorf("feature set = %q/%d, want imported/2", set.Name(), set.Count())
	}
}

func TestParseSTLRejectsGarbage(t *testing.T) {
	if _, err := ParseSTL(strings.NewReader("solid x\nfacet normal 0 0 1\nouter loop\nvertex 0 0\n")); err == nil {
		t.Error("a truncated vertex should error")
	}
	if _, err := ParseSTL(strings.NewReader("solid empty\nendsolid empty\n")); err == nil {
		t.Error("an STL with no facets should error")
	}
}

func TestCoreCavitySplitsBlock(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	// A 10×10×10 tooling block.
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(10), 0, ops.NewBody, func() float64 { return 10 })
	pf := NewCoreCavityFeatures(fs).AddByPartingPlane(PartingZ, 4, 0.02)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("core-cavity unhealthy: %+v", pf.Health())
	}
	bodies := fs.Result()
	if len(bodies) != 2 {
		t.Fatalf("split produced %d bodies, want 2 (core+cavity)", len(bodies))
	}
	core, cavity := bodies[0], bodies[1]
	if !core.IsSolid() || !cavity.IsSolid() {
		t.Error("both core and cavity should be solids")
	}
	if r := ops.Validate(core); !r.Valid {
		t.Errorf("core failed validation: %+v", r)
	}
	// Core spans z∈[0,4], cavity z∈[4,10].
	if cb := core.RangeBox(); !approxEq(cb.Min.Z, 0) || !approxEq(cb.Max.Z, 4) {
		t.Errorf("core z = [%v,%v], want [0,4]", cb.Min.Z, cb.Max.Z)
	}
	if vb := cavity.RangeBox(); !approxEq(vb.Min.Z, 4) || !approxEq(vb.Max.Z, 10) {
		t.Errorf("cavity z = [%v,%v], want [4,10]", vb.Min.Z, vb.Max.Z)
	}
}

func TestCoreCavityGoesSickWhenPartingOutsideBlock(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(10), 0, ops.NewBody, func() float64 { return 10 })
	pf := NewCoreCavityFeatures(fs).AddByPartingPlane(PartingZ, 20, 0)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("parting outside the block = %v, want sick", pf.Health().Status)
	}
}

func TestCoreCavityShrinkageRecorded(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewCoreCavityFeatures(fs).AddByPartingPlane(PartingX, 1, 0.025)
	if d := pf.Definition().(*CoreCavityFeature).Definition(); d.Shrinkage != 0.025 || d.Axis != PartingX {
		t.Errorf("recipe = %+v, want shrinkage 0.025 on X", d)
	}
}
