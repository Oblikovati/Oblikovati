// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Lofted-flange CONVERGE (#2086, carved from #1966). By default a cornered profile carries its
// corners through the whole transition, so a square-to-round wall keeps a rounded corner tube all the
// way across. Inventor's Converge pinches each corner to a point at the far profile instead: the two
// panels meeting at the corner run to a sharp edge that converges away, removing the carried-through
// corner material.
//
// It is modelled by retargeting each corner's far end. A corner band point normally lofts to its own
// far-profile point; converged, it lofts to the midpoint of its two neighbours' far points, which
// lies on the straight edge between them — so the corner protrusion tapers from full at the near
// profile to flush (a point on the panel line) at the far one.

// convergeTurnLo / convergeTurnHi bound the turning angle that counts as a corner: sharp enough to be
// a real corner (a straight run turns ~0), but short of the ~180° reversal at an open profile's
// thickness caps, which are not corners to converge.
const (
	convergeTurnLo = stdmath.Pi / 3    // 60°
	convergeTurnHi = stdmath.Pi * 0.95 // 171°
)

// convergeCorners returns a copy of the far band whose corner points (found on the near band) are
// moved to the midpoint of their neighbours, so each corner tapers to a flush point at the far
// profile, and how many corners it pinched. The near band is unchanged, so the corners stay full
// where the wall meets it. A count of zero means the profiles have no corners to converge.
func convergeCorners(nearBand, farBand []math.Point3) ([]math.Point3, int) {
	out := append([]math.Point3(nil), farBand...)
	n, count := len(nearBand), 0
	for i := range nearBand {
		if isConvergingCorner(nearBand, i) {
			out[i] = farBand[(i-1+n)%n].Midpoint(farBand[(i+1)%n])
			count++
		}
	}
	return out, count
}

// isConvergingCorner reports whether band point i turns sharply enough to be a corner that converges.
func isConvergingCorner(band []math.Point3, i int) bool {
	n := len(band)
	in := band[(i-1+n)%n].VectorTo(band[i])
	out := band[i].VectorTo(band[(i+1)%n])
	turn := angleBetween(in, out)
	return turn > convergeTurnLo && turn < convergeTurnHi
}
