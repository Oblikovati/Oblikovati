// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// A sketch created over the wire must open with the same projected origin the interactive
// Create 2D Sketch gives it. sketch.create used to add the sketch directly, so an add-in — or
// the MCP bridge driving a live test — got a sketch with no (0,0) anchor to constrain against,
// while the same action through the UI got one (#2016).

func TestWireCreatedSketchProjectsTheOrigin(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if got := countKind(ents.Entities, "projectedPoint"); got != 1 {
		t.Fatalf("projected points = %d, want 1 (the origin centre)", got)
	}
	if p := ents.Entities[0].Points; len(p) != 1 || p[0][0] != 0 || p[0][1] != 0 {
		t.Errorf("projected origin at %v, want (0,0)", p)
	}
}

// Turning the option off is honoured on the wire path too, so a client that does not want the
// anchor does not get one.
func TestWireCreatedSketchSkipsTheOriginWhenTheOptionIsOff(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	call(t, r, s, "options.setGroup",
		`{"group":"sketch","sketch":{"gridSpacingCm":1,"gridVisible":true,"gridMajorEvery":5,"snapToPoints":true,"snapToGrid":true,"autoProjectOrigin":false}}`, nil)

	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if got := countKind(ents.Entities, "projectedPoint"); got != 0 {
		t.Errorf("projected points = %d, want 0 with autoProjectOrigin off", got)
	}
}

// Projecting the origin centre by hand into a sketch that already carries it must not stack a
// second reference point on the first — two coincident snap targets at (0,0).
func TestProjectingTheOriginAgainDoesNotDuplicateIt(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.project", `{"sketchIndex":0,"refs":["origin/point/center"]}`,
		&wire.ProjectGeometryResult{})

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if got := countKind(ents.Entities, "projectedPoint"); got != 1 {
		t.Errorf("projected points = %d, want 1 (the same origin, projected once)", got)
	}
}
