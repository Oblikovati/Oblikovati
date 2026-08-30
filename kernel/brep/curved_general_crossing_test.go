// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Unified ruled-crossing driver helpers (ADR-0058 phase 3).

// TestCurvedSideFaceDeclinesPlanarBody: a body with no cone or cylinder side yields ok=false, so the
// unified driver declines rather than trim a non-existent curved side.
func TestCurvedSideFaceDeclinesPlanarBody(t *testing.T) {
	block, _ := SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "block")
	if _, s, _, ok := curvedSideFace(block); ok {
		t.Errorf("a planar block has no curved side; want ok=false, got surface %T", s)
	}
}

// TestCurvedSideSolidSplitRejectsUnsupportedSurface: an unsupported side surface (a plane) is a named
// error, not a silent decline — the message carries the offending type.
func TestCurvedSideSolidSplitRejectsUnsupportedSurface(t *testing.T) {
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	inside := func(math.Point3) bool { return true }
	_, _, err := curvedSideSolidSplit(curvedFace{}, pl, coneSideBand_{}, nil, Intersection, false, inside)
	if err == nil {
		t.Fatal("a plane side surface must be rejected with an error, not handled")
	}
	if !strings.Contains(err.Error(), "Plane") {
		t.Errorf("error must name the offending surface type; got %q", err.Error())
	}
}
