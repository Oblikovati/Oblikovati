// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// Guards for the SSI corrector non-termination guard (M48/C3, Oblikovati/Oblikovati#3477). The guard
// exists so a continuation that cannot resolve a pair STOPS rather than running without bound; it is
// not a tuning knob, and these tests pin both halves of that: the accounting works, and the guard sits
// far enough above the hardest legitimate trace that it cannot bite one.

// TestSSICorrectorGuardCountsAndStops covers the accounting directly: charge spends budget and reports
// whether the trace may continue, and the tracer reads exhausted only once the cap is passed. The
// counter is shared through a pointer because the tracer is copied by value into every sweep — a
// per-copy counter would silently never reach the cap, which is the bug this pins.
func TestSSICorrectorGuardCountsAndStops(t *testing.T) {
	spent := ssiMaxCorrections - 2
	tr := ssiTracer{spent: &spent}
	if !tr.charge() || tr.exhausted() {
		t.Fatalf("with %d of %d spent the trace must continue", spent, ssiMaxCorrections)
	}
	if !tr.charge() || tr.exhausted() {
		t.Fatalf("the charge landing exactly on the cap must still be allowed (spent=%d)", spent)
	}
	if tr.charge() {
		t.Error("the charge past the cap must refuse")
	}
	if !tr.exhausted() {
		t.Error("a tracer past its cap must report exhausted")
	}
	copied := tr // a sweep takes the tracer BY VALUE; the counter must still be shared
	if !copied.exhausted() {
		t.Error("a copied tracer must see the same spend, or no sweep can ever reach the cap")
	}
}

// TestSSICorrectorGuardIgnoresATracerWithNoBudget: the zero tracer some unit tests build carries no
// counter, and must trace rather than divide by a nil budget.
func TestSSICorrectorGuardIgnoresATracerWithNoBudget(t *testing.T) {
	var tr ssiTracer
	if !tr.charge() || tr.exhausted() {
		t.Error("a tracer with no budget installed must never be charged or exhausted")
	}
}

// TestSSICorrectorGuardLeavesTheHardestLegitimateTraceUntouched is the falsifiable half, and the one
// that stops the constant being lowered to make some slow body finish. Torus ∩ plane is the most
// corrector-hungry case in this package that RESOLVES — measured at 17334 corrections against a
// kernel/ops corpus ceiling of 921 — so it must trace to completion with the guard silent. Falsify by
// dropping ssiMaxCorrections near the measured ceiling: this reports a declined trace and goes red.
func TestSSICorrectorGuardLeavesTheHardestLegitimateTraceUntouched(t *testing.T) {
	torus, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 10, 3)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	pl, err := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	traced := TraceSurfaceIntersection(torus, pl, SurfaceGrid{})
	if traced.Declined {
		t.Fatal("the hardest resolving trace in this package must not hit the non-termination guard")
	}
	if len(traced.Curves) == 0 {
		t.Fatal("torus ∩ plane through the tube must trace its section curves")
	}
}

// TestAnOrdinaryPairIsNotReportedAsDeclined is the false-positive half of the guard: the decline is a
// separate flag precisely so a caller never has to infer one from an empty curve set, and the common
// case must leave it clear. A sphere's equator traces in a handful of corrections.
func TestAnOrdinaryPairIsNotReportedAsDeclined(t *testing.T) {
	sp, err := NewSphere(math.P3(0, 0, 0), 5)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	pl, err := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	if traced := TraceSurfaceIntersection(sp, pl, SurfaceGrid{}); traced.Declined {
		t.Error("an ordinary sphere/plane equator must not read as declined")
	}
}
