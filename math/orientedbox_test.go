// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	stdmath "math"
	"testing"
)

// axis-aligned OBB behaves exactly like an AABB.
func TestOrientedBoxAxisAlignedContains(t *testing.T) {
	b := NewOrientedBox(P3(0, 0, 0),
		mustUnit3(t, 1, 0, 0), mustUnit3(t, 0, 1, 0), mustUnit3(t, 0, 0, 1),
		[3]Scalar{2, 1, 0.5})
	if !b.Contains(P3(1.9, 0.9, 0.4)) {
		t.Error("interior point should be contained")
	}
	if b.Contains(P3(2.1, 0, 0)) {
		t.Error("point past the X half-extent should be outside")
	}
}

// A 45°-rotated box contains points its axis-aligned bound would, only along
// its own axes.
func TestOrientedBoxRotated(t *testing.T) {
	d := stdmath.Sqrt2 / 2
	b := NewOrientedBox(P3(0, 0, 0),
		mustUnit3(t, d, d, 0), mustUnit3(t, -d, d, 0), mustUnit3(t, 0, 0, 1),
		[3]Scalar{2, 1, 1})
	// A point 1.5 units along the first (diagonal) axis is comfortably inside.
	along := P3(1.5*d, 1.5*d, 0)
	if !b.Contains(along) {
		t.Errorf("%v should be inside the rotated box", along)
	}
	// The same distance along world +X pokes out the side (only ~1.41 along axis0, but off axis1).
	if b.Contains(P3(2, 0, 0)) {
		t.Error("world-axis point should fall outside the rotated box")
	}
}

func TestOrientedBoxToAABB(t *testing.T) {
	d := stdmath.Sqrt2 / 2
	b := NewOrientedBox(P3(0, 0, 0),
		mustUnit3(t, d, d, 0), mustUnit3(t, -d, d, 0), mustUnit3(t, 0, 0, 1),
		[3]Scalar{1, 1, 1})
	aabb := b.ToAABB()
	// Rotated unit square (half-diagonal √2) bounds to ±√2 in X and Y.
	if !approxEqual(aabb.Max.X, stdmath.Sqrt2, 1e-12) {
		t.Errorf("AABB max X = %v, want √2", aabb.Max.X)
	}
	if !approxEqual(aabb.Min.Z, -1, 1e-12) {
		t.Errorf("AABB min Z = %v, want -1", aabb.Min.Z)
	}
}
