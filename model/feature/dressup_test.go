// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/health"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// prismBody builds a unit-square prism via the extrude generator (for real edges).
func prismBody() *topo.Body {
	return buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}, sketch.XYPlane(), 1, "ext")
}

func TestFilletResolvesEdgeThenDefers(t *testing.T) {
	body := prismBody()
	edgeKey := body.Edges()[0].ReferenceKey()

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(body) // running body the fillet operates on
	fillet := NewDressUpFeatures(fs).AddFillet([][]byte{edgeKey}, func() float64 { return 0.2 })
	fs.Recompute()

	// The edge resolves → the feature is healthy-but-deferred (Warning), not sick.
	if fillet.Health().Status != health.Warning {
		t.Fatalf("fillet health = %v, want warning (input resolved, geometry deferred)", fillet.Health().Status)
	}
}

func TestFilletGoesSickOnLostEdge(t *testing.T) {
	body := prismBody()
	bogus := []byte("not-an-edge-key")

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(body)
	fillet := NewDressUpFeatures(fs).AddFillet([][]byte{bogus}, func() float64 { return 0.2 })
	fs.Recompute()
	if fillet.Health().Status != health.Sick {
		t.Errorf("fillet with a lost edge = %v, want sick", fillet.Health().Status)
	}
}

func TestDressUpWithNoBodyIsSick(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	fillet := NewDressUpFeatures(fs).AddFillet([][]byte{[]byte("x")}, func() float64 { return 1 })
	fs.Recompute() // no preceding body
	if fillet.Health().Status != health.Sick {
		t.Errorf("dress-up with no body = %v, want sick", fillet.Health().Status)
	}
}

func TestChamferShellDraftThreadResolveInputs(t *testing.T) {
	body := prismBody()
	edge := body.Edges()[0].ReferenceKey()
	face := body.Faces()[0].ReferenceKey()

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(body)
	du := NewDressUpFeatures(fs)
	feats := map[string]*PartFeature{
		"chamfer": du.AddChamfer([][]byte{edge}, func() float64 { return 0.1 }),
		"shell":   du.AddShell([][]byte{face}, func() float64 { return 0.2 }),
		"draft":   du.AddDraft([][]byte{face}, func() float64 { return 0.05 }),
		"thread":  du.AddThread(face, "M6x1"),
	}
	fs.Recompute()
	for name, pf := range feats {
		if pf.Health().Status != health.Warning {
			t.Errorf("%s health = %v, want warning (resolved + deferred)", name, pf.Health().Status)
		}
		if pf.Kind() != name {
			t.Errorf("kind = %q, want %q", pf.Kind(), name)
		}
	}
	// A shell referencing a vanished face goes sick.
	bad := du.AddShell([][]byte{[]byte("ghost")}, func() float64 { return 0.2 })
	fs.MarkDirty(bad)
	fs.Recompute()
	if bad.Health().Status != health.Sick {
		t.Error("shell with a lost face should be sick")
	}
}

func TestDressUpDefinitionsAccessible(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	du := NewDressUpFeatures(fs)
	f := du.AddFillet([][]byte{[]byte("e")}, func() float64 { return 0.2 })
	if f.Definition().(*FilletFeature).Definition().Radius() != 0.2 {
		t.Error("fillet radius not accessible via definition")
	}
	if (&ChamferFeature{def: &ChamferDefinition{}}).Definition() == nil ||
		(&ShellFeature{def: &ShellDefinition{}}).Definition() == nil ||
		(&FaceDraftFeature{def: &FaceDraftDefinition{}}).Definition() == nil ||
		(&ThreadFeature{def: &ThreadDefinition{Designation: "M6"}}).Definition().Designation != "M6" {
		t.Error("a dress-up definition is not accessible")
	}
}
