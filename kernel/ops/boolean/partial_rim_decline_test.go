// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/math"
)

// Partial-rim chained-cut decline (EPIC Oblikovati/Oblikovati#1724, ADR-0046). A SECOND curved cut on an
// already-cut target — whose cylinder side now carries a section-arc boundary from the first cut (a notched
// rim), so it is no longer a two-full-circle band — is the partial-rim case the recognizer does NOT build an
// exact solid for (composing a pre-existing boundary with a new SSI imprint is a coupled (u,v)-arrangement
// build, tracked separately). The EPIC's correctness bar is that such a still-declining config declines
// OBSERVABLY: it must fall to the recorded CSG fallback, never panic or error. This pins that bar (and guards
// the orphan-vertex CSG panic fix at the integration level — the chain used to crash here).
// This asserts the DECLINE CODE, never the faceted body: when the configuration this declines on
// lands analytically it becomes a positive corpus case (ADR-0061), so the assertion converts
// rather than being deleted to move a number.
func TestPartialRimChainedCutDeclinesObservably(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3s): `make test-corpus`")
	}
	t.Parallel()
	s := 1 / stdmath.Sqrt2
	target, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	// First cut: a rim-crossing tool that NOTCHES the top rim, leaving a partial-rim side (section-arc boundary).
	tool1, err := brep.SolidCylinder(math.P3(-5.6, 0, 2), math.V3(math.Scalar(s), 0, math.Scalar(s)), 0.9, 16)
	if err != nil {
		t.Fatalf("tool1: %v", err)
	}
	cut1, err := ops.Boolean(ops.Cut, target, tool1)
	if err != nil {
		t.Fatalf("first cut: %v", err)
	}
	// Second cut on the partial-rim body: a transverse drill through the now-notched side.
	tool2, err := brep.SolidCylinder(math.P3(-6, 0, 4), math.V3(1, 0, 0), 1.0, 12)
	if err != nil {
		t.Fatalf("tool2: %v", err)
	}
	rec := &diag.Recorder{}
	cut2, err := ops.BooleanWithDiagnostics(ops.Cut, cut1, tool2, rec)
	if err != nil {
		t.Fatalf("partial-rim second cut must not error (observable decline, not a hard failure): %v", err)
	}
	if cut2 == nil {
		t.Fatal("partial-rim second cut returned nil body")
	}
	if !rec.Has(ops.CodeBooleanCSGFallback) {
		t.Error("partial-rim second cut did not record ops.CodeBooleanCSGFallback — the decline must be observable (#1724)")
	}
}
