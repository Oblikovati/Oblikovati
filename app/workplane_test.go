// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/feature"
	"oblikovati/model/sketch"
)

// planePicker builds a RayPicker over a part's bodies and origin planes.
func planePicker(s *Session, def *compdef.PartComponentDefinition) *RayPicker {
	return NewRayPicker(s.Camera(), func() []*topo.Body { return def.SurfaceBodies().All() }).
		WithPlanes(def.OriginPlanes)
}

func TestPickerSelectsOriginPlane(t *testing.T) {
	s, def := emptyPartSession(t) // camera looks down -Z at the origin
	p := planePicker(s, def)
	sel, ok := p.Pick(100, 100, NewSelectionFilter())
	if !ok {
		t.Fatal("center click hit nothing, expected the XY plane")
	}
	wp, isPlane := sel.(WorkPlaneHandle)
	if !isPlane || wp.Plane.Name() != "XY Plane" {
		t.Errorf("center click selected %T (%v), want the XY plane", sel, sel)
	}
}

func TestPickerPrefersFaceOverPlane(t *testing.T) {
	s, def := emptyPartSession(t)
	// Build a box on the XY plane; its top face is in front of the plane behind it.
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(-2, -2))
	c1 := sk.Points().Add(math.P2(2, -2))
	c2 := sk.Points().Add(math.P2(2, 2))
	c3 := sk.Points().Add(math.P2(-2, 2))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	p := planePicker(s, def)
	sel, ok := p.Pick(100, 100, NewSelectionFilter())
	if !ok {
		t.Fatal("center click hit nothing")
	}
	if _, isFace := sel.(FaceHandle); !isFace {
		t.Errorf("a solid in front should win the pick, got %T", sel)
	}
}

func TestPickerReturnsBodyWhenFilterWantsBodies(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(-2, -2))
	c1 := sk.Points().Add(math.P2(2, -2))
	c2 := sk.Points().Add(math.P2(2, 2))
	c3 := sk.Points().Add(math.P2(-2, 2))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	p := planePicker(s, def)

	if sel, ok := p.Pick(100, 100, NewSelectionFilter(SelectBody)); !ok {
		t.Fatal("body filter hit nothing")
	} else if _, isBody := sel.(BodyHandle); !isBody {
		t.Errorf("body filter returned %T, want BodyHandle", sel)
	}
	// A filter that admits neither the face/body in front nor work planes finds nothing.
	if _, ok := p.Pick(100, 100, NewSelectionFilter(SelectEdge)); ok {
		t.Error("edge-only filter should reject the face hit and the plane")
	}
}

func TestSetCameraSyncsPicker(t *testing.T) {
	s, def := emptyPartSession(t)
	s.SetPicker(planePicker(s, def))
	cam := s.Camera()
	cam.Eye = math.P3(0, 0, 20) // still looking down -Z, just farther
	s.SetCamera(cam)            // must propagate to the picker
	// Picking the origin still resolves the XY plane through the updated camera.
	sel, ok := s.picker.Pick(100, 100, NewSelectionFilter())
	if !ok {
		t.Fatal("pick after SetCamera hit nothing")
	}
	if wp, isPlane := sel.(WorkPlaneHandle); !isPlane || wp.Plane.Name() != "XY Plane" {
		t.Errorf("pick after SetCamera = %T, want the XY plane", sel)
	}
}

func TestPickerPlaneOutsideDisplaySquareMisses(t *testing.T) {
	s, def := emptyPartSession(t)
	p := planePicker(s, def)
	// A pixel far off-center maps far out on the plane, beyond its display square.
	if _, ok := p.Pick(1000, 1000, NewSelectionFilter()); ok {
		t.Error("a click far outside the plane's display square should miss")
	}
}

func TestBrowserHasOriginFolderWithSelectablePlanes(t *testing.T) {
	s, _ := emptyPartSession(t)
	root := BuildBrowser(s)
	var origin *BrowserNode
	for i := range root.Children {
		if root.Children[i].Kind == "origin" {
			origin = &root.Children[i]
		}
	}
	if origin == nil {
		t.Fatal("browser has no Origin folder")
	}
	// The Origin folder holds the full coordinate frame: 3 planes, 3 axes, 1 center point.
	if len(origin.Children) != 7 {
		t.Fatalf("Origin folder has %d elements, want 7 (3 planes, 3 axes, 1 point)", len(origin.Children))
	}
	kinds := map[string]int{}
	for _, n := range origin.Children {
		if n.Select == nil {
			t.Errorf("origin node %q is not selectable", n.Label)
		}
		kinds[n.Kind]++
	}
	if kinds["workplane"] != 3 || kinds["workaxis"] != 3 || kinds["workpoint"] != 1 {
		t.Errorf("Origin folder kinds = %v, want 3 planes / 3 axes / 1 point", kinds)
	}
}

func TestSelectBrowserNodeSelectsPlane(t *testing.T) {
	s, _ := emptyPartSession(t)
	origin := originFolder(BuildBrowser(s))
	xz := origin.Children[1] // XZ Plane
	s.SelectBrowserNode(xz)
	if wp := s.SelectedWorkPlane(); wp == nil || wp.Name() != "XZ Plane" {
		t.Errorf("selecting the XZ browser node gave %v, want the XZ plane", wp)
	}
}

func TestCreateSketchOnSelectedPlaneUsesSelection(t *testing.T) {
	s, _ := emptyPartSession(t)
	origin := originFolder(BuildBrowser(s))
	s.SelectBrowserNode(origin.Children[1]) // XZ Plane
	sk, err := s.CreateSketchOnSelectedPlane()
	if err != nil {
		t.Fatalf("CreateSketchOnSelectedPlane: %v", err)
	}
	// The new sketch is hosted on the XZ plane (normal has a Y component, not Z).
	if n := sk.Plane().Normal().AsVector(); n.Y == 0 {
		t.Errorf("sketch plane normal = %v, expected the XZ plane (Y normal)", n)
	}
}

func TestCreateSketchOnSelectedPlaneFallsBackToXY(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketchOnSelectedPlane() // nothing selected
	if err != nil {
		t.Fatalf("CreateSketchOnSelectedPlane: %v", err)
	}
	if n := sk.Plane().Normal().AsVector(); n.Z == 0 {
		t.Errorf("fallback sketch normal = %v, expected the XY plane (Z normal)", n)
	}
}

// originFolder returns the browser root's Origin folder node.
func originFolder(root BrowserNode) BrowserNode {
	for _, c := range root.Children {
		if c.Kind == "origin" {
			return c
		}
	}
	return BrowserNode{}
}
