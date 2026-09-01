// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"testing"

	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/model/compdef"
)

// TestAdd3DEntitiesMapsPolylineAndEllipse is the regression for the dropped-geometry bug: a
// non-planar DWG used to skip every LWPOLYLINE and ELLIPSE ("no 3D mapping"), so a mostly-
// polyline drawing imported as a near-empty 3D sketch. Every entity must now be placed, with
// no warnings.
func TestAdd3DEntitiesMapsPolylineAndEllipse(t *testing.T) {
	t.Parallel()
	sk := compdef.NewPartComponentDefinition().Sketches3D().Add()
	entities := []drawing.Entity{
		&drawing.Line{Start: [3]float64{0, 0, 0}, End: [3]float64{1, 0, 5}},
		// closed bulged polyline at a non-zero elevation: 2 straight + 1 arc segment.
		&drawing.LwPolyline{Closed: true, Elevation: 3, Points: [][2]float64{{0, 0}, {2, 0}, {2, 2}}, Bulges: []float64{0, 0.5, 0}},
		// full ellipse and a partial one (-> elliptical arc).
		&drawing.Ellipse{Center: [3]float64{0, 0, 1}, MajorAxis: [3]float64{4, 0, 0}, AxisRatio: 0.5, StartAngle: 0, EndAngle: 2 * 3.14159265358979},
		&drawing.Ellipse{Center: [3]float64{5, 0, 1}, MajorAxis: [3]float64{4, 0, 0}, AxisRatio: 0.5, StartAngle: 1, EndAngle: 2.5},
	}
	added, warns := add3DEntities(sk, entities)
	if len(warns) != 0 {
		t.Fatalf("unexpected skip warnings: %v", warns)
	}
	if added != len(entities) {
		t.Errorf("added %d of %d entities", added, len(entities))
	}
	// 1 line + (closed polyline: 3 segments) + 2 ellipses = 6 sketch entities.
	if got := sk.EntityCount(); got != 6 {
		t.Errorf("sketch has %d entities, want 6 (line + 3 polyline segs + 2 ellipses)", got)
	}
}
