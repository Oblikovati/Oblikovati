// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

func TestExtendSurfaceFeatureGrowsSurface(t *testing.T) {
	t.Parallel()
	body := surfaceBodyFrom(t, matchPatch(t, 0, func(i, j int) float64 { return 0.5 * float64(i*i) }))
	f := &ExtendSurfaceFeature{def: &ExtendSurfaceDefinition{Edge: geom.UMaxEdge, Distance: 0.5, Order: 2}, featName: "ExtendSurface"}
	out, err := f.Recompute(Input{Bodies: []*topo.Body{body}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	bs, ok := out.Bodies[0].Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("extended face is %T, want BSplineSurface", out.Bodies[0].Faces()[0].Geometry())
	}
	if _, hi := bs.UDomain(); hi <= 1+1e-9 {
		t.Errorf("extended u-domain max = %g, want > 1", hi)
	}
}

func TestExtendSurfaceFeatureErrorsWithoutBody(t *testing.T) {
	t.Parallel()
	f := &ExtendSurfaceFeature{def: &ExtendSurfaceDefinition{Edge: geom.UMaxEdge, Distance: 1, Order: 2}}
	if _, err := f.Recompute(Input{}); err == nil {
		t.Error("extend with no body should error")
	}
}

func TestUntrimFeatureRecoversFace(t *testing.T) {
	t.Parallel()
	body := surfaceBodyFrom(t, matchPatch(t, 0, func(i, j int) float64 { return 0.3 * float64(i*j) }))
	f := &UntrimFeature{featName: "Untrim"}
	out, err := f.Recompute(Input{Bodies: []*topo.Body{body}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies[0].Faces()) != 1 || len(out.Bodies[0].Edges()) != 4 {
		t.Errorf("untrimmed body = %d faces, %d edges; want 1 face, 4 edges", len(out.Bodies[0].Faces()), len(out.Bodies[0].Edges()))
	}
}

func TestUntrimFeatureErrorsWithoutBody(t *testing.T) {
	t.Parallel()
	f := &UntrimFeature{}
	if _, err := f.Recompute(Input{}); err == nil {
		t.Error("untrim with no body should error")
	}
}

func TestExtendUntrimKinds(t *testing.T) {
	t.Parallel()
	if (&ExtendSurfaceFeature{def: &ExtendSurfaceDefinition{}}).Kind() != "extend-surface" {
		t.Error("extend kind")
	}
	if (&UntrimFeature{}).Kind() != "untrim-surface" {
		t.Error("untrim kind")
	}
}

func TestExtendSurfaceRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewExtendSurfaceFeatures(fs).Add(geom.VMaxEdge, 2.5, 2)
	NewUntrimFeatures(fs).Add()
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	d := fresh.Item(0).Definition().(*ExtendSurfaceFeature).Definition()
	if d.Edge != geom.VMaxEdge || d.Distance != 2.5 || d.Order != 2 {
		t.Errorf("restored extend = %+v, want {VMax 2.5 2}", d)
	}
	if fresh.Item(1).Kind() != "untrim-surface" {
		t.Errorf("second feature kind = %q, want untrim-surface", fresh.Item(1).Kind())
	}
}

func TestRestoreExtendUntrimRejectMissingPayload(t *testing.T) {
	t.Parallel()
	if _, err := restoreExtendSurface(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreExtendSurface(nil) should error")
	}
	if _, err := restoreUntrim(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreUntrim(nil) should error")
	}
}
