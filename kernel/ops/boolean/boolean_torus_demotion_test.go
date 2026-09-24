// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The torus bucket's CONDITIONING DEMOTION reaching the caller as a defect. Split out of
// boolean_torus_skew_test.go, which reached the 500-line limit (Oblikovati/Oblikovati#3515, review
// round 4, Minor-1).
// TestALaneConditioningDemotionIsReported: a rod laid TANGENT to the top of the ring's tube touches it
// along the ring's own top circle and crosses it nowhere. The station's azimuths meet there without
// separating, so the reduction's azimuth census cannot account for the set it would build, and it
// refuses. That refusal is the kind that has to be said out loud: the closed form APPLIED to this pair
// and gave up ground it normally holds, which is a fallback, and a fallback is a diag.Defect that
// reaches feature health, the API and the UI.
//
// The fixture used to be a rod FATTER than the tube. That is no longer a refusal — its four
// full-period branches are built, and skewCorpusFatRod certifies the bodies (Oblikovati/Oblikovati#3515)
// — so the guard is re-pointed at a section that is genuinely ill-conditioned rather than merely one
// the reduction had not been taught. Widening a fast path and quietly losing its demotion would trade
// one defect for a worse one.
//
// The counterpart matters as much: a rod the reduction DOES carry must record nothing. A diagnostic
// that fires on the ordinary case is noise, and "no closed form claims this pair" — a torus against a
// torus, say — is the ordinary case.
func TestALaneConditioningDemotionIsReported(t *testing.T) {
	t.Parallel()
	ring := skewCorpusRing(t)
	// Radius 2 on an axis 3.5 above the ring's plane puts the rod's lowest ruling at z = 1.5, which is
	// exactly the height of the tube's top circle: one tangential touch, no crossing.
	grazing, err := brep.SolidCylinder(math.P3(0, 0, 3.5), math.V3(1, 0, 0), 2, 9)
	if err != nil {
		t.Fatalf("grazing rod: %v", err)
	}
	for _, op := range []ops.PartFeatureOperation{ops.Cut, ops.Join} {
		var rec diag.Recorder
		if _, err := ops.BooleanWithDiagnostics(op, ring, grazing, &rec); err == nil {
			t.Fatalf("%v: an unnameable section must be refused, not built", op)
		}
		if !rec.Has(brep.CodeSectionConditioningDemotion) {
			t.Errorf("%v: the demotion recorded no %q; got %v", op, brep.CodeSectionConditioningDemotion, rec.Records())
		}
		if rec.Count(diag.Defect) == 0 {
			t.Errorf("%v: a conditioning demotion must be a Defect", op)
		}
		// And it names what this pair IS. "The section's curves do not add up" would describe the kernel;
		// these two surfaces touch (Oblikovati/Oblikovati#3515, review round 1).
		assertADiagnosticNames(t, &rec, "touches the torus's tube circle without crossing it")
	}
	assertTheCarriedRodsRecordNoDemotion(t, ring)
}

// assertADiagnosticNames fails unless some record on the recorder carries the phrase.
func assertADiagnosticNames(t *testing.T, rec *diag.Recorder, want string) {
	t.Helper()
	for _, d := range rec.Records() {
		if strings.Contains(d.Detail, want) {
			return
		}
	}
	t.Errorf("no diagnostic names %q; the recorder holds %v", want, rec.Records())
}

// assertTheCarriedRodsRecordNoDemotion requires both rods the reduction carries — the one that pierces
// the tube and the one that swallows it — to build with no demotion recorded at all.
func assertTheCarriedRodsRecordNoDemotion(t *testing.T, ring *topo.Body) {
	t.Helper()
	for _, tool := range []skewRingTool{skewCorpusRod(t), skewCorpusFatRod(t)} {
		var quiet diag.Recorder
		if _, err := ops.BooleanWithDiagnostics(ops.Cut, ring, tool.body, &quiet); err != nil {
			t.Fatalf("the %s the reduction carries must build: %v", tool.name, err)
		}
		if quiet.Has(brep.CodeSectionConditioningDemotion) {
			t.Errorf("%s: a section the closed form named recorded a demotion: %v", tool.name, quiet.Records())
		}
	}
}
