// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// cubeFaceSheet builds a single-face surface body over the 4 corners (wound for an outward plane).
func cubeFaceSheet(feat string, p0, p1, p2, p3 math.Point3) *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	surf, _ := geom.NewPlane(p0, p0.VectorTo(p1).Cross(p1.VectorTo(p2)))
	pts := []math.Point3{p0, p1, p2, p3}
	v := make([]*topo.Vertex, 4)
	for i, p := range pts {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	uses := make([]topo.Use, 4)
	for i := range 4 {
		j := (i + 1) % 4
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(pts[i], pts[j]), v[i], v[j], topo.NewLineage(topo.Tok(feat, "edge", i))))
	}
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// partWithCubeShell gives a part whose running state is the six surface faces of a unit cube.
func partWithCubeShell(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s, def := emptyPartSession(t)
	p := math.P3
	faces := []*topo.Body{
		cubeFaceSheet("bottom", p(0, 0, 0), p(0, 1, 0), p(1, 1, 0), p(1, 0, 0)),
		cubeFaceSheet("top", p(0, 0, 1), p(1, 0, 1), p(1, 1, 1), p(0, 1, 1)),
		cubeFaceSheet("front", p(0, 0, 0), p(1, 0, 0), p(1, 0, 1), p(0, 0, 1)),
		cubeFaceSheet("back", p(0, 1, 0), p(0, 1, 1), p(1, 1, 1), p(1, 1, 0)),
		cubeFaceSheet("left", p(0, 0, 0), p(0, 0, 1), p(0, 1, 1), p(0, 1, 0)),
		cubeFaceSheet("right", p(1, 0, 0), p(1, 1, 0), p(1, 1, 1), p(1, 0, 1)),
	}
	feature.NewBaseFeatures(def.Features()).AddBase(faces...)
	def.Recompute()
	return s, def
}

// TestSculptToolEndToEnd drives the Sculpt UI: with the six cube faces present, OK fills the
// bounded volume into a single unit-volume solid.
func TestSculptToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, def := partWithCubeShell(t)

	s.StartTool(NewSculptTool())
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after sculpt: %d bodies, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if !body.IsSolid() {
		t.Error("sculpt of a closed shell should yield a solid")
	}
	if v := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; stdmath.Abs(v-1) > 1e-6 {
		t.Errorf("sculpted volume = %g, want 1 (unit cube)", v)
	}
}

func TestSculptViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, _ := partWithCubeShell(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Sculpt"); err != nil {
		t.Fatalf("execute Surface.Sculpt: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*SculptTool); !ok {
		t.Fatal("Sculpt command did not start the sculpt tool")
	}
}
