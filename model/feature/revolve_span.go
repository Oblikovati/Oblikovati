// SPDX-License-Identifier: GPL-2.0-only

package feature

import stdmath "math"

// Which way a revolve sweeps (#2019).
//
// The definition has carried a second-direction Angle2 since #313, but nothing named the side the
// FIRST angle grows on, so the only reachable spans were "forward" and "forward plus backward".
// Inventor offers four, the same set an extrude offers, and a revolve reuses [ExtentDirection] for
// them so both features answer the same question the same way:
//
//	Default    (PositiveDir)  [0, +A]        forward from the profile
//	Flipped    (NegativeDir)  [-A, 0]        the same sweep, the other way round
//	Symmetric  (SymmetricDir) [-A/2, +A/2]   A total, half each way
//	Asymmetric (Angle2 set)   [-B, +A]       a separate angle each way
//
// Flipped is not merely cosmetic on a partial revolve: it decides which existing material a Cut or
// Join meets, so [0,90°] and [-90°,0] are different features, and before this the second could not
// be asked for at all.

// revolveSpan resolves the total swept angle and the angular offset it starts at, from the
// definition's angles and direction. A span reaching a full turn collapses to the full revolution
// — the solid is then rotationally complete, so neither the start nor the direction is observable.
func revolveSpan(def *RevolveDefinition) (total, start float64) {
	a1, a2 := callOrZero(def.Angle), callOrZero(def.Angle2)
	if a1 <= 0 {
		return 0, 0 // 0 is the definition's "full revolution" marker, with or without a second angle
	}
	if a2 > 0 {
		return closedSpan(a1+a2, -a2) // asymmetric: a separate angle each way
	}
	switch def.Direction {
	case NegativeDir:
		return closedSpan(a1, -a1)
	case SymmetricDir:
		return closedSpan(a1, -a1/2)
	default:
		return closedSpan(a1, 0)
	}
}

// closedSpan collapses a span that closes on itself to the full revolution, where the start offset
// stops meaning anything.
func closedSpan(total, start float64) (float64, float64) {
	if fullRevolution(total) {
		return 0, 0
	}
	return total, start
}

// fullRevolution reports whether an angle is a complete turn (0 ⇒ full, like revolveSections).
func fullRevolution(angle float64) bool { return angle <= 0 || angle >= 2*stdmath.Pi-1e-9 }
