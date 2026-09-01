// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestWrapsTube distinguishes the two closed-loop windings on a torus: a MERIDIAN circle (fixed
// azimuth, tube angle sweeping 2π) wraps the tube; a LATITUDE circle (fixed tube angle, azimuth
// sweeping 2π) does not.
func TestWrapsTube(t *testing.T) {
	t.Parallel()
	tor := weTestTorus(t, 200, 50)
	meridian := make([]math.Point3, 0, 65)
	latitude := make([]math.Point3, 0, 65)
	for k := 0; k <= 64; k++ {
		a := 2 * stdmath.Pi * float64(k) / 64
		meridian = append(meridian, tor.PointAt(0, a))
		latitude = append(latitude, tor.PointAt(a, 0.3))
	}
	if !wrapsTube(tor, meridian) {
		t.Fatal("a meridian circle winds the tube once — wrapsTube must report true")
	}
	if wrapsTube(tor, latitude) {
		t.Fatal("a latitude circle never winds the tube — wrapsTube must report false")
	}
}

// TestUSpread pins the iso-u census: a constant azimuth set spreads 0; a ±0.1 rad wander spreads 0.2;
// the measure is wrap-safe across the 0/2π seam.
func TestUSpread(t *testing.T) {
	t.Parallel()
	if s := uSpread([]float64{1.5, 1.5, 1.5}); s != 0 {
		t.Fatalf("constant azimuths spread %.3g, want 0", s)
	}
	if s := uSpread([]float64{1.5, 1.6, 1.4}); stdmath.Abs(s-0.2) > 1e-12 {
		t.Fatalf("±0.1 wander spreads %.3g, want 0.2", s)
	}
	if s := uSpread([]float64{0.05, 2*stdmath.Pi - 0.05, 0.05}); stdmath.Abs(s-0.1) > 1e-12 {
		t.Fatalf("seam-straddling wander spreads %.3g, want 0.1", s)
	}
}

// TestTubeBandCircleAndRail pins the ring classification: an iso-u meridian ring is the CIRCLE, a
// wandering ring is the RAIL, and two wandering rings are ambiguous (decline).
func TestTubeBandCircleAndRail(t *testing.T) {
	t.Parallel()
	tor := weTestTorus(t, 200, 50)
	iso := make([]math.Point3, 0, 64)
	wander := make([]math.Point3, 0, 64)
	for k := range 64 {
		v := 2 * stdmath.Pi * float64(k) / 64
		iso = append(iso, tor.PointAt(1.0, v))
		wander = append(wander, tor.PointAt(2.0+0.05*stdmath.Sin(v), v))
	}
	circ, rail, ok := tubeBandCircleAndRail(tor, [][]math.Point3{wander, iso})
	if !ok {
		t.Fatal("iso + wandering rings must classify")
	}
	if s := uSpread(circ.us); s > 1e-6 {
		t.Fatalf("classified circle ring wanders %.3g in u — rings swapped", s)
	}
	if s := uSpread(rail.us); s < 1e-3 {
		t.Fatalf("classified rail ring is iso-u (%.3g) — rings swapped", s)
	}
	if _, _, ok := tubeBandCircleAndRail(tor, [][]math.Point3{wander, wander}); ok {
		t.Fatal("two wandering rings are ambiguous — must decline")
	}
}
