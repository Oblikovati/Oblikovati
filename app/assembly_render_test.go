// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/material"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/scene"
)

// assemblyWithBoxComponent builds a box part [0,2]×[0,2]×[0,4] and places it into a fresh
// assembly at +10 on X, returning the session (assembly active), the assembly definition, and the
// placed occurrence — the fixture for the viewport rendering + picking tests (#769).
func assemblyWithBoxComponent(t *testing.T, tx float64) (*Session, *compdef.AssemblyComponentDefinition, *occurrence.Occurrence) {
	t.Helper()
	s, asm, boxDoc, asmDoc := emptyBoxAssembly(t)
	occ, err := asm.PlaceComponentFromFile(asmDoc, boxDoc, "box:1", math.Translation4(math.V3(math.Scalar(tx), 0, 0)))
	if err != nil {
		t.Fatalf("place box: %v", err)
	}
	return s, asm, occ
}

// emptyBoxAssembly builds a box part [0,2]×[0,2]×[0,4] and a fresh active assembly that can host
// instances of it, returning the session, the assembly definition, the box part document, and the
// assembly document — the shared fixture for the one- and two-instance assembly tests.
func emptyBoxAssembly(t *testing.T) (*Session, *compdef.AssemblyComponentDefinition, *doc.Document, *doc.Document) {
	t.Helper()
	s := extrudedBox(t, 2, 4)
	boxDoc := s.ActiveDocument()
	asmDoc, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(asmDoc); err != nil {
		t.Fatalf("activate assembly: %v", err)
	}
	return s, asmDoc.Content().(*compdef.AssemblyComponentDefinition), boxDoc, asmDoc
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

// TestAssemblySurfaceLookupResolvesOccurrenceAppearance locks the #1103 fix: an active
// assembly's SurfaceLookup resolves each placed body to ITS source part's assigned
// appearance, not the neutral default. Before the fix SurfaceLookup returned nil for an
// assembly, so every occurrence rendered flat default grey.
func TestAssemblySurfaceLookupResolvesOccurrenceAppearance(t *testing.T) {
	s, asm, boxDoc, asmDoc := emptyBoxAssembly(t)
	part := boxDoc.Content().(*compdef.PartComponentDefinition)
	body := part.SurfaceBodies().All()[0]
	part.Assignments().SetBodyAppearance(material.RefKey(body.ReferenceKey()), "steel")
	if _, err := asm.PlaceComponentFromFile(asmDoc, boxDoc, "box:1", math.Translation4(math.V3(5, 0, 0))); err != nil {
		t.Fatalf("place box: %v", err)
	}

	lookup := s.SurfaceLookup() // assembly is active
	if lookup == nil {
		t.Fatal("assembly SurfaceLookup is nil — every occurrence would render default grey (#1103)")
	}
	steel, ok := s.Materials().Appearance("steel")
	if !ok {
		t.Fatal("steel appearance missing from the seeded library")
	}
	got := lookup(body)
	want := appearanceSurface(steel)
	if got.Albedo != want.Albedo || got.Metallic != want.Metallic {
		t.Errorf("placed body resolved to %+v, want the steel appearance (per-occurrence material, #1103)", got)
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

// TestVisibleInstancesGroupsRepeatedComponents pins ADR-0038's dedup: the same component placed
// several times collapses to ONE instance group (one source mesh) with one transform per placement,
// instead of N independent world bodies — so the renderer tessellates/uploads it once.
func TestVisibleInstancesGroupsRepeatedComponents(t *testing.T) {
	s, asm, _ := assemblyWithBoxComponent(t, 10)
	boxDoc := s.Workspace().Documents()[0]
	for i := 2; i <= 5; i++ { // four more copies of the SAME box part
		if _, err := asm.PlaceComponentFromFile(s.ActiveDocument(), boxDoc,
			fmt.Sprintf("box:%d", i), math.Translation4(math.V3(math.Scalar(10*i), 0, 0))); err != nil {
			t.Fatalf("place copy %d: %v", i, err)
		}
	}
	if got := len(s.VisibleBodies()); got != 5 {
		t.Fatalf("VisibleBodies = %d, want 5 (one world body per placement)", got)
	}
	groups := s.VisibleInstances()
	if len(groups) != 1 {
		t.Fatalf("VisibleInstances = %d groups, want 1 (all five copies share one source mesh)", len(groups))
	}
	if got := len(groups[0].Transforms); got != 5 {
		t.Errorf("group has %d transforms, want 5 (one per placement)", got)
	}
}

// TestVisibleInstancesPartIsIdentity: a plain part yields one identity-transform group per body, so
// the part path and the assembly path share one instanced renderer (the K=1 case).
func TestVisibleInstancesPartIsIdentity(t *testing.T) {
	s := extrudedBox(t, 2, 4)
	groups := s.VisibleInstances()
	if len(groups) != 1 || len(groups[0].Transforms) != 1 {
		t.Fatalf("part VisibleInstances = %d groups (transforms %v), want 1 group with 1 identity transform",
			len(groups), groupTransformCounts(groups))
	}
	if !groups[0].Transforms[0].IsEqualTo(math.Identity4(), 1e-9) {
		t.Errorf("part instance transform = %v, want identity", groups[0].Transforms[0])
	}
}

func groupTransformCounts(groups []InstanceGroup) []int {
	out := make([]int, len(groups))
	for i, g := range groups {
		out[i] = len(g.Transforms)
	}
	return out
}
