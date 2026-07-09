// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"math"
	"testing"
)

// Shell direction (inside/outside/both) — Inventor's ShellDirectionEnum, #1864. The kernel op
// (ops.ShellDirected) carries the analytic-volume proofs; these check the wire reaches it with the
// right direction, on a 4×3×1 cm box (12 cm³) shelled 0.2 cm with the top face removed.

// TestShellDirectionVolumes: inside keeps the outer size (4.512 cm³), outside grows the wall
// outward (5.952 cm³), both centres it (5.208 cm³) — three distinct, analytically-known solids.
func TestShellDirectionVolumes(t *testing.T) {
	for _, tc := range []struct {
		dir  string
		want float64
	}{
		{"inside", 12 - 3.6*2.6*0.8},        // 4.512
		{"outside", 4.4*3.4*1.2 - 12},       // 5.952
		{"both", 4.2*3.2*1.1 - 3.8*2.8*0.9}, // 5.208
		{"", 12 - 3.6*2.6*0.8},              // empty ⇒ inside default
	} {
		t.Run(tc.dir, func(t *testing.T) {
			got := shelledBoxVolume(t, tc.dir)
			if math.Abs(got-tc.want) > 1e-6 {
				t.Errorf("shell direction %q volume = %g, want %g", tc.dir, got, tc.want)
			}
		})
	}
}

// TestShellDirectionUnknown: an unknown direction is a clean error.
func TestShellDirectionUnknown(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	top, _ := boxTopFace(t, s)
	if _, err := applyMap(t, s, "shell", map[string]any{
		"faceRefs": []string{top}, "thickness": "2 mm", "direction": "sideways",
	}); err == nil {
		t.Error("unknown shell direction should error")
	}
}

// shelledBoxVolume seeds a 4×3×1 box, shells it 2 mm with the given direction (top face removed),
// and returns the resulting body volume.
func shelledBoxVolume(t *testing.T, dir string) float64 {
	t.Helper()
	s, _, _ := extrudedSolid(t)
	top, _ := boxTopFace(t, s)
	args := map[string]any{"faceRefs": []string{top}, "thickness": "2 mm"}
	if dir != "" {
		args["direction"] = dir
	}
	if _, err := applyMap(t, s, "shell", args); err != nil {
		t.Fatalf("shell %q: %v", dir, err)
	}
	return bodyVolume(t, s)
}
