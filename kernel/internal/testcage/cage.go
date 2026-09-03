// SPDX-License-Identifier: GPL-2.0-only

// Package testcage builds a FACETED copy of a body for tests that pin a planar-only path — the
// state a triangle-CSG fallback leaves a part in. It is test support only: the production
// operation that faceted an analytic operand to satisfy the planar boolean (ops.Facet) is gone
// (ADR-0060, Oblikovati/Oblikovati#3459), and nothing in the kernel facets a body behind a caller's
// back any more.
package testcage

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/boolean"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// Body returns the body's tessellation welded back into a planar solid, coplanar triangles unified
// into maximal faces. nil when the tessellation does not close.
//
// Example:
//
//	plate := testcage.Body(drilledPlate, "facet")
func Body(b *topo.Body, feat string) *topo.Body {
	m, _ := tessellate.TessellateBody(b, tessellate.DefaultQuality())
	facets := make([][]int, 0, len(m.Indices)/3)
	for i := 0; i+2 < len(m.Indices); i += 3 {
		facets = append(facets, []int{m.Indices[i], m.Indices[i+1], m.Indices[i+2]})
	}
	cage := boolean.MeshToBRep(m.Positions, facets, feat)
	if cage == nil {
		return nil
	}
	return brep.UnifyCoplanarFaces(cage, feat)
}
