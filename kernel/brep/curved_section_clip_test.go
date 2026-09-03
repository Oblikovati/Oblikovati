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

// TestSectionInsideOnTheWrapAroundRunIsKept pins the closed-conic half of the clip
// (Oblikovati/Oblikovati#3459). Two crossings cut a circle or an ellipse into TWO arcs, and the runs
// BETWEEN consecutive cuts are only one of them: the other runs from the last cut back round to the
// first. Emitting only the gaps left that arc unrepresented, so when it was the arc lying inside the
// face the clip reported no runs at all and declined a section it had correctly crossed.
//
// The fixture puts the inside arc on the wrap: a bar centred on the circle's SEAM, so the material
// side spans the parameter origin.
func TestSectionInsideOnTheWrapAroundRunIsKept(t *testing.T) {
	t.Parallel()
	circle, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	// A block covering only the side the seam point sits on, so the run inside the face is the one
	// that wraps the parameter origin rather than a gap between cuts. The circle derives its own
	// RefDir, so the side is read off the seam point rather than assumed.
	seam := circle.PointAt(0)
	far := circle.PointAt(0.5)
	lo, hi := math.P3(-6, 2, 0), math.P3(6, 8, 3)
	switch {
	case stdmath.Abs(float64(seam.X)) > stdmath.Abs(float64(seam.Y)) && seam.X > 0:
		lo, hi = math.P3(2, -6, 0), math.P3(8, 6, 3)
	case stdmath.Abs(float64(seam.X)) > stdmath.Abs(float64(seam.Y)):
		lo, hi = math.P3(-8, -6, 0), math.P3(-2, 6, 3)
	case seam.Y < 0:
		lo, hi = math.P3(-6, -8, 0), math.P3(6, -2, 3)
	}
	slab, err := SolidBlock(lo, hi, "slab")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	var face curvedFace
	found := false
	for _, f := range facesOfAny(slab) {
		pl, isPlane := f.surface.(geom.Plane)
		if !isPlane || stdmath.Abs(float64(pl.Origin.Z)) > 1e-9 || stdmath.Abs(float64(pl.Normal().Z)) < 0.9 {
			continue
		}
		face, found = f, true
	}
	if !found {
		t.Fatal("the slab has no face in the z = 0 plane")
	}
	if !pointInFace2D(to2D(facePlane(face), seam), face) {
		t.Fatalf("the fixture must CONTAIN the circle's seam point %v, or the inside run is not the wrap", seam)
	}
	if pointInFace2D(to2D(facePlane(face), far), face) {
		t.Fatalf("the fixture must EXCLUDE the half-turn point %v, or the inside run is a plain gap", far)
	}
	arcs, ok := clipSectionToFace(circle, face)
	if !ok || len(arcs) == 0 {
		t.Fatalf("the clip declined a circle whose inside run wraps the parameter origin (ok=%v, %d arcs)",
			ok, len(arcs))
	}
	// Only INTERIOR samples: a clipped run's endpoints lie ON the face's boundary by construction,
	// and the containment test is strict.
	for _, arc := range arcs {
		for _, s := range []float64{0.25, 0.5, 0.75} {
			if p := arc.PointAt(s); !pointInFace2D(to2D(facePlane(face), p), face) {
				t.Errorf("a clipped run leaves the face at %v", p)
			}
		}
	}
}
