// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// regionPickerForBox builds a RayPicker viewing a [0,2]×[0,2]×[0,4] box head-on, plus the
// body's projected screen rectangle, so the box-select tests can frame the box precisely.
func regionPickerForBox(t *testing.T) (*RayPicker, screenRect) {
	t.Helper()
	s := extrudedBox(t, 2, 4)
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(20, 1, 2)
	cam.Target = math.P3(0, 1, 2)
	p := NewRayPicker(cam, partBodies(s))
	rect, ok := p.projectBodyRect(partBodies(s)()[0])
	if !ok {
		t.Fatal("box did not project to screen — check the camera framing")
	}
	return p, rect
}

func TestPickRegionWindowEnclosesBody(t *testing.T) {
	t.Parallel()
	p, box := regionPickerForBox(t)
	filter := NewSelectionFilter()

	// A window (non-crossing) rect that fully encloses the body's projection selects it.
	pad := 5.0
	hits := p.PickRegion(box.minX-pad, box.minY-pad, box.maxX+pad, box.maxY+pad, false, filter)
	if len(hits) != 1 {
		t.Fatalf("window enclosing the body: got %d hits, want 1", len(hits))
	}
	if _, ok := hits[0].(BodyHandle); !ok {
		t.Errorf("region hit is %T, want BodyHandle", hits[0])
	}

	// A window rect that only clips the body (does not enclose it) selects nothing.
	if got := p.PickRegion(box.minX+pad, box.minY+pad, box.maxX+pad+50, box.maxY+pad+50, false, filter); len(got) != 0 {
		t.Errorf("window that does not fully enclose the body should select nothing, got %d", len(got))
	}
}

func TestPickRegionCrossingTouchesBody(t *testing.T) {
	t.Parallel()
	p, box := regionPickerForBox(t)
	filter := NewSelectionFilter()

	// A crossing rect that merely overlaps the body's projection selects it...
	midX := (box.minX + box.maxX) / 2
	if got := p.PickRegion(midX, box.minY-5, box.maxX+100, box.maxY+100, true, filter); len(got) != 1 {
		t.Fatalf("crossing overlapping the body: got %d hits, want 1", len(got))
	}
	// ...while a crossing rect entirely off the body selects nothing.
	if got := p.PickRegion(box.maxX+50, box.maxY+50, box.maxX+100, box.maxY+100, true, filter); len(got) != 0 {
		t.Errorf("crossing clear of the body should select nothing, got %d", len(got))
	}
}

func TestPickRegionNilBodiesAndBehindCamera(t *testing.T) {
	t.Parallel()
	// No bodies provider → no region hits (the head installs one, but guard the nil case).
	pNil := NewRayPicker(scene.NewCamera(400, 400), nil)
	if got := pNil.PickRegion(0, 0, 100, 100, false, NewSelectionFilter()); got != nil {
		t.Errorf("PickRegion with no bodies provider should return nil, got %d", len(got))
	}

	// A body entirely behind the camera fails to project and is skipped (no hits), even for a
	// huge crossing rectangle.
	s := extrudedBox(t, 2, 4)
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(0, 0, -20)
	cam.Target = math.P3(0, 0, -40) // look away from the box (which sits at z∈[0,4])
	p := NewRayPicker(cam, partBodies(s))
	if got := p.PickRegion(-1e6, -1e6, 1e6, 1e6, true, NewSelectionFilter()); len(got) != 0 {
		t.Errorf("a body behind the camera should not be region-selected, got %d", len(got))
	}
}

// TestPickRegionGranularityByFilter checks the box-select granularity follows the selection
// filter: a body-permissive filter selects the whole body, a face-only filter selects its faces,
// an edge-only filter selects its edges (#909). The box [0,2]×[0,2]×[0,4] has 6 faces and 12 edges.
func TestPickRegionGranularityByFilter(t *testing.T) {
	t.Parallel()
	p, box := regionPickerForBox(t)
	pad := 5.0
	enclose := func(f *SelectionFilter) []Selectable {
		return p.PickRegion(box.minX-pad, box.minY-pad, box.maxX+pad, box.maxY+pad, false, f)
	}

	bodies := enclose(NewSelectionFilter()) // permissive (accepts all) → whole body
	if len(bodies) != 1 {
		t.Fatalf("permissive filter should select the whole body, got %d", len(bodies))
	}
	if _, ok := bodies[0].(BodyHandle); !ok {
		t.Errorf("permissive region hit = %T, want BodyHandle", bodies[0])
	}

	faces := enclose(NewSelectionFilter(SelectFace))
	if len(faces) != 6 {
		t.Errorf("face-only filter over the box = %d hits, want its 6 faces", len(faces))
	} else if _, ok := faces[0].(FaceHandle); !ok {
		t.Errorf("face region hit = %T, want FaceHandle", faces[0])
	}

	edges := enclose(NewSelectionFilter(SelectEdge))
	if len(edges) != 12 {
		t.Errorf("edge-only filter over the box = %d hits, want its 12 edges", len(edges))
	} else if _, ok := edges[0].(EdgeHandle); !ok {
		t.Errorf("edge region hit = %T, want EdgeHandle", edges[0])
	}
}

// TestPickRegionFilterAcceptsNothingRelevant checks a filter that admits none of body/face/edge
// (e.g. vertices only) yields no region hits.
func TestPickRegionFilterAcceptsNothingRelevant(t *testing.T) {
	t.Parallel()
	p, box := regionPickerForBox(t)
	vtx := NewSelectionFilter(SelectVertex)
	if got := p.PickRegion(box.minX-5, box.minY-5, box.maxX+5, box.maxY+5, false, vtx); got != nil {
		t.Errorf("a vertex-only filter should yield no region hits, got %d", len(got))
	}
}

// TestPickRegionGranularityCrossing checks the crossing mode for faces and edges: a rectangle
// over part of the box still catches the faces/edges it overlaps (which a window would miss).
func TestPickRegionGranularityCrossing(t *testing.T) {
	t.Parallel()
	p, box := regionPickerForBox(t)
	midX := (box.minX + box.maxX) / 2
	x0, y0, x1, y1 := box.minX-5, box.minY-5, midX, box.maxY+5

	if faces := p.PickRegion(x0, y0, x1, y1, true, NewSelectionFilter(SelectFace)); len(faces) == 0 {
		t.Error("crossing over part of the box should catch some faces")
	}
	if edges := p.PickRegion(x0, y0, x1, y1, true, NewSelectionFilter(SelectEdge)); len(edges) == 0 {
		t.Error("crossing over part of the box should catch some edges")
	}
}
