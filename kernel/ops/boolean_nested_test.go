// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// Regression for Oblikovati/Oblikovati#1316: Join/Intersect of nested bodies used to concatenate
// shells (MergeBodies), leaving the inner shell as a floating interior wall — an invalid solid whose
// reported volume double-counted the inner body. The union of nested solids is just the outer body.

// nestedCubes returns a large outer cube and a small inner cube strictly inside it, both offset by
// `shift` so the divergence-sum origin sensitivity is exercised (translation invariance).
func nestedCubes(t *testing.T, shift float64) (outer, inner *topo.Body) {
	t.Helper()
	s := m.Scalar(shift)
	var err error
	outer, err = brep.SolidBlock(m.P3(s, s, s), m.P3(s+10, s+10, s+10), "outer")
	if err != nil {
		t.Fatalf("outer: %v", err)
	}
	inner, err = brep.SolidBlock(m.P3(s+3, s+3, s+3), m.P3(s+6, s+6, s+6), "inner")
	if err != nil {
		t.Fatalf("inner: %v", err)
	}
	return outer, inner
}

// TestJoinNestedReturnsOuterOnly asserts Join(outer, inner) with inner ⊂ outer is exactly the outer
// body: outer's volume (1000, not 1027), outer's six faces (no interior wall), and a valid solid.
func TestJoinNestedReturnsOuterOnly(t *testing.T) {
	t.Parallel()
	outer, inner := nestedCubes(t, 0)
	joined, err := Boolean(Join, outer, inner)
	if err != nil {
		t.Fatalf("Boolean(Join): %v", err)
	}
	if got := query.BodyGeometryProperties(joined, DefaultQuality()).Volume; math.Abs(got-1000) > 1e-6 {
		t.Errorf("union volume = %g, want 1000 (no double-counted inner shell)", got)
	}
	if nf := len(joined.Faces()); nf != 6 {
		t.Errorf("union face count = %d, want 6 (no interior wall)", nf)
	}
	if !Validate(joined).ValidSolid() {
		t.Errorf("union is not a valid closed manifold solid")
	}
	if r := Validate(joined); !r.Closed || !r.Manifold {
		t.Errorf("union not closed/manifold: %+v", r)
	}
}

// TestJoinNestedToolContainsTargetReturnsTool covers the mirror branch: when the TOOL contains the
// target, the union is the tool. Volume is the outer cube's (1000), with no interior wall.
func TestJoinNestedToolContainsTargetReturnsTool(t *testing.T) {
	t.Parallel()
	outer, inner := nestedCubes(t, 0)
	joined, err := Boolean(Join, inner, outer) // target=inner (small), tool=outer (large)
	if err != nil {
		t.Fatalf("Boolean(Join): %v", err)
	}
	if got := query.BodyGeometryProperties(joined, DefaultQuality()).Volume; math.Abs(got-1000) > 1e-6 {
		t.Errorf("union volume = %g, want 1000", got)
	}
	if nf := len(joined.Faces()); nf != 6 {
		t.Errorf("union face count = %d, want 6", nf)
	}
}

// TestIntersectNestedReturnsInner asserts Intersect(outer, inner) with inner ⊂ outer is the inner
// body's geometry (volume 27).
func TestIntersectNestedReturnsInner(t *testing.T) {
	t.Parallel()
	outer, inner := nestedCubes(t, 0)
	got, err := Boolean(Intersect, outer, inner)
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	if v := query.BodyGeometryProperties(got, DefaultQuality()).Volume; math.Abs(v-27) > 1e-6 {
		t.Errorf("intersection volume = %g, want 27", v)
	}
	if !Validate(got).ValidSolid() {
		t.Errorf("intersection is not a valid closed manifold solid")
	}
}

// TestJoinNestedTranslationInvariant repeats the nested Join far from the origin: the result volume
// must be unchanged, guarding both the shell-discard fix and the divergence-sum origin handling.
func TestJoinNestedTranslationInvariant(t *testing.T) {
	t.Parallel()
	for _, shift := range []float64{0, 500, -250} {
		outer, inner := nestedCubes(t, shift)
		joined, err := Boolean(Join, outer, inner)
		if err != nil {
			t.Fatalf("shift %g: Boolean(Join): %v", shift, err)
		}
		if got := query.BodyGeometryProperties(joined, DefaultQuality()).Volume; math.Abs(got-1000) > 1e-6 {
			t.Errorf("shift %g: union volume = %g, want 1000", shift, got)
		}
	}
}

// TestCutNestedToolInsideRemovesCavity confirms Cut still routes correctly: cutting an interior tool
// from the target produces a body with a cavity (volume 1000-27 = 973) and remains valid.
func TestCutNestedToolInsideRemovesCavity(t *testing.T) {
	t.Parallel()
	outer, inner := nestedCubes(t, 0)
	got, err := Boolean(Cut, outer, inner)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if v := query.BodyGeometryProperties(got, DefaultQuality()).Volume; math.Abs(v-973) > 1e-6 {
		t.Errorf("cut volume = %g, want 973 (outer minus interior cavity)", v)
	}
}
