// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/math"
)

func TestNopPiCutoutCSG(t *testing.T) {
	left := prismBody(regularPolygonPoints(math.P3(-0.35, 0, 0), 0.18, 32, 0), 0, 0.35, "pi-cutout-left")
	right := prismBody(regularPolygonPoints(math.P3(0.35, 0, 0), 0.18, 32, 0), 0, 0.35, "pi-cutout-right")
	hull, err := ops.ConvexHullOf("pi-cutout-base-hull", left, right)
	if err != nil {
		t.Fatalf("ConvexHullOf(pi holes): %v", err)
	}
	body := joinOrFatal(t, hull, box(-0.45, -0.55, 0, 0.9, 0.12, 0.9), "pi lower stem")
	body = joinOrFatal(t, body, box(-0.45, 0.43, 0, 0.9, 0.12, 0.9), "pi upper stem")

	requireValidNopSolid(t, "pi_cutout", body)
	if got := vol(body); got <= vol(hull) {
		t.Errorf("pi_cutout volume = %.6f, want larger than hole hull %.6f", got, vol(hull))
	}
}
