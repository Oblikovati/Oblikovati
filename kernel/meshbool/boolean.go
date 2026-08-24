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
func Boolean(a, b [][3]Point, op Op) [][3]Point {
	aOut, bOut := CoRefine(a, b)
	var result [][3]Point
	for _, f := range aOut {
		if keepFromA(op, insideMesh(centroid(f), b)) {
			result = append(result, f)
		}
	}
	for _, f := range bOut {
		if keepFromB(op, insideMesh(centroid(f), a)) {
			result = append(result, orientFromB(op, f))
		}
	}
	return result
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
