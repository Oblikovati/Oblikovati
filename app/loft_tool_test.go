// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// seqPicker returns a queued sequence of selectables, one per Pick call — so a test can
// drive several clicks that resolve to different sections.
type seqPicker struct {
	sels []Selectable
	i    int
}

func (p *seqPicker) Pick(_, _ float64, _ *SelectionFilter) (Selectable, bool) {
	if p.i >= len(p.sels) {
		return nil, false
	}
	sel := p.sels[p.i]
	p.i++
	return sel, true
}

// centeredSquareSketch adds a centered square (corners ±half) on the given plane.
func centeredSquareSketch(def *compdef.PartComponentDefinition, plane sketch.Plane, half float64) *sketch.Sketch {
	sk := def.Sketches().Add(plane)
	c0 := sk.Points().Add(math.P2(-half, -half))
	c1 := sk.Points().Add(math.P2(half, -half))
	c2 := sk.Points().Add(math.P2(half, half))
	c3 := sk.Points().Add(math.P2(-half, half))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return sk
}

func planeAtZ(z float64) sketch.Plane {
	p, _ := sketch.NewPlane(math.P3(0, 0, z), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	return p
}

// newPartWithStackedSquares sets up a part with a 4×4 square at z=0 and a 2×2 square at z=5,
// returning the session and the two section handles.
func newPartWithStackedSquares(t *testing.T) (*Session, ProfileHandle, ProfileHandle) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	bottom := centeredSquareSketch(def, sketch.XYPlane(), 2)
	top := centeredSquareSketch(def, planeAtZ(5), 1)
	return s, ProfileHandle{Sketch: bottom, ProfileIndex: 0}, ProfileHandle{Sketch: top, ProfileIndex: 0}
}

// TestLoftToolEndToEnd drives the Loft UI: start the tool, click two sections in order,
// OK — and asserts a validated frustum solid (V = 140/3) lands in the part.
func TestLoftToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, bottom, top := newPartWithStackedSquares(t)
	s.SetPicker(&seqPicker{sels: []Selectable{bottom, top}})

	l := NewLoftTool()
	s.StartTool(l)   // ribbon: click "Loft"
	s.Click(10, 10)  // viewport: click the bottom section
	s.Click(10, 200) // viewport: click the top section
	if !l.CanCommit() {
		t.Fatalf("loft not ready with %d sections", l.SectionCount())
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies after loft, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("lofted body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, 140.0/3) > 0.02 {
		t.Errorf("frustum volume = %g, want ≈46.667", got)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

func TestLoftViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, bottom, top := newPartWithStackedSquares(t)
	s.SetPicker(&seqPicker{sels: []Selectable{bottom, top}})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Create.Loft"); err != nil {
		t.Fatalf("execute Create.Loft: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*LoftTool); !ok {
		t.Fatal("Loft command did not start the loft tool")
	}
	s.Click(0, 0)
	s.Click(0, 0)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Error("ribbon-launched loft produced no body")
	}
}

// TestLoftToolAngleConditionCurves drives the Loft UI with an Angle end condition on both
// sections (two EQUAL squares) and asserts the body curves OUT past the ruled prism — the
// end-to-end S2 behavior the dialog exposes (a Free loft of equal squares is a straight prism).
func TestLoftToolAngleConditionCurves(t *testing.T) {
	t.Parallel()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	bottom := ProfileHandle{Sketch: centeredSquareSketch(def, sketch.XYPlane(), 2), ProfileIndex: 0}
	top := ProfileHandle{Sketch: centeredSquareSketch(def, planeAtZ(4), 2), ProfileIndex: 0}
	s.SetPicker(&seqPicker{sels: []Selectable{bottom, top}})

	l := NewLoftTool()
	s.StartTool(l)
	s.Click(10, 10)
	s.Click(10, 200)
	// The dialog would set these from the condition controls; drive them directly here.
	end := feature.LoftEnd{Condition: feature.LoftAngle, Angle: 45 * stdmath.Pi / 180, Impact: 1}
	l.SetFirstCondition(end)
	l.SetLastCondition(end)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	body := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("angled loft body not a valid solid: %+v", r)
	}
	if maxX := float64(body.RangeBox().Max.X); maxX < 2.15 {
		t.Errorf("angled loft did not curve: max x = %.3f, want > 2.15 (ruled prism would be 2.0)", maxX)
	}
}

