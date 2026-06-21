// SPDX-License-Identifier: GPL-2.0-only

package benchgen

import (
	"math"

	obkmath "oblikovati.org/math"
)

// carBoundsCm is the synthetic vehicle envelope (length × width × height, cm) the bays
// are scattered through, so the assembly has real spatial spread for the frustum-cull
// and raycast/selection scenarios rather than collapsing to the origin.
var carBoundsCm = obkmath.V3(450, 180, 150)

// jitterCm offsets parts within a single bay so co-located placements do not perfectly
// overlap (kept small relative to the bay grid so bays stay spatially distinct).
const jitterCm = 6.0

// bayTransform places the i-th of n leaf bays in a grid cell spanning the car bounds —
// the spatial spread the frustum-cull and picking benchmarks need. Returned as a pure
// translation; orientation is left identity (the benchmark stresses count, not pose).
func bayTransform(i, n int) obkmath.Matrix4 {
	return obkmath.Translation4(gridPosition(i, n, carBoundsCm))
}

// partTransform offsets the k-th part within a bay by a small deterministic jitter so a
// bay's parts are distinguishable yet stay inside the bay's cell.
func partTransform(k int) obkmath.Matrix4 {
	off := gridPosition(k, 27, obkmath.V3(jitterCm*3, jitterCm*3, jitterCm*3))
	return obkmath.Translation4(off)
}

// gridPosition maps a linear index in [0,count) to the center of a cell of an
// approximately cubic grid scaled to bounds, so successive indices fan out across the
// volume deterministically. count<=1 returns the volume center.
func gridPosition(index, count int, bounds obkmath.Vector3) obkmath.Vector3 {
	if count <= 1 {
		return obkmath.V3(bounds.X/2, bounds.Y/2, bounds.Z/2)
	}
	side := int(math.Ceil(math.Cbrt(float64(count))))
	x := index % side
	y := (index / side) % side
	z := (index / (side * side)) % side
	cell := obkmath.V3(bounds.X/float64(side), bounds.Y/float64(side), bounds.Z/float64(side))
	return obkmath.V3((float64(x)+0.5)*cell.X, (float64(y)+0.5)*cell.Y, (float64(z)+0.5)*cell.Z)
}
