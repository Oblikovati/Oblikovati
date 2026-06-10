// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// selectPlanes adds the given origin/work planes to the session selection.
func selectPlanes(s *Session, planes ...*feature.WorkPlane) {
	for _, p := range planes {
		s.Selection().Add(WorkPlaneHandle{Plane: p})
	}
}

// addUserPoint adds a datum point at p to the part and returns it (a selectable input
// for three-point planes).
func addUserPoint(def *compdef.PartComponentDefinition, p math.Point3) *feature.WorkPoint {
	return def.WorkPoints().AddByPosition(func() math.Point3 { return p })
}

// boxTopFace extrudes a unit box and returns one of its (planar) faces with its body, a
// FaceHandle input for the Tangent command's wiring.
func boxTopFace(t *testing.T, def *compdef.PartComponentDefinition) (*topo.Face, *topo.Body) {
	t.Helper()
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(-1, -1))
	c1 := sk.Points().Add(math.P2(1, -1))
	c2 := sk.Points().Add(math.P2(1, 1))
	c3 := sk.Points().Add(math.P2(-1, 1))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 3 })
	def.Recompute()
	body := def.SurfaceBodies().All()[0]
	return body.Faces()[0], body
}

func TestOffsetWorkPlaneToolWaitsForDistanceThenCreates(t *testing.T) {
	s, def := emptyPartSession(t)
	before := def.WorkPlanes().Count()
	tool := NewOffsetWorkPlaneTool()
	s.StartTool(tool)
	// Pick the base plane: not committable yet — the distance is still unset.
	tool.Pick(s, WorkPlaneHandle{Plane: def.OriginPlanes()[0]}) // XY plane (normal +Z)
	if tool.CanCommit() {
		t.Fatal("offset tool should not be committable before a distance is entered")
	}
	// Enter 25 mm via the session bridge (what the dialog field does), then commit.
	s.SetOffsetDistanceDisplay(25)
	if !tool.CanCommit() {
		t.Fatal("offset tool should be committable once base and distance are set")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if def.WorkPlanes().Count() != before+1 {
		t.Errorf("offset created %d planes, want 1", def.WorkPlanes().Count()-before)
	}
	// 25 mm is 2.5 cm in model units; the XY plane offset +Z lands at z=2.5.
	wp := tool.AddedPlane()
	if wp == nil || !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 2.5), 1e-9) {
		t.Errorf("offset plane origin = %v, want (0,0,2.5)", wp.Plane().Origin())
	}
}

// TestPickableWorkPlanesIncludesVisibleUserPlanes is the issue-#132 regression: the viewport
// picker must offer ribbon-created user planes (not only the origin frame), so a new sketch
// can be hosted on one by clicking it in the 3D view. A hidden user plane drops out.
func TestPickableWorkPlanesIncludesVisibleUserPlanes(t *testing.T) {
	s, def := emptyPartSession(t)
	origin := len(def.OriginPlanes()) // the always-pickable coordinate-system planes
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })

	got := s.PickableWorkPlanes()
	if len(got) != origin+1 {
		t.Fatalf("PickableWorkPlanes = %d, want %d (origin) + 1 user plane", len(got), origin)
	}
	if !containsPlane(got, wp) {
		t.Errorf("PickableWorkPlanes omitted the visible user plane %p", wp)
	}

	wp.SetVisible(false)
	if got := s.PickableWorkPlanes(); len(got) != origin || containsPlane(got, wp) {
		t.Errorf("a hidden user plane must not be pickable: got %d planes (want %d)", len(got), origin)
	}
}

// TestPickableWorkPlanesHonorsEditScope: a user plane created after the node being edited is
// hidden by the edit scope, so it must drop out of the picker too — geometry the overlays do
// not draw must not be clickable.
func TestPickableWorkPlanesHonorsEditScope(t *testing.T) {
	s, def, f1, _ := twoExtrudePart(t)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Recompute()

	s.BeginEditFeature(FeatureHandle{Feature: f1})
	if containsPlane(s.PickableWorkPlanes(), wp) {
		t.Error("a plane hidden by the edit scope must not be pickable")
	}
	s.CancelTool()
	if !containsPlane(s.PickableWorkPlanes(), wp) {
		t.Error("the plane must be pickable again after the edit closes")
	}
}

func containsPlane(planes []*feature.WorkPlane, want *feature.WorkPlane) bool {
	for _, p := range planes {
		if p == want {
			return true
		}
	}
	return false
}

