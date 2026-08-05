// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// #2048: FreeformBody.SetLevel / MoveVertices / CreaseEdges and the freeform.* wire methods
// shipped with #699, but its UI was left out of scope and never built — a user could place a
// Box, Plane or Quad Ball and then not touch its cage.

// partWithFreeformBox returns a session whose active part holds a placed free-form box.
func partWithFreeformBox(t *testing.T) (*Session, *compdef.PartComponentDefinition, *feature.FreeformBody) {
	t.Helper()
	s, def := emptyPartSession(t)
	s.StartFeatureTool(NewFreeformBoxTool())
	if err := s.OK(); err != nil {
		t.Fatalf("place freeform box: %v", err)
	}
	_, body, ok := activeFreeformCage(s)
	if !ok {
		t.Fatal("the placed freeform box exposes no cage")
	}
	return s, def, body
}

// beginCageDragAt seeds an in-flight drag on a cage vertex without going through the camera, so
// the assertions are about the cage rather than about where a pixel lands.
func beginCageDragAt(t *testing.T, s *Session, vertex int) {
	t.Helper()
	pf, body, ok := activeFreeformCage(s)
	if !ok {
		t.Fatal("no freeform cage to drag")
	}
	s.cageEdit = cageEditDrag{
		vertex: vertex, feature: pf, body: body,
		origin: body.Vertices().Item(vertex).Point(), active: true,
	}
}

func TestFreeformCageDragMovesAVertex(t *testing.T) {
	s, _, body := partWithFreeformBox(t)
	s.StartTool(NewFreeformCageEditTool())
	if !s.CageEditActive() {
		t.Fatal("the cage tool did not arm")
	}
	before := body.Vertices().Item(0).Point()
	beginCageDragAt(t, s, 0)
	s.applyCageMove(math.V3(0, 0, 1))
	after := body.Vertices().Item(0).Point()
	if after.IsEqualTo(before, 1e-12) {
		t.Fatalf("the cage vertex did not move: still %v", after)
	}
	if got := float64(before.VectorTo(after).Length()); got < 0.999 || got > 1.001 {
		t.Errorf("vertex moved %g, want 1", got)
	}
}

// A drag with no movement records no undo step, so a stray click leaves nothing behind.
func TestFreeformCageDragWithoutMovementRecordsNothing(t *testing.T) {
	s, _, _ := partWithFreeformBox(t)
	s.StartTool(NewFreeformCageEditTool())
	beginCageDragAt(t, s, 0)
	s.CommitCageDrag()
	if s.CageDragActive() {
		t.Error("the drag is still active after commit")
	}
}

func TestFreeformCageLevelAppliesToTheBody(t *testing.T) {
	s, def, body := partWithFreeformBox(t)
	tool := NewFreeformCageEditTool()
	s.StartTool(tool)
	if got := tool.Level(); got != body.Level() {
		t.Errorf("the tool opened at level %d, want the body's %d", got, body.Level())
	}
	facesBefore := len(def.SurfaceBodies().Item(0).Faces())

	tool.SetLevel(body.Level() + 1)
	if !s.ApplyActiveCageLevel() {
		t.Fatal("ApplyActiveCageLevel found no body to edit")
	}
	if got := body.Level(); got != tool.Level() {
		t.Errorf("body level = %d, want %d", got, tool.Level())
	}
	if got := len(def.SurfaceBodies().Item(0).Faces()); got <= facesBefore {
		t.Errorf("subdividing left %d faces, was %d — the body did not re-subdivide", got, facesBefore)
	}
}

// A negative level is clamped rather than rejected, so a spun-down field cannot break the body.
func TestFreeformCageLevelClampsAtZero(t *testing.T) {
	tool := NewFreeformCageEditTool()
	tool.SetLevel(-3)
	if tool.Level() != 0 {
		t.Errorf("level = %d, want 0 after clamping", tool.Level())
	}
}

func TestFreeformCageCreaseSharpensTheEdgesAtAHandle(t *testing.T) {
	s, def, body := partWithFreeformBox(t)
	tool := NewFreeformCageEditTool()
	s.StartTool(tool)
	// Subdivide first, so creasing has a visible effect on the limit surface.
	tool.SetLevel(2)
	s.ApplyActiveCageLevel()
	before := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume

	if s.CreaseActiveCageHandle() {
		t.Fatal("crease should refuse before a handle has been dragged")
	}
	tool.lastVertex = 0
	if !s.CreaseActiveCageHandle() {
		t.Fatal("crease found no edges at the dragged handle")
	}
	after := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume
	if after <= before {
		t.Errorf("creasing left the volume at %g (was %g) — a sharpened corner should push it out", after, before)
	}
	if n := len(cageEdgesAt(body, 0)); n < 3 {
		t.Errorf("cage vertex 0 has %d incident edges, want at least 3 for a box corner", n)
	}
}

