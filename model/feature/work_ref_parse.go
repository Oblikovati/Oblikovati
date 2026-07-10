// SPDX-License-Identifier: GPL-2.0-only

package feature

import "strings"

// workFeatureRefPrefixes are the reference-string prefixes that name a work feature — an origin
// datum, a user plane/axis/point, a UCS, a model vertex, or a B-rep edge (a lineage-key "edge/…" or
// an ADR-0040 geometric "edge-geom/…" descriptor) — rather than a raw B-rep face key. The two edge
// forms are kept verbatim so the edge resolver (work_edge_ref.go) can bind them; without this a
// wire edge ref was mis-wrapped as a face key and every edge-based datum went unhealthy (#1840, #1842).
var workFeatureRefPrefixes = []string{"origin/", "plane/", "axis/", "point/", "ucs/", "vertex/", "edge/", "edge-geom/"}

// ParseWorkRef classifies a raw reference string from the wire: a work-feature reference is kept
// verbatim; any other string is treated as a raw B-rep face key and wrapped as a face ref. It is
// the single place the wire reference vocabulary is interpreted, shared by the add-in router and
// the operation registry (so the two never drift).
func ParseWorkRef(r string) WorkRef {
	for _, p := range workFeatureRefPrefixes {
		if strings.HasPrefix(r, p) {
			return WorkRef(r)
		}
	}
	return faceRef([]byte(r))
}

// PlaneTargetFromRef resolves a raw plane reference — a work plane ("plane/N"), an origin plane
// ("origin/plane/xy"), or a planar B-rep face key — to a *WorkPlane usable as a feature target,
// e.g. an extrude's to-face termination. A work/origin plane resolves to its datum; a planar face
// key yields a transient fixed plane ([NewFixedWorkPlane]). An unresolvable reference is an error.
func (g *WorkGeometry) PlaneTargetFromRef(ref string) (*WorkPlane, error) {
	wr := ParseWorkRef(ref)
	if wp, err := g.WorkPlaneByRef(wr); err == nil {
		return wp, nil
	}
	pl, err := g.plane(wr)
	if err != nil {
		return nil, err
	}
	return NewFixedWorkPlane(pl), nil
}
