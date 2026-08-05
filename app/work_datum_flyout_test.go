// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Every Work Features flyout entry: the enable predicate that decides whether a click builds
// straight from the selection or starts the guided pick, and the tool it starts. A predicate
// that reads the wrong selection kind would leave its entry permanently in guided-pick mode —
// working, but never taking the shortcut the panel is designed around (#2043, #2044).

// datumEntry is one flyout entry under test: its guided tool and its enable predicate.
type datumEntry struct {
	name  string
	tool  func() *DatumPickTool
	ready func(*Session) bool
	// seed puts the selection the predicate wants in place; nil means the entry needs picks
	// this fixture cannot supply (a revolved or toroidal face), so only the negative is asserted.
	seed func(*testing.T, *Session, *compdef.PartComponentDefinition)
}

func datumEntries() []datumEntry {
	selPoints := func(n int) func(*testing.T, *Session, *compdef.PartComponentDefinition) {
		return func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			for i := 0; i < n; i++ {
				s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(float64(i), 0, 0))})
			}
		}
	}
	selPlanes := func(n int) func(*testing.T, *Session, *compdef.PartComponentDefinition) {
		return func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
			selectPlanes(s, def.OriginPlanes()[:n]...)
		}
	}
	selEdge := func(t *testing.T, s *Session, def *compdef.PartComponentDefinition) {
		s.Selection().Add(linearEdgeOf(t, def))
	}
	selAxisAndPoint := func(t *testing.T, s *Session, def *compdef.PartComponentDefinition) {
		selectAxes(s, originAxis(t, def, feature.OriginXAxis))
		s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(0, 2, 0))})
	}
	selPlaneAndPoint := func(_ *testing.T, s *Session, def *compdef.PartComponentDefinition) {
		selectPlanes(s, def.OriginPlanes()[0])
		s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(0, 0, 4))})
	}
	selAxisAndPlane := func(t *testing.T, s *Session, def *compdef.PartComponentDefinition) {
		selectAxes(s, originAxis(t, def, feature.OriginXAxis))
		selectPlanes(s, def.OriginPlanes()[1])
	}
	return []datumEntry{
		{"axis on edge", newEdgeWorkAxisTool, canEdgeWorkAxis, selEdge},
		{"axis through two points", newTwoPointWorkAxisTool, canTwoPointWorkAxis, selPoints(2)},
		{"axis at two planes", newPlaneIntersectionWorkAxisTool, canPlaneIntersectionWorkAxis, selPlanes(2)},
		{"axis normal to plane", newNormalToPlaneWorkAxisTool, canNormalToPlaneWorkAxis, selPlaneAndPoint},
		{"axis parallel to axis", newParallelToAxisWorkAxisTool, canParallelToAxisWorkAxis, selAxisAndPoint},
		{"axis projected to plane", newAxisOnPlaneWorkAxisTool, canAxisOnPlaneWorkAxis, selAxisAndPlane},
		{"axis of revolved face", newRevolvedFaceWorkAxisTool, canRevolvedFaceWorkAxis, nil},
		{"point at vertex", newVertexWorkPointTool, canVertexWorkPoint, selPoints(1)},
		{"point at edge midpoint", newMidpointWorkPointTool, canMidpointWorkPoint, selEdge},
		{"point at centroid", newCentroidWorkPointTool, canCentroidWorkPoint, selEdge},
		{"point at three planes", newThreePlaneWorkPointTool, canThreePlaneWorkPoint, selPlanes(3)},
		{"point at two axes", newTwoAxisWorkPointTool, canTwoAxisWorkPoint, nil},
		{"point at plane and axis", newPlaneAndAxisWorkPointTool, canPlaneAndAxisWorkPoint, selAxisAndPlane},
		{"point at face centre", newFaceCenterWorkPointTool, canFaceCenterWorkPoint, nil},
		{"point at curve and surface", newCurveAndEntityWorkPointTool, canCurveAndEntityWorkPoint, nil},
		{"plane parallel through point", newParallelThroughPointWorkPlaneTool, canParallelThroughPointWorkPlane, selPlaneAndPoint},
		{"plane through axis and point", newLineAndPointWorkPlaneTool, canLineAndPointWorkPlane, selAxisAndPoint},
		{"plane tangent through point", newPointAndTangentWorkPlaneTool, canPointAndTangentWorkPlane, nil},
		{"plane tangent through axis", newLineAndTangentWorkPlaneTool, canLineAndTangentWorkPlane, nil},
		{"plane torus midplane", newTorusMidPlaneWorkPlaneTool, canTorusMidPlaneWorkPlane, nil},
	}
}

// TestDatumFlyoutEntriesGateOnTheirSelection: each entry is NOT ready with an empty selection,
// IS ready once its inputs are selected, and names a prompt telling the user what to pick.
func TestDatumFlyoutEntriesGateOnTheirSelection(t *testing.T) {
	for _, e := range datumEntries() {
		t.Run(e.name, func(t *testing.T) {
			s, def := emptyPartSession(t)
			tool := e.tool()
			if tool.Name() == "" || tool.Prompt(s) == "" {
				t.Error("the entry names no tool or gives no prompt")
			}
			if e.ready(s) {
				t.Error("the entry reports ready with nothing selected")
			}
			if e.seed == nil {
				return // needs geometry this fixture has no analytic form of
			}
			e.seed(t, s, def)
			if !e.ready(s) {
				t.Error("the entry is not ready with its inputs selected")
			}
		})
	}
}

