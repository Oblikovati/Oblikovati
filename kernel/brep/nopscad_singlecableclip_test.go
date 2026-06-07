// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"oblikovati/kernel/ops"
	"oblikovati/math"
	"testing"
)

func TestNopSingleCableClipCSG(t *testing.T) {
	base := prismBody(roundedRectPoints(1.6, 0.18, 0.08, 4), 0, 0.5, "cable-clip-foot")
	post := prismBody(roundedRectPoints(0.4, 0.9, 0.08, 4), 0, 0.5, "cable-clip-post")
	top := prismBody(regularPolygonPoints(math.P3(-0.55, 0.62, 0), 0.22, 24, 0), 0, 0.5, "cable-clip-loop")
	body, err := ops.ConvexHullOf("single-cable-clip-hull", base, post, top)
	if err != nil {
		t.Fatalf("ConvexHullOf(single cable clip): %v", err)
	}
	body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(-0.45, 0.45, 0), 0.18, 24, 0), -0.05, 0.55, "cable-channel"), "single cable channel")
	body = cutOrFatal(t, body, cylinderZAt(0.45, 0.45, -0.05, 0.55, 0.15, "cable-clip-screw"), "single cable clip screw")

	requireValidNopSolid(t, "single_cable_clip", body)
	if got := vol(body); got <= 0 {
		t.Errorf("single_cable_clip volume = %.6f, want positive clip", got)
	}
}
