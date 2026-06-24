// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// flatNeighbourBody builds a flat 5×5 NURBS surface body over [x0,x0+1]×[y0,y0+1] at z=0.
func flatNeighbourBody(t *testing.T, tag string, x0, y0 float64) *topo.Body {
	t.Helper()
	const n = 5
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := 0; i < n; i++ {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			ctrl[i][j] = math.P3(math.Scalar(x0+float64(i)*0.25), math.Scalar(y0+float64(j)*0.25), 0)
			w[i][j] = 1
		}
	}
	k := []float64{0, 0, 0, 0, 0.5, 1, 1, 1, 1}
	s, err := geom.NewBSplineSurface(3, 3, ctrl, w, k, k)
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(tag, "body", 0)))
	c := [4]math.Point3{s.PointAt(0, 0), s.PointAt(1, 0), s.PointAt(1, 1), s.PointAt(0, 1)}
	v := make([]*topo.Vertex, 4)
	for i, p := range c {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(tag, "v", i)))
	}
	uses := make([]topo.Use, 4)
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(c[i], c[j]), v[i], v[j], topo.NewLineage(topo.Tok(tag, "e", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(s, topo.NewLineage(topo.Tok(tag, "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// partWithFourSidedOpening seeds a part with four planar surfaces bounding a unit-square opening.
func partWithFourSidedOpening(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "fill.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	def.Features().Add(&seedSurfaceFeature{body: flatNeighbourBody(t, "west", -1, 0)})
	def.Features().Add(&seedSurfaceFeature{body: flatNeighbourBody(t, "east", 1, 0)})
	def.Features().Add(&seedSurfaceFeature{body: flatNeighbourBody(t, "south", 0, -1)})
	def.Features().Add(&seedSurfaceFeature{body: flatNeighbourBody(t, "north", 0, 1)})
	def.Recompute()
	return s, def
}

func TestFillSurfaceToolClosesOpening(t *testing.T) {
	s, def := partWithFourSidedOpening(t)
	tool := NewFillSurfaceTool() // defaults to G2
	s.StartTool(tool)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if got := def.SurfaceBodies().Count(); got != 5 {
		t.Fatalf("after fill = %d surface bodies, want 5 (4 neighbours + fill)", got)
	}
	bs, ok := nurbsFaceSurface(def.SurfaceBodies().Item(def.SurfaceBodies().Count() - 1))
	if !ok {
		t.Fatal("fill body has no NURBS face")
	}
	if p := bs.PointAt(0.5, 0.5); p.Z < -1e-6 || p.Z > 1e-6 {
		t.Errorf("planar opening fill center left z=0: %v", p)
	}
}

func TestFillSurfaceToolParams(t *testing.T) {
	tool := NewFillSurfaceTool()
	if tool.Prompt(nil) == "" || !tool.CanCommit() {
		t.Error("fill tool should prompt and be committable by default")
	}
	if tool.Name() != "Fill Surface" {
		t.Errorf("tool name = %q", tool.Name())
	}
	p := tool.Params()
	p.Choices[0].Set(0)
	if p.Choices[0].Get() != 0 {
		t.Error("continuity get/set round-trip mismatch")
	}
	p.Ints[0].Set(5)
	if tool.sides != 5 {
		t.Errorf("sides get/set round-trip mismatch: %d", tool.sides)
	}
	tool.continuity = -1
	if tool.CanCommit() {
		t.Error("out-of-range continuity should block commit")
	}
	tool.continuity, tool.sides = 2, 2
	if tool.CanCommit() {
		t.Error("fewer than three sides should block commit")
	}
}

func TestFillSurfaceViaRibbonCommand(t *testing.T) {
	s, _ := partWithFourSidedOpening(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Fill"); err != nil {
		t.Fatalf("execute Surface.Fill: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Fill Surface" {
		t.Errorf("Surface.Fill started tool %q, want Fill Surface", got)
	}
	if tool, ok := s.ActiveTool().Tool().(*FillSurfaceTool); ok {
		_ = tool.AddedFeature() // nil until committed
	}
}
