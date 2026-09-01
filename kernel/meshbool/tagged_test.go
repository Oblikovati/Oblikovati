// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "testing"

// cubeTagged returns the axis-aligned cube [x0,x0+2]³ as a tagged soup whose six
// faces each carry a distinct tag (base+0 .. base+5), so a test can assert which
// original face a kept output triangle descends from. boxMesh emits the six faces in
// a fixed order, two triangles each, so triangle i belongs to face i/2.
func cubeTagged(x0, y0, z0 float64, base int) TaggedSoup {
	c := boxMesh([3]float64{x0, y0, z0}, [3]float64{x0 + 2, y0 + 2, z0 + 2})
	var s TaggedSoup
	for i, tri := range c {
		s.add(tri, base+i/2)
	}
	return s
}

// TestBooleanTaggedMatchesUntagged proves the tagged path is a strict refactor: the
// result triangles are identical (same set, same order) to the untagged Boolean for
// every operation, so delegating Boolean through BooleanTagged changed nothing.
func TestBooleanTaggedMatchesUntagged(t *testing.T) {
	t.Parallel()
	a := boxMesh([3]float64{0, 0, 0}, [3]float64{2, 2, 2})
	b := boxMesh([3]float64{1, 1, 1}, [3]float64{3, 3, 3})
	for _, op := range []Op{Union, Difference, Intersection} {
		want := Boolean(a, b, op)
		got := BooleanTagged(untagged(a), untagged(b), op)
		if len(got.Tris) != len(want) {
			t.Fatalf("op %d: tagged tri count %d != untagged %d", op, len(got.Tris), len(want))
		}
		if len(got.Tags) != len(got.Tris) {
			t.Fatalf("op %d: len(Tags)=%d != len(Tris)=%d — invariant broken", op, len(got.Tags), len(got.Tris))
		}
		for i := range want {
			if !triEqual(got.Tris[i], want[i]) {
				t.Fatalf("op %d tri %d: tagged %v != untagged %v", op, i, got.Tris[i], want[i])
			}
		}
	}
}

// TestBooleanTaggedProvenance proves every kept triangle carries a tag that traces to
// exactly one originating operand face, and that a's tags and b's tags stay disjoint
// (b's ids are offset past a's), so reconstruction can attribute each output triangle
// to a known surface. The union of two offset cubes keeps faces from both operands.
func TestBooleanTaggedProvenance(t *testing.T) {
	t.Parallel()
	const naFaces = 6
	a := cubeTagged(0, 0, 0, 0)
	b := cubeTagged(1, 1, 1, naFaces)
	got := BooleanTagged(a, b, Union)
	if len(got.Tris) == 0 {
		t.Fatal("union produced no triangles")
	}
	var fromA, fromB int
	for i, tag := range got.Tags {
		if tag < 0 || tag >= 2*naFaces {
			t.Fatalf("tri %d: tag %d out of range [0,%d)", i, tag, 2*naFaces)
		}
		if tag < naFaces {
			fromA++
		} else {
			fromB++
		}
	}
	if fromA == 0 || fromB == 0 {
		t.Fatalf("union must keep faces from both operands: fromA=%d fromB=%d", fromA, fromB)
	}
}

// TestBooleanTaggedDifferenceKeepsBTag proves a cavity wall (a kept, reversed b face
// for Difference) keeps b's tag — the reversed face still belongs to b's surface, the
// invariant reconstruction relies on to give the cavity wall b's analytic surface.
func TestBooleanTaggedDifferenceKeepsBTag(t *testing.T) {
	t.Parallel()
	const naFaces = 6
	a := cubeTagged(0, 0, 0, 0)
	b := cubeTagged(1, 1, 1, naFaces)
	got := BooleanTagged(a, b, Difference)
	var sawB bool
	for _, tag := range got.Tags {
		if tag >= naFaces {
			sawB = true
		}
	}
	if !sawB {
		t.Fatal("a−b must retain b faces as cavity walls, each with a b tag")
	}
}

func triEqual(a, b [3]Point) bool {
	return a[0].Equal(b[0]) && a[1].Equal(b[1]) && a[2].Equal(b[2])
}
