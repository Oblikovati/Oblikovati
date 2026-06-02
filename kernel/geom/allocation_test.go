// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

// PBI-020 (the COM TransientGeometry "single construction point") is satisfied
// by the geom/math package surface itself: per architecture/core/03, value
// types are plain Go structs created by ordinary constructors, so the factory
// and its marshaling are gone. The remaining testable acceptance criterion is
// "high-volume creation stays allocation-light" — the value-type constructors
// and evaluators must not touch the heap. (Slice-backed types — Polyline,
// BSpline* — necessarily allocate their backing storage and are excluded.)
func TestValueGeometryConstructionIsAllocationFree(t *testing.T) {
	var sink float64
	avg := testing.AllocsPerRun(1000, func() {
		p := math.P3(1, 2, 3)
		v := math.V3(4, 5, 6)
		line, _ := NewLine(p, v)
		seg := NewLineSegment(p, p.TranslateBy(v))
		circle, _ := NewCircle(p, v, 2)
		arc := NewArc2d(math.P2(0, 0), 1, 0, 1)
		plane, _ := NewPlane(p, v)
		sphere, _ := NewSphere(p, 3)
		sink += line.PointAt(0.5).Z + seg.PointAt(0.5).X + circle.PointAt(0.25).Y +
			arc.PointAt(0.5).X + plane.NormalAt(0, 0).Z + sphere.PointAt(1, 0.5).X
	})
	if avg != 0 {
		t.Errorf("value geometry construction/evaluation allocated %.1f times per run, want 0", avg)
	}
	_ = sink
}
