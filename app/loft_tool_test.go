// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/doc"
	"oblikovati/model/sketch"
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
	s, bottom, top := newPartWithStackedSquares(t)
	s.SetPicker(&seqPicker{sels: []Selectable{bottom, top}})

	l := NewLoftTool()
	s.StartTool(l)   // ribbon: click "Loft"
	s.Click(10, 10)  // viewport: click the bottom section
	s.Click(10, 200) // viewport: click the top section
	if !l.CanCommit() {
		t.Fatalf("loft not ready with %d sections", len(l.Sections()))
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

func TestLoftToolNeedsTwoSections(t *testing.T) {
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
