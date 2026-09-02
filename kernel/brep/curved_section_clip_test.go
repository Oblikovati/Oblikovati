// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestSectionCrossingAFaceTwiceGivesTwoArcs pins Oblikovati/Oblikovati#3459's clip step. A section that
// enters and leaves a face MORE THAN ONCE was bounded to the span between its OUTERMOST crossings,
// which spans the gap as well — for a bore's section circle crossing a bar's footprint, the span ran
// straight through the material the bar does not cover.
func TestSectionCrossingAFaceTwiceGivesTwoArcs(t *testing.T) {
	t.Parallel()
	bar, err := SolidBlock(math.P3(-8, -1, 0), math.P3(8, 1, 3), "bar")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	var footprint curvedFace
	found := false
	for _, f := range facesOfAny(bar) {
		pl, isPlane := f.surface.(geom.Plane)
		if !isPlane || stdmath.Abs(float64(pl.Origin.Z)) > 1e-9 || stdmath.Abs(float64(pl.Normal().Z)) < 0.9 {
			continue
		}
		footprint, found = f, true
	}
	if !found {
		t.Fatal("the bar has no face in the z = 0 plane")
	}
	// A bore of radius 2 at x = −4: its section circle reaches y = ±2, so it crosses the bar's
	// y = ±1 sides four times and lies inside the bar over two separate runs.
	circle, err := geom.NewCircle(math.P3(-4, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	arcs, ok := clipSectionToFace(circle, footprint)
	if !ok {
		t.Fatal("a circle crossing the footprint four times must clip to the runs inside it")
	}
	if len(arcs) != 2 {
		t.Fatalf("got %d arcs, want 2 — the circle enters and leaves the bar twice", len(arcs))
	}
	for _, arc := range arcs {
		for _, s := range []float64{0, 0.25, 0.5, 0.75, 1} {
			p := arc.PointAt(s)
			if stdmath.Abs(float64(p.Y)) > 1+1e-9 || stdmath.Abs(float64(p.X)+4) > 2+1e-9 {
				t.Errorf("an arc leaves the bar at %v; every clipped run must lie inside the face", p)
			}
			if off := stdmath.Abs(float64(math.P3(-4, 0, 0).DistanceTo(p)) - 2); off > 1e-9 {
				t.Errorf("a clipped run stands %g off the circle it came from", off)
			}
		}
	}
}
