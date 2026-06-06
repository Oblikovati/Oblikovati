// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati/math"
)

func TestLinePointAndDomain(t *testing.T) {
	l, err := NewLine(math.P3(1, 0, 0), math.V3(0, 0, 2)) // dir normalized to +Z
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := l.PointAt(3); !got.IsEqualTo(math.P3(1, 0, 3), eqScalar) {
		t.Errorf("PointAt(3) = %v, want {1 0 3}", got)
	}
	if got := l.TangentAt(3); !got.IsEqualTo(math.V3(0, 0, 1), eqScalar) {
		t.Errorf("TangentAt = %v, want unit +Z", got)
	}
	lo, hi := l.Domain()
	if !stdmath.IsInf(lo, -1) || !stdmath.IsInf(hi, 1) {
		t.Errorf("Domain = [%v,%v], want unbounded", lo, hi)
	}
}

func TestNewLineZeroDirFails(t *testing.T) {
	if _, err := NewLine(math.P3(0, 0, 0), math.V3(0, 0, 0)); err == nil {
		t.Fatal("expected error for zero direction")
	}
}

func TestLineSegmentEvaluation(t *testing.T) {
	s := NewLineSegment(math.P3(0, 0, 0), math.P3(3, 4, 0))
	if got := s.PointAt(0); got != (math.Point3{}) {
		t.Errorf("PointAt(0) = %v, want origin", got)
	}
	if got := s.PointAt(1); !got.IsEqualTo(math.P3(3, 4, 0), eqScalar) {
		t.Errorf("PointAt(1) = %v, want {3 4 0}", got)
	}
	if got := s.PointAt(0.5); !got.IsEqualTo(math.P3(1.5, 2, 0), eqScalar) {
		t.Errorf("PointAt(0.5) = %v, want {1.5 2 0}", got)
	}
	approxScalar(t, s.Length(), 5, "Length")
}

func TestLine2dThroughAndSegment(t *testing.T) {
	l, err := Line2dThrough(math.P2(0, 0), math.P2(0, 5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := l.PointAt(2); !got.IsEqualTo(math.P2(0, 2), eqScalar) {
		t.Errorf("PointAt(2) = %v, want {0 2}", got)
	}
	seg := NewLineSegment2d(math.P2(0, 0), math.P2(6, 8))
	approxScalar(t, seg.Length(), 10, "Length2d")
}
