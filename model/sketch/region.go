// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// EntityOutline returns sample points (in sketch coordinates) approximating an entity's outline —
// the points a box / region selection projects to screen and tests against the rubber-band
// rectangle. A standalone point yields itself; every curve reuses the shared polyline samplers
// (entityPolyline), so the resolution matches the rendered overlay. Entities with no outline
// (text, images) return nil.
func EntityOutline(e Entity) []math.Point2 {
	if p, ok := e.(*Point); ok {
		return []math.Point2{p.Position()}
	}
	pts, _ := entityPolyline(e)
	return pts
}
