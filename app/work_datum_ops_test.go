// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// #2043 / #2044: the Work Features panel shipped five buttons, all work PLANES. There was no
// Work Axis command anywhere and no general Work Point one, so 0 of 9 axis and 1 of 10 point
// constructors were reachable, and 11 of 17 plane constructors were API-only.

// linearEdgeOf returns one edge of the body — the linear-edge input the edge-driven datum
// constructors take.
func linearEdgeOf(t *testing.T, def *compdef.PartComponentDefinition) EdgeHandle {
	t.Helper()
	_, body := boxTopFace(t, def)
	for _, e := range body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if a.IsEqualTo(b, 1e-9) {
			continue
		}
		return EdgeHandle{Edge: e}
	}
	t.Fatal("the block has no linear edge")
	return EdgeHandle{}
}

// originAxis returns one of the part's origin datum axes by reference.
func originAxis(t *testing.T, def *compdef.PartComponentDefinition, ref feature.WorkRef) *feature.WorkAxis {
	t.Helper()
	axes := def.WorkAxes()
	for i := 0; i < axes.Count(); i++ {
		if a := axes.Item(i); a.Key() == ref {
			return a
		}
	}
	t.Fatalf("no origin axis %q", ref)
	return nil
}

// selectAxes adds the given work axes to the session selection.
func selectAxes(s *Session, axes ...*feature.WorkAxis) {
	for _, a := range axes {
		s.Selection().Add(WorkAxisHandle{Axis: a})
	}
}

// The datum-axis constructors: each builds from the current selection and lands a healthy axis.
func TestWorkAxisConstructorsBuildFromTheSelection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		setup func(*testing.T, *Session, *compdef.PartComponentDefinition)
		build func(*Session) (*feature.WorkAxis, error)
	}{
		{"on edge", func(t *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			s.Selection().Add(linearEdgeOf(t, def))
		}, (*Session).CreateEdgeWorkAxis},
		{"through two points", func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(0, 0, 0))})
			s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(1, 2, 3))})
		}, (*Session).CreateTwoPointWorkAxis},
		{"at two planes", func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			selectPlanes(s, def.OriginPlanes()[0], def.OriginPlanes()[1])
		}, (*Session).CreatePlaneIntersectionWorkAxis},
		{"normal to plane", func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			selectPlanes(s, def.OriginPlanes()[0])
			s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(1, 1, 0))})
		}, (*Session).CreateNormalToPlaneWorkAxis},
		{"parallel through point", func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			selectAxes(s, originAxis(t, def, feature.OriginXAxis))
			s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(0, 2, 0))})
		}, (*Session).CreateParallelToAxisWorkAxis},
		{"projected to plane", func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			selectAxes(s, originAxis(t, def, feature.OriginXAxis))
			selectPlanes(s, def.OriginPlanes()[1])
		}, (*Session).CreateAxisOnPlaneWorkAxis},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, def := emptyPartSession(t)
			tc.setup(t, s, def) // setup seeds its own datums, so count AFTER it
			before := def.WorkAxes().Count()
			wa, err := tc.build(s)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			if def.WorkAxes().Count() != before+1 {
				t.Fatalf("created %d axes, want 1", def.WorkAxes().Count()-before)
			}
			if !wa.Health().OK() {
				t.Errorf("axis is sick: %+v", wa.Health())
			}
		})
	}
}

// The datum-point constructors, same shape.
func TestWorkPointConstructorsBuildFromTheSelection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		setup func(*testing.T, *Session, *compdef.PartComponentDefinition)
		build func(*Session) (*feature.WorkPoint, error)
	}{
		{"at vertex", func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(1, 1, 1))})
		}, (*Session).CreateVertexWorkPoint},
		{"at edge midpoint", func(t *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			s.Selection().Add(linearEdgeOf(t, def))
		}, (*Session).CreateMidpointWorkPoint},
		{"at centroid", func(t *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			s.Selection().Add(linearEdgeOf(t, def))
		}, (*Session).CreateCentroidWorkPoint},
		{"at three planes", func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			selectPlanes(s, def.OriginPlanes()[0], def.OriginPlanes()[1], def.OriginPlanes()[2])
		}, (*Session).CreateThreePlaneWorkPoint},
		{"at plane and axis", func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			selectPlanes(s, def.OriginPlanes()[0])
			selectAxes(s, originAxis(t, def, feature.OriginZAxis)) // Z axis pierces XY
		}, (*Session).CreatePlaneAndAxisWorkPoint},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, def := emptyPartSession(t)
			tc.setup(t, s, def) // setup seeds its own datums, so count AFTER it
			before := def.WorkPoints().Count()
			wp, err := tc.build(s)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			if def.WorkPoints().Count() != before+1 {
				t.Fatalf("created %d points, want 1", def.WorkPoints().Count()-before)
			}
			if !wp.Health().OK() {
				t.Errorf("point is sick: %+v", wp.Health())
			}
		})
	}
}

