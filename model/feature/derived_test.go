// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// fakeBodySource is a local BodySource (compdef.PartComponentDefinition satisfies
// the real interface in production, but importing compdef here would cycle since
// compdef now owns a feature engine).
type fakeBodySource struct {
	bodies  *topo.SurfaceBodies
	version int
}

func newFakeBodySource() *fakeBodySource                     { return &fakeBodySource{bodies: topo.NewSurfaceBodies()} }
func (f *fakeBodySource) SurfaceBodies() *topo.SurfaceBodies { return f.bodies }
func (f *fakeBodySource) ModelGeometryVersion() string       { return fmt.Sprintf("v%d", f.version) }
func (f *fakeBodySource) edit()                              { f.version++ }

func oneBody() *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("src", "body", 0)))
	v := bld.AddVertex(math.P3(0, 0, 0), topo.NewLineage(topo.Tok("src", "vertex", 0)))
	e := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), v, v, topo.NewLineage(topo.Tok("src", "edge", 0)))
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, topo.NewLineage(topo.Tok("src", "face", 0)), topo.OuterLoop(topo.Fwd(e)))
	return bld.Build()
}

func TestDerivedComponentPullsSourceAssociatively(t *testing.T) {
	// The source is a BodySource (compdef.PartComponentDefinition in production).
	source := newFakeBodySource()
	source.SurfaceBodies().Add(oneBody())

	fs := NewPartFeatures(nil, nil)
	pf := NewDerivedComponents(fs).AddDerived(source, math.Identity4(), DeriveSourceLink{})
	fs.Recompute()
	if !pf.Health().OK() || len(fs.Result()) != 1 {
		t.Fatalf("derived pull: health=%+v bodies=%d, want ok/1", pf.Health(), len(fs.Result()))
	}
	v0 := pf.Definition().(*DerivedPartComponent).SourceVersion()

	// Edit the source (add a body, bump its version) → re-derive reflects it.
	source.SurfaceBodies().Add(oneBody())
	source.edit()
	fs.MarkDirty(pf)
	fs.Recompute()
	if len(fs.Result()) != 2 {
		t.Errorf("after source edit, derived has %d bodies, want 2", len(fs.Result()))
	}
	if pf.Definition().(*DerivedPartComponent).SourceVersion() == v0 {
		t.Error("source version did not advance on edit")
	}
}

func TestNonParametricBaseFeatureIsDownstreamEditable(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	base := NewBaseFeatures(fs).AddBase(oneBody())
	// A downstream derived/extrude feature consumes the base body's running state.
	source := newFakeBodySource()
	source.SurfaceBodies().Add(oneBody())
	NewDerivedComponents(fs).AddDerived(source, math.Identity4(), DeriveSourceLink{})

	fs.Recompute()
	if !base.Health().OK() {
		t.Fatalf("base feature sick: %+v", base.Health())
	}
	if got := base.Definition().(*NonParametricBaseFeature); len(got.Bodies()) != 1 {
		t.Errorf("base feature wraps %d bodies, want 1", len(got.Bodies()))
	}
	// Base body + derived body flow through the history into the result.
	if len(fs.Result()) != 2 {
		t.Errorf("result has %d bodies, want 2 (base + derived)", len(fs.Result()))
	}
	if base.Kind() != "base" {
		t.Errorf("base kind = %q", base.Kind())
	}
}
