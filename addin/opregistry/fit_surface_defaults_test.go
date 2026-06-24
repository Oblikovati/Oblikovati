// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import "testing"

// TestFitSurfaceDefaults: omitted (zero) degree/spans fall back to the bicubic 6×6 defaults, while
// explicit values pass through unchanged.
func TestFitSurfaceDefaults(t *testing.T) {
	d, nu, nv := fitSurfaceDefaults(fitSurfaceArgs{})
	if d != 3 || nu != 6 || nv != 6 {
		t.Errorf("zero args = (%d,%d,%d), want the (3,6,6) defaults", d, nu, nv)
	}
	d, nu, nv = fitSurfaceDefaults(fitSurfaceArgs{Degree: 5, NU: 8, NV: 7})
	if d != 5 || nu != 8 || nv != 7 {
		t.Errorf("explicit args = (%d,%d,%d), want (5,8,7) passed through", d, nu, nv)
	}
}