// A datum entry with its inputs already selected builds immediately instead of starting the
// guided tool — the shortcut every Work Features click is supposed to take.
func TestStartDatumBuildsFromAPreSelection(t *testing.T) {
	s, def := emptyPartSession(t)
	selectPlanes(s, def.OriginPlanes()[0], def.OriginPlanes()[1])
	before := def.WorkAxes().Count()

	if err := startDatum(newPlaneIntersectionWorkAxisTool)(s); err != nil {
		t.Fatalf("startDatum: %v", err)
	}
	if def.WorkAxes().Count() != before+1 {
		t.Errorf("created %d axes, want 1 — the pre-selection shortcut did not fire",
			def.WorkAxes().Count()-before)
	}
	if s.ActiveTool() != nil {
		t.Error("a pre-selected build should not also start the guided tool")
	}
}

// With nothing selected the same click starts the guided pick instead, so it is never inert.
func TestStartDatumFallsBackToTheGuidedPick(t *testing.T) {
	s, _ := emptyPartSession(t)
	if err := startDatum(newPlaneIntersectionWorkAxisTool)(s); err != nil {
		t.Fatalf("startDatum: %v", err)
	}
	if s.ActiveTool() == nil {
		t.Fatal("with nothing selected the entry should start the guided tool")
	}
	if got := s.ActiveTool().Name(); got != "Axis at Two Planes" {
		t.Errorf("started %q, want the plane-intersection tool", got)
	}
	s.CancelTool()
}

// canStartWorkFeature gates the whole panel: live on a part, dead inside a sketch.
func TestWorkFeaturePanelEnableGate(t *testing.T) {
	s, _ := emptyPartSession(t)
	if !canStartWorkFeature(s) {
		t.Error("the Work Features panel should be live on a part")
	}
	if _, err := s.CreateSketchOnOrigin(OriginXY); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	if canStartWorkFeature(s) {
		t.Error("the Work Features panel should be dead inside a sketch")
	}
	if canStartWorkFeature(NewSession()) {
		t.Error("the Work Features panel should be dead with no part")
	}
}

