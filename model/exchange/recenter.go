// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"math"
	"sort"

	"oblikovati.org/kernel/exchange/drawing"
)

// recenterThreshold is the distance from the origin (in database units, cm) beyond which an
// imported drawing is shifted back toward the origin. Georeferenced/survey DWGs carry
// coordinates in the tens of millions (state-plane, UTM); the viewport uploads positions to
// the GPU as float32, whose ~7 significant digits give only ~6-unit precision at 5.4e7, so
// the drawing renders as garbage or nothing. 1e5 cm = 1 km: no mechanical part legitimately
// sits a kilometre from its own origin, so ordinary near-origin imports (and the DWG/DXF
// export round-trip) are never recentered, while survey-scale data always is.
const recenterThreshold = 1e5

// recenterFarFromOrigin shifts a drawing whose geometry sits far from the origin back toward
// it, returning the (possibly) shifted entities, the applied offset, and whether a shift was
// made. The shift is the per-axis MEDIAN of the entity anchors, which is robust: a handful of
// far-flung off-sheet entities cannot drag the centre off the main drawing (unlike a mean or
// a bounding-box centre, which an outlier on the opposite side pulls toward zero). Drawings
// already near the origin are returned unchanged.
//
// The geometry is moved into the part frame rather than kept at its survey coordinates — the
// import convention for far-from-origin data — so the absolute position is not preserved on a
// subsequent export; that is acceptable for the precision it buys (see recenterThreshold).
func recenterFarFromOrigin(entities []drawing.Entity) ([]drawing.Entity, [3]float64, bool) {
	c, ok := robustCenter(entities)
	if !ok || math.Hypot(math.Hypot(c[0], c[1]), c[2]) <= recenterThreshold {
		return entities, [3]float64{}, false
	}
	return drawing.TranslateEntities(entities, -c[0], -c[1], -c[2]), c, true
}

// robustCenter returns the per-axis median of the entities' anchor points, or ok=false when
// no entity carries an anchor. The median (not the mean) keeps the centre on the bulk of the
// geometry even when a small fraction of entities sit far away.
func robustCenter(entities []drawing.Entity) ([3]float64, bool) {
	xs := make([]float64, 0, len(entities))
	ys := make([]float64, 0, len(entities))
	zs := make([]float64, 0, len(entities))
	for _, e := range entities {
		if p, ok := drawing.EntityAnchor(e); ok {
			xs, ys, zs = append(xs, p[0]), append(ys, p[1]), append(zs, p[2])
		}
	}
	if len(xs) == 0 {
		return [3]float64{}, false
	}
	return [3]float64{median(xs), median(ys), median(zs)}, true
}

// median returns the middle value of v (averaging the two middle values for an even count).
// It sorts v in place; callers pass a scratch slice they own.
func median(v []float64) float64 {
	sort.Float64s(v)
	n := len(v)
	if n%2 == 1 {
		return v[n/2]
	}
	return (v[n/2-1] + v[n/2]) / 2
}

// recenterWarning describes a recenter so the import surfaces it to the user (the absolute
// position changed). offset is the original centre that was moved to the origin.
func recenterWarning(offset [3]float64) string {
	return fmt.Sprintf("import: recentered drawing by (%.1f, %.1f, %.1f) to keep precision (geometry was far from the origin)",
		offset[0], offset[1], offset[2])
}
