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

// Two defects that both end in the same place — a body faceted where it should have stayed analytic —
// and that compound on the same shape, so they are pinned together here.
//
// #33: a multi-region extrude must apply its regions' prisms ONE AT A TIME rather than merge them into
// a single multi-lump tool. Same set operation — A−(B₁∪B₂) = ((A−B₁)−B₂) — different computation: the
// merged tool is not a bare analytic primitive, so combine's gate fails and BOTH operands are faceted.
// At scale it broke outright (TorquimeterDisk's 52-region cut came back 1630 faces / 25 shells / not
// closed, from a planar boolean that had itself classified the result invalid).
//
// #34: combine's gate additionally demanded an ALL-PLANAR target, which defeated the kernel's own
// chaining — every bore after the first meets a target already carrying a cylinder wall, so both were
// faceted again and the earlier bore's wall was shattered in passing. brep.drillThroughCurved was
// built precisely to chain onto an already-curved target (#1336).

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

// TestMultiRegionCutKeepsTheAnalyticPath pins the sharpest SYNTHETIC signal of #33 and #34 together:
// a drilled plate must come back as its EXACT B-rep — 6 box planes + one analytic cylinder wall per
// bore, 9 faces, no facets anywhere.
//
// Each defect alone shredded it. Merged into one multi-lump tool (#33), the tool is not a bare
// primitive, combine facets BOTH operands, and every bore lands as a 24-gon: ZERO cylinder faces, 102
// total. Per-lump but with combine's old target-must-be-planar gate (#34), only the 1st and 3rd bores
// stayed round — bore 2 met an already-curved target, so both were faceted again and bore 1's wall was
// shattered in passing: 1 cylinder face, 63 total.
//
// This case does NOT reproduce #33's headline failure (the non-closed body): merged stays valid and
// closed here, and even at 40 clean bores / 1226 faces. That needs the real geometry —
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
	if n := cylinderFaceCount(b); n != 3 {
		t.Errorf("drilled plate kept %d analytic cylinder walls, want 3 (one per bore): "+
			"0 means the merged multi-lump tool faceted every bore (#33); 1 means the chain broke on the "+
			"already-curved target after bore 1 (#34)", n)
	}
	if n := len(b.Faces()); n != 9 {
		t.Errorf("drilled plate has %d faces, want 9 — the exact B-rep is 6 box planes + 3 hole walls; "+
			"more means something was faceted (63 per-lump-only, 102 merged)", n)
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

// TestWasherKeepsBothWallsAnalytic pins the simplest shape #34's gate used to shred: a washer is
// cylinder − cylinder, so BOTH operands are curved. The old gate demanded that exactly one of them be
// a bare primitive and the OTHER be all-planar, so the commonest tube in CAD never took the curved
// path — both walls were faceted into 24-gons (measured 9.3175 cm³ and ZERO analytic faces, against an
// analytic 9.4248). Nothing multi-region is involved; this is the plain two-cylinder case.
func TestWasherKeepsBothWallsAnalytic(t *testing.T) {
	const rOut, rIn, h = 2.0, 1.0, 1.0
	fs := feature.NewPartFeatures(param.NewParameters())
	feature.NewExtrudeFeatures(fs).AddExtrude(boreSketch(t, rOut, 0), []int{0}, ops.NewBody,
		feature.Extent{Type: feature.DistanceExtent, Direction: feature.PositiveDir, Distance: func() float64 { return h }}, 0)
	fs.Recompute()
	feature.NewExtrudeFeatures(fs).AddExtrude(boreSketch(t, rIn, 0), []int{0}, ops.Cut,
		feature.Extent{Type: feature.ThroughAllExtent, Direction: feature.SymmetricDir}, 0)
	fs.Recompute()

	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("got %d bodies, want 1", len(bodies))
	}
	b := bodies[0]
	if !b.IsSolid() || !ops.Validate(b).Closed {
		t.Errorf("washer is not a closed solid (IsSolid=%v)", b.IsSolid())
	}
	if n := cylinderFaceCount(b); n != 2 {
		t.Errorf("washer kept %d analytic cylinder walls, want 2 (outer + bore) — "+
			"a both-operands-curved pair must still take the exact curved path", n)
	}
	want := stdmath.Pi * (rOut*rOut - rIn*rIn) * h
	got := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesHigh).VolumeMm3 / 1000
	if stdmath.Abs(got-want)/want > 0.005 {
		t.Errorf("washer volume = %g cm³, want %g ±0.5%% (faceted walls read 9.3175)", got, want)
	}
}

// boreSketch is one circle of radius r on the plane z.
func boreSketch(t *testing.T, r, z float64) *sketch.Sketch {
	t.Helper()
	ux, _ := math.NewUnitVector3(1, 0, 0)
	uy, _ := math.NewUnitVector3(0, 1, 0)
	pl, err := sketch.NewPlane(math.P3(0, 0, math.Scalar(z)), ux, uy)
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	s := sketch.NewSketches().Add(pl)
	s.Circles().AddByCenterRadius(math.P2(0, 0), math.Scalar(r))
	return s
}
