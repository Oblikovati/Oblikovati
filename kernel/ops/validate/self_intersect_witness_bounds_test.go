// SPDX-License-Identifier: GPL-2.0-only

package validate

import (
	"testing"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Two invariants of the exact scan that a reported crossing has to satisfy, both broken on the OCCT
// blend-parity corpus (Oblikovati/Oblikovati#3477):
//
//  1. A witness lies on BOTH trimmed faces, so it lies in both faces' range boxes. The scan sampled a
//     BOUNDED intersection curve over its whole domain — probe.SampleRange narrows the parameter interval
//     only for the unbounded plane-pair line — so a closed curve two curved faces meet on was sampled
//     far outside either of them, and only the trim test stood between the scan and a fabricated
//     crossing. On corpus simple/B3 the trim test on a torus and a sphere patch answered the far side
//     of their own surface and three "crossings" were reported at points outside both boxes.
//  2. Faces that touch TANGENTIALLY are in contact, not interpenetrating, and how far apart two
//     independently built faces read is a Sew() question (ADR-0042). Read at Weld() instead, marcher
//     noise of a few times 1e-7 on a blend running tangent to the face it blends passed the gate and
//     put geom's SSI seed field on a NURBS surface — corpus bfuseblend/B5 did not return in nine
//     minutes.

// TestSelfIntersectionWitnessLiesInBothFaceBoxes: whatever the scan reports, the witness is a point
// both faces own, so both range boxes contain it.
func TestSelfIntersectionWitnessLiesInBothFaceBoxes(t *testing.T) {
	t.Parallel()
	a := brepfixture.Tetra(1, math.V3(0, 0, 0))
	b := brepfixture.Tetra(1, math.V3(0.2, 0.2, 0.2))
	merged := topo.MergeBodies(topo.NewLineage(topo.Tok("imp", "body", 0)), true, a, b)
	hits := SelfIntersections(merged, mesh.DefaultQuality())
	if len(hits) == 0 {
		t.Fatal("interpenetrating shells must report self-intersections")
	}
	for _, h := range hits {
		if !h.FaceA.RangeBox().Contains(h.Witness) || !h.FaceB.RangeBox().Contains(h.Witness) {
			t.Errorf("witness %v is outside a reported face's own range box (A %v, B %v)",
				h.Witness, h.FaceA.RangeBox(), h.FaceB.RangeBox())
		}
	}
}

// grazeDepth is how far the probe reaches past the face in the tolerance-class fixture: three decades
// above float noise, and below the Sew() of a body this size — the band a tangent contact's marcher
// residue lands in.
const grazeDepth = 1e-6

// TestProbeInsideMaterialSeparatesTheToleranceClasses pins the class of the material-overlap gate. A
// probe reaching grazeDepth past a face is INSIDE it at Weld() (float noise) and OUTSIDE it at Sew()
// (independent computations), and Sew() is what two separately built faces have to be compared at.
func TestProbeInsideMaterialSeparatesTheToleranceClasses(t *testing.T) {
	t.Parallel()
	p := math.P3
	wall := brepfixture.QuadBody("wall", p(0, 0, 0), p(2, 0, 0), p(2, 0, 2), p(0, 0, 2)).Faces()[0]
	res := geom.ResolutionForBox(wall.RangeBox())
	if res.Weld() >= grazeDepth || res.Sew() <= grazeDepth {
		t.Fatalf("the fixture no longer straddles the classes: Weld()=%g Sew()=%g against %g",
			res.Weld(), res.Sew(), grazeDepth)
	}
	near := []math.Point3{p(1, grazeDepth, 1), p(1, -grazeDepth, 1)} // one side is the material side
	if !probeInsideMaterial(wall, near, res.Weld()) {
		t.Error("a probe grazeDepth past the wall must read as inside its material at Weld()")
	}
	if probeInsideMaterial(wall, near, res.Sew()) {
		t.Error("the same probe must read as CONTACT at Sew(): that depth is not interpenetration")
	}
}
