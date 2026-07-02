// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// sketchRegionFixture builds a sketch viewed head-on by a 200×200 camera centred on the origin,
// plus a projector from sketch coordinates to screen pixels (matching pickSketchRegion).
func sketchRegionFixture(t *testing.T) (*Session, *sketch.Sketch, func(math.Point2) (float64, float64)) {
	t.Helper()
	s, sk := sketchSession(t)
	s.Grid().SnapToGrid = false
	cam := scene.NewCamera(200, 200)
	cam.Eye, cam.Target, cam.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)
	plane := sk.Plane()
	proj := func(p math.Point2) (float64, float64) {
		x, y, _ := renderer.Project(s.Camera(), regionNear, regionFar, plane.ToModel(p))
		return x, y
	}
	return s, sk, proj
}

func TestPickSketchRegionWindowEnclosesEntities(t *testing.T) {
	s, sk, proj := sketchRegionFixture(t)
	ln := sk.Lines().AddByTwoPoints(math.P2(-1, 0), math.P2(1, 0))
	sk.Points().Add(math.P2(40, 40)) // far off to the side

	ax, ay := proj(math.P2(-1, 0))
	bx, by := proj(math.P2(1, 0))
	pad := 6.0
	x0, y0 := min(ax, bx)-pad, min(ay, by)-pad
	x1, y1 := max(ax, bx)+pad, max(ay, by)+pad

	hits := s.pickSketchRegion(x0, y0, x1, y1, false) // window
	if len(hits) != 1 {
		t.Fatalf("window enclosing the line should select exactly it, got %d hits", len(hits))
	}
	if h, ok := hits[0].(SketchEntityHandle); !ok || h.Entity != ln {
		t.Errorf("window hit = %v, want the line", hits[0])
	}
}

func TestPickSketchRegionCrossingCatchesPartialLine(t *testing.T) {
	s, sk, proj := sketchRegionFixture(t)
	ln := sk.Lines().AddByTwoPoints(math.P2(-2, 0), math.P2(2, 0))

	// A rectangle around the line's MIDDLE: both endpoints are outside it, so a window misses,
	// but the segment crosses it → crossing selects it.
	mx, my := proj(math.P2(0, 0))
	x0, y0, x1, y1 := mx-5, my-5, mx+5, my+5

	if got := s.pickSketchRegion(x0, y0, x1, y1, false); len(got) != 0 {
		t.Fatalf("window over the line's middle should select nothing (endpoints outside), got %d", len(got))
	}
	hits := s.pickSketchRegion(x0, y0, x1, y1, true) // crossing
	if len(hits) != 1 {
		t.Fatalf("crossing over the line's middle should select it, got %d", len(hits))
	}
	if h := hits[0].(SketchEntityHandle); h.Entity != ln {
		t.Errorf("crossing hit = %v, want the line", hits[0])
	}
}

func TestPickSketchRegionNoActiveSketch(t *testing.T) {
	if got := NewSession().pickSketchRegion(0, 0, 10, 10, false); got != nil {
		t.Errorf("pickSketchRegion with no active sketch should return nil, got %d", len(got))
	}
}

// TestBoxSelectStateMachineInSketch drives the box-select state machine in a sketch (no
// RegionPicker installed) and checks BeginBoxSelect is allowed and CommitBoxSelect routes to the
// sketch region pick, selecting the enclosed entity.
func TestBoxSelectStateMachineInSketch(t *testing.T) {
	s, sk, proj := sketchRegionFixture(t)
	ln := sk.Lines().AddByTwoPoints(math.P2(-1, 0), math.P2(1, 0))
	ax, ay := proj(math.P2(-1, 0))
	bx, by := proj(math.P2(1, 0))

	s.BeginBoxSelect(min(ax, bx)-6, min(ay, by)-6) // editing a sketch, so this is allowed
	if !s.BoxSelectActive() {
		t.Fatal("BeginBoxSelect should start a box in the sketch editor (no RegionPicker needed)")
	}
	s.UpdateBoxSelect(max(ax, bx)+6, max(ay, by)+6) // L→R window enclosing the line
	if n := s.CommitBoxSelect(0); n != 1 {
		t.Fatalf("commit should select the enclosed line, got %d hits", n)
	}
	if s.Selection().Count() != 1 || !s.Selection().Contains(SketchEntityHandle{Entity: ln}) {
		t.Errorf("the line should be the selection after the box: count=%d", s.Selection().Count())
	}
}
