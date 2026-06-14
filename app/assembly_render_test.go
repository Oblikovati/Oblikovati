// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/scene"
)

// assemblyWithBoxComponent builds a box part [0,2]×[0,2]×[0,4] and places it into a fresh
// assembly at +10 on X, returning the session (assembly active), the assembly definition, and the
// placed occurrence — the fixture for the viewport rendering + picking tests (#769).
func assemblyWithBoxComponent(t *testing.T, tx float64) (*Session, *compdef.AssemblyComponentDefinition, *occurrence.Occurrence) {
	t.Helper()
	s := extrudedBox(t, 2, 4) // active part = box [0,2]×[0,2]×[0,4]
	boxDoc := s.ActiveDocument()
	asmDoc, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(asmDoc); err != nil {
		t.Fatalf("activate assembly: %v", err)
	}
	asm := asmDoc.Content().(*compdef.AssemblyComponentDefinition)
	occ, err := asm.PlaceComponentFromFile(asmDoc, boxDoc, "box:1", math.Translation4(math.V3(math.Scalar(tx), 0, 0)))
	if err != nil {
		t.Fatalf("place box: %v", err)
	}
	return s, asm, occ
}

// TestVisibleBodiesTransformsPlacedComponents: an assembly's VisibleBodies are its components
// transformed into assembly space — a box placed at +10 on X has its range box shifted there, so
// the viewport (and picker) see the placed geometry, not the component at its own origin (#769).
func TestVisibleBodiesTransformsPlacedComponents(t *testing.T) {
	s, _, _ := assemblyWithBoxComponent(t, 10)

	bodies := s.VisibleBodies()
	if len(bodies) != 1 {
		t.Fatalf("assembly VisibleBodies = %d, want 1 placed component body", len(bodies))
	}
	rb := bodies[0].RangeBox()
	if got := float64(rb.Min.X); got < 9.99 || got > 10.01 {
		t.Errorf("placed body min X = %g, want 10 (box origin 0 + placement 10)", got)
	}
	if got := float64(rb.Max.X); got < 11.99 || got > 12.01 {
		t.Errorf("placed body max X = %g, want 12 (10 + 2 side)", got)
	}
	if got := float64(rb.Max.Z); got < 3.99 || got > 4.01 {
		t.Errorf("placed body max Z = %g, want 4 (the extrude height, unmoved on Z)", got)
	}
}

// TestVisibleBodiesEmptyForUnplacedAssembly: an assembly with no components renders nothing.
func TestVisibleBodiesEmptyForUnplacedAssembly(t *testing.T) {
	s := NewSession()
	asmDoc, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	_ = s.Workspace().SetActiveDocument(asmDoc)
	if got := len(s.VisibleBodies()); got != 0 {
		t.Errorf("empty assembly VisibleBodies = %d, want 0", got)
	}
}

// TestOccurrenceOfBodyMapsBackToComponent: each world-space assembly body resolves to the
// occurrence it was placed from, so a viewport click on a component selects that occurrence.
func TestOccurrenceOfBodyMapsBackToComponent(t *testing.T) {
	s, _, occ := assemblyWithBoxComponent(t, 10)
	bodies := s.VisibleBodies()
	got, ok := s.OccurrenceOfBody(bodies[0])
	if !ok || got != occ {
		t.Errorf("OccurrenceOfBody = (%v, %v), want the placed occurrence", got, ok)
	}
}

// TestAssemblyBodyCacheIsStableUntilEdited: VisibleBodies returns the SAME body pointers between
// calls (so the head's per-body tessellation cache keeps hitting), and rebuilds when the
// occurrence structure changes.
func TestAssemblyBodyCacheIsStableUntilEdited(t *testing.T) {
	s, asm, _ := assemblyWithBoxComponent(t, 10)
	first := s.VisibleBodies()
	second := s.VisibleBodies()
	if len(first) != 1 || first[0] != second[0] {
		t.Fatal("VisibleBodies should return cached, stable body pointers between unedited calls")
	}
	// Placing another component bumps the occurrence revision → the cache rebuilds with more bodies.
	box2 := s.Workspace().Documents()[0] // the box part document
	if _, err := asm.PlaceComponentFromFile(s.ActiveDocument(), box2, "box:2", math.Translation4(math.V3(20, 0, 0))); err != nil {
		t.Fatalf("place second: %v", err)
	}
	if got := len(s.VisibleBodies()); got != 2 {
		t.Errorf("after a second placement VisibleBodies = %d, want 2 (cache rebuilt)", got)
	}
}

// TestViewportClickSelectsOccurrence: a click on a placed component's body selects the OCCURRENCE
// by default (component-level selection), and a face-filtered pick yields the face instead — the
// occurrence vs face precedence the picker resolves (#769).
func TestViewportClickSelectsOccurrence(t *testing.T) {
	s, _, occ := assemblyWithBoxComponent(t, 10)

	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(11, 1, 20) // above the placed box (x∈[10,12], y∈[0,2]), looking down
	cam.Target = math.P3(11, 1, 0)
	cam.Up = math.V3(0, 1, 0)
	picker := NewRayPicker(cam, func() []*topo.Body { return s.VisibleBodies() }).
		WithOccurrenceLookup(s.OccurrenceOfBody)

	// Default (all-accepting) filter: the whole component is selected.
	sel, ok := picker.Pick(200, 200, NewSelectionFilter())
	if !ok {
		t.Fatal("a center click should hit the placed component")
	}
	oh, ok := sel.(OccurrenceHandle)
	if !ok || oh.Occurrence != occ {
		t.Fatalf("default click selected %T, want the OccurrenceHandle for box:1", sel)
	}

	// A face filter (what a machining tool sets) bypasses occurrence selection and picks the face.
	faceSel, ok := picker.Pick(200, 200, NewSelectionFilter(SelectFace))
	if !ok {
		t.Fatal("a face-filtered click should hit the top cap")
	}
	if _, ok := faceSel.(FaceHandle); !ok {
		t.Errorf("face-filtered click selected %T, want FaceHandle", faceSel)
	}
}
