// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

func TestFairFeatureReplacesSurface(t *testing.T) {
	body := surfaceBodyFrom(t, matchPatch(t, 0, func(i, j int) float64 { return 0.3 * float64((i*3+j)%2) }))
	f := &FairFeature{def: &FairDefinition{HoldOrder: 1, Strength: 0.5, Iterations: 10}, featName: "Fair"}
	out, err := f.Recompute(Input{Bodies: []*topo.Body{body}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 1 {
		t.Fatalf("fair should replace the body, got %d", len(out.Bodies))
	}
	if _, ok := out.Bodies[0].Faces()[0].Geometry().(geom.BSplineSurface); !ok {
		t.Fatalf("faired face is %T, want BSplineSurface", out.Bodies[0].Faces()[0].Geometry())
	}
}

func TestFairFeatureErrorsWithoutBody(t *testing.T) {
	f := &FairFeature{def: &FairDefinition{HoldOrder: 1, Strength: 0.5, Iterations: 5}}
	if _, err := f.Recompute(Input{}); err == nil {
		t.Error("fair with no body should error")
	}
}

func TestFairKind(t *testing.T) {
	if (&FairFeature{def: &FairDefinition{}}).Kind() != "fair-surface" {
		t.Error("fair kind")
	}
}

func TestFairSurfaceRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewFairFeatures(fs).Add(2, 0.4, 25)
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	d := fresh.Item(0).Definition().(*FairFeature).Definition()
	if d.HoldOrder != 2 || d.Strength != 0.4 || d.Iterations != 25 {
		t.Errorf("restored fair = %+v, want {2 0.4 25}", d)
	}
}

func TestRestoreFairRejectsMissingPayload(t *testing.T) {
	if _, err := restoreFairSurface(NewPartFeatures(nil, nil), nil); err == nil {
		t.Error("restoreFairSurface(nil) should error")
	}
}
