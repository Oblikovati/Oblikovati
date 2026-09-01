// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/model/compdef"
)

// sketch3d.edit used to mark the sketch OBJECT edited without entering the session's 3D-sketch
// environment: it answered {"editing": true} while InSketch3D stayed false, so the contextual 3D
// Sketch tab never appeared and every command gated on it stayed disabled. The 3D sketch was not
// driveable over the API at all — found by the EPIC #2053 live test, whose whole 3D ribbon read
// as inert for that reason. The planar setSketchEdit always went through the session; this is
// the same seam for 3D.
func TestEditSketch3DEntersTheSessionEnvironment(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	s.ActiveDocument().Content().(*compdef.PartComponentDefinition).Sketches3D().Add()

	var res wire.EditSketch3DResult
	call(t, r, s, wire.MethodSketch3DEdit, `{"sketchIndex":0}`, &res)
	if !res.Editing {
		t.Fatal("sketch3d.edit did not report the sketch as editing")
	}
	if !s.InSketch3D() {
		t.Error("sketch3d.edit left the session outside the 3D-sketch environment, so every " +
			"command gated on InSketch3D stays disabled")
	}

	call(t, r, s, wire.MethodSketch3DExitEdit, `{"sketchIndex":0}`, &res)
	if s.InSketch3D() {
		t.Error("sketch3d.exitEdit left the session inside the 3D-sketch environment")
	}
	if res.Editing {
		t.Error("sketch3d.exitEdit still reports the sketch as editing")
	}
}