// TestLoftToolPointSectionCone drives the Loft UI picking a circle region then a WORK POINT as
// an apex (Sharp condition) — the tool must build a cone (V = πr²h/3), exercising point-section
// picking end to end.
func TestLoftToolPointSectionCone(t *testing.T) {
	t.Parallel()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	base := def.Sketches().Add(sketch.XYPlane())
	base.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	apex := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 4) })
	s.SetPicker(&seqPicker{sels: []Selectable{ProfileHandle{Sketch: base, ProfileIndex: 0}, WorkPointHandle{Point: apex}}})

	l := NewLoftTool()
	s.StartTool(l)
	s.Click(10, 10) // the circle region
	s.Click(0, 200) // the apex work point
	if l.SectionCount() != 2 {
		t.Fatalf("loft has %d sections, want 2 (a profile + an apex)", l.SectionCount())
	}
	l.SetLastCondition(feature.LoftEnd{Condition: feature.LoftSharpPoint})
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	body := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("cone body not a valid solid: %+v", r)
	}
	want := stdmath.Pi * 4 / 3 * 4 // πr²h/3, r=2 h=4
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, want) > 0.03 {
		t.Errorf("cone volume = %g, want ≈%g", got, want)
	}
}

// TestLoftToolFaceSectionTangent drives the Loft UI picking a body FACE (a cylinder's top) then a
// circle above, with a Tangent condition on the face — the loft must flare out tangent to the
// planar top (exact G1), beyond the ruled radius. Exercises face-section picking end to end.
func TestLoftToolFaceSectionTangent(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~6s): `make test-corpus`")
	}
	t.Parallel()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	base := def.Sketches().Add(sketch.XYPlane())
	base.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(base, 0, ops.NewBody, func() float64 { return 3 })
	def.Recompute()
	cyl := def.SurfaceBodies().Item(0)

	var topKey []byte
	bestZ := -1e30
	for _, f := range cyl.Faces() {
		bb := f.RangeBox()
		if float64(bb.Max.Z-bb.Min.Z) < 1e-6 {
			if zc := float64(bb.Min.Z+bb.Max.Z) / 2; zc > bestZ {
				bestZ, topKey = zc, f.ReferenceKey()
			}
		}
	}
	topFace, ok := cyl.FindFaceByKey(topKey)
	if !ok {
		t.Fatal("could not re-find the cylinder top face by key")
	}
	topSketch := def.Sketches().Add(planeAtZ(6))
	topSketch.Circles().AddByCenterRadius(math.P2(0, 0), 1)
	def.Recompute()

	s.SetPicker(&seqPicker{sels: []Selectable{
		FaceHandle{Face: topFace, Body: cyl},
		ProfileHandle{Sketch: topSketch, ProfileIndex: 0},
	}})
	l := NewLoftTool()
	s.StartTool(l)
	s.Click(0, 0)   // the cylinder top face
	s.Click(10, 10) // the small circle
	if l.SectionCount() != 2 {
		t.Fatalf("loft has %d sections, want 2 (face + profile)", l.SectionCount())
	}
	l.SetFirstCondition(feature.LoftEnd{Condition: feature.LoftTangent})
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	loft := def.SurfaceBodies().Item(def.SurfaceBodies().Count() - 1)
	if r := ops.Validate(loft); !r.Valid || !loft.IsSolid() {
		t.Fatalf("face-section loft not a valid solid: %+v", r)
	}
	if maxX := float64(loft.RangeBox().Max.X); maxX < 2.15 {
		t.Errorf("tangent face loft did not flare: max x = %.3f, want > 2.15 (ruled would be 2.0)", maxX)
	}
}

