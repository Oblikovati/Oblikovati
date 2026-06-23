// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

func TestEditControlPointsLiftsNurbsFace(t *testing.T) {
	body := surfaceFaceBody(t, multiSpanPatch(t)) // helpers in rebuild_faces_test.go
	out, err := ops.EditControlPoints(body, []geom.ControlPointDelta{{U: 2, V: 2, Delta: math.V3(0, 0, 1.5)}})
	if err != nil {
		t.Fatalf("EditControlPoints: %v", err)
	}
	bs, ok := out.Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("edited face geometry is %T, want geom.BSplineSurface", out.Faces()[0].Geometry())
	}
	// The control net keeps its shape (same dims/degree); only a control position moved.
	if len(bs.Ctrl) != len(multiSpanPatch(t).Ctrl) {
		t.Errorf("net rows changed to %d", len(bs.Ctrl))
	}
}

func TestEditControlPointsErrorsWithoutNurbsFace(t *testing.T) {
	box := csgBox(math.P3(0, 0, 0), 1, 1, 1) // planar faces only
	if _, err := ops.EditControlPoints(box, []geom.ControlPointDelta{{U: 0, V: 0, Delta: math.V3(0, 0, 1)}}); err == nil {
		t.Error("a body with no NURBS face should error")
	}
}

func TestEditControlPointsRejectsOutOfRange(t *testing.T) {
	body := surfaceFaceBody(t, multiSpanPatch(t))
	if _, err := ops.EditControlPoints(body, []geom.ControlPointDelta{{U: 99, V: 0, Delta: math.V3(0, 0, 1)}}); err == nil {
		t.Error("an out-of-range control index should error")
	}
}
