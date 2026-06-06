// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"
)

// TestThreadCatalog pins the standards/sizes/pitches and the parseable designations they build.
func TestThreadCatalog(t *testing.T) {
	if got := ThreadStandards(); len(got) != 3 {
		t.Fatalf("standards = %v, want 3 (ISO/ANSI/JIS)", got)
	}
	if StandardSystem(StandardANSI) != SystemImperial || StandardSystem(StandardISO) != SystemMetric || StandardSystem(StandardJIS) != SystemMetric {
		t.Error("system mapping wrong (ANSI imperial; ISO/JIS metric)")
	}
	if len(ThreadSizes(StandardISO)) == 0 || len(ThreadSizes(StandardANSI)) == 0 {
		t.Fatal("empty size tables")
	}

	// Metric designation round-trips through the parser.
	dm, err := ThreadDesignation(StandardISO, "M8", 1.25)
	if err != nil || dm != "M8x1.25" {
		t.Fatalf("metric designation = %q, %v", dm, err)
	}
	if s, _ := ParseThreadDesignation(dm); s.MajorDiameter != 8 || s.Pitch != 1.25 {
		t.Errorf("M8x1.25 parsed = %+v", s)
	}

	// Imperial: 1/4 @ 20 TPI → "1/4-20", major 6.35 mm, pitch 1.27 mm.
	di, err := ThreadDesignation(StandardANSI, "1/4", inchPerMM/20)
	if err != nil || di != "1/4-20" {
		t.Fatalf("imperial designation = %q, %v", di, err)
	}
	si, err := ParseThreadDesignation(di)
	if err != nil || stdmath.Abs(si.MajorDiameter-6.35) > 1e-6 || stdmath.Abs(si.Pitch-1.27) > 1e-6 {
		t.Errorf("1/4-20 parsed = %+v, %v", si, err)
	}

	// Numbered gauge.
	if sg, err := ParseThreadDesignation("#8-32"); err != nil || stdmath.Abs(sg.MajorDiameter-0.164*inchPerMM) > 1e-6 {
		t.Errorf("#8-32 parsed = %+v, %v", sg, err)
	}

	// A pitch a size does not offer is rejected.
	if _, err := ThreadDesignation(StandardISO, "M8", 99); err == nil {
		t.Error("an unavailable pitch should error")
	}
}
