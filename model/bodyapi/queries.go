// SPDX-License-Identifier: GPL-2.0-only

package bodyapi

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

var _ contract.BodyQueries = (*BodyQueriesAdapter)(nil)

// BodyQueriesAdapter exposes one body's query surface (M07-F07, #630).
type BodyQueriesAdapter struct {
	body *topo.Body
	q    ops.Quality
}

// NewBodyQueries wraps a body at the given quality.
func NewBodyQueries(b *topo.Body, q ops.Quality) *BodyQueriesAdapter {
	return &BodyQueriesAdapter{body: b, q: q}
}

// IsPointInside classifies a point against the body's material.
func (a *BodyQueriesAdapter) IsPointInside(x, y, z float64) types.Containment {
	c := query.BodyContainment(a.body, math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)), a.q, onTolDefault)
	return containmentOf(c)
}

// ConvexEdgeCount and ConcaveEdgeCount report the dihedral classification.
func (a *BodyQueriesAdapter) ConvexEdgeCount() int {
	return len(blend.BodyEdgeConvexity(a.body)[blend.EdgeConvex])
}

func (a *BodyQueriesAdapter) ConcaveEdgeCount() int {
	return len(blend.BodyEdgeConvexity(a.body)[blend.EdgeConcave])
}

// IsEntityValid checks the body at the given level (1 topology, 2 + the
// self-intersection scan).
func (a *BodyQueriesAdapter) IsEntityValid(checkLevel int) bool {
	ok, _ := ops.ValidateBodyEntities(a.body, entityCheckLevel(checkLevel), a.q)
	return ok
}

// entityCheckLevel maps the wire integer onto the kernel level (0 → topology).
func entityCheckLevel(level int) ops.EntityCheckLevel {
	if level >= int(ops.CheckGeometry) {
		return ops.CheckGeometry
	}
	return ops.CheckTopology
}

// BindTransientKey resolves a session id to its entity kind and persistent key.
func (a *BodyQueriesAdapter) BindTransientKey(key uint64) (string, []byte, bool) {
	ref, ok := a.body.BindTransientKey(key)
	if !ok {
		return "", nil, false
	}
	return ref.Kind.String(), transientRefKey(ref), true
}

// transientRefKey returns the bound entity's persistent reference key.
func transientRefKey(ref topo.TransientRef) []byte {
	switch ref.Kind {
	case topo.KindVertex:
		return ref.Vertex.ReferenceKey()
	case topo.KindEdge:
		return ref.Edge.ReferenceKey()
	case topo.KindFace:
		return ref.Face.ReferenceKey()
	case topo.KindShell:
		return ref.Shell.ReferenceKey()
	default:
		return ref.Wire.ReferenceKey()
	}
}
