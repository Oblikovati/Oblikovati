// SPDX-License-Identifier: GPL-2.0-only

package viewport

import (
	"math"
	"testing"
)

// mul4x4 multiplies two column-major 4×4 matrices (for the inverse round-trip check).
func mul4x4(a, b [16]float32) [16]float32 {
	var out [16]float32
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			var s float32
			for k := 0; k < 4; k++ {
				s += a[k*4+row] * b[col*4+k]
			}
			out[col*4+row] = s
		}
	}
	return out
}

// TestInvert4x4RoundTrip checks M·M⁻¹ ≈ I for a non-trivial perspective-like matrix, the
// property the skybox relies on (ADR-0026 §5).
func TestInvert4x4RoundTrip(t *testing.T) {
	m := [16]float32{
		2, 0, 0, 0,
		0, 1.5, 0, 0,
		1, 0.5, -1.002, -1,
		0, 0, -0.2, 0,
	}
	inv, ok := Invert4x4(m)
	if !ok {
		t.Fatal("matrix reported singular")
	}
	prod := mul4x4(m, inv)
	for i := 0; i < 16; i++ {
		want := float32(0)
		if i%5 == 0 {
			want = 1
		}
		if math.Abs(float64(prod[i]-want)) > 1e-4 {
			t.Errorf("(M·M⁻¹)[%d] = %g, want %g", i, prod[i], want)
		}
	}
}

// TestInvert4x4Singular checks a singular matrix (zero column) is reported, not silently
// returning garbage.
func TestInvert4x4Singular(t *testing.T) {
	if _, ok := Invert4x4([16]float32{}); ok {
		t.Error("zero matrix should be singular")
	}
}
