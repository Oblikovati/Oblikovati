// SPDX-License-Identifier: GPL-2.0-only

package envimage

import (
	"math"

	gmath "oblikovati.org/math"
)

// mix linearly interpolates a→b by t∈[0,1] (t is clamped).
func mix(a, b, t float32) float32 { return a + (b-a)*gmath.Clamp(t, 0, 1) }

// smoothstep is the Hermite 0→1 ramp between edges e0 and e1.
func smoothstep(e0, e1, x float32) float32 {
	t := gmath.Clamp((x-e0)/(e1-e0), 0, 1)
	return t * t * (3 - 2*t)
}

// absf is the float32 absolute value.
func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// expf is math.Exp at float32.
func expf(v float32) float32 { return float32(math.Exp(float64(v))) }
