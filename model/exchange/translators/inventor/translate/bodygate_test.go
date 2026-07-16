// SPDX-License-Identifier: GPL-2.0-only

package translate

import "testing"

// The gate's whole job is to tell a mis-decoded body from a correct one using only the extents, so
// the cases are stated as extents. Numbers are the real ReelToReel measurements: BigChunkyPlate's
// body came out 40 cm thick against a 3 cm tessellation (26.3x the true volume), while the
// volume-correct ReelMotorBearingShaft sits inside a tessellation inflated by its sketch patches.
func TestEscapingAxis(t *testing.T) {
	cases := []struct {
		name    string
		body    [3]float64
		mesh    [3]float64
		escaped bool
		axis    string
	}{
		{"BigChunkyPlate extruded 40cm into a 3cm plate", [3]float64{40, 40, 48}, [3]float64{3.0, 76.8, 83.96}, true, "smallest"},
		{"MainFrameSingleHeadBlock, 1.2cm plate", [3]float64{40, 48, 48}, [3]float64{1.2, 76.8, 83.37}, true, "smallest"},
		{"CapstonMotorMachinedHolder overshoots its largest extent", [3]float64{2.0, 9.2, 9.2}, [3]float64{3.19, 7.92, 7.92}, true, "largest"},
		{"ReelMotorBearingShaft (volume-correct) fits", [3]float64{1.85, 3.4, 3.4}, [3]float64{1.85, 3.74, 3.81}, false, ""},
		{"PressureRollerMainShaft (volume-correct) fits", [3]float64{0.8, 0.8, 4.4}, [3]float64{0.88, 4.4, 5.06}, false, ""},
		{"Capston (volume-correct) fits", [3]float64{2.6, 2.6, 4.75}, [3]float64{3.66, 4.75, 6.98}, false, ""},
		{"equal extents are not an escape", [3]float64{1, 2, 3}, [3]float64{1, 2, 3}, false, ""},
		{"within tolerance is not an escape", [3]float64{1, 2, 3.05}, [3]float64{1, 2, 3.0}, false, ""},
		{"just outside tolerance escapes", [3]float64{1, 2, 3.2}, [3]float64{1, 2, 3.0}, true, "largest"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			axis, over, escaped := escapingAxis(c.body, c.mesh)
			if escaped != c.escaped {
				t.Fatalf("escaped = %v, want %v (body %v, mesh %v)", escaped, c.escaped, c.body, c.mesh)
			}
			if escaped && axis != c.axis {
				t.Errorf("axis = %q, want %q", axis, c.axis)
			}
			if escaped && over <= 1 {
				t.Errorf("overshoot = %v, want > 1", over)
			}
		})
	}
}

// A zero-sized mesh extent carries no information and must not be read as an escape (it would
// otherwise divide by zero and reject everything).
func TestEscapingAxisIgnoresZeroMeshExtent(t *testing.T) {
	if _, _, escaped := escapingAxis([3]float64{0, 1, 2}, [3]float64{0, 1, 2}); escaped {
		t.Fatal("a zero mesh extent must not count as an escape")
	}
}

func TestSortedSides(t *testing.T) {
	got := sortedSides([3]float64{-1, 0, 5}, [3]float64{2, 10, 6})
	want := [3]float64{1, 3, 10}
	if got != want {
		t.Fatalf("sortedSides = %v, want %v", got, want)
	}
}
