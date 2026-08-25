// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"testing"
)

func TestEmission(t *testing.T) {
	got := Emission(3, NewColor3(1, 0.5, 0.2))
	want := NewColor3(3, 1.5, 0.6)
	if math.Abs(got.R-want.R) > 1e-9 || math.Abs(got.G-want.G) > 1e-9 || math.Abs(got.B-want.B) > 1e-9 {
		t.Errorf("Emission(3, (1,0.5,0.2)) = %+v, want %+v", got, want)
	}
}

func TestEmissionZeroLuminanceIsBlack(t *testing.T) {
	got := Emission(0, NewColor3(1, 1, 1))
	if got != (Color3{}) {
		t.Errorf("Emission(0, white) = %+v, want black", got)
	}
}
