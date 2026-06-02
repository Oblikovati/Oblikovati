// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	stdmath "math"
	"testing"
)

func TestVector2AccessorsAndConversions(t *testing.T) {
	v := V2(1, 2)
	if got := v.Negate(); got != (Vector2{-1, -2}) {
		t.Errorf("Negate = %v, want {-1 -2}", got)
	}
	if got := v.AsPoint(); got != (Point2{1, 2}) {
		t.Errorf("AsPoint = %v, want {1 2}", got)
	}
	if !v.IsEqualTo(V2(1, 2+1e-12), 0) {
		t.Error("near-equal vectors should compare equal")
	}
}

func TestPoint2AsVectorAndDistanceSquared(t *testing.T) {
	if got := P2(3, 4).AsVector(); got != (Vector2{3, 4}) {
		t.Errorf("AsVector = %v, want {3 4}", got)
	}
	if got := P2(0, 0).DistanceSquaredTo(P2(3, 4)); got != 25 {
		t.Errorf("DistanceSquaredTo = %v, want 25", got)
	}
}

func TestUnitVector2Accessors(t *testing.T) {
	u, err := UnitVector2FromVector(V2(0, 3))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.X() != 0 || u.Y() != 1 {
		t.Errorf("accessors = (%v,%v), want (0,1)", u.X(), u.Y())
	}
	if got := u.AngleTo(mustUnit2(t, 1, 0)); !approxEqual(got, stdmath.Pi/2, 1e-12) {
		t.Errorf("AngleTo = %v, want π/2", got)
	}
	if !u.IsEqualTo(mustUnit2(t, 0, 1), 0) {
		t.Error("equal unit vectors should compare equal")
	}
}

func TestUnitVector3Accessors(t *testing.T) {
	u := mustUnit3(t, 0, 0, 5)
	if u.X() != 0 || u.Y() != 0 || u.Z() != 1 {
		t.Errorf("accessors = (%v,%v,%v), want (0,0,1)", u.X(), u.Y(), u.Z())
	}
	if got := u.Negate(); got.Z() != -1 {
		t.Errorf("Negate Z = %v, want -1", got.Z())
	}
}

func TestMatrix4AtAndCells(t *testing.T) {
	m := Translation4(V3(7, 8, 9))
	if got := m.At(0, 3); got != 7 {
		t.Errorf("At(0,3) = %v, want 7", got)
	}
	if got := m.Cells()[11]; got != 9 {
		t.Errorf("Cells[11] = %v, want 9", got)
	}
}

func TestMatrix3AccessorsAndCoordinateSystem(t *testing.T) {
	m := CoordinateSystem3(P2(1, 0), V2(0, 1), V2(-1, 0))
	if got := m.TransformPoint(P2(0, 0)); !got.IsEqualTo(P2(1, 0), 1e-12) {
		t.Errorf("origin maps to %v, want {1 0}", got)
	}
	if got := m.TransformVector(V2(1, 0)); !got.IsEqualTo(V2(0, 1), 1e-12) {
		t.Errorf("local X maps to %v, want {0 1}", got)
	}
	if got := m.At(0, 2); got != 1 {
		t.Errorf("At(0,2) = %v, want 1", got)
	}
	if m.Cells()[8] != 1 {
		t.Errorf("Cells[8] = %v, want 1", m.Cells()[8])
	}
	if got := Translation3(V2(2, 3)).Translation(); got != (Vector2{2, 3}) {
		t.Errorf("Translation = %v, want {2 3}", got)
	}
}

func TestExplicitToleranceOverride(t *testing.T) {
	// A loose explicit tolerance accepts a gap the default would reject.
	if !P3(0, 0, 0).IsEqualTo(P3(0.01, 0, 0), 0.1) {
		t.Error("explicit loose tolerance should accept 0.01 gap")
	}
	if V2(1, 0).IsEqualTo(V2(1.5, 0), 0.1) {
		t.Error("explicit tolerance should still reject 0.5 gap")
	}
}
