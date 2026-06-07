// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"testing"
)

// TestNopSmdResistorCSG pins smd_resistor as the union of its ceramic body and
// two metal end caps; printed text is intentionally not material geometry.
func TestNopSmdResistorCSG(t *testing.T) {
	body := box(-0.28, -0.125, 0, 0.56, 0.25, 0.12)
	for _, cap := range []*topo.Body{box(-0.50, -0.125, 0, 0.22, 0.25, 0.12), box(0.28, -0.125, 0, 0.22, 0.25, 0.12)} {
		var err error
		body, err = ops.Boolean(ops.Join, body, cap)
		if err != nil {
			t.Fatalf("Boolean(Join cap): %v", err)
		}
	}

	requireValidNopSolid(t, "smd_resistor", body)
	want := 1.0 * 0.25 * 0.12
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("smd_resistor volume = %.6f, want %.6f", got, want)
	}
}
