// SPDX-License-Identifier: GPL-2.0-only

package facetstore

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
)

func storeBox(t *testing.T) *topo.Body {
	t.Helper()
	return subd.ToBody(subd.Box(2, 1, 1), "box")
}

// Calculate caches: the second call at the same tolerance returns the SAME set
// (pointer identity — no re-faceting), and Existing sees it.
func TestCalculateFacetsCachesPerTolerance(t *testing.T) {
	st := NewFacetStore()
	b := storeBox(t)
	first := st.CalculateFacets(b, 0.01)
	if again := st.CalculateFacets(b, 0.01); again != first {
		t.Error("same tolerance must return the cached facet set, not re-facet")
	}
	if _, ok := st.ExistingFacets(b, 0.01); !ok {
		t.Error("ExistingFacets should find the calculated set")
	}
	if _, ok := st.ExistingFacets(b, 0.05); ok {
		t.Error("ExistingFacets must not invent a set for an uncalculated tolerance")
	}
}

// Tolerances list every cached set ascending; strokes track independently.
func TestToleranceListsAscending(t *testing.T) {
	st := NewFacetStore()
	b := storeBox(t)
	st.CalculateFacets(b, 0.05)
	st.CalculateFacets(b, 0.01)
	got := st.FacetTolerances(b)
	if len(got) != 2 || got[0] != 0.01 || got[1] != 0.05 {
		t.Errorf("FacetTolerances = %v, want [0.01 0.05]", got)
	}
	if n := len(st.StrokeTolerances(b)); n != 0 {
		t.Errorf("stroke tolerances = %d entries, want 0 (none calculated)", n)
	}
}

// The facet set carries per-face index counts summing to the merged mesh, and
// texture coordinates pair off 2-per-vertex.
func TestFacetSetShape(t *testing.T) {
	st := NewFacetStore()
	b := storeBox(t)
	fs := st.CalculateFacets(b, 0.01)
	sum := 0
	for _, c := range fs.IndexCountPerFace {
		sum += c
	}
	if sum != len(fs.Mesh.Indices) {
		t.Errorf("per-face index counts sum %d != merged indices %d", sum, len(fs.Mesh.Indices))
	}
	if uv := fs.TextureCoordinates(); len(uv) != 2*len(fs.Mesh.Positions) {
		t.Errorf("texture coords = %d floats, want %d (2 per vertex)", len(uv), 2*len(fs.Mesh.Positions))
	}
}

// Strokes cache like facets and cover every edge.
func TestCalculateStrokesCaches(t *testing.T) {
	st := NewFacetStore()
	b := storeBox(t)
	ss := st.CalculateStrokes(b, 0.01)
	if len(ss.Polylines) != len(b.Edges()) {
		t.Errorf("strokes cover %d edges, want %d", len(ss.Polylines), len(b.Edges()))
	}
	if again := st.CalculateStrokes(b, 0.01); again != ss {
		t.Error("same tolerance must return the cached stroke set")
	}
}

// Face-level retrieval rides the body cache; DropBody evicts everything.
func TestFaceFacetsAndDrop(t *testing.T) {
	st := NewFacetStore()
	b := storeBox(t)
	f := b.Faces()[0]
	mesh, ok := st.FaceFacets(b, f, 0.01)
	if !ok || len(mesh.Indices) == 0 {
		t.Fatalf("FaceFacets = (%v, %v), want a non-empty mesh", mesh, ok)
	}
	if foreign := faceOfAnotherBody(t); func() bool { _, ok := st.FaceFacets(b, foreign, 0.01); return ok }() {
		t.Error("a face of another body must not resolve")
	}
	st.DropBody(b)
	if _, ok := st.ExistingFacets(b, 0.01); ok {
		t.Error("DropBody should evict the facet sets")
	}
}

func faceOfAnotherBody(t *testing.T) *topo.Face {
	t.Helper()
	return storeBox(t).Faces()[0]
}

// FaceStrokes samples the boundary without touching the cache.
func TestFaceStrokesUncached(t *testing.T) {
	st := NewFacetStore()
	b := storeBox(t)
	pl := st.FaceStrokes(b.Faces()[0], 0.01)
	if len(pl) != len(b.Faces()[0].Edges()) {
		t.Errorf("face strokes = %d polylines, want %d", len(pl), len(b.Faces()[0].Edges()))
	}
	if n := len(st.StrokeTolerances(b)); n != 0 {
		t.Errorf("face strokes must not populate the body cache (got %d tolerances)", n)
	}
}

// The default-quality angle bound carries into facet quality, so a coarse
// chord tolerance still rounds small curves (the Quality contract).
func TestQualityKeepsAngleBound(t *testing.T) {
	q := quality(0.5)
	if q.AngleTolerance != ops.DefaultQuality().AngleTolerance {
		t.Errorf("angle tolerance = %g, want display default %g", q.AngleTolerance, ops.DefaultQuality().AngleTolerance)
	}
}
