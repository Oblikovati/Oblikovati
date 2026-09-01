// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// nurbsGridBody wraps a flat n×n bicubic B-spline (z=0) in a one-face surface body — the
// freeform quilt a control-net edit shapes.
func nurbsGridBody(t *testing.T, n int) *topo.Body {
	t.Helper()
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := range n {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := range n {
			ctrl[i][j] = math.P3(float64(i)/float64(n-1), float64(j)/float64(n-1), 0)
			w[i][j] = 1
		}
	}
	k := uniformClampedKnots(n-1, 3)
	s, err := geom.NewBSplineSurface(3, 3, ctrl, w, k, k)
	if err != nil {
		t.Fatalf("nurbs grid: %v", err)
	}
	return surfaceBodyFrom(t, s) // helper in rebuild_surface_test.go (same package)
}

// uniformClampedKnots is a clamped degree-p knot vector with evenly spaced interiors for n+1 CVs.
func uniformClampedKnots(n, p int) []float64 {
	nctrl := n + 1
	knots := make([]float64, nctrl+p+1)
	interior := nctrl - p - 1
	for j := 1; j <= interior; j++ {
		knots[p+j] = float64(j) / float64(interior+1)
	}
	for i := nctrl; i < nctrl+p+1; i++ {
		knots[i] = 1
	}
	return knots
}

func TestControlPointEditLiftsSurfaceKeepingStructure(t *testing.T) {
	t.Parallel()
	body := nurbsGridBody(t, 5)
	deltas := []geom.ControlPointDelta{{U: 2, V: 2, Delta: math.V3(0, 0, 1)}}
	f := &ControlPointEditFeature{def: &ControlPointEditDefinition{Deltas: deltas}, featName: "EditControlPoints"}
	out, err := f.Recompute(Input{Bodies: []*topo.Body{body}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	bs, ok := out.Bodies[0].Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("edited face geometry is %T, want geom.BSplineSurface", out.Bodies[0].Faces()[0].Geometry())
	}
	if bs.UDegree != 3 || bs.VDegree != 3 || len(bs.Ctrl) != 5 || len(bs.Ctrl[0]) != 5 {
		t.Errorf("edit changed structure: degree %dx%d net %dx%d, want 3x3 / 5x5", bs.UDegree, bs.VDegree, len(bs.Ctrl), len(bs.Ctrl[0]))
	}
	if z := bs.PointAt(0.5, 0.5).Z; z <= 0.1 {
		t.Errorf("limit surface barely rose after lifting the centre CV: z=%g", z)
	}
}

func TestControlPointEditErrorsWithoutBody(t *testing.T) {
	t.Parallel()
	f := &ControlPointEditFeature{def: &ControlPointEditDefinition{}}
	if _, err := f.Recompute(Input{}); err == nil {
		t.Error("control-point edit with no target body should error")
	}
}

func TestControlPointEditKind(t *testing.T) {
	t.Parallel()
	f := &ControlPointEditFeature{def: &ControlPointEditDefinition{}}
	if f.Kind() != "control-point-edit" {
		t.Errorf("Kind = %q, want control-point-edit", f.Kind())
	}
}

func TestControlPointEditRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewControlPointEditFeatures(fs).Add([]geom.ControlPointDelta{
		{U: 1, V: 2, Delta: math.V3(0.1, -0.2, 0.3)},
		{U: 3, V: 0, Delta: math.V3(0, 0, 0.5)},
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	d := fresh.Item(0).Definition().(*ControlPointEditFeature).Definition()
	if len(d.Deltas) != 2 {
		t.Fatalf("restored %d deltas, want 2", len(d.Deltas))
	}
	if d.Deltas[0].U != 1 || d.Deltas[0].V != 2 || !d.Deltas[0].Delta.IsEqualTo(math.V3(0.1, -0.2, 0.3), 1e-12) {
		t.Errorf("restored delta[0] = %+v, want {1 2 (0.1,-0.2,0.3)}", d.Deltas[0])
	}
}

func TestRestoreControlPointEditRejectsMissingPayload(t *testing.T) {
	t.Parallel()
	if _, err := restoreControlPointEdit(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreControlPointEdit(nil) should error")
	}
}
