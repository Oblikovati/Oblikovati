// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestRevolveProfileSketchNamesProfile: the Revolution feature names its profile via a BoundaryPatch
// property; RevolveProfileSketch resolves it to a valid GraphSketches index. On a single-revolve
// fixture that is the sole profile sketch.
func TestRevolveProfileSketchNamesProfile(t *testing.T) {
	for _, f := range []string{"24_revolve_270.ipt", "16_revolve.ipt"} {
		d := openDoc(t, f)
		idx, ok := RevolveProfileSketch(d)
		if !ok {
			t.Errorf("%s: expected the revolve's profile sketch to resolve", f)
			continue
		}
		if g := GraphSketches(d); idx < 0 || idx >= len(g) {
			t.Errorf("%s: profile index %d out of range (%d sketches)", f, idx, len(g))
		}
	}
}
