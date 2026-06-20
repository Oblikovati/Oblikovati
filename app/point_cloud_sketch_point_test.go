// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// TestCreateSketchPointAtSelectedCloudPoint: inside a sketch, with a snapped scan point selected,
// the command adds a sketch point at the scan point projected onto the sketch plane; outside a
// sketch or with nothing selected it is disabled and errors (#645).
func TestCreateSketchPointAtSelectedCloudPoint(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(3, 4, 7)})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}

	// Not in a sketch yet → disabled and errors even with a scan point selected.
	s.Select(PointCloudPointHandle{Cloud: pc, Point: math.P3(3, 4, 7)})
	if canSketchPointAtCloudPoint(s) {
		t.Error("Sketch Point should be disabled outside a sketch")
	}
	if _, err := s.CreateSketchPointAtSelectedCloudPoint(); err == nil {
		t.Error("expected an error outside a sketch")
	}

	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	// Entering the sketch clears the selection; re-select the snapped scan point.
	s.Select(PointCloudPointHandle{Cloud: pc, Point: math.P3(3, 4, 7)})
	if !canSketchPointAtCloudPoint(s) {
		t.Error("Sketch Point should be enabled in a sketch with a scan point selected")
	}

	before := sk.Points().Count()
	p, err := s.CreateSketchPointAtSelectedCloudPoint()
	if err != nil {
		t.Fatalf("CreateSketchPointAtSelectedCloudPoint: %v", err)
	}
	if sk.Points().Count() != before+1 {
		t.Errorf("sketch point count = %d, want %d", sk.Points().Count(), before+1)
	}
	// On the XY plane the scan point (3,4,7) projects to sketch (3,4).
	if got := p.Position(); got != (math.P2(3, 4)) {
		t.Errorf("sketch point at %v, want (3,4) — (3,4,7) projected onto XY", got)
	}
}