// TestLoftToolRailGuides drives the Loft UI picking two circle sections plus an open PATH (a rail
// that bulges to x=3.5) — the loft must follow the rail and bulge past the ruled radius.
func TestLoftToolRailGuides(t *testing.T) {
	t.Parallel()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	base := def.Sketches().Add(sketch.XYPlane())
	base.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	top := def.Sketches().Add(planeAtZ(4))
	top.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	// Rail on XZ: (u,v)→(u,0,v), bulging to x=3.5 at mid height; touches both circle +X corners.
	railSketch := def.Sketches().Add(sketch.XZPlane())
	a := railSketch.Points().Add(math.P2(2, 0))
	mid := railSketch.Points().Add(math.P2(3.5, 2))
	b := railSketch.Points().Add(math.P2(2, 4))
	railSketch.Lines().Add(a, mid)
	railSketch.Lines().Add(mid, b)

	s.SetPicker(&seqPicker{sels: []Selectable{
		ProfileHandle{Sketch: base, ProfileIndex: 0},
		ProfileHandle{Sketch: top, ProfileIndex: 0},
		PathHandle{Sketch: railSketch, PathIndex: 0},
	}})
	l := NewLoftTool()
	s.StartTool(l)
	s.Click(0, 0) // base circle
	s.Click(0, 0) // top circle
	s.Click(0, 0) // the rail path
	if l.SectionCount() != 2 || l.RailCount() != 1 {
		t.Fatalf("loft picks: %d sections, %d rails; want 2 + 1", l.SectionCount(), l.RailCount())
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("railed loft not a valid solid: %+v", r)
	}
	if maxX := float64(body.RangeBox().Max.X); maxX < 3.4 {
		t.Errorf("loft did not follow the rail: max x = %.3f, want ≈3.5 (ruled would be 2.0)", maxX)
	}
}

// TestLoftToolCenterlineBends drives the Loft UI in centerline mode: two circle sections plus a
// spine PATH that bows to x=2 — the loft must bend along it (its centroid moves off-axis).
func TestLoftToolCenterlineBends(t *testing.T) {
	t.Parallel()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	base := def.Sketches().Add(sketch.XYPlane())
	base.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	top := def.Sketches().Add(planeAtZ(4))
	top.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	// Spine on XZ: (u,v)→(u,0,v), bowing to x=2 at mid; touches both circle centres.
	spine := def.Sketches().Add(sketch.XZPlane())
	a := spine.Points().Add(math.P2(0, 0))
	mid := spine.Points().Add(math.P2(2, 2))
	b := spine.Points().Add(math.P2(0, 4))
	spine.Lines().Add(a, mid)
	spine.Lines().Add(mid, b)

	s.SetPicker(&seqPicker{sels: []Selectable{
		ProfileHandle{Sketch: base, ProfileIndex: 0},
		ProfileHandle{Sketch: top, ProfileIndex: 0},
		PathHandle{Sketch: spine, PathIndex: 0},
	}})
	l := NewLoftTool()
	s.StartTool(l)
	s.Click(0, 0) // base circle
	s.Click(0, 0) // top circle
	l.SetUseCenterline(true)
	s.Click(0, 0) // the spine path → centerline
	if l.SectionCount() != 2 || !l.HasCenterline() {
		t.Fatalf("loft picks: %d sections, centerline=%v; want 2 + a centerline", l.SectionCount(), l.HasCenterline())
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("centerlined loft not a valid solid: %+v", r)
	}
	if cx := float64(ops.BodyGeometryProperties(body, ops.DefaultQuality()).Centroid.X); cx < 0.5 {
		t.Errorf("loft did not bend along the centerline: centroid x = %.3f, want > 0.5", cx)
	}
}

func TestLoftToolNeedsTwoSections(t *testing.T) {
	t.Parallel()
	s, bottom, top := newPartWithStackedSquares(t)
	s.SetPicker(&seqPicker{sels: []Selectable{bottom, top}})
	l := NewLoftTool()
	s.StartTool(l)
	s.Click(0, 0) // one section only
	if l.CanCommit() {
		t.Error("loft ready with a single section")
	}
	s.Click(0, 0) // second section
	if !l.CanCommit() {
		t.Error("loft not ready after two sections")
	}
}
