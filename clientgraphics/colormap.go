// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

// At maps a scalar value to a color by piecewise-linear interpolation between the mapper's
// ascending Values stops, clamping to the end colors outside the range (Inventor's
// GraphicsColorMapper). The mapper is assumed validated (len(Colors)==len(Values)>0) by
// DecodeGroup, so the heatmap path never sees a malformed legend.
func (m *ColorMapper) At(value float64) [4]float32 {
	if value <= m.Values[0] {
		return m.Colors[0]
	}
	last := len(m.Values) - 1
	if value >= m.Values[last] {
		return m.Colors[last]
	}
	hi := upperStop(m.Values, value)
	return lerpColor(m.Colors[hi-1], m.Colors[hi], stopFraction(m.Values[hi-1], m.Values[hi], value))
}

// upperStop returns the index of the first stop strictly greater than value (value is
// known to be inside the open range, so this is in [1, last]).
func upperStop(values []float64, value float64) int {
	for i := 1; i < len(values); i++ {
		if values[i] > value {
			return i
		}
	}
	return len(values) - 1
}

// stopFraction returns value's position in [lo, hi] as 0..1 (0 if the interval is empty).
func stopFraction(lo, hi, value float64) float32 {
	span := hi - lo
	if span == 0 {
		return 0
	}
	return float32((value - lo) / span)
}

// lerpColor linearly interpolates two rgba colors by t in 0..1.
func lerpColor(a, b [4]float32, t float32) [4]float32 {
	return [4]float32{
		a[0] + (b[0]-a[0])*t,
		a[1] + (b[1]-a[1])*t,
		a[2] + (b[2]-a[2])*t,
		a[3] + (b[3]-a[3])*t,
	}
}
