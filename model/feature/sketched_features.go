// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// The remaining sketched features carry their full Definition (the triangle) and
// extent/operation surface. Revolve and coil generate real (faceted) solids of
// revolution via the shared swept-solid primitive; sweep/loft generation is below.
// Curved profile edges and exact analytic/NURBS surfaces are a later refinement —
// phase A approximates the swept surfaces as planar facets, a real watertight solid.

// resolveSingleProfile re-derives one closed region from a sketch, erroring (→ sick)
// when it is missing or open.
func resolveSingleProfile(skt *sketch.Sketch, index int, feat string) (*sketch.Profile, error) {
	all := skt.Profiles()
	if index < 0 || index >= all.Count() {
		return nil, fmt.Errorf("%s: profile %d not found (sketch has %d)", feat, index, all.Count())
	}
	p := all.Item(index)
	if !p.IsClosed() {
		return nil, fmt.Errorf("%s: profile is open (cannot form a solid)", feat)
	}
	return p, nil
}

// featOr returns name if set, else fallback — the lineage feature id for generated
// topology (a unique per-feature name keeps reference keys distinct).
func featOr(name, fallback string) string {
	if name == "" {
		return fallback
	}
	return name
}

// ensureCCW2 returns the polygon wound counter-clockwise (positive signed area) so the
// prism builder gives it outward normals.
func ensureCCW2(poly []math.Point2) []math.Point2 {
	var area float64
	for i := range poly {
		j := (i + 1) % len(poly)
		area += float64(poly[i].X*poly[j].Y - poly[j].X*poly[i].Y)
	}
	if area >= 0 {
		return poly
	}
	rev := make([]math.Point2, len(poly))
	for i, p := range poly {
		rev[len(poly)-1-i] = p
	}
	return rev
}
