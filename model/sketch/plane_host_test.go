// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati/math"
)

// TestSketchTracksPlaneHost verifies a sketch with a plane host re-reads it on
// RefreshPlane — the mechanism that carries a sketch with a moving work plane.
func TestSketchTracksPlaneHost(t *testing.T) {
	s := NewSketches().Add(XYPlane())

	xAxis, _ := math.NewUnitVector3(1, 0, 0)
	yAxis, _ := math.NewUnitVector3(0, 1, 0)
	z := math.Scalar(0)
	host := func() Plane {
		p, _ := NewPlane(math.P3(0, 0, z), xAxis, yAxis)
		return p
	}
	s.SetPlaneHost(host)
	if got := s.Plane().Origin().Z; got != 0 {
		t.Fatalf("initial plane origin Z = %v, want 0", got)
	}

	z = 5 // the host (a work plane) moved up
	s.RefreshPlane()
	if got := s.Plane().Origin().Z; got != 5 {
		t.Errorf("after host moved, plane origin Z = %v, want 5 (sketch did not track)", got)
	}
}
