// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

func TestBridgeFeatureConnectsPanels(t *testing.T) {
	a := surfaceBodyFrom(t, matchPatch(t, 0, func(i, j int) float64 { return 0.4 * float64(i*i) }))
	b := surfaceBodyFrom(t, matchPatch(t, 2, func(i, j int) float64 { return 0.3 * float64((4-i)*(4-i)) }))
	f := &BridgeFeature{def: &BridgeDefinition{OrderA: 2, OrderB: 2}, featName: "Bridge"}
	out, err := f.Recompute(Input{Bodies: []*topo.Body{a, b}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 3 {
		t.Fatalf("bridge should append a body (2 panels + bridge), got %d", len(out.Bodies))
	}
	bs, ok := out.Bodies[2].Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("bridge face is %T, want BSplineSurface", out.Bodies[2].Faces()[0].Geometry())
	}
	if x := float64(bs.PointAt(0.5, 0.5).X); x < 1.2 || x > 1.8 {
		t.Errorf("bridge mid x = %g, want between the panels (~1.5)", x)
	}
}

func TestBridgeFeatureErrorsWithoutTwoBodies(t *testing.T) {
	f := &BridgeFeature{def: &BridgeDefinition{}}
	a := surfaceBodyFrom(t, matchPatch(t, 0, func(i, j int) float64 { return 0 }))
	if _, err := f.Recompute(Input{Bodies: []*topo.Body{a}}); err == nil {
		t.Error("bridge with fewer than two bodies should error")
	}
}

func TestBridgeKind(t *testing.T) {
	if (&BridgeFeature{def: &BridgeDefinition{}}).Kind() != "bridge-surface" {
		t.Error("bridge kind")
	}
}

func TestBridgeSurfaceRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewBridgeFeatures(fs).Add(1, 2)
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	d := fresh.Item(0).Definition().(*BridgeFeature).Definition()
	if d.OrderA != 1 || d.OrderB != 2 {
		t.Errorf("restored bridge = %+v, want {1 2}", d)
	}
}

func TestRestoreBridgeRejectsMissingPayload(t *testing.T) {
	if _, err := restoreBridgeSurface(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreBridgeSurface(nil) should error")
	}
}
