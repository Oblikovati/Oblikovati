// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// multiSpanSurfaceBody is a genuinely-cubic bicubic patch carried on extra spans (knots
// inserted via F01) wrapped in a single-face surface body — the over-defined input Rebuild
// cleans up to a single span.
func multiSpanSurfaceBody(t *testing.T) *topo.Body {
	t.Helper()
	ctrl := make([][]math.Point3, 4)
	w := make([][]float64, 4)
	for i := 0; i < 4; i++ {
		ctrl[i] = make([]math.Point3, 4)
		w[i] = []float64{1, 1, 1, 1}
		for j := 0; j < 4; j++ {
			ctrl[i][j] = math.P3(float64(i), float64(j), float64((i-1)*(j-1))*0.4)
		}
	}
	bez := []float64{0, 0, 0, 0, 1, 1, 1, 1}
	s, err := geom.NewBSplineSurface(3, 3, ctrl, w, bez, bez)
	if err != nil {
		t.Fatalf("bicubic: %v", err)
	}
	if s, err = s.InsertKnotU(0.5, 1); err != nil {
		t.Fatalf("InsertKnotU: %v", err)
	}
	if s, err = s.InsertKnotV(0.5, 1); err != nil {
		t.Fatalf("InsertKnotV: %v", err)
	}
	return surfaceBodyFrom(t, s)
}

// surfaceBodyFrom wraps a B-spline surface in a one-face surface body (straight corner edges).
func surfaceBodyFrom(t *testing.T, s geom.BSplineSurface) *topo.Body {
	t.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("rebuild", "body", 0)))
	corners := [4]math.Point3{s.PointAt(0, 0), s.PointAt(1, 0), s.PointAt(1, 1), s.PointAt(0, 1)}
	v := make([]*topo.Vertex, 4)
	for i, p := range corners {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("rebuild", "v", i)))
	}
	uses := make([]topo.Use, 4)
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(corners[i], corners[j]), v[i], v[j], topo.NewLineage(topo.Tok("rebuild", "e", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(s, topo.NewLineage(topo.Tok("rebuild", "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

func TestRebuildFeatureCleansFaceAndReportsDeviation(t *testing.T) {
	r := &RebuildFeature{def: &RebuildDefinition{UDegree: 3, VDegree: 3, UCount: 4, VCount: 4}, featName: "Rebuild"}
	out, err := r.Recompute(Input{Bodies: []*topo.Body{multiSpanSurfaceBody(t)}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 1 {
		t.Fatalf("expected 1 body, got %d", len(out.Bodies))
	}
	bs, ok := out.Bodies[0].Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("rebuilt face geometry is %T, want geom.BSplineSurface", out.Bodies[0].Faces()[0].Geometry())
	}
	if len(bs.Ctrl) != 4 || len(bs.Ctrl[0]) != 4 {
		t.Errorf("rebuilt net = %dx%d, want 4x4 single span", len(bs.Ctrl), len(bs.Ctrl[0]))
	}
	if r.Deviation() > 1e-6 {
		t.Errorf("rebuilding a multi-span bicubic to a single span should be near-exact, dev=%g", r.Deviation())
	}
}

func TestRebuildFeatureErrorsWithoutBody(t *testing.T) {
	r := &RebuildFeature{def: &RebuildDefinition{UDegree: 3, VDegree: 3, UCount: 4, VCount: 4}}
	if _, err := r.Recompute(Input{}); err == nil {
		t.Error("rebuild with no target body should error")
	}
}

func TestRebuildFeatureValidatesRecipe(t *testing.T) {
	r := &RebuildFeature{def: &RebuildDefinition{UDegree: 3, VDegree: 3, UCount: 3, VCount: 4}}
	if _, err := r.Recompute(Input{Bodies: []*topo.Body{multiSpanSurfaceBody(t)}}); err == nil {
		t.Error("uCount < uDegree+1 should error")
	}
}

func TestRebuildFeatureKind(t *testing.T) {
	r := &RebuildFeature{def: &RebuildDefinition{}}
	if r.Kind() != "rebuild-surface" {
		t.Errorf("Kind = %q, want rebuild-surface", r.Kind())
	}
}

func TestRestoreRebuildRejectsMissingPayload(t *testing.T) {
	if _, err := restoreRebuild(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreRebuild(nil) should error on a missing payload")
	}
}

func TestRebuildFeatureRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewRebuildFeatures(fs).Add(3, 2, 5, 4)

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 {
		t.Fatalf("feature count = %d, want 1", fresh.Count())
	}
	d := fresh.Item(0).Definition().(*RebuildFeature).Definition()
	if d.UDegree != 3 || d.VDegree != 2 || d.UCount != 5 || d.VCount != 4 {
		t.Errorf("restored recipe = %+v, want {3 2 5 4}", d)
	}
}
