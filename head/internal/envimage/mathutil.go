// SPDX-License-Identifier: GPL-2.0-only

package envimage

import "math"

// clamp01 clamps v to [0,1].
func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// mix linearly interpolates a→b by t∈[0,1] (t is clamped).
func mix(a, b, t float32) float32 { return a + (b-a)*clamp01(t) }

// smoothstep is the Hermite 0→1 ramp between edges e0 and e1.
func smoothstep(e0, e1, x float32) float32 {
	t := clamp01((x - e0) / (e1 - e0))
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
