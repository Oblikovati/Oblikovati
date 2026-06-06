// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	stdmath "math"

	"oblikovati/api/types"
	"oblikovati/kernel/ops"
)

// QualityFor maps an export resolution preset to the kernel facetting tolerances.
// Higher resolution ⇒ tighter chord + angle ⇒ more triangles for CURVED bodies (a
// planar body triangulates exactly regardless, so its count is resolution-independent).
// Medium equals ops.DefaultQuality (the display density). The zero value normalizes to
// medium.
//
// Example:
//
//	q := meshio.QualityFor(types.ResolutionHigh) // 0.0125 mm chord, 5° angle
func QualityFor(r types.MeshResolution) ops.Quality {
	switch r.Normalized() {
	case types.ResolutionLow:
		return ops.Quality{ChordTolerance: 0.20, AngleTolerance: deg(30)}
	case types.ResolutionHigh:
		return ops.Quality{ChordTolerance: 0.0125, AngleTolerance: deg(5)}
	default: // medium
		return ops.DefaultQuality()
	}
}

// deg converts degrees to radians (the unit ops.Quality.AngleTolerance expects).
func deg(d float64) float64 { return d * stdmath.Pi / 180 }
