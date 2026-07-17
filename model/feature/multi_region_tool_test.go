// SPDX-License-Identifier: GPL-2.0-only

package feature_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// A multi-region extrude must apply its regions' prisms ONE AT A TIME, not merge them into a single
// multi-lump tool and run one boolean (#33).
//
// The two are the same set operation — A−(B₁∪B₂) = ((A−B₁)−B₂) — but not the same computation. The
// merged tool is not a bare analytic primitive, so combine's exactlyOneCurvedPrimitive test fails and
// BOTH operands are faceted into the planar path; per lump, the exact curved boolean applies instead.
//
// The real cost was not cosmetic. TorquimeterDisk cuts 52 regions at once: merged, that is an
// 878-face 52-shell tool, and the planar boolean returned a body it had ITSELF classified invalid
// (booleanGeneral ships the planar result unchecked once the operands exceed csgFallbackFaceLimit) —
// 1630 faces, 25 shells, not closed. Lump by lump: 461 faces, one shell, a closed solid within 5% of
// Inventor.

func cylinderFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}

// plateSketch is a 10x6 plate on XY.
func plateSketch() *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	c := [][2]float64{{0, 0}, {10, 0}, {10, 6}, {0, 6}}
	pts := make([]*sketch.Point, 0, 4)
	for _, p := range c {
		pts = append(pts, s.Points().Add(math.P2(math.Scalar(p[0]), math.Scalar(p[1]))))
	}
	for i := range pts {
		s.Lines().Add(pts[i], pts[(i+1)%len(pts)])
	}
	return s
}

// threeBoresSketch is three separate circles — three distinct regions of ONE sketch, the
// multi-region selection an Extrude gathers with Ctrl+click. z places the sketch plane.
func threeBoresSketch(t *testing.T, r, z float64) *sketch.Sketch {
	t.Helper()
	ux, _ := math.NewUnitVector3(1, 0, 0)
	uy, _ := math.NewUnitVector3(0, 1, 0)
	pl, err := sketch.NewPlane(math.P3(0, 0, math.Scalar(z)), ux, uy)
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	s := sketch.NewSketches().Add(pl)
	for _, cx := range []float64{2, 5, 8} {
		s.Circles().AddByCenterRadius(math.P2(math.Scalar(cx), 3), math.Scalar(r))
	}
	return s
}

// TestMultiRegionCutKeepsTheAnalyticPath pins the fix by its sharpest SYNTHETIC signal: a bore must
// survive as an analytic cylinder wall. Merged, the tool is not a bare primitive, so combine facets
// BOTH operands and every bore comes back a 24-gon — measured: ZERO cylinder faces and 102 total,
// against 1 and 63 per-lump.
//
// It asserts >= 1, not one per bore, because only some survive today: after the first bore the TARGET
// carries a cylinder wall, so exactlyOneCurvedPrimitive fails on the next one and combine facets both
// again — even though drillThroughCurved exists precisely to chain bores onto an already-curved
// target (#1336). That gate is a separate defect (tracked in #34); this test pins the #33 fix and
// would catch a regression of it, without freezing today's alternating behaviour as if it were right.
//
// This synthetic case does NOT reproduce #33's headline failure (the non-closed body): merged stays
// valid and closed here, and even at 40 clean bores / 1226 faces. That needs the real geometry —
// translate.TestMultipointDiskRebuildsAsAClosedSolid.
func TestMultiRegionCutKeepsTheAnalyticPath(t *testing.T) {
	const r = 0.5
	fs := feature.NewPartFeatures(param.NewParameters())
	feature.NewExtrudeFeatures(fs).AddExtrude(plateSketch(), []int{0}, ops.NewBody,
		feature.Extent{Type: feature.DistanceExtent, Direction: feature.PositiveDir, Distance: func() float64 { return 2 }}, 0)
	fs.Recompute()

	feature.NewExtrudeFeatures(fs).AddExtrude(threeBoresSketch(t, r, 0), []int{0, 1, 2}, ops.Cut,
		feature.Extent{Type: feature.ThroughAllExtent, Direction: feature.SymmetricDir}, 0)
	fs.Recompute()

	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("got %d bodies, want 1", len(bodies))
	}
	b := bodies[0]
	rep := ops.Validate(b)
	if !b.IsSolid() || !rep.Closed || !rep.Valid {
		t.Errorf("drilled plate: IsSolid=%v Valid=%v Closed=%v — a multi-region cut must stay a closed solid",
			b.IsSolid(), rep.Valid, rep.Closed)
	}
	if n := cylinderFaceCount(b); n < 1 {
		t.Errorf("drilled plate kept %d analytic cylinder walls, want at least 1 — a merged multi-lump "+
			"tool is not a bare primitive, so combine facets every bore into a 24-gon", n)
	}
	want := 10*6*2 - 3*stdmath.Pi*r*r*2
	got := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesHigh).VolumeMm3 / 1000 // mm³ → cm³
	if stdmath.Abs(got-want)/want > 0.01 {
		t.Errorf("drilled plate volume = %g cm³, want %g ±1%%", got, want)
	}
}

// TestMultiRegionJoinStaysOneClosedSolid pins the same for a Join: three separate bosses raised on a
// plate must each weld on exactly, leaving one closed solid of the exact summed volume.
func TestMultiRegionJoinStaysOneClosedSolid(t *testing.T) {
	const r = 0.5
	fs := feature.NewPartFeatures(param.NewParameters())
	feature.NewExtrudeFeatures(fs).AddExtrude(plateSketch(), []int{0}, ops.NewBody,
		feature.Extent{Type: feature.DistanceExtent, Direction: feature.PositiveDir, Distance: func() float64 { return 2 }}, 0)
	fs.Recompute()

	feature.NewExtrudeFeatures(fs).AddExtrude(threeBoresSketch(t, r, 2), []int{0, 1, 2}, ops.Join,
		feature.Extent{Type: feature.DistanceExtent, Direction: feature.PositiveDir, Distance: func() float64 { return 3 }}, 0)
	fs.Recompute()

	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("got %d bodies, want 1", len(bodies))
	}
	if b := bodies[0]; !b.IsSolid() || !ops.Validate(b).Closed {
		t.Errorf("bossed plate is not a closed solid (IsSolid=%v)", b.IsSolid())
	}
	// The bosses sit ON the plate's top face (z=2) and rise 3, so they add their full volume.
	want := 10*6*2 + 3*stdmath.Pi*r*r*3
	got := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesHigh).VolumeMm3 / 1000
	if stdmath.Abs(got-want)/want > 0.01 {
		t.Errorf("bossed plate volume = %g cm³, want %g ±1%%", got, want)
	}
}
