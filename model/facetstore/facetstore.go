// SPDX-License-Identifier: GPL-2.0-only

// Package facetstore caches facet and stroke sets per body and tolerance, so
// API clients can retrieve a previously calculated set without re-faceting —
// the reference API's GetExistingFacets / GetExistingStrokes semantics
// (M07-F03 remainder, Oblikovati/Oblikovati#293).
//
// Entries are keyed by *topo.Body identity: a recompute mints new bodies, so
// stale entries simply stop being reachable. DropBody evicts explicitly when a
// caller replaces bodies in place.
package facetstore

import (
	"sort"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// FacetStore is the per-session facet/stroke cache. Zero value is NOT ready —
// use NewFacetStore.
type FacetStore struct {
	facets  map[*topo.Body]map[float64]*tessellate.BodyFacets
	strokes map[*topo.Body]map[float64]*tessellate.BodyStrokes
}

// NewFacetStore returns an empty store.
//
// Example: st := facetstore.NewFacetStore(); fs := st.CalculateFacets(body, 0.01)
func NewFacetStore() *FacetStore {
	return &FacetStore{
		facets:  map[*topo.Body]map[float64]*tessellate.BodyFacets{},
		strokes: map[*topo.Body]map[float64]*tessellate.BodyStrokes{},
	}
}

// quality maps a chordal tolerance onto the kernel quality (angle bound stays
// the display default so round features keep their minimum segment count).
func quality(tolerance float64) ops.Quality {
	return ops.Quality{ChordTolerance: tolerance, AngleTolerance: ops.DefaultQuality().AngleTolerance}
}

// CalculateFacets facets the body at the tolerance and caches the set under
// that tolerance; an already-cached set returns without re-faceting.
func (st *FacetStore) CalculateFacets(b *topo.Body, tolerance float64) *tessellate.BodyFacets {
	if fs, ok := st.ExistingFacets(b, tolerance); ok {
		return fs
	}
	fs := tessellate.CalculateBodyFacets(b, quality(tolerance))
	perBody, ok := st.facets[b]
	if !ok {
		perBody = map[float64]*tessellate.BodyFacets{}
		st.facets[b] = perBody
	}
	perBody[tolerance] = fs
	return fs
}

// ExistingFacets retrieves the facet set previously calculated at exactly this
// tolerance, without faceting.
func (st *FacetStore) ExistingFacets(b *topo.Body, tolerance float64) (*tessellate.BodyFacets, bool) {
	fs, ok := st.facets[b][tolerance]
	return fs, ok
}

// FacetTolerances lists the tolerances facet sets exist at for the body,
// ascending.
func (st *FacetStore) FacetTolerances(b *topo.Body) []float64 {
	return sortedTolerances(keysOfFacets(st.facets[b]))
}

// CalculateStrokes samples the body's edges at the tolerance and caches the
// stroke set; an already-cached set returns without re-sampling.
func (st *FacetStore) CalculateStrokes(b *topo.Body, tolerance float64) *tessellate.BodyStrokes {
	if ss, ok := st.ExistingStrokes(b, tolerance); ok {
		return ss
	}
	ss := tessellate.CalculateBodyStrokes(b, quality(tolerance))
	perBody, ok := st.strokes[b]
	if !ok {
		perBody = map[float64]*tessellate.BodyStrokes{}
		st.strokes[b] = perBody
	}
	perBody[tolerance] = ss
	return ss
}

// ExistingStrokes retrieves the stroke set previously calculated at exactly
// this tolerance.
func (st *FacetStore) ExistingStrokes(b *topo.Body, tolerance float64) (*tessellate.BodyStrokes, bool) {
	ss, ok := st.strokes[b][tolerance]
	return ss, ok
}

// StrokeTolerances lists the tolerances stroke sets exist at, ascending.
func (st *FacetStore) StrokeTolerances(b *topo.Body) []float64 {
	return sortedTolerances(keysOfStrokes(st.strokes[b]))
}

// FaceFacets returns one face's mesh from the body facet set at the tolerance,
// calculating the body set if absent. The face-level reference calls ride the
// body cache: a body set already carries every face's mesh.
func (st *FacetStore) FaceFacets(b *topo.Body, f *topo.Face, tolerance float64) (*ops.Mesh, bool) {
	fs := st.CalculateFacets(b, tolerance)
	for i, face := range fs.Faces {
		if face == f {
			return fs.FaceMeshes[i], true
		}
	}
	return nil, false
}

// FaceStrokes samples one face's boundary edges at the tolerance (uncached —
// boundary sampling is cheap next to faceting).
func (st *FacetStore) FaceStrokes(f *topo.Face, tolerance float64) [][]math.Point3 {
	return tessellate.CalculateFaceStrokes(f, quality(tolerance)).Polylines
}

// DropBody evicts every cached set of a body (a caller replacing bodies in
// place rather than minting new ones).
func (st *FacetStore) DropBody(b *topo.Body) {
	delete(st.facets, b)
	delete(st.strokes, b)
}

func keysOfFacets(m map[float64]*tessellate.BodyFacets) []float64 {
	out := make([]float64, 0, len(m))
	for t := range m {
		out = append(out, t)
	}
	return out
}

func keysOfStrokes(m map[float64]*tessellate.BodyStrokes) []float64 {
	out := make([]float64, 0, len(m))
	for t := range m {
		out = append(out, t)
	}
	return out
}

func sortedTolerances(ts []float64) []float64 {
	sort.Float64s(ts)
	return ts
}
