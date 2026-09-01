// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
)

func TestNurbsPlaneFeatureBuildsFlatNurbsBody(t *testing.T) {
	t.Parallel()
	f := &NurbsPlaneFeature{def: &NurbsPlaneDefinition{Width: 10, Height: 6, UCount: 5, VCount: 4}, featName: "NurbsPlane"}
	out, err := f.Recompute(Input{})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 1 || out.Bodies[0].IsSolid() {
		t.Fatalf("want one surface body, got %d (solid=%v)", len(out.Bodies), len(out.Bodies) > 0 && out.Bodies[0].IsSolid())
	}
	bs, ok := out.Bodies[0].Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("face geometry is %T, want geom.BSplineSurface", out.Bodies[0].Faces()[0].Geometry())
	}
	if bs.UDegree != 3 || bs.VDegree != 3 || len(bs.Ctrl) != 5 || len(bs.Ctrl[0]) != 4 {
		t.Errorf("plane net = degree %dx%d, %dx%d CVs; want 3x3 / 5x4", bs.UDegree, bs.VDegree, len(bs.Ctrl), len(bs.Ctrl[0]))
	}
	// Flat at z=0, spanning the requested size.
	if p := bs.PointAt(1, 1); p.Z != 0 || float64(p.X) != 10 || float64(p.Y) != 6 {
		t.Errorf("far corner = %v, want (10,6,0)", p)
	}
}

func TestNurbsPlaneFeatureValidates(t *testing.T) {
	t.Parallel()
	if _, err := (&NurbsPlaneFeature{def: &NurbsPlaneDefinition{Width: 0, Height: 1, UCount: 4, VCount: 4}}).Recompute(Input{}); err == nil {
		t.Error("non-positive width should error")
	}
	if _, err := (&NurbsPlaneFeature{def: &NurbsPlaneDefinition{Width: 1, Height: 1, UCount: 3, VCount: 4}}).Recompute(Input{}); err == nil {
		t.Error("a control count below 4 (cubic) should error")
	}
}

func TestNurbsPlaneRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewNurbsPlaneFeatures(fs).Add(8, 4, 6, 5)
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	d := fresh.Item(0).Definition().(*NurbsPlaneFeature).Definition()
	if d.Width != 8 || d.Height != 4 || d.UCount != 6 || d.VCount != 5 {
		t.Errorf("restored recipe = %+v, want {8 4 6 5}", d)
	}
}

func TestRestoreNurbsPlaneRejectsMissingPayload(t *testing.T) {
	t.Parallel()
	if _, err := restoreNurbsPlane(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreNurbsPlane(nil) should error")
	}
}
