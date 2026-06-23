// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// fillNeighbour builds a flat 5×5 bicubic surface body over [x0,x0+1]×[y0,y0+1] at z=0 — one of the
// four neighbours bounding a unit-square opening.
func fillNeighbour(t *testing.T, x0, y0 float64) *topo.Body {
	t.Helper()
	const n = 5
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := 0; i < n; i++ {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			ctrl[i][j] = math.P3(math.Scalar(x0+float64(i)*0.25), math.Scalar(y0+float64(j)*0.25), 0)
			w[i][j] = 1
		}
	}
	k := uniformClampedKnots(n-1, 3)
	s, err := geom.NewBSplineSurface(3, 3, ctrl, w, k, k)
	if err != nil {
		t.Fatalf("neighbour patch: %v", err)
	}
	return surfaceBodyFrom(t, s)
}

func openingNeighbours(t *testing.T) []*topo.Body {
	return []*topo.Body{
		fillNeighbour(t, -1, 0), // west
		fillNeighbour(t, 1, 0),  // east
		fillNeighbour(t, 0, -1), // south
		fillNeighbour(t, 0, 1),  // north
	}
}

func TestFillFeatureClosesOpening(t *testing.T) {
	f := &FillFeature{def: &FillDefinition{Order: 0}, featName: "Fill"}
	out, err := f.Recompute(Input{Bodies: openingNeighbours(t)})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 5 {
		t.Fatalf("fill should append a body (4 neighbours + 1 fill), got %d", len(out.Bodies))
	}
	bs, ok := out.Bodies[4].Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("fill face is %T, want BSplineSurface", out.Bodies[4].Faces()[0].Geometry())
	}
	for i := 0; i <= 4; i++ {
		for j := 0; j <= 4; j++ {
			if p := bs.PointAt(float64(i)/4, float64(j)/4); p.Z < -1e-7 || p.Z > 1e-7 {
				t.Fatalf("planar opening fill left z=0 at (%d,%d): z=%g", i, j, p.Z)
			}
		}
	}
}

func TestFillFeatureErrorsWithoutFourBodies(t *testing.T) {
	f := &FillFeature{def: &FillDefinition{Order: 0}}
	if _, err := f.Recompute(Input{Bodies: openingNeighbours(t)[:3]}); err == nil {
		t.Error("fill with fewer than four bodies should error")
	}
}

func TestFillKind(t *testing.T) {
	if (&FillFeature{def: &FillDefinition{}}).Kind() != "fill-surface" {
		t.Error("fill kind")
	}
}

func TestFillSurfaceRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewFillFeatures(fs).Add(2)
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	d := fresh.Item(0).Definition().(*FillFeature).Definition()
	if d.Order != 2 {
		t.Errorf("restored fill order = %d, want 2", d.Order)
	}
}

func TestRestoreFillRejectMissingPayload(t *testing.T) {
	if _, err := restoreFillSurface(NewPartFeatures(nil, nil), nil); err == nil {
		t.Error("restoreFillSurface(nil) should error")
	}
}
