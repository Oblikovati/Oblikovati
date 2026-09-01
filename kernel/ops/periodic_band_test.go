// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestPeriodicBandGridRejectsNonBand guards the sphere-cap-over-the-pole fix: periodicBandGrid must
// only accept a genuine latitude band (boundary = two full-longitude circles), not a boundary that
// wanders across latitudes (a cap straddling the pole). Accepting the latter gridded its whole
// us×vs box and tore at the pole (340-triangle full-sphere fan on the EDF bell-mouth).
func TestPeriodicBandGridRejectsNonBand(t *testing.T) {
	t.Parallel()
	s, err := geom.NewSphere(math.P3(0, 0, 0), 10)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	// A genuine band: points on two constant-latitude circles spanning full longitude → accepted.
	var band []math.Point3
	for _, lat := range []float64{-0.3, 0.3} {
		for k := range 8 {
			band = append(band, s.PointAt(2*stdmath.Pi*float64(k)/8, lat))
		}
	}
	if _, _, ok := periodicBandGrid(s, band, nil); !ok {
		t.Error("two full-longitude latitude circles should be accepted as a band")
	}
	// A boundary spiralling across many latitudes is NOT a band (a cap near the pole) → rejected,
	// so it falls through to the boundary triangulator instead of a torn full-domain grid.
	var spiral []math.Point3
	for k := range 16 {
		lat := -0.3 + 1.2*float64(k)/16 // wanders across latitudes, up toward the pole
		spiral = append(spiral, s.PointAt(2*stdmath.Pi*float64(k)/16, lat))
	}
	if _, _, ok := periodicBandGrid(s, spiral, nil); ok {
		t.Error("a boundary wandering across latitudes is not a band; must be rejected")
	}
}
