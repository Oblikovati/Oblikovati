// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// Facet/stroke calculation and retrieval over the wire (M07-F03 remainder,
// #293), backed by the session's tolerance-keyed facet store.

// bodyCalculateFacets serves wire.MethodBodyCalculateFacets.
func bodyCalculateFacets(s *app.Session, part *compdef.PartComponentDefinition, in wire.CalculateFacetsArgs) (wire.FacetSetResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.FacetSetResult{}, err
	}
	fs := s.FacetStore().CalculateFacets(b, in.Tolerance)
	return facetSetReply(fs, in.IncludeTextureMap), nil
}

// bodyExistingFacets serves wire.MethodBodyExistingFacets — retrieval only.
func bodyExistingFacets(s *app.Session, part *compdef.PartComponentDefinition, in wire.CalculateFacetsArgs) (wire.FacetSetResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.FacetSetResult{}, err
	}
	fs, ok := s.FacetStore().ExistingFacets(b, in.Tolerance)
	if !ok {
		return wire.FacetSetResult{}, fmt.Errorf("no facet set exists at tolerance %g (calculate it first; cached: %v)",
			in.Tolerance, s.FacetStore().FacetTolerances(b))
	}
	return facetSetReply(fs, in.IncludeTextureMap), nil
}

func facetSetReply(fs *tessellate.BodyFacets, withUV bool) wire.FacetSetResult {
	out := wire.FacetSetResult{
		VertexCount: len(fs.Mesh.Positions), FacetCount: len(fs.Mesh.Indices) / 3,
		VertexIndices: fs.Mesh.Indices, IndexCountPerFace: fs.IndexCountPerFace,
	}
	out.VertexCoordinates, out.NormalVectors = meshArrays(fs.Mesh)
	if withUV {
		out.TextureCoordinates = fs.TextureCoordinates()
	}
	return out
}

func meshArrays(m *ops.Mesh) ([]float64, []float64) {
	coords := make([]float64, 0, 3*len(m.Positions))
	normals := make([]float64, 0, 3*len(m.Normals))
	for _, p := range m.Positions {
		coords = append(coords, float64(p.X), float64(p.Y), float64(p.Z))
	}
	for _, n := range m.Normals {
		normals = append(normals, float64(n.X), float64(n.Y), float64(n.Z))
	}
	return coords, normals
}

// bodyFacetTolerances serves wire.MethodBodyFacetTolerances.
func bodyFacetTolerances(s *app.Session, part *compdef.PartComponentDefinition, in wire.BodyIndexArgs) (wire.FacetTolerancesResult, error) {
	return toleranceReply(part, in.BodyIndex, func(b *topo.Body) []float64 { return s.FacetStore().FacetTolerances(b) })
}

// bodyStrokeTolerances serves wire.MethodBodyStrokeTolerances.
func bodyStrokeTolerances(s *app.Session, part *compdef.PartComponentDefinition, in wire.BodyIndexArgs) (wire.FacetTolerancesResult, error) {
	return toleranceReply(part, in.BodyIndex, func(b *topo.Body) []float64 { return s.FacetStore().StrokeTolerances(b) })
}

func toleranceReply(part *compdef.PartComponentDefinition, index int, list func(*topo.Body) []float64) (wire.FacetTolerancesResult, error) {
	b, err := bodyAt(part, index)
	if err != nil {
		return wire.FacetTolerancesResult{}, err
	}
	return wire.FacetTolerancesResult{Tolerances: list(b)}, nil
}

// bodyCalculateStrokes serves wire.MethodBodyCalculateStrokes.
func bodyCalculateStrokes(s *app.Session, part *compdef.PartComponentDefinition, in wire.CalculateStrokesArgs) (wire.StrokeSetResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.StrokeSetResult{}, err
	}
	return strokeSetReply(s.FacetStore().CalculateStrokes(b, in.Tolerance).Polylines), nil
}

// bodyExistingStrokes serves wire.MethodBodyExistingStrokes — retrieval only.
func bodyExistingStrokes(s *app.Session, part *compdef.PartComponentDefinition, in wire.CalculateStrokesArgs) (wire.StrokeSetResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.StrokeSetResult{}, err
	}
	ss, ok := s.FacetStore().ExistingStrokes(b, in.Tolerance)
	if !ok {
		return wire.StrokeSetResult{}, fmt.Errorf("no stroke set exists at tolerance %g (calculate it first; cached: %v)",
			in.Tolerance, s.FacetStore().StrokeTolerances(b))
	}
	return strokeSetReply(ss.Polylines), nil
}

func strokeSetReply(polylines [][]math.Point3) wire.StrokeSetResult {
	out := wire.StrokeSetResult{PolylineCount: len(polylines)}
	for _, pl := range polylines {
		out.PolylineLengths = append(out.PolylineLengths, len(pl))
		for _, p := range pl {
			out.VertexCoordinates = append(out.VertexCoordinates, float64(p.X), float64(p.Y), float64(p.Z))
		}
		out.VertexCount += len(pl)
	}
	return out
}

// faceCalculateFacets serves wire.MethodFaceCalculateFacets.
func faceCalculateFacets(s *app.Session, part *compdef.PartComponentDefinition, in wire.FaceFacetsArgs) (wire.FacetSetResult, error) {
	b, f, err := resolveFace(part, in)
	if err != nil {
		return wire.FacetSetResult{}, err
	}
	mesh, ok := s.FacetStore().FaceFacets(b, f, in.Tolerance)
	if !ok {
		return wire.FacetSetResult{}, fmt.Errorf("face not found in the body's facet set")
	}
	out := wire.FacetSetResult{
		VertexCount: len(mesh.Positions), FacetCount: len(mesh.Indices) / 3,
		VertexIndices: mesh.Indices, IndexCountPerFace: []int{len(mesh.Indices)},
	}
	out.VertexCoordinates, out.NormalVectors = meshArrays(mesh)
	return out, nil
}

// faceCalculateStrokes serves wire.MethodFaceCalculateStrokes.
func faceCalculateStrokes(s *app.Session, part *compdef.PartComponentDefinition, in wire.FaceFacetsArgs) (wire.StrokeSetResult, error) {
	_, f, err := resolveFace(part, in)
	if err != nil {
		return wire.StrokeSetResult{}, err
	}
	return strokeSetReply(s.FacetStore().FaceStrokes(f, in.Tolerance)), nil
}

// resolveFace resolves the face-addressed args to their body and face on the active part.
func resolveFace(part *compdef.PartComponentDefinition, in wire.FaceFacetsArgs) (*topo.Body, *topo.Face, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return nil, nil, err
	}
	f, ok := b.FindFaceByKey([]byte(in.FaceKey))
	if !ok {
		return nil, nil, fmt.Errorf("no face with key %q on body %d", in.FaceKey, in.BodyIndex)
	}
	return b, f, nil
}
