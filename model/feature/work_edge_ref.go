// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"encoding/base64"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A work feature can be built on a B-rep EDGE (an axis along a straight edge, a point at an edge
// midpoint). Like a face, an edge is addressed by a single [WorkRef], re-bound to the running body
// each recompute so the datum follows the edge through upstream edits. Two reference forms resolve
// through the one edge resolver (ADR-0040):
//   - a lineage-key form "edge/<key>" — an interactive pick, re-bound by exact key via FindEdgeByKey,
//     the edge mirror of the face/vertex forms;
//   - a geometric form "edge-geom/<descriptor>" — authored from geometry by an external author (an
//     exporter) that has no Oblikovati lineage, re-bound by nearness via FindEdgeByGeometry.

const edgeRefPrefix = "edge/"

// edgeRef encodes a B-rep edge reference key as a WorkRef (the lineage-key form), mirroring faceRef.
func edgeRef(key []byte) WorkRef {
	return WorkRef(edgeRefPrefix + base64.RawURLEncoding.EncodeToString(key))
}

// EdgeRef is the exported edge-key encoder for the selection layer (a picked edge → a work-feature
// edge reference), the edge mirror of [FaceRef]/[VertexRef].
func EdgeRef(key []byte) WorkRef { return edgeRef(key) }

// EdgeRefKey decodes a lineage-key edge WorkRef back to its reference key, reporting ok=false for a
// ref that is not a lineage-key edge reference.
func EdgeRefKey(ref WorkRef) ([]byte, bool) { return decodeRefKey(string(ref), edgeRefPrefix) }

// edge resolves an edge WorkRef to its B-rep edge against the running body. It accepts both the
// lineage-key "edge/<key>" form and the ADR-0040 geometric "edge-geom/<descriptor>" form. It errors
// when ref is not an edge reference, when no body has been built yet, or when the reference no
// longer binds (the work feature then goes Sick) — #1840, #1842.
func (g *WorkGeometry) edge(ref WorkRef) (*topo.Edge, error) {
	if desc, ok := types.ParseGeometricEdgeRef(string(ref)); ok {
		return g.edgeByGeometry(desc, ref)
	}
	key, ok := decodeRefKey(string(ref), edgeRefPrefix)
	if !ok {
		return nil, fmt.Errorf("work geometry: %q is not an edge reference", ref)
	}
	if len(g.bodies) == 0 {
		return nil, fmt.Errorf("work geometry: no body yet to resolve edge %q", ref)
	}
	for _, b := range g.bodies {
		if e, ok := b.FindEdgeByKey(key); ok {
			return e, nil
		}
	}
	return nil, fmt.Errorf("work geometry: edge reference %q is lost", ref)
}

// edgeByGeometry binds an ADR-0040 geometric edge descriptor to an edge on the running body by
// nearness (FindEdgeByGeometry), returning it only when the binder finds it unambiguously.
func (g *WorkGeometry) edgeByGeometry(desc types.GeometricEdgeRef, ref WorkRef) (*topo.Edge, error) {
	if len(g.bodies) == 0 {
		return nil, fmt.Errorf("work geometry: no body yet to resolve edge %q", ref)
	}
	gr := topo.GeometricEdgeRef{
		Midpoint:  math.P3(desc.Midpoint[0], desc.Midpoint[1], desc.Midpoint[2]),
		Direction: math.V3(desc.Direction[0], desc.Direction[1], desc.Direction[2]),
	}
	for _, b := range g.bodies {
		if e, ok := b.FindEdgeByGeometry(gr, geomEdgeBindTol); ok {
			return e, nil
		}
	}
	return nil, fmt.Errorf("work geometry: geometric edge reference %q binds no edge", ref)
}
