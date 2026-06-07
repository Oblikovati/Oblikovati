// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/math"
)

// TestNopSbrRailCSG pins the SBR16S source geometry: the translated concave rail
// base section, its screw clearance cuts, and the supported rod drawn by sbr_rail().
func TestNopSbrRailCSG(t *testing.T) {
	base := prismBody(sbrRailSectionPoints(), -1.75, 1.75, "sbr-rail-base")
	requireValidNopSolid(t, "sbr_rail base", base)
	previousVolume := vol(base)
	for _, hole := range []struct {
		x, z   float64
		y0, y1 float64
	}{
		{-1.5, 0, 1.95, 2.55},
		{1.5, 0, 1.95, 2.55},
		{0, 0, 1.11, 2.91},
	} {
		tool := prismBodyAlongY(regularPolygonXZ(hole.x, hole.z, 0.265, 32, 0), hole.y0, hole.y1, "sbr-screw-clearance")
		requireValidNopSolid(t, "sbr_rail screw clearance", tool)
		var err error
		base, err = ops.Boolean(ops.Cut, base, tool)
		if err != nil {
			t.Fatalf("Boolean(Cut SBR screw hole at x=%g z=%g): %v", hole.x, hole.z, err)
		}
		requireValidNopSolid(t, "sbr_rail cut base", base)
		if got := vol(base); got <= 0 || got >= previousVolume {
			t.Fatalf("SBR screw cut at x=%g z=%g volume = %.6f, want between 0 and %.6f", hole.x, hole.z, got, previousVolume)
		}
		previousVolume = vol(base)
	}
	sourceBaseVolume := nopPolygonArea(sbrRailSectionPoints()) * 3.5
	cutBaseVolume := vol(base)
	if cutBaseVolume <= 0 || cutBaseVolume >= sourceBaseVolume {
		t.Fatalf("sbr_rail cut base volume = %.6f, want less than uncut %.6f", cutBaseVolume, sourceBaseVolume)
	}

	rod := prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.8, 64, 0), -2, 2, "sbr-rod")
	body, err := ops.Boolean(ops.Join, base, rod)
	if err != nil {
		t.Fatalf("Boolean(Join SBR base+rod): %v", err)
	}

	requireValidNopSolid(t, "sbr_rail", body)
	rodVolume := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.8, 64, 0)) * 4.0
	if got := vol(body); got <= cutBaseVolume || got >= cutBaseVolume+rodVolume {
		t.Errorf("sbr_rail volume = %.6f, want between %.6f and %.6f", got, cutBaseVolume, cutBaseVolume+rodVolume)
	}
}
