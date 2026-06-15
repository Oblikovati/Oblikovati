// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// MeshSolidData is the serialized form of a MeshSolidFeature (M20-F15 #492): the source mesh
// stored inline as welded vertices plus facet index loops, so the conversion round-trips
// without depending on the original import file.
type MeshSolidData struct {
	Vertices [][]float64 `yaml:"vertices"`
	Facets   [][]int     `yaml:"facets"`
}

// serializeMeshSolid captures the source mesh inline.
func serializeMeshSolid(g *MeshGeometry) *MeshSolidData {
	return &MeshSolidData{Vertices: encodePoints(g.Vertices), Facets: copyFacets(g.Facets)}
}

// restoreMeshSolid rebuilds a MeshSolidFeature from its inline mesh.
func restoreMeshSolid(fs *PartFeatures, d *MeshSolidData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("mesh-solid feature is missing its payload")
	}
	g := &MeshGeometry{Vertices: decodePoints(d.Vertices), Facets: copyFacets(d.Facets)}
	return NewMeshFeatures(fs).AddSolid(g), nil
}

// copyFacets deep-copies facet index loops so the serialized and live forms never alias.
func copyFacets(facets [][]int) [][]int {
	out := make([][]int, len(facets))
	for i, f := range facets {
		out[i] = append([]int(nil), f...)
	}
	return out
}
