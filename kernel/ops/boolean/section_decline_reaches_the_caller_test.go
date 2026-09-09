// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/math"
)

// The boolean's own decline says only "no exact analytic path claims this configuration", which is the
// same sentence for every refusal there is. Oblikovati/Oblikovati#3525: the recorder the caller threaded
// in — the one a feature reply reads, and with it the API and the UI — must also carry WHICH gate
// refused, on the same operation.

// TestTheDeclineSaysWhichGateRefused: a ring sitting inside a rod whose wall is TANGENT to the ring's
// outer equator demotes for CONDITIONING — their surfaces touch and never cross, so the section's
// azimuths meet without separating — and the operation reports both the generic decline and the gate
// behind it. The fixture used to be a rod merely thicker than the ring's tube, which is built now
// (Oblikovati/Oblikovati#3515) and so no longer drives this gate.
func TestTheDeclineSaysWhichGateRefused(t *testing.T) {
	t.Parallel()
	const major, minor = 6.0, 1.5
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(1, 0, 0), major, minor, "ring")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	rod, err := brep.SolidCylinder(math.P3(0, 0, -10), math.V3(0, 0, 1), major+minor, 20)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	rec := &diag.Recorder{}
	if _, err := BooleanWithDiagnostics(Cut, rod, ring, rec); err == nil {
		t.Fatal("the grazing ring-in-rod cut built; the fixture no longer exercises the decline")
	}
	assertRecorded(t, rec, CodeBooleanNoExactCurvedPath, "no exact analytic path")
	assertRecorded(t, rec, brep.CodeSectionConditioningDemotion, "geom.Cylinder cylinder:f#2 ∩ geom.Torus ring:face#0")
	assertRecorded(t, rec, brep.CodeSectionConditioningDemotion, "touches the torus's tube circle without crossing it")
}

// TestATorusPairsRefusalNamesItsOwnGate: the other example of #3525. A torus PAIR's refusal has to
// reach the caller naming the TORUS PAIR's faces and the torus pair's own gate, so it does not read like
// the ring-on-rod's above.
//
// The row asserted CodeSectionUnclaimedPair until ADR-0066 (#3514) claimed the torus pair. That code's
// route now has no fixture at all in the kernel's primitive vocabulary — a sweep of {torus, sphere,
// block, cylinder, cone} against each other in all three operations reaches it from nothing — so the
// ordinary-refusal routing is tested where the decision is made, in kernel/brep's
// TestAnOrdinaryRefusalIsRecordedAsInfo, and what is left to test end to end is this: the refusal that
// DOES happen carries its own gate's sentence to the caller's recorder.
func TestATorusPairsRefusalNamesItsOwnGate(t *testing.T) {
	t.Parallel()
	a, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "a")
	if err != nil {
		t.Fatalf("SolidTorus a: %v", err)
	}
	b, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(1, 0, 0), 5, 1.5, "b")
	if err != nil {
		t.Fatalf("SolidTorus b: %v", err)
	}
	rec := &diag.Recorder{}
	if _, err := BooleanWithDiagnostics(Join, a, b, rec); err == nil {
		t.Fatal("the torus pair built; the fixture no longer exercises the decline")
	}
	assertRecorded(t, rec, brep.CodeSectionConditioningDemotion, "geom.Torus a:face#0 ∩ geom.Torus b:face#0")
	assertRecorded(t, rec, brep.CodeSectionConditioningDemotion, "do not satisfy the form it was solved from")
}

// TestARefusalIsReportedOnce: one boolean asks the same face pair up to four times — each pairing runs
// in both operand orders, and booleanGeneralExact enters brep.BooleanDiag twice — so a single refusal
// reached a user four identical times. A fix whose whole subject is what a user reads must not print
// itself four times (#3525, review round 1).
func TestARefusalIsReportedOnce(t *testing.T) {
	t.Parallel()
	a, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "a")
	if err != nil {
		t.Fatalf("SolidTorus a: %v", err)
	}
	b, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(1, 0, 0), 5, 1.5, "b")
	if err != nil {
		t.Fatalf("SolidTorus b: %v", err)
	}
	rec := &diag.Recorder{}
	if _, err := BooleanWithDiagnostics(Join, a, b, rec); err == nil {
		t.Fatal("the torus pair built; the fixture no longer exercises the decline")
	}
	seen := map[string]int{}
	for _, d := range rec.Records() {
		seen[string(d.Code)+": "+d.Detail]++
	}
	for line, n := range seen {
		if n > 1 {
			t.Errorf("one refusal was reported %d times: %s", n, line)
		}
	}
}

// assertRecorded fails unless the recorder carries a diagnostic of this code whose detail contains want.
func assertRecorded(t *testing.T, rec *diag.Recorder, code diag.Code, want string) {
	t.Helper()
	for _, d := range rec.Records() {
		if d.Code == code && strings.Contains(d.Detail, want) {
			return
		}
	}
	t.Errorf("no %s diagnostic naming %q; the recorder holds %v", code, want, rec.Records())
}
