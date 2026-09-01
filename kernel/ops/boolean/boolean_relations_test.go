// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops/query"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// TestBooleanDisjointRelations exercises the disjoint classify path through every op.
func TestBooleanDisjointRelations(t *testing.T) {
	t.Parallel()
	a := brepfixture.Box(math.P3(0, 0, 0), 1, 1, 1)
	b := brepfixture.Box(math.P3(10, 10, 10), 1, 1, 1)

	// Cut by a tool that misses leaves the target unchanged.
	if res, err := ops.Boolean(ops.Cut, a, b); err != nil || stdmath.Abs(query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume-1) > 1e-6 {
		t.Fatalf("disjoint cut = vol %v, err %v; want 1", query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume, err)
	}
	// Intersect of disjoint bodies is empty.
	if res, err := ops.Boolean(ops.Intersect, a, b); err != nil || query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume > 1e-6 {
		t.Fatalf("disjoint intersect = vol %v, err %v; want 0", query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume, err)
	}
	// Join of disjoint bodies merges both.
	if res, err := ops.Boolean(ops.Join, a, b); err != nil || stdmath.Abs(query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume-2) > 1e-6 {
		t.Fatalf("disjoint join = vol %v, err %v; want 2", query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume, err)
	}
}

// TestBooleanContainmentRelations exercises the one-contains-the-other classify paths.
func TestBooleanContainmentRelations(t *testing.T) {
	t.Parallel()
	big := brepfixture.Box(math.P3(0, 0, 0), 4, 4, 4)
	small := brepfixture.Box(math.P3(1, 1, 1), 1, 1, 1) // strictly inside big

	// Cut(small, big): the tool contains the target ⇒ everything removed.
	if res, err := ops.Boolean(ops.Cut, small, big); err != nil || query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume > 1e-6 {
		t.Fatalf("toolContainsTarget cut = vol %v, err %v; want 0", query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume, err)
	}
	// Intersect(big, small): target contains tool ⇒ the tool.
	if res, err := ops.Boolean(ops.Intersect, big, small); err != nil || stdmath.Abs(query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume-1) > 1e-6 {
		t.Fatalf("targetContainsTool intersect = vol %v, err %v; want 1", query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume, err)
	}
	// Intersect(small, big): tool contains target ⇒ the target.
	if res, err := ops.Boolean(ops.Intersect, small, big); err != nil || stdmath.Abs(query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume-1) > 1e-6 {
		t.Fatalf("toolContainsTarget intersect = vol %v, err %v; want 1", query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume, err)
	}
	// Cut(big, small): target contains tool ⇒ face-splitting cavity path runs without error.
	if _, err := ops.Boolean(ops.Cut, big, small); err != nil {
		t.Fatalf("targetContainsTool cut errored: %v", err)
	}
}

// TestBooleanNewBodyReturnsTool covers the NewBody short-circuit.
func TestBooleanNewBodyReturnsTool(t *testing.T) {
	t.Parallel()
	a := brepfixture.Box(math.P3(0, 0, 0), 1, 1, 1)
	b := brepfixture.Box(math.P3(5, 5, 5), 2, 2, 2)
	res, err := ops.Boolean(ops.NewBody, a, b)
	if err != nil || res != b {
		t.Fatalf("NewBody boolean = %p, err %v; want the tool %p", res, err, b)
	}
}
