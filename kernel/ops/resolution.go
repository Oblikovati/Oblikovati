// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// ADR-0042 Phase 1: the model-relative tolerance primitive lives in kernel/geom (so the
// brep planar-arrangement boolean can share it; see geom/resolution.go). ops re-exports
// it and adds the constructors that need topo/CSG types geom cannot see.

// Resolution is the model-relative coincidence scale (alias of geom.Resolution).
type Resolution = geom.Resolution

// ResolutionForSize / ResolutionForPoints re-export the geom constructors so ops call
// sites need only one import.
func ResolutionForSize(size float64) Resolution { return geom.ResolutionForSize(size) }
func ResolutionForPoints(pts []math.Point3) Resolution {
	return geom.ResolutionForPoints(pts)
}

// ResolutionForBody builds a Resolution from a body's true bounding-box diagonal (curved
// edges included, via RangeBox) — the entry point for B-rep ops (boolean, CSG, sew, weld).
func ResolutionForBody(b *topo.Body) Resolution {
	if b == nil {
		return geom.ResolutionForSize(0) // floors to minModelSize
	}
	return geom.ResolutionForBox(b.RangeBox())
}

// ResolutionForBodies builds a Resolution from the LARGEST of several operands'
// bounding-box diagonals — the right scale for a multi-body op (e.g. a boolean, whose
// tolerance must suit the bigger operand rather than a tiny tool).
func ResolutionForBodies(bodies ...*topo.Body) Resolution {
	size := 0.0
	for _, b := range bodies {
		if s := ResolutionForBody(b).Size(); s > size {
			size = s
		}
	}
	return geom.ResolutionForSize(size)
}

// resolutionForTris builds a Resolution from CSG triangles' combined bounding-box diagonal
// — the entry point for the BSP-CSG / triangle-weld path, where the triangles themselves
// are the geometry being welded.
func resolutionForTris(tris []tri) Resolution {
	box := math.EmptyBox()
	for _, t := range tris {
		box = box.ExtendPoint(t.a).ExtendPoint(t.b).ExtendPoint(t.c)
	}
	return geom.ResolutionForBox(box)
}
