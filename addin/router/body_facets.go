// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Facet/stroke calculation and retrieval over the wire (M07-F03 remainder,
// #293), backed by the session's tolerance-keyed facet store.

// bodyCalculateFacets serves wire.MethodBodyCalculateFacets.
func bodyCalculateFacets(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.CalculateFacetsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	fs := s.FacetStore().CalculateFacets(b, in.Tolerance)
	return json.Marshal(facetSetReply(fs, in.IncludeTextureMap))
}

// bodyExistingFacets serves wire.MethodBodyExistingFacets — retrieval only.
func bodyExistingFacets(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.CalculateFacetsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	fs, ok := s.FacetStore().ExistingFacets(b, in.Tolerance)
	if !ok {
		return nil, fmt.Errorf("no facet set exists at tolerance %g (calculate it first; cached: %v)",
			in.Tolerance, s.FacetStore().FacetTolerances(b))
	}
	return json.Marshal(facetSetReply(fs, in.IncludeTextureMap))
}

func facetSetReply(fs *ops.BodyFacets, withUV bool) wire.FacetSetResult {
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
func bodyFacetTolerances(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return toleranceReply(s, raw, func(b *topo.Body) []float64 { return s.FacetStore().FacetTolerances(b) })
}

// bodyStrokeTolerances serves wire.MethodBodyStrokeTolerances.
func bodyStrokeTolerances(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return toleranceReply(s, raw, func(b *topo.Body) []float64 { return s.FacetStore().StrokeTolerances(b) })
}

func toleranceReply(s *app.Session, raw json.RawMessage, list func(*topo.Body) []float64) (json.RawMessage, error) {
	var in wire.BodyIndexArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.FacetTolerancesResult{Tolerances: list(b)})
}

// bodyCalculateStrokes serves wire.MethodBodyCalculateStrokes.
func bodyCalculateStrokes(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.CalculateStrokesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	return json.Marshal(strokeSetReply(s.FacetStore().CalculateStrokes(b, in.Tolerance).Polylines))
}

// bodyExistingStrokes serves wire.MethodBodyExistingStrokes — retrieval only.
func bodyExistingStrokes(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.CalculateStrokesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	ss, ok := s.FacetStore().ExistingStrokes(b, in.Tolerance)
	if !ok {
		return nil, fmt.Errorf("no stroke set exists at tolerance %g (calculate it first; cached: %v)",
			in.Tolerance, s.FacetStore().StrokeTolerances(b))
	}
	return json.Marshal(strokeSetReply(ss.Polylines))
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
func faceCalculateFacets(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	b, f, tol, err := resolveFaceArgs(s, raw)
	if err != nil {
		return nil, err
	}
	mesh, ok := s.FacetStore().FaceFacets(b, f, tol)
	if !ok {
		return nil, fmt.Errorf("face not found in the body's facet set")
	}
	out := wire.FacetSetResult{
		VertexCount: len(mesh.Positions), FacetCount: len(mesh.Indices) / 3,
		VertexIndices: mesh.Indices, IndexCountPerFace: []int{len(mesh.Indices)},
	}
	out.VertexCoordinates, out.NormalVectors = meshArrays(mesh)
	return json.Marshal(out)
}

// faceCalculateStrokes serves wire.MethodFaceCalculateStrokes.
func faceCalculateStrokes(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	_, f, tol, err := resolveFaceArgs(s, raw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(strokeSetReply(s.FacetStore().FaceStrokes(f, tol)))
}

// resolveFaceArgs decodes the face-addressed facet args.
func resolveFaceArgs(s *app.Session, raw json.RawMessage) (*topo.Body, *topo.Face, float64, error) {
	var in wire.FaceFacetsArgs
	if err := decode(raw, &in); err != nil {
		return nil, nil, 0, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, nil, 0, err
	}
	f, ok := b.FindFaceByKey([]byte(in.FaceKey))
	if !ok {
		return nil, nil, 0, fmt.Errorf("no face with key %q on body %d", in.FaceKey, in.BodyIndex)
	}
	return b, f, in.Tolerance, nil
}