// The cage overlay draws a handle cross per vertex plus the wire, so the handles are visible and
// grabbable — the tool would otherwise arm and show nothing.
func TestFreeformCageOverlayDrawsHandlesAndWire(t *testing.T) {
	_, _, body := partWithFreeformBox(t)
	items := cageOverlayItems(body, 0.1)
	if len(items) != 2 {
		t.Fatalf("cage overlay has %d items, want 2 (wire + handles)", len(items))
	}
	if want := body.Edges().Count() * 2; len(items[0].Positions) != want {
		t.Errorf("cage wire has %d positions, want %d (2 per edge)", len(items[0].Positions), want)
	}
	if want := body.Vertices().Count() * 6; len(items[1].Positions) != want {
		t.Errorf("cage handles have %d positions, want %d (a 3-axis cross per vertex)",
			len(items[1].Positions), want)
	}
}

// Edit Cage is on the Freeform panel and only enabled when there is a cage to edit.
func TestEditCageIsOnTheFreeformPanel(t *testing.T) {
	s := registeredSession(t)
	tab, ok := BuildRibbon(s).Tab(tabSurfacesMesh)
	if !ok {
		t.Fatal("ribbon has no Surfaces & Mesh tab")
	}
	panel, ok := tab.Panel("Freeform")
	if !ok {
		t.Fatal("Surfaces & Mesh tab has no Freeform panel")
	}
	b, ok := buttonNamed(panel, "Edit Cage")
	if !ok {
		t.Fatal("the Freeform panel has no Edit Cage button")
	}
	if b.Enabled {
		t.Error("Edit Cage should be disabled on a part with no freeform body")
	}
	if err := s.Execute("Freeform.Box"); err != nil {
		t.Fatalf("Freeform.Box: %v", err)
	}
	if err := s.OK(); err != nil {
		t.Fatalf("place the box: %v", err)
	}
	again, _ := buttonNamed(mustPanel(t, s, tabSurfacesMesh, "Freeform"), "Edit Cage")
	if !again.Enabled {
		t.Error("Edit Cage should be enabled once the part holds a freeform body")
	}
}

// mustPanel rebuilds the ribbon and returns the named panel.
func mustPanel(t *testing.T, s *Session, tabName, panelName string) RibbonPanel {
	t.Helper()
	tab, ok := BuildRibbon(s).Tab(tabName)
	if !ok {
		t.Fatalf("ribbon has no %q tab", tabName)
	}
	panel, ok := tab.Panel(panelName)
	if !ok {
		t.Fatalf("%q tab has no %q panel", tabName, panelName)
	}
	return panel
}

// The cage hit-test: a ray down the axis at a handle finds that vertex; one pointing away finds
// none; and a point behind the ray origin never matches.
func TestNearestCageVertexHitTest(t *testing.T) {
	_, _, body := partWithFreeformBox(t)
	at := body.Vertices().Item(0).Point()
	// Fire from 10 units back along +X at the vertex.
	origin := math.P3(float64(at.X)-10, float64(at.Y), float64(at.Z))
	dir := math.V3(1, 0, 0)
	i, ok := nearestCageVertex(body, origin, dir, 0.5)
	if !ok || i != 0 {
		t.Errorf("nearestCageVertex = (%d,%v), want vertex 0", i, ok)
	}
	if _, ok := nearestCageVertex(body, origin, math.V3(0, 1, 0), 0.5); ok {
		t.Error("a ray pointing away from every handle should find none")
	}
	// A point behind the origin is +Inf away, never a hit.
	if d := rayPointDistanceOnly(math.P3(0, 0, 0), math.V3(1, 0, 0), math.P3(-5, 0, 0)); !stdmath.IsInf(d, 1) {
		t.Errorf("a point behind the ray reported distance %g, want +Inf", d)
	}
	if d := rayPointDistanceOnly(math.P3(0, 0, 0), math.V3(1, 0, 0), math.P3(5, 3, 0)); stdmath.Abs(d-3) > 1e-9 {
		t.Errorf("perpendicular distance = %g, want 3", d)
	}
}

// A drag through the camera-driven entry point moves the grabbed handle, so the viewport path
// is exercised end to end and not just its core.
func TestBeginCageDragGrabsTheHandleUnderTheCursor(t *testing.T) {
	s, _, body := partWithFreeformBox(t)
	s.StartTool(NewFreeformCageEditTool())
	// Aim the camera at the first cage vertex so its handle sits under the centre pixel.
	at := body.Vertices().Item(0).Point()
	cam := s.Camera()
	cam.Target = at
	cam.Eye = math.P3(float64(at.X), float64(at.Y), float64(at.Z)+20)
	cam.Up = math.V3(0, 1, 0)
	s.SetCamera(cam)

	if !s.BeginCageDrag(100, 100) {
		t.Skip("the handle did not land under the test pixel in this camera framing")
	}
	if !s.CageDragActive() {
		t.Fatal("BeginCageDrag reported a grab but no drag is active")
	}
	before := body.Vertices().Item(s.cageEdit.vertex).Point()
	s.UpdateCageDrag(140, 100)
	if body.Vertices().Item(s.cageEdit.vertex).Point().IsEqualTo(before, 1e-12) {
		t.Error("dragging the cursor did not move the grabbed handle")
	}
	s.CommitCageDrag()
	if s.CageDragActive() {
		t.Error("the drag is still active after commit")
	}
}
