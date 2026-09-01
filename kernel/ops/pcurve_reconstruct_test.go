// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/tessellate"
)

func TestReconstructFacePcurvesRoundTrips(t *testing.T) {
	t.Parallel()
	f := halfDiskFace(t, 2)
	q := Quality{ChordTolerance: 1e-3}
	ReconstructFacePcurves(f, q)
	s := f.Geometry()
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			pc := u.Pcurve()
			pts := tessellate.DiscretizeEdge(u.Edge(), q)
			if u.Reversed() {
				pts = probe.ReversedPoints(pts)
			}
			if len(pc) != len(pts) {
				t.Fatalf("pcurve has %d points, edge discretization has %d", len(pc), len(pts))
			}
			for i := range pc {
				got := s.PointAt(float64(pc[i].X), float64(pc[i].Y))
				if d := float64(got.DistanceTo(pts[i])); d > 1e-3 {
					t.Errorf("pcurve[%d] does not round-trip its edge point (off %.5f)", i, d)
				}
			}
		}
	}
}

// The arc edge must yield a multi-point pcurve (the curved boundary captured in (u,v)), not a chord.
func TestReconstructFacePcurvesCapturesCurvedEdge(t *testing.T) {
	t.Parallel()
	f := halfDiskFace(t, 2)
	ReconstructFacePcurves(f, Quality{ChordTolerance: 1e-3})
	maxPts := 0
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			if n := len(u.Pcurve()); n > maxPts {
				maxPts = n
			}
		}
	}
	if maxPts <= 2 {
		t.Errorf("arc edge pcurve not subdivided: longest pcurve is %d points", maxPts)
	}
}

func TestReconstructFacePcurvesIdempotent(t *testing.T) {
	t.Parallel()
	f := halfDiskFace(t, 2)
	q := Quality{ChordTolerance: 1e-3}
	ReconstructFacePcurves(f, q)
	first := len(f.Loops()[0].EdgeUses()[0].Pcurve())
	ReconstructFacePcurves(f, q)
	if again := len(f.Loops()[0].EdgeUses()[0].Pcurve()); again != first {
		t.Errorf("re-running changed the pcurve length: %d then %d", first, again)
	}
}
