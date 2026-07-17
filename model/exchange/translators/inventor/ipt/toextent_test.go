// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	stdmath "math"
	"testing"
)

// TestToExtentTargetDecodes pins the "To <face>" target (#30). real_multipoint_disk's ex5 terminates
// at a face, and its stated length (0.2) is a stale leftover that must NOT be built as a depth.
//
// The target was found by DIFFERENTIAL, because the GPL reference does not implement this extent at
// all (Create_FxExtrude_New special-cases only extend==5/ALL and otherwise feeds dimLength1 straight
// to the extrusion): across BigChunkyPlate's 31 Dimension and 11 To features, exactly two properties
// separate them — 0x0d (a boolean flag) and 0x12, whose type the reference comments only as "???".
func TestToExtentTargetDecodes(t *testing.T) {
	ex := DecodeExtrudes(openDoc(t, "real_multipoint_disk.ipt"))
	var to []Extrude
	for _, e := range ex {
		if e.ToPlaneOK {
			to = append(to, e)
		}
	}
	if len(to) != 1 {
		t.Fatalf("decoded %d To-extent extrudes, want 1 (of %d)", len(to), len(ex))
	}
	e := to[0]
	if got, want := e.ToPlane.Origin, [3]float64{0, 0, 0.3}; !samePoint3(got, want) {
		t.Errorf("To target origin = %v, want %v", got, want)
	}
	if got, want := e.ToPlane.PlaneNormal(), [3]float64{0, 0, -1}; !samePoint3(got, want) {
		t.Errorf("To target normal = %v, want %v (xAxis x yAxis)", got, want)
	}
	// The stale length is still decoded — it just must not be USED (see translate.extentOf).
	if e.Distance != 0.2 {
		t.Errorf("stale Distance = %v, want 0.2 — the fixture changed", e.Distance)
	}
}

// TestToTargetsAreSelfValidating pins the oracles that make the layout provable rather than guessed:
// the frame's axes are unit vectors (a wrong offset yields arbitrary magnitudes), and each target's
// normal is PARALLEL to its own extrude's direction — as a face the extrude runs TO must be, or it
// could never reach it. Both hold on every To extrude in the corpus fixtures.
func TestToTargetsAreSelfValidating(t *testing.T) {
	for _, f := range []string{"real_multipoint_disk.ipt", "real_screw_slot.ipt", "real_arc_linkage.ipt"} {
		for i, e := range DecodeExtrudes(openDoc(t, f)) {
			if !e.ToPlaneOK {
				continue
			}
			if !isUnit(e.ToPlane.XAxis) || !isUnit(e.ToPlane.YAxis) {
				t.Errorf("%s ex%d: target axes are not unit vectors (%v, %v) — wrong offset?",
					f, i, e.ToPlane.XAxis, e.ToPlane.YAxis)
			}
			if !e.DirOK {
				continue
			}
			n := e.ToPlane.PlaneNormal()
			dot := n[0]*e.Dir[0] + n[1]*e.Dir[1] + n[2]*e.Dir[2]
			if stdmath.Abs(stdmath.Abs(dot)-1) > 1e-6 {
				t.Errorf("%s ex%d: target normal %v is not parallel to the extrude direction %v (dot %.6f) — "+
					"an extrude cannot terminate at a face it never meets", f, i, n, e.Dir, dot)
			}
		}
	}
}

// TestNonToExtrudesStateNoTarget pins that the target is To-SPECIFIC: a Dimension extrude must not
// claim one, or extentOf would throw away a length that is real.
func TestNonToExtrudesStateNoTarget(t *testing.T) {
	for _, e := range DecodeExtrudes(openDoc(t, "real_screw_slot.ipt")) {
		if e.ToPlaneOK {
			t.Errorf("a screw extrude claims a To target, but the part has no To extent")
		}
	}
}

func samePoint3(a, b [3]float64) bool {
	for i := range a {
		if stdmath.Abs(a[i]-b[i]) > 1e-6 {
			return false
		}
	}
	return true
}
