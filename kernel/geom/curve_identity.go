// SPDX-License-Identifier: GPL-2.0-only

package geom

import "reflect"

// SameCurveObject reports whether two curves are ONE curve object — the same dynamic type and value,
// or the same pointer — and never panics doing it. `a == b` on two Curve3 interfaces panics at run time
// when both hold the same UNCOMPARABLE dynamic type: a value Polyline or BSplineCurve carries slices,
// and the kernel stores a marched section curve as a value Polyline on the edge that carries it
// (MarchedDeviation), so a boolean chained on a boolean's result reaches an identity test with two of
// them. Such a value has no identity to compare, and the answer is false: it is not the same object.
//
// The guard reads the VALUE, not the type: a struct type with a Curve3 field is comparable as a type
// while a value of it holding a Polyline is not (SubCurve over a marched section), and only the value
// says which. Both nil is one (absent) curve.
//
// Example: if geom.SameCurveObject(prev.curve, next.curve) { /* one run of one curve */ }
func SameCurveObject(a, b Curve3) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if !reflect.ValueOf(a).Comparable() || !reflect.ValueOf(b).Comparable() {
		return false
	}
	return a == b
}
