// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/health"
)

// addBody is a fake feature that appends one new body to the running state.
type addBody struct {
	kind string
	mk   func() *topo.Body
}

func (f addBody) Kind() string { return f.kind }
func (f addBody) Recompute(in Input) (Output, error) {
	return Output{Bodies: append(append([]*topo.Body(nil), in.Bodies...), f.mk())}, nil
}

// failer is a fake feature that always fails (→ sick).
type failer struct{}

func (failer) Kind() string                    { return "failer" }
func (failer) Recompute(Input) (Output, error) { return Output{}, errors.New("boom") }

func makeBody() *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("f", "body", 0)))
	v := bld.AddVertex(math.P3(0, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 0)))
	e := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), v, v, topo.NewLineage(topo.Tok("f", "edge", 0)))
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(e)))
	return bld.Build()
}

func body() addBody { return addBody{kind: "box", mk: makeBody} }

func TestRecomputeProducesBodies(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	fs.Add(body())
	fs.Add(body())
	fs.Recompute()
	if len(fs.Result()) != 2 {
		t.Fatalf("result has %d bodies, want 2", len(fs.Result()))
	}
	for i := 0; i < fs.Count(); i++ {
		if !fs.Item(i).Health().OK() {
			t.Errorf("feature %d not healthy", i)
		}
	}
}

func TestEditingEarlyFeatureReusesCleanPrefix(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	a := fs.Add(body())
	b := fs.Add(body())
	c := fs.Add(body())
	fs.Recompute() // all evaluated once
	for _, f := range []*PartFeature{a, b, c} {
		if f.RecomputeCount() != 1 {
			t.Fatalf("after first recompute, %s count = %d, want 1", f.Name(), f.RecomputeCount())
		}
	}
	// Edit the middle feature: it and its tail re-evaluate; the prefix (a) does not.
	fs.MarkDirty(b)
	fs.Recompute()
	if a.RecomputeCount() != 1 {
		t.Errorf("clean prefix feature a recomputed again: count=%d", a.RecomputeCount())
	}
	if b.RecomputeCount() != 2 || c.RecomputeCount() != 2 {
		t.Errorf("dirty tail not re-evaluated: b=%d c=%d, want 2/2", b.RecomputeCount(), c.RecomputeCount())
	}
}

func TestFailingFeatureGoesSickWithoutAbortingRebuild(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	good := fs.Add(body())
	bad := fs.Add(failer{})
	after := fs.Add(body()) // independent of bad → must still evaluate
	fs.Recompute()

	if !good.Health().OK() {
		t.Error("healthy feature before the failure went sick")
	}
	if bad.Health().Status != health.Sick {
		t.Errorf("failing feature health = %v, want sick", bad.Health().Status)
	}
	if !after.Health().OK() {
		t.Error("independent feature after the failure should still be healthy (rebuild not aborted)")
	}
}

func TestSickFeaturePoisonsDependents(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	bad := fs.Add(failer{})
	dependent := fs.Add(body(), bad.ID()) // depends on the failing feature
	fs.Recompute()
	if dependent.Health().Status != health.Sick {
		t.Errorf("dependent of a sick feature = %v, want sick (poisoned)", dependent.Health().Status)
	}
	if dependent.RecomputeCount() != 0 {
		t.Error("poisoned feature should not run its recompute")
	}
}

func TestLookupAndRename(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	f := fs.Add(body())
	f.SetName("Extrusion1")
	id := f.ID()
	f.SetName("Boss") // rename is id-stable
	if f.ID() != id {
		t.Error("rename changed the feature id")
	}
	if got, ok := fs.ByName("Boss"); !ok || got != f {
		t.Error("ByName did not find the renamed feature")
	}
	if got, ok := fs.ByID(id); !ok || got != f {
		t.Error("ByID failed after rename")
	}
}
