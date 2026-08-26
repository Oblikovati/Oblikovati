// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// matchPatch builds a degree-3 5×5 B-spline patch at x∈[xoff,xoff+1] with height field z(i,j),
// clamped uniform knots (compatible for matching).
func matchPatch(t *testing.T, xoff float64, z func(i, j int) float64) geom.BSplineSurface {
	t.Helper()
	const n = 5
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := range n {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := range n {
			ctrl[i][j] = math.P3(math.Scalar(xoff+float64(i)*0.25), math.Scalar(float64(j)*0.25), math.Scalar(z(i, j)))
			w[i][j] = 1
		}
	}
	k := uniformClampedKnots(n-1, 3) // helper in control_point_edit_test.go (same package)
	s, err := geom.NewBSplineSurface(3, 3, ctrl, w, k, k)
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	return s
}

func TestMatchFeatureMakesSurfaceTangent(t *testing.T) {
	target := surfaceBodyFrom(t, matchPatch(t, 0, func(i, j int) float64 { return 0.3 * float64(i) }))
	src := surfaceBodyFrom(t, matchPatch(t, 1, func(i, j int) float64 { return 0 }))

	f := &MatchFeature{def: &MatchDefinition{Order: 1, SourceEdge: geom.UMinEdge, TargetEdge: geom.UMaxEdge}, featName: "Match"}
	out, err := f.Recompute(Input{Bodies: []*topo.Body{target, src}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 2 {
		t.Fatalf("expected 2 bodies, got %d", len(out.Bodies))
	}
	matched, ok := out.Bodies[1].Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("matched face is %T, want BSplineSurface", out.Bodies[1].Faces()[0].Geometry())
	}
	// G1: the matched surface's u=0 first derivative equals the target's u=1 first derivative.
	tgt := matchPatch(t, 0, func(i, j int) float64 { return 0.3 * float64(i) })
	for _, v := range []float64{0, 0.5, 1} {
		sd := matched.SurfaceDersAt(0, v, 1, 0)[1][0]
		td := tgt.SurfaceDersAt(1, v, 1, 0)[1][0]
		if !sd.IsEqualTo(td, 1e-7) {
			t.Fatalf("G1 tangent mismatch at v=%g: %v vs %v", v, sd, td)
		}
	}
}

func TestMatchFeatureErrorsWithoutTarget(t *testing.T) {
	src := surfaceBodyFrom(t, matchPatch(t, 1, func(i, j int) float64 { return 0 }))
	f := &MatchFeature{def: &MatchDefinition{Order: 1}}
	if _, err := f.Recompute(Input{Bodies: []*topo.Body{src}}); err == nil {
		t.Error("match with only one body should error (no target)")
	}
}

func TestMatchFeatureKind(t *testing.T) {
	f := &MatchFeature{def: &MatchDefinition{}}
	if f.Kind() != "match-surface" {
		t.Errorf("Kind = %q, want match-surface", f.Kind())
	}
}

func TestMatchFeatureRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewMatchFeatures(fs).Add(2, geom.UMinEdge, geom.UMaxEdge)
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	d := fresh.Item(0).Definition().(*MatchFeature).Definition()
	if d.Order != 2 || d.SourceEdge != geom.UMinEdge || d.TargetEdge != geom.UMaxEdge {
		t.Errorf("restored recipe = %+v, want {2 UMin UMax}", d)
	}
}

func TestRestoreMatchRejectsMissingPayload(t *testing.T) {
	if _, err := restoreMatch(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreMatch(nil) should error")
	}
}
