// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// TestSweepPathParameterEditResweeps pins the sweep path attribution (Oblikovati#1693): a
// parameter that drives the sweep's PATH sketch — not its profile sketch — must re-sweep the
// body. The path reaches the feature through an opaque live provider, so without the
// definition's PathSketch attribution the #1414 tail invalidation saw no dependent feature: the
// rail sketch re-solved to its new length while the swept body silently kept the old one (the
// resized NopSCADlib tubing measured its pre-edit volume).
func TestSweepPathParameterEditResweeps(t *testing.T) {
	t.Parallel()
	def := compdef.NewPartComponentDefinition()
	if _, err := def.Parameters().AddUserParameter("rail", "20 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}

	// Profile: a 4×4 mm square about the origin on XY.
	prof := def.Sketches().Add(sketch.XYPlane())
	c0 := prof.Points().Add(math.P2(-0.2, -0.2))
	c1 := prof.Points().Add(math.P2(0.2, -0.2))
	c2 := prof.Points().Add(math.P2(0.2, 0.2))
	c3 := prof.Points().Add(math.P2(-0.2, 0.2))
	prof.Lines().Add(c0, c1)
	prof.Lines().Add(c1, c2)
	prof.Lines().Add(c2, c3)
	prof.Lines().Add(c3, c0)

	// Path: a straight rail up +Z on XZ, its length driven by the user parameter.
	railSk := def.Sketches().Add(sketch.XZPlane())
	p0 := railSk.Points().Add(math.P2(0, 0))
	p1 := railSk.Points().Add(math.P2(0, 2))
	railSk.Lines().Add(p0, p1)
	if _, err := railSk.DimensionConstraints().AddDistance(p0, p1, "rail"); err != nil {
		t.Fatalf("AddDistance(rail): %v", err)
	}

	sweep := feature.NewSweepFeatures(def.Features()).AddDefinition(&feature.SweepDefinition{
		Sketch:       prof,
		ProfileIndex: 0,
		Path: func() *sketch.Path3D {
			paths := railSk.Paths()
			if len(paths) == 0 {
				return nil
			}
			pts := paths[0].Points()
			chain := make([]*sketch.Point3D, len(pts))
			for i, q := range pts {
				chain[i] = sketch.NewPoint3D(railSk.Plane().ToModel(q))
			}
			return sketch.NewPath3D(chain, paths[0].IsClosed())
		},
		PathSketch: railSk, // the attribution under test
		Operation:  ops.NewBody,
	})
	def.Recompute()
	if !sweep.Health().OK() {
		t.Fatalf("sweep sick: %+v", sweep.Health())
	}
	volumeAt := func() float64 {
		return ops.BodyGeometryProperties(def.Features().Result()[0], ops.DefaultQuality()).Volume
	}
	before := volumeAt()

	if err := def.Parameters().SetExpression(userParamID(t, def, "rail"), "30 mm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	def.RecomputeAfterChange()
	after := volumeAt()

	// The rail grew 20→30 mm, so the swept volume must grow by the same 1.5 factor.
	if want := before * 1.5; stdmath.Abs(after-want) > 1e-6*want {
		t.Errorf("swept volume after rail edit = %.6f, want %.6f (1.5× of %.6f) — the path edit did not re-sweep", after, want, before)
	}
}
