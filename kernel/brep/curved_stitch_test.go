// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// curvedStitch topology round-trip (M2 Phase 1, Oblikovati/Oblikovati#1334): flattening a closed
// analytic body to curvedFaces and stitching them back must reproduce a solid with the same faces and
// shared edges (volume is checked through the exported HalfSpaceCut in ops_test, which can use ops).

// TestCurvedStitchRoundTripsCylinder: facesOfAny → curvedStitch on a cylinder rebuilds a solid with its
// 3 faces (2 caps + side) and 3 shared edges (2 seam circles + the axial seam), each edge used twice.
func TestCurvedStitchRoundTripsCylinder(t *testing.T) {
	t.Parallel()
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	got := curvedStitch(facesOfAny(cyl))
	if got == nil || !got.IsSolid() {
		t.Fatalf("stitched cylinder is not a solid: %+v", got)
	}
	if n := len(got.Faces()); n != 3 {
		t.Errorf("stitched cylinder has %d faces, want 3", n)
	}
	if n := len(got.Edges()); n != 3 {
		t.Errorf("stitched cylinder has %d edges, want 3 (shared, not duplicated)", n)
	}
	// A closed manifold has exactly two edge-uses per edge (counting multiplicity — the periodic side's
	// axial seam is used twice by the SAME side face, so it borders one face but still has two uses).
	for _, e := range got.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
	}
}

// TestCurvedStitchRoundTripsSphere: a bare sphere (one boundary-less face) round-trips to one solid face.
func TestCurvedStitchRoundTripsSphere(t *testing.T) {
	t.Parallel()
	sphere, err := SolidSphere(math.P3(0, 0, 0), 5, "s")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	got := curvedStitch(facesOfAny(sphere))
	if got == nil || !got.IsSolid() {
		t.Fatalf("stitched sphere is not a solid: %+v", got)
	}
	if n := len(got.Faces()); n != 1 {
		t.Errorf("stitched sphere has %d faces, want 1", n)
	}
}

// TestEdgeCurveForCircleSubRangeIsArc: a circle sub-range must become an Arc3d (whole domain = the arc),
// so the edge tessellates over the arc, not the full circle (the #1334 arc-edge trap).
func TestEdgeCurveForCircleSubRangeIsArc(t *testing.T) {
	t.Parallel()
	circle, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	le := loopEdge{curve: circle, t0: 0, t1: 0.25} // a quarter arc
	got := edgeCurveFor(le)
	arc, ok := got.(geom.Arc3d)
	if !ok {
		t.Fatalf("circle sub-range edge curve is %T, want geom.Arc3d", got)
	}
	// The arc's endpoints must match the circle at t0 and t1.
	if d := float64(arc.PointAt(0).DistanceTo(circle.PointAt(0))); d > 1e-9 {
		t.Errorf("arc start off the circle's t0 by %g", d)
	}
	if d := float64(arc.PointAt(1).DistanceTo(circle.PointAt(0.25))); d > 1e-9 {
		t.Errorf("arc end off the circle's t1 by %g", d)
	}
}

// TestEdgeCurveForFullCircleStaysCircle: a closed full-circle edge keeps the Circle (no arc conversion).
func TestEdgeCurveForFullCircleStaysCircle(t *testing.T) {
	t.Parallel()
	circle, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	le := loopEdge{curve: circle, t0: 0, t1: 1}
	if _, ok := edgeCurveFor(le).(geom.Circle); !ok {
		t.Errorf("full-circle edge curve is %T, want geom.Circle", edgeCurveFor(le))
	}
}

// TestCurvedStitchWeldsProducerNoiseAtScale is the mechanism regression for #1602: on a ~2 m part
// the SSI tracer accepts an on-curve point anywhere within 1e-7·extent ≈ 2.8e-5 cm, so two
// independently computed copies of one seam point can sit ~28× apart the retired absolute 1e-6
// weld grid — and failed to merge, tearing the seam. The curved stitch grid now scales with the
// stitched geometry (geom.Resolution.Stitch), so the copies weld to one vertex.
func TestCurvedStitchWeldsProducerNoiseAtScale(t *testing.T) {
	t.Parallel()
	const extent = 280.0
	w := newWelder3(geom.ResolutionForSize(extent).Stitch())
	p := math.P3(100, 57.3, -42.1)
	noisy := p.TranslateBy(math.V3(1.6e-5, -1.6e-5, 1.6e-5)) // |Δ| ≈ 2.8e-5 = the SSI acceptance at this extent
	if w.add(p) != w.add(noisy) {
		t.Errorf("two SSI-noise copies of one seam point (%g apart) welded to distinct vertices on a %g cm part",
			float64(p.DistanceTo(noisy)), extent)
	}
	far := p.TranslateBy(math.V3(0.1, 0, 0)) // a genuinely distinct vertex stays distinct
	if w.add(far) == w.add(p) {
		t.Error("a point 0.1 cm away merged with the seam point; the stitch grid is too coarse")
	}
}

// TestClosedPolylineRecognitionAtScale: a traced SSI loop on a large part whose closure gap is
// producer noise (not zero) must still be recognised as closed by the (u,v) arrangement — the old
// exact-cell key equality misread it as open once the gap outgrew the absolute grid, so both
// operand sides emitted different edge runs and the seam tore (#1602).
func TestClosedPolylineRecognitionAtScale(t *testing.T) {
	t.Parallel()
	pts := []math.Point3{math.P3(100, 0, 0), math.P3(0, 100, 0), math.P3(-100, 0, 0), math.P3(0, -100, 0), math.P3(100, 2.8e-5, 0)}
	pl, err := geom.NewPolyline(pts)
	if err != nil {
		t.Fatalf("NewPolyline: %v", err)
	}
	if !isClosedCurve(&pl) {
		t.Error("a loop closed to within SSI producer noise on a 200 cm part was misread as open")
	}
	open := []math.Point3{math.P3(100, 0, 0), math.P3(0, 100, 0), math.P3(-100, 0, 0), math.P3(0, -100, 0), math.P3(100, 1, 0)}
	plOpen, _ := geom.NewPolyline(open)
	if isClosedCurve(&plOpen) {
		t.Error("a chain with a 1 cm endpoint gap was misread as closed")
	}
}
