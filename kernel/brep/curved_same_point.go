// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// When two points on a section are ONE point. This outlives the analytic half-space pipeline ADR-0061
// deleted, where it was a shared helper of the general stage, and is used by the drill, the corner
// junction and the coaxial builders.

// samePoint reports whether two points coincide within the model-relative weld tolerance (#1399).
// Section/trace endpoints carry more accumulated round-off than an exact vertex weld, so they merge at
// the looser on-line tolerance res.Plane() (1e-7 at unit scale) rather than res.Weld(); deriving it from
// the body's extent keeps loop-closure and arc-chaining watertight on a km-scale part.
func samePoint(a, b math.Point3, res geom.Resolution) bool {
	return float64(a.DistanceTo(b)) < res.Plane()
}
