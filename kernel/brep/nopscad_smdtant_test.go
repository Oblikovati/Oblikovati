// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"
)

func TestNopSmdTantCSG(t *testing.T) {
	t.Parallel()
	body := taperedBoxBody(0.72, 0.42, 0.64, 0.34, 0.02, 0.26, "smd-tant-body")
	for _, x := range []float64{-0.41, 0.41} {
		body = joinOrFatal(t, body, box(x-0.12, -0.17, 0.02, 0.24, 0.34, 0.05), "smd tant lead")
	}
	body = cutOrFatal(t, body, box(-0.17, -0.21, -0.01, 0.34, 0.42, 0.1), "smd tant lead gap")
	body = joinOrFatal(t, body, box(-0.31, -0.15, 0.26, 0.08, 0.3, 0.01), "smd tant stripe")

	requireValidNopSolid(t, "smd_tant", body)
	if got := vol(body); got <= 0 {
		t.Errorf("smd_tant volume = %.6f, want positive tantalum package", got)
	}
}
