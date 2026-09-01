// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// springStations is a 6-turn variable-pitch spring: tight 1 mm-pitch ends,
// open 4 mm-pitch middle, constant 5 mm radius.
var springStations = []HelixStation{
	{Turn: 0, Radius: 0.5, Pitch: 0.1},
	{Turn: 2, Radius: 0.5, Pitch: 0.4},
	{Turn: 4, Radius: 0.5, Pitch: 0.4},
	{Turn: 6, Radius: 0.5, Pitch: 0.1},
}

func variableSpring(t *testing.T) VariableHelix3d {
	t.Helper()
	h, err := NewVariableHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), springStations, false)
	if err != nil {
		t.Fatalf("NewVariableHelix3d: %v", err)
	}
	return h
}

// TestVariableHelixStationPassThrough: at each station's parameter the curve
// sits at exactly the station radius and the trapezoid-integrated height.
func TestVariableHelixStationPassThrough(t *testing.T) {
	t.Parallel()
	h := variableSpring(t)
	// Trapezoid heights: 0→2: (0.1+0.4)/2·2 = 0.5; 2→4: 0.4·2 = 0.8; 4→6: 0.5.
	wantHeights := []float64{0, 0.5, 1.3, 1.8}
	for i, st := range springStations {
		p := h.PointAt(st.Turn / h.TotalTurns())
		r := stdmath.Hypot(float64(p.X), float64(p.Y))
		if stdmath.Abs(r-st.Radius) > 1e-12 {
			t.Errorf("station %d radius = %v, want %v", i, r, st.Radius)
		}
		if stdmath.Abs(float64(p.Z)-wantHeights[i]) > 1e-12 {
			t.Errorf("station %d height = %v, want %v", i, p.Z, wantHeights[i])
		}
	}
}

// TestVariableHelixLengthMonotonic: arc length must strictly accumulate along
// the parameter (the property-test guard against interpolation glitches).
func TestVariableHelixLengthMonotonic(t *testing.T) {
	t.Parallel()
	h := variableSpring(t)
	prev := 0.0
	for i := 1; i <= 64; i++ {
		t1 := float64(i) / 64
		partial := simpsonLength(func(u float64) float64 {
			return float64(h.TangentAt(u).Length())
		}, 0, t1, 256)
		if partial <= prev {
			t.Fatalf("arc length not monotonic at t=%v (%v after %v)", t1, partial, prev)
		}
		prev = partial
	}
	if total := h.Length(); stdmath.Abs(total-prev) > 1e-9 {
		t.Errorf("Length() = %v, want the full integral %v", total, prev)
	}
}

// TestVariableHelixMatchesConstantHelix: a two-station table with constant
// pitch/radius must reproduce the analytic constant helix exactly.
func TestVariableHelixMatchesConstantHelix(t *testing.T) {
	t.Parallel()
	v, err := NewVariableHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), []HelixStation{
		{Turn: 0, Radius: 0.8, Pitch: 1},
		{Turn: 5, Radius: 0.8, Pitch: 1},
	}, false)
	if err != nil {
		t.Fatalf("variable: %v", err)
	}
	c, err := NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 0.8, 1, 0, 5, false)
	if err != nil {
		t.Fatalf("constant: %v", err)
	}
	for i := 0; i <= 32; i++ {
		u := float64(i) / 32
		if d := float64(v.PointAt(u).DistanceTo(c.PointAt(u))); d > 1e-12 {
			t.Fatalf("curves diverge at t=%v by %g", u, d)
		}
	}
	if stdmath.Abs(v.Length()-c.Length()) > 1e-9 {
		t.Errorf("lengths diverge: %v vs %v", v.Length(), c.Length())
	}
}

// TestVariableHelixRejectsBadStations: too few rows, a nonzero first turn,
// and non-increasing turns are all rejected with the offending values.
func TestVariableHelixRejectsBadStations(t *testing.T) {
	t.Parallel()
	axis, ref := math.V3(0, 0, 1), math.V3(1, 0, 0)
	cases := [][]HelixStation{
		{{Turn: 0, Radius: 1, Pitch: 1}},
		{{Turn: 1, Radius: 1, Pitch: 1}, {Turn: 2, Radius: 1, Pitch: 1}},
		{{Turn: 0, Radius: 1, Pitch: 1}, {Turn: 0, Radius: 2, Pitch: 1}},
	}
	for i, rows := range cases {
		if _, err := NewVariableHelix3d(math.P3(0, 0, 0), axis, ref, rows, false); err == nil {
			t.Errorf("case %d: invalid stations accepted", i)
		}
	}
}
