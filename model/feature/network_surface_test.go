// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// gridPolylines builds a 3×3 grid of U- and V-direction polylines in z=0 (each a 5-point line).
func gridPolylines() (u, v [][]math.Point3) {
	line := func(x0, y0, x1, y1 float64) []math.Point3 {
		out := make([]math.Point3, 5)
		for k := range 5 {
			t := math.Scalar(float64(k) / 4)
			out[k] = math.P3(math.Scalar(x0)+(math.Scalar(x1)-math.Scalar(x0))*t, math.Scalar(y0)+(math.Scalar(y1)-math.Scalar(y0))*t, 0)
		}
		return out
	}
	for j := range 3 { // u-curves along x at y=0,1,2
		u = append(u, line(0, float64(j), 2, float64(j)))
	}
	for i := range 3 { // v-curves along y at x=0,1,2
		v = append(v, line(float64(i), 0, float64(i), 2))
	}
	return u, v
}

func TestNetworkFeatureBuildsSurface(t *testing.T) {
	t.Parallel()
	u, v := gridPolylines()
	f := &NetworkFeature{def: &NetworkDefinition{UCurves: u, VCurves: v}, featName: "Network"}
	out, err := f.Recompute(Input{})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 1 {
		t.Fatalf("network should append one body, got %d", len(out.Bodies))
	}
	bs, ok := out.Bodies[0].Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("network face is %T, want BSplineSurface", out.Bodies[0].Faces()[0].Geometry())
	}
	if p := bs.PointAt(0.5, 0.5); p.Z < -1e-6 || p.Z > 1e-6 {
		t.Errorf("flat network left z=0: %v", p)
	}
}

func TestNetworkFeatureErrorsWithoutGrid(t *testing.T) {
	t.Parallel()
	u, v := gridPolylines()
	f := &NetworkFeature{def: &NetworkDefinition{UCurves: u[:1], VCurves: v}}
	if _, err := f.Recompute(Input{}); err == nil {
		t.Error("network with one u-curve should error")
	}
}

func TestNetworkKind(t *testing.T) {
	t.Parallel()
	if (&NetworkFeature{def: &NetworkDefinition{}}).Kind() != "network-surface" {
		t.Error("network kind")
	}
}

func TestNetworkSurfaceRoundTrip(t *testing.T) {
	t.Parallel()
	u, v := gridPolylines()
	fs := NewPartFeatures(nil)
	NewNetworkFeatures(fs).Add(u, v)
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	d := fresh.Item(0).Definition().(*NetworkFeature).Definition()
	if len(d.UCurves) != 3 || len(d.VCurves) != 3 || len(d.UCurves[0]) != 5 {
		t.Errorf("restored network = %d u / %d v curves (u0 %d pts), want 3/3/5", len(d.UCurves), len(d.VCurves), len(d.UCurves[0]))
	}
}

func TestRestoreNetworkRejectsMissingPayload(t *testing.T) {
	t.Parallel()
	if _, err := restoreNetworkSurface(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreNetworkSurface(nil) should error")
	}
}
