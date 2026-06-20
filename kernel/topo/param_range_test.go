// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestSurfaceEvaluatorParamRange: a bounded surface reports a finite domain, and an
// unbounded one (a plane) is clamped to a finite search window rather than returning ±Inf.
func TestSurfaceEvaluatorParamRange(t *testing.T) {
	sphere, _ := geom.NewSphere(math.P3(0, 0, 0), 3)
	uLo, uHi, vLo, vHi := NewSurfaceEvaluator(sphere).ParamRange()
	if uHi <= uLo || vHi <= vLo {
		t.Errorf("sphere param range invalid: u[%g,%g] v[%g,%g]", uLo, uHi, vLo, vHi)
	}

	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	puLo, puHi, pvLo, pvHi := NewSurfaceEvaluator(plane).ParamRange()
	for _, b := range []float64{puLo, puHi, pvLo, pvHi} {
		if stdmath.IsInf(b, 0) {
			t.Errorf("plane param range must be clamped finite, got u[%g,%g] v[%g,%g]", puLo, puHi, pvLo, pvHi)
		}
	}
}
