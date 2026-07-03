// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	"sort"

	"oblikovati.org/kernel/subd"
)

// Serialized forms of the direct-geometry features (M10-F03/F04): mesh (imported tessellation),
// free-form / Alias free-form (sub-D control cage), hull (OpenSCAD-style convex hull), and mold
// core/cavity. Like the surface edits, these kinds were creatable but had no serialization codec,
// so a part containing one refused to marshal (#1416/#1617). The sub-D cage's creases and level
// survive the round-trip so sharp edges are not lost on reload.

// MeshData is imported tessellated geometry: shared vertices and facets (each an ordered loop of
// vertex indices). A mesh is reference geometry, so nothing else needs storing.
type MeshData struct {
	Vertices [][]float64 `yaml:"vertices"`
	Facets   [][]int     `yaml:"facets"`
}

func serializeMesh(g *MeshGeometry) *MeshData {
	return &MeshData{Vertices: encodePoints(g.Vertices), Facets: cloneFacets(g.Facets)}
}

func restoreMesh(fs *PartFeatures, d *MeshData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("mesh feature is missing its payload")
	}
	g := &MeshGeometry{Vertices: decodePoints(d.Vertices), Facets: cloneFacets(d.Facets)}
	return NewMeshFeatures(fs).Add(g), nil
}

// cloneFacets deep-copies a facet list so the serialized/restored geometry does not alias the
// live cage (facets are variable-length loops, not fixed triangles).
func cloneFacets(facets [][]int) [][]int {
	out := make([][]int, len(facets))
	for i, f := range facets {
		out[i] = append([]int(nil), f...)
	}
	return out
}

// CreaseData is one sub-D cage edge's crease: its undirected endpoint indices and sharpness.
type CreaseData struct {
	A         int     `yaml:"a"`
	B         int     `yaml:"b"`
	Sharpness float64 `yaml:"sharpness"`
}

// FreeformData is a sub-D free-form body's recipe: the control cage (vertices, polygon faces, and
// creases) and the subdivision level. Shared by the freeform and alias-freeform codecs.
type FreeformData struct {
	Vertices [][]float64  `yaml:"vertices"`
	Faces    [][]int      `yaml:"faces"`
	Creases  []CreaseData `yaml:"creases,omitempty"`
	Level    int          `yaml:"level"`
}

func serializeFreeform(b *FreeformBody) *FreeformData {
	return &FreeformData{
		Vertices: encodePoints(b.cage.Verts),
		Faces:    cloneFacets(b.cage.Faces),
		Creases:  encodeCreases(b.cage.Creases),
		Level:    b.level,
	}
}

// freeformCage rebuilds the sub-D cage from serialized data (both freeform kinds share it).
func freeformCage(d *FreeformData) subd.Mesh {
	cage := subd.Mesh{Verts: decodePoints(d.Vertices), Faces: cloneFacets(d.Faces)}
	for _, c := range d.Creases {
		cage.SetCrease(c.A, c.B, c.Sharpness)
	}
	return cage
}

func restoreFreeform(fs *PartFeatures, d *FreeformData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("freeform feature is missing its payload")
	}
	return NewFreeformFeatures(fs).add(freeformCage(d), d.Level), nil
}

func restoreAliasFreeform(fs *PartFeatures, d *FreeformData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("alias-freeform feature is missing its payload")
	}
	return NewAliasFreeformFeatures(fs).add(freeformCage(d), d.Level), nil
}

// encodeCreases flattens the crease map into a slice sorted by (a,b) so the serialized order is
// deterministic (a map iterates in random order, which would churn the document on every save).
func encodeCreases(creases map[[2]int]float64) []CreaseData {
	if len(creases) == 0 {
		return nil
	}
	out := make([]CreaseData, 0, len(creases))
	for k, s := range creases {
		out = append(out, CreaseData{A: k[0], B: k[1], Sharpness: s})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].A != out[j].A {
			return out[i].A < out[j].A
		}
		return out[i].B < out[j].B
	})
	return out
}

// HullData is a hull's recipe. The hull is taken over the running solids with no parameters, so
// the payload is empty — but a codec must still exist (a kind with no payload struct silently
// dropped on save; #1416).
type HullData struct{}

func restoreHull(fs *PartFeatures, d *HullData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("hull feature is missing its payload")
	}
	return NewHullFeatures(fs).Add(), nil
}

// CoreCavityData is a mold split's recipe: the parting axis, its position, and the shrinkage
// allowance (fractional oversize compensating part shrink).
type CoreCavityData struct {
	Axis      int     `yaml:"axis"`
	Position  float64 `yaml:"position"`
	Shrinkage float64 `yaml:"shrinkage"`
}

func serializeCoreCavity(def *CoreCavityDefinition) *CoreCavityData {
	return &CoreCavityData{Axis: int(def.Axis), Position: evalFloat(def.Position), Shrinkage: def.Shrinkage}
}

func restoreCoreCavity(fs *PartFeatures, d *CoreCavityData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("core-cavity feature is missing its payload")
	}
	return NewCoreCavityFeatures(fs).AddByPartingPlaneFn(PartingAxis(d.Axis), constFloat(d.Position), d.Shrinkage), nil
}