func TestOffsetWorkPlaneToolNotCommittableWithoutBase(t *testing.T) {
	s, _ := emptyPartSession(t)
	tool := NewOffsetWorkPlaneTool()
	s.StartTool(tool)
	s.SetOffsetDistanceDisplay(10) // a distance but no plane picked
	if tool.CanCommit() {
		t.Error("offset tool should not commit without a base plane")
	}
}

func TestCreateMidplaneWorkPlane(t *testing.T) {
	s, def := emptyPartSession(t)
	selectPlanes(s, def.OriginPlanes()[0], def.OriginPlanes()[1]) // XY + XZ
	wp, err := s.CreateMidplaneWorkPlane()
	if err != nil {
		t.Fatalf("CreateMidplaneWorkPlane: %v", err)
	}
	if !wp.Health().OK() {
		t.Errorf("midplane sick: %+v", wp.Health())
	}
}

func TestCreateMidplaneNeedsTwoPlanes(t *testing.T) {
	s, def := emptyPartSession(t)
	selectPlanes(s, def.OriginPlanes()[0]) // only one
	if _, err := s.CreateMidplaneWorkPlane(); err == nil {
		t.Error("midplane with one plane selected should error")
	}
}

func TestCreatedWorkPlanesAppearInBrowserWithUniqueNames(t *testing.T) {
	s, def := emptyPartSession(t)
	for i := 0; i < 2; i++ {
		s.Selection().Clear()
		selectPlanes(s, def.OriginPlanes()[0], def.OriginPlanes()[1])
		if _, err := s.CreateMidplaneWorkPlane(); err != nil {
			t.Fatalf("CreateMidplaneWorkPlane #%d: %v", i+1, err)
		}
	}
	var labels []string
	for _, c := range BuildBrowser(s).Children {
		if c.Kind != "workplane" {
			continue
		}
		if c.Select == nil {
			t.Errorf("work plane node %q is not selectable", c.Label)
		}
		labels = append(labels, c.Label)
	}
	if len(labels) != 2 || labels[0] == labels[1] {
		t.Errorf("browser work plane labels = %v, want two distinct nodes", labels)
	}
}

func TestCreateThreePointWorkPlaneFromPoints(t *testing.T) {
	s, def := emptyPartSession(t)
	a := addUserPoint(def, math.P3(0, 0, 0))
	b := addUserPoint(def, math.P3(2, 0, 0))
	c := addUserPoint(def, math.P3(0, 2, 0))
	s.Selection().Add(WorkPointHandle{Point: a})
	s.Selection().Add(WorkPointHandle{Point: b})
	s.Selection().Add(WorkPointHandle{Point: c})
	wp, err := s.CreateThreePointWorkPlane()
	if err != nil {
		t.Fatalf("CreateThreePointWorkPlane: %v", err)
	}
	if !wp.Health().OK() || !wp.Plane().Normal().AsVector().IsParallelTo(math.V3(0, 0, 1), 1e-9) {
		t.Errorf("three-point plane health=%v normal=%v, want healthy +Z", wp.Health(), wp.Plane().Normal())
	}
}

func TestCreateThreePointNeedsThreePoints(t *testing.T) {
	s, def := emptyPartSession(t)
	s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(0, 0, 0))})
	if _, err := s.CreateThreePointWorkPlane(); err == nil {
		t.Error("three-point with one point should error")
	}
}

func TestCreateNormalToAxisWorkPlane(t *testing.T) {
	s, def := emptyPartSession(t)
	// Origin X axis (axis 0) + origin center point (point 0): a plane through the origin,
	// normal to X (a YZ-oriented plane).
	s.Selection().Add(WorkAxisHandle{Axis: def.WorkAxes().Item(0)})
	s.Selection().Add(WorkPointHandle{Point: def.WorkPoints().Item(0)})
	wp, err := s.CreateNormalToAxisWorkPlane()
	if err != nil {
		t.Fatalf("CreateNormalToAxisWorkPlane: %v", err)
	}
	if !wp.Health().OK() || !wp.Plane().Normal().AsVector().IsParallelTo(math.V3(1, 0, 0), 1e-9) {
		t.Errorf("normal-to-axis plane health=%v normal=%v, want healthy +X", wp.Health(), wp.Plane().Normal())
	}
}

