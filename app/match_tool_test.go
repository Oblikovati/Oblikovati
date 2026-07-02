// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// matchPatchBody builds a degree-3 5×5 NURBS patch at x∈[xoff,xoff+1] with height z(i,j), wrapped in
// a one-face surface body (clamped uniform knots → compatible across the seam).
func matchPatchBody(t *testing.T, xoff float64, z func(i, j int) float64) *topo.Body {
	t.Helper()
	const n = 5
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := 0; i < n; i++ {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			ctrl[i][j] = math.P3(math.Scalar(xoff+float64(i)*0.25), math.Scalar(float64(j)*0.25), math.Scalar(z(i, j)))
			w[i][j] = 1
		}
	}
	k := []float64{0, 0, 0, 0, 0.5, 1, 1, 1, 1}
	s, err := geom.NewBSplineSurface(3, 3, ctrl, w, k, k)
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("m", "body", 0)))
	c := [4]math.Point3{s.PointAt(0, 0), s.PointAt(1, 0), s.PointAt(1, 1), s.PointAt(0, 1)}
	v := make([]*topo.Vertex, 4)
	for i, p := range c {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("m", "v", i)))
	}
	uses := make([]topo.Use, 4)
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(c[i], c[j]), v[i], v[j], topo.NewLineage(topo.Tok("m", "e", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(s, topo.NewLineage(topo.Tok("m", "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// partWithTwoPatches seeds a part with a curved target patch then a flat source patch to its right.
func partWithTwoPatches(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "match.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	def.Features().Add(&seedSurfaceFeature{body: matchPatchBody(t, 0, func(i, j int) float64 { return 0.5 * float64(i*i) })})
	def.Features().Add(&seedSurfaceFeature{body: matchPatchBody(t, 1, func(i, j int) float64 { return 0 })})
	def.Recompute()
	return s, def
}

func TestMatchToolG2VerifiedByContinuityChecker(t *testing.T) {
	s, def := partWithTwoPatches(t)
	tool := NewMatchTool() // defaults: G2, source U Min, target U Max
	s.StartTool(tool)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	// The matched (last) surface vs the target (previous): F13 must report G2 continuity.
	src, _ := nurbsFaceSurface(def.SurfaceBodies().Item(def.SurfaceBodies().Count() - 1))
	tgt, _ := nurbsFaceSurface(def.SurfaceBodies().Item(def.SurfaceBodies().Count() - 2))
	rep := analysis.CrossEdgeContinuity(src, tgt, edgeParamOf(geom.UMinEdge), edgeParamOf(geom.UMaxEdge), 21)
	if rep.MaxGap > 1e-7 {
		t.Errorf("Match-G2 should close the gap (G0), got max %g", rep.MaxGap)
	}
	if rep.MaxNormalDeg > 1e-4 {
		t.Errorf("Match-G2 should be tangent (G1), got max angle %g°", rep.MaxNormalDeg)
	}
	if rep.MaxCurvPct > 0.5 {
		t.Errorf("Match-G2 should be curvature-continuous (G2), got max %g%%", rep.MaxCurvPct)
	}
}

func TestMatchToolG1LeavesCurvatureBreak(t *testing.T) {
	s, def := partWithTwoPatches(t)
	tool := NewMatchTool()
	tool.order = 1 // G1 only
	s.StartTool(tool)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	src, _ := nurbsFaceSurface(def.SurfaceBodies().Item(def.SurfaceBodies().Count() - 1))
	tgt, _ := nurbsFaceSurface(def.SurfaceBodies().Item(def.SurfaceBodies().Count() - 2))
	rep := analysis.CrossEdgeContinuity(src, tgt, edgeParamOf(geom.UMinEdge), edgeParamOf(geom.UMaxEdge), 21)
	if rep.MaxNormalDeg > 1e-4 {
		t.Errorf("Match-G1 should be tangent, got %g°", rep.MaxNormalDeg)
	}
	if rep.MaxCurvPct < 1 {
		t.Errorf("Match-G1 (only) should leave a curvature break against a curved target, got %g%%", rep.MaxCurvPct)
	}
}

func TestMatchToolParams(t *testing.T) {
	tool := NewMatchTool()
	if tool.Prompt(nil) == "" || !tool.CanCommit() {
		t.Error("match tool should prompt and be committable")
	}
	p := tool.Params()
	p.Choices[0].Set(3)
	p.Choices[1].Set(2)
	p.Choices[2].Set(1)
	if p.Choices[0].Get() != 3 || p.Choices[1].Get() != 2 || p.Choices[2].Get() != 1 {
		t.Error("param get/set round-trip mismatch")
	}
}

func TestEdgeParamOfAllEdges(t *testing.T) {
	cases := []struct {
		edge           geom.Boundary
		wantU0, wantV0 float64 // (u,v) at t=0
		wantU1, wantV1 float64 // (u,v) at t=1
	}{
		{geom.UMinEdge, 0, 0, 0, 1},
		{geom.UMaxEdge, 1, 0, 1, 1},
		{geom.VMinEdge, 0, 0, 1, 0},
		{geom.VMaxEdge, 0, 1, 1, 1},
	}
	for _, c := range cases {
		ep := edgeParamOf(c.edge)
		u0, v0 := ep(0)
		u1, v1 := ep(1)
		if u0 != c.wantU0 || v0 != c.wantV0 || u1 != c.wantU1 || v1 != c.wantV1 {
			t.Errorf("edge %d: got (%g,%g)→(%g,%g), want (%g,%g)→(%g,%g)", c.edge, u0, v0, u1, v1, c.wantU0, c.wantV0, c.wantU1, c.wantV1)
		}
	}
}

func TestMatchToolErrorsWithoutTarget(t *testing.T) {
	s, _ := emptyPartSession(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	def.Features().Add(&seedSurfaceFeature{body: matchPatchBody(t, 0, func(i, j int) float64 { return 0 })})
	def.Recompute()
	tool := NewMatchTool()
	s.StartTool(tool)
	if err := s.OK(); err == nil {
		t.Error("matching with only one surface body should error (no target)")
	}
}

func TestMatchViaRibbonCommand(t *testing.T) {
	s, _ := partWithTwoPatches(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Match"); err != nil {
		t.Fatalf("execute Surface.Match: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Match Surface" {
		t.Errorf("Surface.Match started tool %q, want Match Surface", got)
	}
}

// TestMatchToolDraftFeature pins the #1626 commit-gate seam: the match tool is always
// commit-ready (its choices are always valid), so the draft is always available — a missing
// target body is caught by the gate's preview, not by the draft.
func TestMatchToolDraftFeature(t *testing.T) {
	if draft, ok := NewMatchTool().DraftFeature(nil); !ok || draft == nil {
		t.Fatalf("DraftFeature = (%v, %v), want a non-nil draft from the defaults", draft, ok)
	}
}
