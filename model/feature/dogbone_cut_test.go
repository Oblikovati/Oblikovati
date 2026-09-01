// SPDX-License-Identifier: GPL-2.0-only

package feature_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// plateSketch20x10 is a 20x10 plate on XY.
func plateSketch20x10() *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	c := [][2]float64{{-10, -5}, {10, -5}, {10, 5}, {-10, 5}}
	p := make([]*sketch.Point, 4)
	for i, q := range c {
		p[i] = s.Points().Add(math.P2(math.Scalar(q[0]), math.Scalar(q[1])))
	}
	for i := range p {
		s.Lines().Add(p[i], p[(i+1)%4])
	}
	return s
}

// dogboneSketch is Inventor's dog-bone slot: a rounded-rect slot whose four corners are relieved by
// small discs. The arrangement splits it into a slot cell abutting four disc cells — five profiles
// sharing arcs, the shape that cracked BigChunkyPlate's top face when cut one prism at a time (#38).
func dogboneSketch(t *testing.T, z float64) *sketch.Sketch {
	t.Helper()
	ux, _ := math.NewUnitVector3(1, 0, 0)
	uy, _ := math.NewUnitVector3(0, 1, 0)
	pl, _ := sketch.NewPlane(math.P3(0, 0, math.Scalar(z)), ux, uy)
	s := sketch.NewSketches().Add(pl)
	roundedRect(s, -4, 4, -2, 2, 0.5)
	for _, c := range [][2]float64{{-3.5, 1.5}, {3.5, 1.5}, {-3.5, -1.5}, {3.5, -1.5}} {
		s.Circles().AddByCenterRadius(math.P2(math.Scalar(c[0]), math.Scalar(c[1])), 0.5)
	}
	return s
}

// roundedRect adds a rounded rectangle [x0,x1]x[y0,y1] with corner radius r as lines + quarter arcs.
func roundedRect(s *sketch.Sketch, x0, x1, y0, y1, r float64) {
	pts := [][2]float64{{x0 + r, y0}, {x1 - r, y0}, {x1, y0 + r}, {x1, y1 - r}, {x1 - r, y1}, {x0 + r, y1}, {x0, y1 - r}, {x0, y0 + r}}
	p := make([]*sketch.Point, len(pts))
	for i, q := range pts {
		p[i] = s.Points().Add(math.P2(math.Scalar(q[0]), math.Scalar(q[1])))
	}
	s.Lines().Add(p[0], p[1])
	s.Lines().Add(p[2], p[3])
	s.Lines().Add(p[4], p[5])
	s.Lines().Add(p[6], p[7])
	centres := [][2]float64{{x1 - r, y0 + r}, {x1 - r, y1 - r}, {x0 + r, y1 - r}, {x0 + r, y0 + r}}
	ends := [][2]int{{1, 2}, {3, 4}, {5, 6}, {7, 0}}
	for i, c := range centres {
		s.Arcs().AddByCenterStartEnd(math.P2(math.Scalar(c[0]), math.Scalar(c[1])), p[ends[i][0]].Position(), p[ends[i][1]].Position(), true)
	}
}

// TestDogboneCutClosesWithDissolve: a plate cut through-all by the dog-bone slot rebuilds as a closed
// solid of the right volume — the dissolve fuses the five abutting cells so no coincident wall is left
// behind (#38). NoDissolve is the pre-fix path, kept as the whole-part fallback.
func TestDogboneCutClosesWithDissolve(t *testing.T) {
	t.Parallel()
	fs := feature.NewPartFeatures(param.NewParameters())
	feature.NewExtrudeFeatures(fs).AddExtrude(plateSketch20x10(), []int{0}, ops.NewBody,
		feature.Extent{Type: feature.DistanceExtent, Direction: feature.PositiveDir, Distance: func() float64 { return 3 }}, 0)
	fs.Recompute()

	sk := dogboneSketch(t, 3)
	n := sk.Profiles().Count()
	if n < 5 {
		t.Fatalf("expected ≥5 dog-bone profiles, got %d", n)
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	feature.NewExtrudeFeatures(fs).AddExtrudeFeature(&feature.ExtrudeDefinition{
		Sketch: sk, ProfileIndices: idx, Operation: ops.Cut,
		Extent: feature.Extent{Type: feature.ThroughAllExtent, Direction: feature.NegativeDir},
	})
	fs.Recompute()

	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("got %d bodies, want 1", len(bodies))
	}
	b := bodies[0]
	if rep := ops.Validate(b); !b.IsSolid() || !rep.Closed {
		t.Errorf("dog-bone cut is not a closed solid (IsSolid=%v Closed=%v shells=%d)", b.IsSolid(), rep.Closed, len(b.Shells()))
	}
	slot := 8.0*4.0 - (4-stdmath.Pi)*0.5*0.5 // rounded rect 8x4 r0.5
	want := 20.0*10.0*3.0 - slot*3.0
	got := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesHigh).VolumeMm3 / 1000
	if stdmath.Abs(got-want)/want > 0.01 {
		t.Errorf("dog-bone cut volume = %g cm³, want %g ±1%%", got, want)
	}
}
