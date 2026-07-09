// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestBuildExtrusionShellSheetIsOpen: with caps=false buildExtrusionShell builds an OPEN,
// non-solid wall sheet — the four side walls of a 2×2 square, no start/end caps — the tool for a
// Surface-operation extrude (Inventor kSurfaceOperation, #1858). The same profile with caps=true
// is the pre-existing closed solid prism (6 faces), proving the refactor left the solid path
// intact.
func TestBuildExtrusionShellSheetIsOpen(t *testing.T) {
	poly := []math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}

	sheet := buildExtrusionShell(poly, sketch.XYPlane(), span{near: 0, far: 3}, 0, "s", false)
	if sheet.IsSolid() {
		t.Error("caps-off shell is solid, want an open sheet")
	}
	if got := len(sheet.Faces()); got != 4 {
		t.Errorf("sheet has %d faces, want 4 walls (no caps)", got)
	}

	solid := buildExtrusionShell(poly, sketch.XYPlane(), span{near: 0, far: 3}, 0, "s", true)
	if !solid.IsSolid() || len(solid.Faces()) != 6 {
		t.Errorf("caps-on shell = solid?%v with %d faces, want a solid with 6", solid.IsSolid(), len(solid.Faces()))
	}
}
