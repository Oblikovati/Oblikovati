// SPDX-License-Identifier: GPL-2.0-only

package geom

// Internals the external geom_test package needs to assert a property of the real implementation
// rather than of a copy of it. Nothing here widens the package's exported surface.

// CurveSpanSamplesForTest is the step count [CurveSpanBox] walks a span with no closed-form extent
// in. A test that has to show the bare walk UNDER-bounds a curve must walk the same stations, and
// hard-coding the number there would let the two drift apart silently.
const CurveSpanSamplesForTest = curveSpanSamples
