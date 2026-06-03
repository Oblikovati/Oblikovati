// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/doc"
	"github.com/Oblikovati/oblikovati/model/feature"
)

// newPartWithSurface returns a session whose active part's running body is a 2×3 planar
// surface patch (a sheet) — the input a thicken turns into a solid.
func newPartWithSurface(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	feature.NewBaseFeatures(def.Features()).AddBase(patchSurface(2, 3))
	def.Recompute()
	if def.SurfaceBodies().Count() != 1 || def.SurfaceBodies().Item(0).IsSolid() {
		t.Fatal("expected one surface (non-solid) base body")
	}
	return s
}

// patchSurface builds a one-face planar surface body [0,w]×[0,h] at z=0.
func patchSurface(w, h float64) *topo.Body {
	lin := topo.NewLineage(topo.Tok("test", "patch", 0))
	bld := topo.NewBuilder(false, lin)
	p := []math.Point3{{X: 0, Y: 0}, {X: w, Y: 0}, {X: w, Y: h}, {X: 0, Y: h}}
	v := make([]*topo.Vertex, 4)
	for i, q := range p {
		v[i] = bld.AddVertex(q, lin)
	}
	uses := make([]topo.Use, 4)
	for i := range p {
		e := bld.AddEdge(geom.NewLineSegment(p[i], p[(i+1)%4]), v[i], v[(i+1)%4], lin)
		uses[i] = topo.Use{Edge: e}
	}
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, lin, topo.OuterLoop(uses...))
	return bld.Build()
}

// TestThickenToolEndToEnd drives the Thicken UI: with a 2×3 surface patch active, start the
// tool, set thickness 0.5, OK — and asserts a valid slab solid of volume 3.
func TestThickenToolEndToEnd(t *testing.T) {
	s := newPartWithSurface(t)
	th := NewThickenTool()
	s.StartTool(th)
	th.SetThickness(0.5)
	if !th.CanCommit() {
		t.Fatal("thicken not ready after thickness")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("thickened body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, 3) > 1e-6 {
		t.Errorf("slab volume = %g, want 3", got)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

// TestThickenViaRibbonCommand starts the tool from its ribbon command.
func TestThickenViaRibbonCommand(t *testing.T) {
	s := newPartWithSurface(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Modify.Thicken"); err != nil {
		t.Fatalf("execute Modify.Thicken: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*ThickenTool); !ok {
		t.Fatal("Thicken command did not start the tool")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if !activePartDef(t, s).SurfaceBodies().Item(0).IsSolid() {
		t.Error("thicken did not produce a solid")
	}
}
