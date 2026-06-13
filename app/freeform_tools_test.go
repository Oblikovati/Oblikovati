// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestFreeformBoxToolEndToEnd places a sub-D box from the tool: at level 1 the cage
// subdivides but the body stays a valid closed solid.
func TestFreeformBoxToolEndToEnd(t *testing.T) {
	s, def := emptyPartSession(t)

	tool := NewFreeformBoxTool()
	s.StartTool(tool)
	if !tool.CanCommit() {
		t.Fatal("freeform box should commit with its default sizes")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after freeform box: %d bodies, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("freeform box not a valid solid: %+v", r)
	}
}

// TestFreeformPlaneToolMakesASurface places the open plane cage — a sheet, not a solid.
func TestFreeformPlaneToolMakesASurface(t *testing.T) {
	s, def := emptyPartSession(t)

	s.StartTool(NewFreeformPlaneTool())
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 || def.SurfaceBodies().Item(0).IsSolid() {
		t.Fatal("freeform plane should leave one open (surface) body")
	}
}

// TestFreeformQuadBallToolMakesASolid places the closed sphere-like cage.
func TestFreeformQuadBallToolMakesASolid(t *testing.T) {
	s, def := emptyPartSession(t)

	tool := NewFreeformQuadBallTool()
	s.StartTool(tool)
	tool.radius, tool.level = 3, 2
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("freeform quad ball not a valid solid: %+v", r)
	}
}

// TestFreeformToolsViaRibbonCommands asserts each Freeform panel command starts its tool.
func TestFreeformToolsViaRibbonCommands(t *testing.T) {
	s, _ := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	for id, want := range map[string]string{
		"Freeform.Box":      "Freeform Box",
		"Freeform.Plane":    "Freeform Plane",
		"Freeform.QuadBall": "Freeform Quad Ball",
	} {
		if err := s.Execute(id); err != nil {
			t.Fatalf("execute %s: %v", id, err)
		}
		if got := s.ActiveTool().Name(); got != want {
			t.Errorf("%s started tool %q, want %q", id, got, want)
		}
		s.CancelTool()
	}
}
