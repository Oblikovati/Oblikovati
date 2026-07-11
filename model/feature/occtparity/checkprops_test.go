// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

type fakeT struct{ errs, logs []string }

func (f *fakeT) Errorf(s string, a ...any) { f.errs = append(f.errs, s) }
func (f *fakeT) Logf(s string, a ...any)   { f.logs = append(f.logs, s) }

func TestAssertAreaTolerance(t *testing.T) {
	// within 1% => no error; >0.1% => a drift warning
	f := &fakeT{}
	assertArea(f, "c", 100.5, 100.0, 0.01) // 0.5% off: pass, warn
	if len(f.errs) != 0 {
		t.Fatalf("unexpected error at 0.5%%: %v", f.errs)
	}
	if len(f.logs) != 1 {
		t.Fatalf("expected drift warning at 0.5%%, got %d", len(f.logs))
	}
	// >1% => error
	f2 := &fakeT{}
	assertArea(f2, "c", 102.0, 100.0, 0.01) // 2% off
	if len(f2.errs) != 1 {
		t.Fatalf("expected error at 2%%, got %d", len(f2.errs))
	}
	// within 0.1% => no error, no warning
	f3 := &fakeT{}
	assertArea(f3, "c", 100.05, 100.0, 0.01) // 0.05% off
	if len(f3.errs) != 0 || len(f3.logs) != 0 {
		t.Fatalf("clean case warned/errored: %v %v", f3.errs, f3.logs)
	}
}