func TestCreateTangentWorkPlaneWiring(t *testing.T) {
	s, def := emptyPartSession(t)
	face, body := boxTopFace(t, def)
	before := def.WorkPlanes().Count()
	selectPlanes(s, def.OriginPlanes()[0])
	s.Selection().Add(FaceHandle{Face: face, Body: body})
	if _, err := s.CreateTangentWorkPlane(); err != nil {
		t.Fatalf("CreateTangentWorkPlane: %v", err)
	}
	if def.WorkPlanes().Count() != before+1 {
		t.Errorf("tangent command added %d planes, want 1", def.WorkPlanes().Count()-before)
	}
}

func TestOffsetWorkPlaneButtonOpensDistanceDialogNotInstantCreate(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	before := def.WorkPlanes().Count()
	if err := s.Execute("WorkPlane.Offset"); err != nil {
		t.Fatalf("execute Offset: %v", err)
	}
	// The button must NOT drop a plane immediately — it opens the offset tool/dialog.
	if s.ActiveOffsetPlane() == nil {
		t.Fatal("Offset should start the offset-plane tool")
	}
	if def.WorkPlanes().Count() != before {
		t.Error("Offset created a plane without asking for a distance")
	}
	// Pick a plane and a distance in the dialog, then OK.
	origin := originFolder(BuildBrowser(s))
	s.SelectBrowserNode(origin.Children[0]) // XY Plane → fed to the tool as the base
	if def.WorkPlanes().Count() != before {
		t.Error("picking the base plane should not yet create the plane (distance pending)")
	}
	s.SetOffsetDistanceDisplay(15)
	if err := s.OK(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if def.WorkPlanes().Count() != before+1 {
		t.Errorf("offset created %d planes after OK, want 1", def.WorkPlanes().Count()-before)
	}
}

func TestOffsetWorkPlaneButtonSeedsPreselectedPlane(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	selectPlanes(s, def.OriginPlanes()[0])
	if err := s.Execute("WorkPlane.Offset"); err != nil {
		t.Fatalf("execute Offset: %v", err)
	}
	tool := s.ActiveOffsetPlane()
	if tool == nil || !tool.BasePicked() {
		t.Fatal("a pre-selected plane should seed the offset tool's base, leaving only the distance")
	}
}

func TestMidplaneToolCommitsAfterTwoPicks(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	before := def.WorkPlanes().Count()
	if err := s.Execute("WorkPlane.Midplane"); err != nil {
		t.Fatalf("execute Midplane: %v", err)
	}
	origin := originFolder(BuildBrowser(s))
	s.SelectBrowserNode(origin.Children[0]) // XY plane — not enough yet
	if s.ActiveTool() == nil {
		t.Fatal("Midplane tool should stay open after one plane")
	}
	s.SelectBrowserNode(origin.Children[1]) // XZ plane — now commit
	if s.ActiveTool() != nil {
		t.Error("Midplane tool should auto-commit after the second plane")
	}
	if def.WorkPlanes().Count() != before+1 {
		t.Errorf("midplane tool created %d planes, want 1", def.WorkPlanes().Count()-before)
	}
}

func TestToggleWorkPlaneVisibilitySessionAndKeyboard(t *testing.T) {
	s, def := emptyPartSession(t)
	xy := def.OriginPlanes()[0]
	selectPlanes(s, xy)
	if xy.Visible() {
		t.Fatal("origin plane should start hidden")
	}
	// The session method toggles every selected plane.
	s.ToggleSelectedWorkPlaneVisibility()
	if !xy.Visible() {
		t.Error("ToggleSelectedWorkPlaneVisibility should show the selected plane")
	}
	// The V keyboard shortcut toggles it back.
	if err := s.PressKey(KeyEvent{Key: "V"}); err != nil {
		t.Fatalf("PressKey V: %v", err)
	}
	if xy.Visible() {
		t.Error("V should toggle the selected plane's visibility back off")
	}
}

func TestWorkPlaneBrowserVisibilityMenuToggles(t *testing.T) {
	s, _ := emptyPartSession(t)
	node := originFolder(BuildBrowser(s)).Children[0] // XY plane node
	wp := node.Select.(WorkPlaneHandle).Plane
	menu := BrowserMenu(node)
	var vis BrowserMenuItem
	for _, m := range menu {
		if m.Label == "Visibility" {
			vis = m
		}
	}
	if vis.Invoke == nil {
		t.Fatal("workplane menu has no Visibility item")
	}
	before := wp.Visible()
	if err := vis.Invoke(s); err != nil {
		t.Fatalf("invoke Visibility: %v", err)
	}
	if wp.Visible() == before {
		t.Error("Visibility menu item should toggle the plane's visibility")
	}
}

func TestWorkPlaneButtonsDisabledInSketch(t *testing.T) {
	s, _ := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	if _, err := s.CreateSketchOnOrigin(OriginXY); err != nil {
		t.Fatalf("enter sketch: %v", err)
	}
	if c, _ := s.Commands().ByID("WorkPlane.Offset"); c.IsEnabled(s) {
		t.Error("Work Plane buttons should be disabled inside the sketch environment")
	}
}

func TestPickerSnapsToWorkPoint(t *testing.T) {
	s, def := emptyPartSession(t)
	p := NewRayPicker(s.Camera(), func() []*topo.Body { return def.SurfaceBodies().All() }).
		WithPoints(func() []*feature.WorkPoint { return []*feature.WorkPoint{def.WorkPoints().Item(0)} })
	sel, ok := p.Pick(100, 100, NewSelectionFilter()) // center pixel maps to the origin center
	if !ok {
		t.Fatal("center pick hit nothing, expected the origin point")
	}
	if _, isPoint := sel.(WorkPointHandle); !isPoint {
		t.Errorf("center pick = %T, want WorkPointHandle", sel)
	}
}

func TestPickerSnapsToWorkAxis(t *testing.T) {
	s, def := emptyPartSession(t)
	p := NewRayPicker(s.Camera(), func() []*topo.Body { return def.SurfaceBodies().All() }).
		WithAxes(func() []*feature.WorkAxis { return []*feature.WorkAxis{def.WorkAxes().Item(0)} })
	sel, ok := p.Pick(100, 100, NewSelectionFilter(SelectWorkAxis)) // X axis passes through the origin
	if !ok {
		t.Fatal("center pick missed the origin axis")
	}
	if _, isAxis := sel.(WorkAxisHandle); !isAxis {
		t.Errorf("center pick = %T, want WorkAxisHandle", sel)
	}
}

func TestWorkFeatureRibbonCommandsRegistered(t *testing.T) {
	s, _ := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	for _, id := range []string{
		"WorkPlane.Offset", "WorkPlane.Midplane",
		"WorkPlane.ThreePoints", "WorkPlane.Tangent", "WorkPlane.NormalToAxis",
	} {
		c, ok := s.Commands().ByID(id)
		if !ok {
			t.Fatalf("command %q not registered", id)
		}
		if c.Tab() != "3D Model" || c.Category() != "Work Features" {
			t.Errorf("%q is on tab %q/panel %q, want 3D Model/Work Features", id, c.Tab(), c.Category())
		}
	}
	// Every Work Features button is always live in the part environment (it guides the
	// pick when nothing is selected), so none is greyed out.
	for _, id := range []string{
		"WorkPlane.Offset", "WorkPlane.Midplane",
		"WorkPlane.ThreePoints", "WorkPlane.Tangent", "WorkPlane.NormalToAxis",
	} {
		if c, _ := s.Commands().ByID(id); !c.IsEnabled(s) {
			t.Errorf("%q should be enabled in the part environment", id)
		}
	}
}

// TestOffsetWorkPlaneFromFace offsets a work plane from a picked planar face: the box's top
// face (z=4) offset by 2 lands at z=6. Regression for the offset-plane tool ignoring face
// picks (it filtered/handled work planes only, despite its "or planar face" prompt).
func TestOffsetWorkPlaneFromFace(t *testing.T) {
	s := extrudedBox(t, 2, 4) // box, top face at z=4
	body := partBodies(s)()[0]
	top := topFaceOf(t, body)

	tool := NewOffsetWorkPlaneTool()
	s.StartTool(tool)
	tool.Pick(s, FaceHandle{Face: top, Body: body})
	if !tool.BasePicked() {
		t.Fatal("offset tool did not accept the picked planar face")
	}
	tool.SetDistance(2)
	if err := s.OK(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	wp := tool.AddedPlane()
	if wp == nil {
		t.Fatal("no plane created")
	}
	if z := wp.Plane().Origin().Z; z < 6-1e-9 || z > 6+1e-9 {
		t.Errorf("offset-from-face plane Z = %g, want 6 (4 + 2)", z)
	}
}
