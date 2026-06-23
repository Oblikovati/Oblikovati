// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// seedSurfaceFeature injects a fixed surface body into the running state — a test seam for
// putting a freeform (NURBS) surface in front of the Rebuild tool, which the standard part
// pipeline cannot produce without an import.
type seedSurfaceFeature struct{ body *topo.Body }

func (f *seedSurfaceFeature) Kind() string { return "seed-surface" }
func (f *seedSurfaceFeature) Recompute(in feature.Input) (feature.Output, error) {
	return feature.Output{Bodies: append(append([]*topo.Body(nil), in.Bodies...), f.body)}, nil
}

// multiSpanNurbsBody is a genuinely single-span bicubic patch carried on extra spans (F01 knot
// insertion), wrapped as a one-face surface body — the over-defined surface Rebuild cleans up.
func multiSpanNurbsBody(t *testing.T) *topo.Body {
	t.Helper()
	ctrl := make([][]math.Point3, 4)
	w := make([][]float64, 4)
	for i := 0; i < 4; i++ {
		ctrl[i] = make([]math.Point3, 4)
		w[i] = []float64{1, 1, 1, 1}
		for j := 0; j < 4; j++ {
			ctrl[i][j] = math.P3(float64(i), float64(j), float64((i-1)*(j-1))*0.4)
		}
	}
	bez := []float64{0, 0, 0, 0, 1, 1, 1, 1}
	s, err := geom.NewBSplineSurface(3, 3, ctrl, w, bez, bez)
	if err != nil {
		t.Fatalf("bicubic: %v", err)
	}
	if s, err = s.InsertKnotU(0.5, 1); err != nil {
		t.Fatalf("InsertKnotU: %v", err)
	}
	if s, err = s.InsertKnotV(0.5, 1); err != nil {
		t.Fatalf("InsertKnotV: %v", err)
	}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("seed", "body", 0)))
	corners := [4]math.Point3{s.PointAt(0, 0), s.PointAt(1, 0), s.PointAt(1, 1), s.PointAt(0, 1)}
	v := make([]*topo.Vertex, 4)
	for i, p := range corners {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("seed", "v", i)))
	}
	uses := make([]topo.Use, 4)
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(corners[i], corners[j]), v[i], v[j], topo.NewLineage(topo.Tok("seed", "e", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(s, topo.NewLineage(topo.Tok("seed", "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// partWithNurbsSurface returns a session whose active part's running state holds one freeform
// (NURBS) surface body, ready for the Rebuild tool.
func partWithNurbsSurface(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	def.Features().Add(&seedSurfaceFeature{body: multiSpanNurbsBody(t)})
	def.Recompute()
	return s, def
}

func TestSurfaceRebuildToolEndToEnd(t *testing.T) {
	s, def := partWithNurbsSurface(t)
	tool := NewSurfaceRebuildTool()
	s.StartTool(tool)
	if !tool.CanCommit() {
		t.Fatal("rebuild tool should be committable with the default 3,3,4,4 target")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after rebuild: %d surface bodies, want 1 (rebuild replaces the running surface)", def.SurfaceBodies().Count())
	}
	bs, ok := def.SurfaceBodies().Item(0).Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("rebuilt face geometry is %T, want geom.BSplineSurface", def.SurfaceBodies().Item(0).Faces()[0].Geometry())
	}
	if len(bs.Ctrl) != 4 || len(bs.Ctrl[0]) != 4 {
		t.Errorf("rebuilt net = %dx%d, want a clean 4x4 single span", len(bs.Ctrl), len(bs.Ctrl[0]))
	}
	if !scrollbackMentions(s, "deviation") {
		t.Error("the Command Window should report the achieved rebuild deviation")
	}
}

// scrollbackMentions reports whether any Command Window line contains substr.
func scrollbackMentions(s *Session, substr string) bool {
	for _, l := range s.CommandLine().Scrollback().Lines() {
		if strings.Contains(l.Text, substr) {
			return true
		}
	}
	return false
}

func TestSurfaceRebuildToolReportsNoFreeformFace(t *testing.T) {
	// A planar boundary patch has only an unbounded-domain analytic face: nothing to rebuild.
	s, def, region := partWithSquareRegion(t)
	feature.NewBoundaryPatchFeatures(def.Features()).Add(region.Sketch, region.ProfileIndex, feature.PatchFree)
	def.Recompute()

	tool := NewSurfaceRebuildTool()
	s.StartTool(tool)
	if err := s.OK(); err == nil {
		t.Error("rebuilding a planar patch should error (no rebuildable face)")
	}
}

func TestSurfaceRebuildToolCanCommit(t *testing.T) {
	tool := NewSurfaceRebuildTool()
	if !tool.CanCommit() {
		t.Error("defaults (3,3,4,4) should be committable")
	}
	tool.Params().Ints[2].Set(3) // U control points = 3 < uDegree+1 = 4
	if tool.CanCommit() {
		t.Error("a U control count below degree+1 must block commit")
	}
}

func TestSurfaceRebuildToolParamsRoundTrip(t *testing.T) {
	tool := NewSurfaceRebuildTool()
	if tool.Prompt(nil) == "" {
		t.Error("rebuild tool should have a non-empty prompt")
	}
	p := tool.Params()
	if len(p.Ints) != 4 {
		t.Fatalf("want 4 integer params, got %d", len(p.Ints))
	}
	for i, ip := range p.Ints {
		ip.Set(5 + i)
		if ip.Get() != 5+i {
			t.Errorf("param %q get/set mismatch: got %d", ip.Label, ip.Get())
		}
	}
}

func TestSurfaceRebuildViaRibbonCommand(t *testing.T) {
	s, _ := partWithNurbsSurface(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Rebuild"); err != nil {
		t.Fatalf("execute Surface.Rebuild: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Rebuild Surface" {
		t.Errorf("Surface.Rebuild started tool %q, want Rebuild Surface", got)
	}
}
