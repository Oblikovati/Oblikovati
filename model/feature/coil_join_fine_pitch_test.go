// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// coreDiscSketch is a faceted disc of radius r on XY — a thread core profile.
func coreDiscSketch(r float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	s.Circles().AddByCenterRadius(math.P2(0, 0), r)
	return s
}

// roundThreadSketch is a round wire of radius wireR centred at radius meanR — a coil thread.
func roundThreadSketch(meanR, wireR float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	s.Circles().AddByCenterRadius(math.P2(meanR, 0), wireR)
	return s
}

// TestCoilJoinFinePitchWatertight is the integration regression for #879. A helical thread
// coiled onto a core and JOINED produced a body shredded into coincident, unpaired OPEN edges
// at fine pitch — the dense self-proximate geometry put thousands of coincident vertices on
// opposite sides of the stitch weld grid's cell boundaries, so they failed to merge. The weld
// now searches neighbouring cells, so the coil-join is one watertight solid across the pitches
// of a real thread (the 3 mm pitch on a Ø6 core was the worst case: ~9600 open edges before).
func TestCoilJoinFinePitchWatertight(t *testing.T) {
	t.Parallel()
	for _, pitch := range []float64{0.4, 0.3, 0.25, 0.2} { // cm: 4, 3, 2.5, 2 mm
		fs := NewPartFeatures(nil)
		NewExtrudeFeatures(fs).AddByDistanceExtent(coreDiscSketch(0.3), 0, ops.NewBody, func() float64 { return 1.2 })
		NewCoilFeatures(fs).AddDefinition(&CoilDefinition{
			Sketch: roundThreadSketch(0.35, 0.1), Axis: zWorkAxis(),
			Pitch: angleConst(pitch), Height: angleConst(1.2), Operation: ops.Join,
		})
		fs.Recompute()

		bodies := fs.Result()
		if len(bodies) != 1 {
			t.Fatalf("pitch %g cm: coil-join produced %d bodies, want 1", pitch, len(bodies))
		}
		open := 0
		for _, e := range bodies[0].Edges() {
			if len(e.Faces()) < 2 {
				open++
			}
		}
		if r := ops.Validate(bodies[0]); !r.Valid || !r.Closed || open != 0 {
			t.Errorf("pitch %g cm: threaded rod not watertight: valid=%v closed=%v open=%d",
				pitch, r.Valid, r.Closed, open)
		}
	}
}
