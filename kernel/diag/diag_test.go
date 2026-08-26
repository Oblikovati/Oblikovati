// SPDX-License-Identifier: GPL-2.0-only

package diag

import (
	"sync"
	"testing"
)

func TestRecorderCollectsInOrder(t *testing.T) {
	var r Recorder
	r.Recordf("a.one", Warning, "first %d", 1)
	r.Record(Diagnostic{Code: "a.two", Severity: Defect, Detail: "second"})

	got := r.Records()
	if len(got) != 2 {
		t.Fatalf("recorded %d, want 2", len(got))
	}
	if got[0].Code != "a.one" || got[0].Detail != "first 1" || got[0].Severity != Warning {
		t.Errorf("first diagnostic = %+v, want a.one/warning/\"first 1\"", got[0])
	}
	if got[1].Code != "a.two" || got[1].Severity != Defect {
		t.Errorf("second diagnostic = %+v, want a.two/defect", got[1])
	}
}

func TestRecorderCountAndHas(t *testing.T) {
	var r Recorder
	r.Recordf("x.warn", Warning, "w")
	r.Recordf("x.defect", Defect, "d")
	r.Recordf("x.defect2", Defect, "d2")

	if n := r.Count(Defect); n != 2 {
		t.Errorf("defect count = %d, want 2", n)
	}
	if n := r.Count(Warning); n != 1 {
		t.Errorf("warning count = %d, want 1", n)
	}
	if !r.Has("x.defect") {
		t.Error("Has(x.defect) = false, want true")
	}
	if r.Has("x.absent") {
		t.Error("Has(x.absent) = true, want false")
	}
}

// TestNilRecorderDiscards is the contract that lets an emission site stay branch-free: every method is
// safe on a nil *Recorder and reports empty.
func TestNilRecorderDiscards(t *testing.T) {
	var r *Recorder // nil
	r.Record(Diagnostic{Code: "ignored"})
	r.Recordf("ignored", Defect, "x")
	if got := r.Records(); got != nil {
		t.Errorf("nil recorder Records() = %v, want nil", got)
	}
	if r.Count(Defect) != 0 || r.Has("ignored") {
		t.Error("nil recorder reported a diagnostic it discarded")
	}
}

// TestRecorderIsConcurrencySafe records from many goroutines at once — the mesher tessellates faces in
// parallel into one shared recorder — and asserts none are lost or race.
func TestRecorderIsConcurrencySafe(t *testing.T) {
	var r Recorder
	const goroutines, each = 16, 64
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for range each {
				r.Recordf("c.parallel", Defect, "from a goroutine")
			}
		})
	}
	wg.Wait()
	if n := r.Count(Defect); n != goroutines*each {
		t.Errorf("concurrent records = %d, want %d (lost under the race)", n, goroutines*each)
	}
}

func TestDiagnosticString(t *testing.T) {
	d := Diagnostic{Code: "boolean.csg-fallback", Severity: Defect, Detail: "Cut on a cylinder operand"}
	if got, want := d.String(), "defect boolean.csg-fallback: Cut on a cylinder operand"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