// The plane constructors that had no ribbon path.
func TestNewWorkPlaneConstructorsBuildFromTheSelection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		setup func(*testing.T, *Session, *compdef.PartComponentDefinition)
		build func(*Session) (*feature.WorkPlane, error)
	}{
		{"parallel through point", func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			selectPlanes(s, def.OriginPlanes()[0])
			s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(0, 0, 4))})
		}, (*Session).CreateParallelThroughPointWorkPlane},
		{"through axis and point", func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			selectAxes(s, originAxis(t, def, feature.OriginXAxis))
			s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(0, 1, 1))})
		}, (*Session).CreateLineAndPointWorkPlane},
		{"through two axes", func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			selectAxes(s, originAxis(t, def, feature.OriginXAxis), originAxis(t, def, feature.OriginYAxis))
		}, (*Session).CreateTwoLinesWorkPlane},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, def := emptyPartSession(t)
			tc.setup(t, s, def) // setup seeds its own datums, so count AFTER it
			before := def.WorkPlanes().Count()
			wp, err := tc.build(s)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			if def.WorkPlanes().Count() != before+1 {
				t.Fatalf("created %d planes, want 1", def.WorkPlanes().Count()-before)
			}
			if !wp.Health().OK() {
				t.Errorf("plane is sick: %+v", wp.Health())
			}
		})
	}
}

// The angle plane gathers its picks and its angle before committing, and lands at that angle.
func TestAngleWorkPlaneToolNeedsPicksAndAngle(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	before := def.WorkPlanes().Count()
	tool := NewAngleWorkPlaneTool()
	s.StartTool(tool)

	tool.Pick(s, WorkAxisHandle{Axis: originAxis(t, def, feature.OriginXAxis)}) // X axis
	if tool.CanCommit() {
		t.Fatal("angle plane should not commit with only an axis")
	}
	tool.Pick(s, WorkPlaneHandle{Plane: def.OriginPlanes()[0]}) // XY
	if tool.CanCommit() {
		t.Fatal("angle plane should not commit before an angle is entered")
	}
	tool.SetAngleDegrees(30)
	if !tool.CanCommit() {
		t.Fatal("angle plane should commit once axis, plane and angle are set")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if def.WorkPlanes().Count() != before+1 {
		t.Fatalf("created %d planes, want 1", def.WorkPlanes().Count()-before)
	}
	wp := tool.AddedPlane()
	if wp == nil || !wp.Health().OK() {
		t.Fatalf("angle plane is sick: %+v", wp)
	}
	// Rotating XY by 30° about X tilts the normal off +Z by 30°.
	got := float64(wp.Plane().Normal().AsVector().Dot(math.V3(0, 0, 1)))
	if want := stdmath.Cos(30 * stdmath.Pi / 180); stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("angle plane normal·Z = %g, want cos30° = %g", got, want)
	}
}

// Datum names must be unique: Dear ImGui derives a browser node id from the label and asserts on
// duplicates, and every work point used to be minted as the same "WorkPoint".
func TestCreatedDatumsGetUniqueNames(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	for i := range 3 {
		s.Selection().Clear()
		s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(float64(i), 0, 0))})
		if _, err := s.CreateVertexWorkPoint(); err != nil {
			t.Fatalf("CreateVertexWorkPoint %d: %v", i, err)
		}
	}
	seen := map[string]bool{}
	for i := 0; i < def.WorkPoints().Count(); i++ {
		name := def.WorkPoints().Item(i).Name()
		if seen[name] && name != "WorkPoint" { // the raw seeds share the model's default name
			t.Errorf("duplicate datum-point name %q", name)
		}
		seen[name] = true
	}
}

// The Work Features panel carries one split button per datum kind, each with its constructor
// flavours in the flyout. It used to be five flat buttons, all work planes.
func TestWorkFeaturesPanelCarriesAllThreeDatumKinds(t *testing.T) {
	t.Parallel()
	s := registeredSession(t)
	tab, ok := BuildRibbon(s).Tab(tabCreateModify)
	if !ok {
		t.Fatal("ribbon has no Create & Modify tab")
	}
	panel, ok := tab.Panel(panelWorkFeatures)
	if !ok {
		t.Fatal("Create & Modify tab has no Work Features panel")
	}
	if len(panel.Buttons) != 3 {
		t.Errorf("Work Features has %d buttons, want 3 (Plane / Axis / Point)", len(panel.Buttons))
	}
	for _, want := range []struct {
		button   string
		variants int
	}{{"Work Plane", 11}, {"Work Axis", 6}, {"Work Point", 7}} {
		b, ok := buttonNamed(panel, want.button)
		if !ok {
			t.Errorf("Work Features has no %q button", want.button)
			continue
		}
		if len(b.Variants) != want.variants {
			t.Errorf("%s carries %d flyout entries, want %d", want.button, len(b.Variants), want.variants)
		}
	}
}

// Every Work Features entry is always live in the part environment — click it and it either
// builds from the selection or starts a guided pick, never nothing.
func TestWorkFeatureEntriesAreEnabledWithNoSelection(t *testing.T) {
	t.Parallel()
	s := registeredSession(t)
	tab, _ := BuildRibbon(s).Tab(tabCreateModify)
	panel, _ := tab.Panel(panelWorkFeatures)
	for _, b := range panel.Buttons {
		if !b.Enabled {
			t.Errorf("%s is disabled with nothing selected", b.Command.DisplayName())
		}
		for _, v := range b.Variants {
			if !v.Enabled {
				t.Errorf("%s ▸ %s is disabled with nothing selected", b.Command.DisplayName(), v.Label)
			}
		}
	}
}
