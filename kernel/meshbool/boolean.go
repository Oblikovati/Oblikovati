// SPDX-License-Identifier: GPL-2.0-only

package meshbool

// Op is a boolean set operation between two solids.
type Op int

const (
	Union        Op = iota // a ∪ b
	Difference             // a − b
	Intersection           // a ∩ b
)

// Boolean computes `a op b` as a triangle mesh. It co-refines the two operands so
// they share their intersection curve exactly, classifies each resulting face
// inside/outside the other solid by generalized winding number, and keeps and
// orients the faces the operation calls for. The result is watertight BY
// CONSTRUCTION: because the operands conform, a kept face of one and the kept face
// of the other meet exactly along the shared curve — the property whose absence
// tears #2084. PRECONDITION: a and b are closed, outward-oriented, non-degenerate
// meshes. Coplanar face pairs are conformed but their coplanar-keep selection is a
// later layer.
//
// Boolean is the untagged entry point; it delegates to [BooleanTagged] with a
// single-tag soup so the keep/coplanar/orient logic lives in exactly one place. The
// result triangles are identical (same order) to the pre-tagging implementation.
func Boolean(a, b [][3]Point, op Op) [][3]Point {
	return BooleanTagged(untagged(a), untagged(b), op).Tris
}

// keepCoplanar decides a face of a that is coincident with a face of b, by whether
// their outward normals agree. Same-direction coincidence (both solids on the same
// side of the plane) is a real boundary for Union/Intersection and internal to a
// Difference; opposite-direction coincidence (solids face-to-face) is internal to
// Union/Intersection and the retained a-boundary of a Difference. b's coincident
// copy is always dropped so the result carries exactly one face there.
func keepCoplanar(op Op, sameDir bool) bool {
	if op == Difference {
		return !sameDir
	}
	return sameDir
}

// keepFromA reports whether a face of a (inB = its centroid lies inside b) belongs
// to the result. Union/Difference keep a's faces OUTSIDE b; Intersection keeps
// those INSIDE b.
func keepFromA(op Op, inB bool) bool {
	if op == Intersection {
		return inB
	}
	return !inB
}

// keepFromB reports whether a face of b (inA = its centroid lies inside a) belongs
// to the result. Union keeps b's faces OUTSIDE a; Difference/Intersection keep
// those INSIDE a.
func keepFromB(op Op, inA bool) bool {
	if op == Union {
		return !inA
	}
	return inA
}

// orientFromB flips a kept face of b for Difference — the retained b faces bound
// the removed cavity, so their outward normal for `a − b` is reversed.
func orientFromB(op Op, f [3]Point) [3]Point {
	if op == Difference {
		return [3]Point{f[0], f[2], f[1]}
	}
	return f
}
