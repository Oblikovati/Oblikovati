// SPDX-License-Identifier: GPL-2.0-only

// Package tol carries the model-relative coincidence scale for the operation layer.
//
// The primitive itself lives in kernel/geom (ADR-0042 Phase 1) so the brep planar
// arrangement can share it. What lives here are the constructors that need topo, which
// geom cannot import — and they live BELOW kernel/ops because every operation family
// needs them: leaving them in kernel/ops made a tolerance lookup a reason to depend on
// the whole operation layer.
package tol

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Resolution is the model-relative coincidence scale (alias of geom.Resolution).
type Resolution = geom.Resolution

// ForSize / ForPoints re-export the geom constructors so a call site needs one import.
func ForSize(size float64) Resolution { return geom.ResolutionForSize(size) }

// ForPoints builds a Resolution from a point set's bounding box.
func ForPoints(pts []math.Point3) Resolution { return geom.ResolutionForPoints(pts) }

// ForBody builds a Resolution from a body's true bounding-box diagonal (curved edges
// included, via RangeBox) — the entry point for B-rep ops (boolean, CSG, sew, weld).
func ForBody(b *topo.Body) Resolution {
	if b == nil {
		return geom.ResolutionForSize(0) // floors to minModelSize
	}
	return geom.ResolutionForBox(b.RangeBox())
}

// ForBodies builds a Resolution from the LARGEST of several operands' bounding-box
// diagonals — the right scale for a multi-body op, whose tolerance must suit the bigger
// operand rather than a tiny tool.
func ForBodies(bodies ...*topo.Body) Resolution {
	size := 0.0
	for _, b := range bodies {
		if s := ForBody(b).Size(); s > size {
			size = s
		}
	}
	return geom.ResolutionForSize(size)
}

// ForTris builds a Resolution from CSG triangles' combined bounding-box diagonal — the entry
// point for the BSP-CSG / triangle-weld path, where the triangles themselves are the geometry
// being welded.
//
// Example: res := tol.ForTris(tris) // the weld scale for a soup with no body to measure
func ForTris(tris []mesh.Tri) Resolution {
	box := math.EmptyBox()
	for _, t := range tris {
		box = box.ExtendPoint(t.A).ExtendPoint(t.B).ExtendPoint(t.C)
	}
	return geom.ResolutionForBox(box)
}
