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

// A slotted screw, end to end — the shape that exposed #31 (a real part, ReelToReel's
// CompressionRollerArmActuatorScrew, rebuilt here from hand-built sketches so the test owns its
// geometry and no .ipt decode is involved).
//
// What it actually catches, verified by reverting each fix in turn — NOT both, despite both being #31:
//   - ops.Facet collapsing the Ø3mm shaft to a square prism (losing 38.5 of its 106.03 mm³ before the
//     boolean ran): CAUGHT — reverting it fails "cross-hole removal" and the Inventor check.
//   - the drill boring the SLOT'S VOID and plugging it: NOT caught here, because this feature ORDER
//     joins the shaft before the hole, so the cut's target always carries a faceted cylinder and the
//     drill declines on its cap count regardless. That defect is pinned where it is reachable, by
//     brep.TestDrillRejectsCapsAcrossAVoid.
//
// Every stage is checked against an INDEPENDENT analytic value, never against our own output — the two
// #31 defects CANCELLED to 1.033x of truth on the real part, so only outside values can catch that. The
// 2% band covers the ~0.64% inscribed-N-gon tessellation bias with headroom; the defects were 36% and
// 135% of their stages.

// hexHeadSketch is the head: a hexagon (across-flats 0.6, circumradius 0.346) on the world x=0 plane,
// whose sketch X axis is world +Y and Y axis world +Z — so its normal is +X.
func hexHeadSketch(t *testing.T) *sketch.Sketch {
	t.Helper()
	s := sketch.NewSketches().Add(planeAt(t, math.P3(0, 0, 0), math.V3(0, 1, 0), math.V3(0, 0, 1)))
	corners := [][2]float64{{0.3, -0.173}, {0, -0.346}, {-0.3, -0.173}, {-0.3, 0.173}, {0, 0.346}, {0.3, 0.173}}
	pts := make([]*sketch.Point, 0, len(corners))
	for _, c := range corners {
		pts = append(pts, s.Points().Add(math.P2(math.Scalar(c[0]), math.Scalar(c[1]))))
	}
	for i := range pts {
		s.Lines().Add(pts[i], pts[(i+1)%len(pts)])
	}
	return s
}

// slotSketch is the screwdriver slot: a 1.6mm x 10mm rectangle on the head's outer face (x=0.7),
// cut back INTO the head (its extrude is reversed).
func slotSketch(t *testing.T) *sketch.Sketch {
	t.Helper()
	s := sketch.NewSketches().Add(planeAt(t, math.P3(0.7, 0, 0), math.V3(0, 0, -1), math.V3(0, 1, 0)))
	corners := [][2]float64{{-0.08, 0.5}, {0.08, 0.5}, {0.08, -0.5}, {-0.08, -0.5}}
	pts := make([]*sketch.Point, 0, len(corners))
	for _, c := range corners {
		pts = append(pts, s.Points().Add(math.P2(math.Scalar(c[0]), math.Scalar(c[1]))))
	}
	for i := range pts {
		s.Lines().Add(pts[i], pts[(i+1)%len(pts)])
	}
	return s
}

// crossHoleSketch is the transverse hole: r=0.15 at world (0.4, 0) on the z=0.08 plane, cut through all.
// Its axis crosses the slot, so the material it meets is in TWO DISJOINT pieces — the #31 trigger.
func crossHoleSketch(t *testing.T) *sketch.Sketch {
	t.Helper()
	s := sketch.NewSketches().Add(planeAt(t, math.P3(0.7, -0.3, 0.08), math.V3(-1, 0, 0), math.V3(0, 1, 0)))
	s.Circles().AddByCenterRadius(math.P2(0.3, 0.3), 0.15)
	return s
}

// shaftSketch is the Ø3mm shaft: r=0.15 at the origin on the x=0 plane, normal -X, joined 1.5 long.
func shaftSketch(t *testing.T) *sketch.Sketch {
	t.Helper()
	s := sketch.NewSketches().Add(planeAt(t, math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(0, 1, 0)))
	s.Circles().AddByCenterRadius(math.P2(0, 0), 0.15)
	return s
}

