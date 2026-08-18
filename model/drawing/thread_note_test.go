// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// threadedCylinderBody returns a solid cylinder (radius cm, 5 cm tall along +Z) whose lateral face is
// retyped to a machined thread carrying the given designation — the same retype the thread feature
// performs, so a hole note reads a real threaded face, not a stub.
func threadedCylinderBody(t *testing.T, radius float64, designation string, rightHanded bool) *topo.Body {
	t.Helper()
	body, err := brep.SolidCylinder(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), radius, 5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	var key []byte
	var cyl geom.Cylinder
	for _, f := range body.Faces() {
		if c, ok := f.Geometry().(geom.Cylinder); ok {
			key, cyl = f.ReferenceKey(), c
			break
		}
	}
	if key == nil {
		t.Fatal("no cylindrical face on the solid cylinder")
	}
	threaded := geom.ThreadedCylinder{
		Cylinder: cyl, Pitch: 0.125, Depth: 0.05, Designation: designation,
		RightHanded: rightHanded, VMin: 0, VMax: 5,
	}
	tb, err := ops.ReplaceFaceSurface(body, key, threaded)
	if err != nil {
		t.Fatalf("ReplaceFaceSurface: %v", err)
	}
	return tb
}

// TestThreadDesignationText: the callout is the authored designation for a right-handed thread, and
// a left-handed thread appends "- LH" unless it already carries it; an empty designation stays empty.
func TestThreadDesignationText(t *testing.T) {
	base := geom.ThreadedCylinder{Designation: "M8x1.25", RightHanded: true}
	if got := threadDesignationText(base); got != "M8x1.25" {
		t.Errorf("right-handed callout = %q, want M8x1.25", got)
	}
	left := geom.ThreadedCylinder{Designation: "M8x1.25", RightHanded: false}
	if got := threadDesignationText(left); got != "M8x1.25 - LH" {
		t.Errorf("left-handed callout = %q, want M8x1.25 - LH", got)
	}
	already := geom.ThreadedCylinder{Designation: "M8x1.25-LH", RightHanded: false}
	if got := threadDesignationText(already); got != "M8x1.25-LH" {
		t.Errorf("callout = %q, want the designation unchanged (LH not doubled)", got)
	}
	if got := threadDesignationText(geom.ThreadedCylinder{RightHanded: false}); got != "" {
		t.Errorf("empty-designation callout = %q, want empty", got)
	}
}

// TestThreadAtMatchesByCentre: a callout matches a hole recovered near its projected axis and not one
// a full diameter away.
func TestThreadAtMatchesByCentre(t *testing.T) {
	threads := []threadCallout{{center: gmath.P2(3, 4), designation: "M6x1"}}
	if got := threadAt(threads, gmath.P2(3.02, 3.98), 0.4); got != "M6x1" {
		t.Errorf("near hole = %q, want M6x1", got)
	}
	if got := threadAt(threads, gmath.P2(3.5, 4), 0.4); got != "" {
		t.Errorf("far hole = %q, want plain (no match)", got)
	}
}

// TestThreadCalloutsFromBody: a threaded cylinder projects one callout whose designation is read off
// the surface, anchored at the axis (the origin projected into the TOP view).
func TestThreadCalloutsFromBody(t *testing.T) {
	body := threadedCylinderBody(t, 2, "M8x1.25", true)
	basis := baseBasis(types.BaseViewTop, bodyCenter(body))
	callouts := threadCalloutsFrom(body, basis)
	if len(callouts) != 1 || callouts[0].designation != "M8x1.25" {
		t.Fatalf("thread callouts = %+v, want one M8x1.25", callouts)
	}
}

// TestHoleNoteRendersThreadDesignation: a tapped hole's note reads as its thread designation and the
// note reports one threaded hole, while a plain cylinder still reads Ø<d> with no threaded holes.
func TestHoleNoteRendersThreadDesignation(t *testing.T) {
	c := drawingWithCylinder(t, 2) // plain Ø40
	topBase(t, c.Sheets().Active().Views())
	hn, err := c.Sheets().Active().Annotations().AddHoleNotes("HN", "TOP", types.HoleNotePerHole, "")
	if err != nil {
		t.Fatalf("AddHoleNotes: %v", err)
	}
	if hn.Labels()[0].Text != "Ø40.00" || hn.ThreadCount() != 0 {
		t.Fatalf("plain note = %q (%d threaded), want Ø40.00 / 0", hn.Labels()[0].Text, hn.ThreadCount())
	}

	// Retype the model's wall to a machined thread and recompute: the same note now reads the
	// designation and counts one threaded hole (associative to the model).
	c.SetBodyResolver(fakeBodyResolver{body: threadedCylinderBody(t, 2, "M8x1.25", true)})
	c.RecomputeViews()
	if hn.Labels()[0].Text != "M8x1.25" || hn.ThreadCount() != 1 {
		t.Errorf("threaded note = %q (%d threaded), want M8x1.25 / 1", hn.Labels()[0].Text, hn.ThreadCount())
	}
}
