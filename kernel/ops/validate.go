// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/topo"
)

// ValidationReport is the outcome of [Validate]: whether the body is a valid
// manifold, plus precise issues for any offending edges.
type ValidationReport struct {
	Valid         bool
	Manifold      bool
	Closed        bool
	OrientationOK bool
	Issues        []string
}

// Validate checks a body's topology: every edge of a manifold solid must be used by
// exactly two faces with opposite orientation, and a solid must be closed (no
// boundary edges). It reports each offending edge precisely (PBI-084) — a surface
// body is allowed to be open.
func Validate(b *topo.Body) ValidationReport {
	r := ValidationReport{Manifold: true, Closed: true, OrientationOK: true}
	for _, e := range b.Edges() {
		switch uses := e.Uses(); {
		case len(uses) < 2:
			r.Closed = false
			if b.IsSolid() {
				r.Issues = append(r.Issues, fmt.Sprintf("boundary (open) edge %d on a solid", e.ID()))
			}
		case len(uses) > 2:
			r.Manifold = false
			r.Issues = append(r.Issues, fmt.Sprintf("non-manifold edge %d used by %d faces", e.ID(), len(uses)))
		default:
			if uses[0].Reversed() == uses[1].Reversed() {
				r.OrientationOK = false
				r.Issues = append(r.Issues, fmt.Sprintf("inconsistent orientation at edge %d", e.ID()))
			}
		}
	}
	r.Valid = r.Manifold && r.OrientationOK && (!b.IsSolid() || r.Closed)
	return r
}

// BoundaryEdges returns the open (boundary) edges of a body — those used by fewer
// than two faces. An empty result means the body is closed.
func BoundaryEdges(b *topo.Body) []*topo.Edge {
	var open []*topo.Edge
	for _, e := range b.Edges() {
		if len(e.Uses()) < 2 {
			open = append(open, e)
		}
	}
	return open
}