func planeAt(t *testing.T, origin math.Point3, x, y math.Vector3) sketch.Plane {
	t.Helper()
	ux, err := math.NewUnitVector3(float64(x.X), float64(x.Y), float64(x.Z))
	if err != nil {
		t.Fatalf("x axis: %v", err)
	}
	uy, err := math.NewUnitVector3(float64(y.X), float64(y.Y), float64(y.Z))
	if err != nil {
		t.Fatalf("y axis: %v", err)
	}
	pl, err := sketch.NewPlane(origin, ux, uy)
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	return pl
}

func TestSlottedScrewRebuildsWithoutOffsettingErrors(t *testing.T) {
	fs := feature.NewPartFeatures(param.NewParameters())
	vol := func() float64 {
		return analysis.MassPropertiesOf(fs.Result(), 1, types.MassPropertiesHigh).VolumeMm3
	}
	dist := func(d float64) feature.Extent {
		return feature.Extent{Type: feature.DistanceExtent, Direction: feature.PositiveDir, Distance: func() float64 { return d }}
	}
	check := func(what string, got, want float64) {
		t.Helper()
		if stdmath.Abs(got-want)/want > 0.02 {
			t.Errorf("%s = %.3f mm³, want %.3f +/-2%%", what, got, want)
		}
	}

	feature.NewExtrudeFeatures(fs).AddExtrude(hexHeadSketch(t), []int{0}, ops.NewBody, dist(0.7), 0)
	fs.Recompute()
	head := vol()
	// A hexagon of across-flats 0.6 has area (sqrt(3)/2)*0.6² = 0.3118 cm², times 0.7 deep.
	check("head", head, stdmath.Sqrt(3)/2*0.36*0.7*1000)

	slot := dist(0.6)
	slot.Direction = feature.NegativeDir // cut back into the head from its outer face
	feature.NewExtrudeFeatures(fs).AddExtrude(slotSketch(t), []int{0}, ops.Cut, slot, 0)
	fs.Recompute()
	afterSlot := vol()
	// The slot spans the full hexagon width, so it removes 0.16 (z) x 0.6 (y) x 0.6 (deep) cm³.
	check("slot removal", head-afterSlot, 0.16*0.6*0.6*1000)

	feature.NewExtrudeFeatures(fs).AddExtrude(shaftSketch(t), []int{0}, ops.Join, dist(1.5), 0)
	fs.Recompute()
	afterShaft := vol()
	// pi*r²*h. This is the stage ops.Facet used to shred: a square prism would add only 67.5.
	check("shaft join", afterShaft-afterSlot, stdmath.Pi*0.15*0.15*1.5*1000)

	feature.NewExtrudeFeatures(fs).AddExtrude(crossHoleSketch(t), []int{0}, ops.Cut,
		feature.Extent{Type: feature.ThroughAllExtent, Direction: feature.SymmetricDir}, 0)
	fs.Recompute()
	afterHole := vol()
	// The r=0.15 cylinder along Z at (x=0.4, y=0) through the hexagon, MINUS what the slot already took:
	//   integral over the disc of the hex height 2h(y), h(y) = 0.346 - 0.5767|y|  = 43.725 mm³
	//   less the slot's 0.16 thickness over the disc                              = 11.310 mm³
	// The drill used to PLUG the slot here (removing -11.3) or over-cut to 70.6 on a shredded shaft.
	check("cross-hole removal", afterShaft-afterHole, 43.725-11.310)

	for _, b := range fs.Result() {
		if !b.IsSolid() {
			t.Errorf("rebuilt screw is not a solid")
		}
	}
	// Inventor's own value for this part is 234.7 mm³ (measured via COM on the real .ipt).
	check("finished screw vs Inventor", afterHole, 234.7)
}
