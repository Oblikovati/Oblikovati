// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// SameCurveObject must answer identity for every curve kind the kernel stores on an edge and panic on
// none of them — `==` on two value Polylines is a run-time panic ("comparing uncomparable type").
func TestSameCurveObjectNeverPanicsAndReadsIdentity(t *testing.T) {
	t.Parallel()
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0)}
	byValue, again := Polyline{Vertices: pts}, Polyline{Vertices: pts}
	if SameCurveObject(byValue, again) {
		t.Error("two value polylines have no identity to share; they must read as different objects")
	}
	byPointer := &Polyline{Vertices: pts}
	if !SameCurveObject(byPointer, byPointer) {
		t.Error("one pointer polyline is not read as itself")
	}
	if SameCurveObject(byPointer, &Polyline{Vertices: pts}) {
		t.Error("two pointer polylines of equal content are two objects")
	}
	seg, other := NewLineSegment(pts[0], pts[1]), NewLineSegment(pts[0], pts[2])
	if !SameCurveObject(seg, seg) || SameCurveObject(seg, other) || SameCurveObject(seg, byValue) {
		t.Error("comparable curves must compare by value, and a different kind is never the same object")
	}
	if SameCurveObject(SubCurve(byValue, 0, 0.5), SubCurve(byValue, 0, 0.5)) {
		t.Error("a sub-curve over a value polyline is uncomparable by VALUE though comparable by type")
	}
	if !SameCurveObject(nil, nil) || SameCurveObject(nil, seg) {
		t.Error("two absent curves are one absence; an absent and a present one are not")
	}
}