// The revolved-face and face-centre entries build from a real cylindrical face, which the box
// fixture cannot supply — this is the analytic half of the axis/point sets.
func TestRevolvedFaceDatumsBuildFromACylinder(t *testing.T) {
	s, cyl := newPartWithCylinder(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	s.Selection().Add(FaceHandle{Face: cylinderFaceOf(t, cyl), Body: cyl})

	if !canRevolvedFaceWorkAxis(s) {
		t.Fatal("a selected cylindrical face should enable the revolved-face axis")
	}
	before := def.WorkAxes().Count()
	wa, err := s.CreateRevolvedFaceWorkAxis()
	if err != nil {
		t.Fatalf("CreateRevolvedFaceWorkAxis: %v", err)
	}
	if def.WorkAxes().Count() != before+1 || !wa.Health().OK() {
		t.Errorf("the revolved-face axis did not land healthy: %+v", wa.Health())
	}
	// The axis of a +Z cylinder points along Z.
	if d := wa.Direction().AsVector(); stdmath.Abs(stdmath.Abs(float64(d.Z))-1) > 1e-6 {
		t.Errorf("revolved-face axis direction = %v, want ±Z", d)
	}
}

// The tangent and torus-midplane entries need a curved face, which only the cylinder fixture
// supplies. This is their analytic half, and it covers the plane constructors the box-based
// table can only assert the negative of.
func TestCurvedFaceWorkPlanesBuildFromACylinder(t *testing.T) {
	s, cyl := newPartWithCylinder(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	face := cylinderFaceOf(t, cyl)

	// Tangent through a point: a curved face plus a point.
	s.Selection().Add(FaceHandle{Face: face, Body: cyl})
	s.Selection().Add(WorkPointHandle{Point: addUserPoint(def, math.P3(0.5, 0, 1))})
	if !canPointAndTangentWorkPlane(s) {
		t.Fatal("a face plus a point should enable the tangent-through-point plane")
	}
	before := def.WorkPlanes().Count()
	if _, err := s.CreatePointAndTangentWorkPlane(); err != nil {
		t.Fatalf("CreatePointAndTangentWorkPlane: %v", err)
	}
	if def.WorkPlanes().Count() != before+1 {
		t.Error("the tangent-through-point plane was not created")
	}

	// Tangent through an axis: a curved face plus an axis.
	s.Selection().Clear()
	s.Selection().Add(FaceHandle{Face: face, Body: cyl})
	selectAxes(s, originAxis(t, def, feature.OriginZAxis))
	if !canLineAndTangentWorkPlane(s) {
		t.Fatal("a face plus an axis should enable the tangent-through-axis plane")
	}
	if _, err := s.CreateLineAndTangentWorkPlane(); err != nil {
		t.Fatalf("CreateLineAndTangentWorkPlane: %v", err)
	}

	// Torus midplane takes the face alone.
	s.Selection().Clear()
	s.Selection().Add(FaceHandle{Face: face, Body: cyl})
	if !canTorusMidPlaneWorkPlane(s) {
		t.Fatal("a selected face should enable the torus midplane")
	}
	if _, err := s.CreateTorusMidPlaneWorkPlane(); err != nil {
		t.Fatalf("CreateTorusMidPlaneWorkPlane: %v", err)
	}

	// The curve∩surface point takes an edge plus a plane or face.
	s.Selection().Clear()
	s.Selection().Add(FaceHandle{Face: face, Body: cyl})
	s.Selection().Add(EdgeHandle{Edge: cyl.Edges()[0]})
	if !canCurveAndEntityWorkPoint(s) {
		t.Fatal("an edge plus a face should enable the curve∩surface point")
	}
	if _, err := s.CreateCurveAndEntityWorkPoint(); err != nil {
		t.Fatalf("CreateCurveAndEntityWorkPoint: %v", err)
	}

	// Face centre likewise takes the face alone.
	s.Selection().Clear()
	s.Selection().Add(FaceHandle{Face: face, Body: cyl})
	if !canFaceCenterWorkPoint(s) {
		t.Fatal("a selected face should enable the face-centre point")
	}
	if _, err := s.CreateFaceCenterWorkPoint(); err != nil {
		t.Fatalf("CreateFaceCenterWorkPoint: %v", err)
	}
}

// Two axes: the plane through them, and the point where they meet.
func TestTwoAxisDatumsBuild(t *testing.T) {
	s, def := emptyPartSession(t)
	selectAxes(s, originAxis(t, def, feature.OriginXAxis), originAxis(t, def, feature.OriginYAxis))
	if !canTwoLinesWorkPlane(s) || !canTwoAxisWorkPoint(s) {
		t.Fatal("two selected axes should enable both the plane and the point")
	}
	if _, err := s.CreateTwoLinesWorkPlane(); err != nil {
		t.Fatalf("CreateTwoLinesWorkPlane: %v", err)
	}
	wp, err := s.CreateTwoAxisWorkPoint()
	if err != nil {
		t.Fatalf("CreateTwoAxisWorkPoint: %v", err)
	}
	// X and Y meet at the origin.
	if !wp.Point().IsEqualTo(math.P3(0, 0, 0), 1e-9) {
		t.Errorf("X∩Y point = %v, want the origin", wp.Point())
	}
}

// Each constructor refuses with the wrong selection rather than building a garbage datum.
func TestDatumConstructorsRefuseTheWrongSelection(t *testing.T) {
	s, _ := emptyPartSession(t)
	for name, build := range map[string]func() error{
		"revolved face":    func() error { _, err := s.CreateRevolvedFaceWorkAxis(); return err },
		"edge axis":        func() error { _, err := s.CreateEdgeWorkAxis(); return err },
		"two points":       func() error { _, err := s.CreateTwoPointWorkAxis(); return err },
		"plane ∩ plane":    func() error { _, err := s.CreatePlaneIntersectionWorkAxis(); return err },
		"normal to plane":  func() error { _, err := s.CreateNormalToPlaneWorkAxis(); return err },
		"parallel axis":    func() error { _, err := s.CreateParallelToAxisWorkAxis(); return err },
		"axis on plane":    func() error { _, err := s.CreateAxisOnPlaneWorkAxis(); return err },
		"vertex point":     func() error { _, err := s.CreateVertexWorkPoint(); return err },
		"midpoint":         func() error { _, err := s.CreateMidpointWorkPoint(); return err },
		"centroid":         func() error { _, err := s.CreateCentroidWorkPoint(); return err },
		"face centre":      func() error { _, err := s.CreateFaceCenterWorkPoint(); return err },
		"three planes":     func() error { _, err := s.CreateThreePlaneWorkPoint(); return err },
		"two axes point":   func() error { _, err := s.CreateTwoAxisWorkPoint(); return err },
		"plane ∩ axis":     func() error { _, err := s.CreatePlaneAndAxisWorkPoint(); return err },
		"curve ∩ surface":  func() error { _, err := s.CreateCurveAndEntityWorkPoint(); return err },
		"parallel plane":   func() error { _, err := s.CreateParallelThroughPointWorkPlane(); return err },
		"axis+point plane": func() error { _, err := s.CreateLineAndPointWorkPlane(); return err },
		"two axes plane":   func() error { _, err := s.CreateTwoLinesWorkPlane(); return err },
		"tangent at point": func() error { _, err := s.CreatePointAndTangentWorkPlane(); return err },
		"tangent at axis":  func() error { _, err := s.CreateLineAndTangentWorkPlane(); return err },
		"torus midplane":   func() error { _, err := s.CreateTorusMidPlaneWorkPlane(); return err },
	} {
		if err := build(); err == nil {
			t.Errorf("%s built a datum from an empty selection", name)
		}
	}
}
