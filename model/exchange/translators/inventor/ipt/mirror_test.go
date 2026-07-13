// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestDecodeMirror checks the mirror plane decodes to the offset work plane at x=3 with a
// +X normal (the plane the pocket is mirrored across).
func TestDecodeMirror(t *testing.T) {
	mir, ok := DecodeMirror(openDoc(t, "23_pocket_mirror.ipt"))
	if !ok {
		t.Fatal("no mirror decoded")
	}
	if absf(mir.Origin[0]-3) > 1e-6 || absf(mir.Origin[1]) > 1e-6 || absf(mir.Origin[2]) > 1e-6 {
		t.Errorf("origin = %v, want (3,0,0)", mir.Origin)
	}
	if mir.Normal != [3]float64{1, 0, 0} {
		t.Errorf("normal = %v, want (1,0,0)", mir.Normal)
	}
}

// TestDecodeMirrorAbsent confirms non-mirror parts (including patterns) report no mirror.
func TestDecodeMirrorAbsent(t *testing.T) {
	for _, file := range []string{"10_box.ipt", "21_pocket_rect.ipt", "22_pocket_circ.ipt"} {
		if _, ok := DecodeMirror(openDoc(t, file)); ok {
			t.Errorf("%s: decoded a mirror where there is none", file)
		}
	}
}
