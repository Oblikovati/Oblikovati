// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// capRegion samples an n×n spherical-cap height field — a scanned region for the fit feature.
func capRegion(n int, r, half float64) []math.Point3 {
	pts := make([]math.Point3, 0, n*n)
	for i := range n {
		for j := range n {
			x := -half + 2*half*float64(i)/float64(n-1)
			y := -half + 2*half*float64(j)/float64(n-1)
			pts = append(pts, math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(stdmath.Sqrt(r*r-x*x-y*y))))
		}
	}
	return pts
}

func TestFitFeatureBuildsNurbsBody(t *testing.T) {
	f := &FitFeature{def: &FitDefinition{Points: capRegion(12, 10, 3), Degree: 3, NU: 5, NV: 5}}
	out, err := f.Recompute(Input{})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 1 {
		t.Fatalf("fit should append one body, got %d", len(out.Bodies))
	}
	if _, ok := out.Bodies[0].Faces()[0].Geometry().(geom.BSplineSurface); !ok {
		t.Fatalf("fitted face is %T, want BSplineSurface", out.Bodies[0].Faces()[0].Geometry())
	}
}

func TestFitFeatureErrorsWithoutEnoughPoints(t *testing.T) {
	f := &FitFeature{def: &FitDefinition{Points: capRegion(4, 10, 3), Degree: 3, NU: 5, NV: 5}}
	if _, err := f.Recompute(Input{}); err == nil {
		t.Error("fewer points than the control net should error")
	}
}

func TestFitKind(t *testing.T) {
	if (&FitFeature{def: &FitDefinition{}}).Kind() != "fit-surface" {
		t.Error("fit kind")
	}
}

func TestFitSurfaceRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewFitFeatures(fs).Add(capRegion(8, 10, 3), 3, 5, 5)
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	d := fresh.Item(0).Definition().(*FitFeature).Definition()
	if d.Degree != 3 || d.NU != 5 || d.NV != 5 || len(d.Points) != 64 {
		t.Errorf("restored fit = {deg %d nu %d nv %d pts %d}, want {3 5 5 64}", d.Degree, d.NU, d.NV, len(d.Points))
	}
}

func TestRestoreFitRejectsMissingPayload(t *testing.T) {
	if _, err := restoreFitSurface(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreFitSurface(nil) should error")
	}
}
