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

// TestTheDeclineSaysWhichGateRefused: a ring driven across a rod demotes for CONDITIONING, and the
// operation reports both the generic decline and the gate behind it.
func TestTheDeclineSaysWhichGateRefused(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(1, 0, 0), 6, 1.5, "ring")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	rod, err := brep.SolidCylinder(math.P3(0, 0, -10), math.V3(0, 0, 1), 3, 20)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	rec := &diag.Recorder{}
	if _, err := BooleanWithDiagnostics(Cut, rod, ring, rec); err == nil {
		t.Fatal("the ring-on-rod cut built; the fixture no longer exercises the decline")
	}
	assertRecorded(t, rec, CodeBooleanNoExactCurvedPath, "no exact analytic path")
	assertRecorded(t, rec, brep.CodeSectionConditioningDemotion, "the torus section's")
}

// TestAnUnclaimedPairSaysSoOnTheSameRecorder: the other example of #3525 — a torus pair, which no
// closed form claims — reaches the caller by name too, so the two refusals no longer read alike.
func TestAnUnclaimedPairSaysSoOnTheSameRecorder(t *testing.T) {
	t.Parallel()
	a, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 4, 1, "a")
	if err != nil {
		t.Fatalf("SolidTorus a: %v", err)
	}
	b, err := brep.SolidTorus(math.P3(4, 0, 0), math.V3(1, 0, 0), 4, 1, "b")
	if err != nil {
		t.Fatalf("SolidTorus b: %v", err)
	}
	rec := &diag.Recorder{}
	if _, err := BooleanWithDiagnostics(Join, a, b, rec); err == nil {
		t.Fatal("the torus pair built; the fixture no longer exercises the decline")
	}
	assertRecorded(t, rec, brep.CodeSectionUnclaimedPair, "geom.Torus ∩ geom.Torus")
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
