// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// userPlane rebuilds a sketch plane from a saved document's exact origin + axes (the .opd stores
// the same triple), so this test lofts the issue #1495 geometry verbatim rather than an idealization.
func userPlane(t *testing.T, ox, oy, oz, xx, xy, xz, yx, yy, yz float64) sketch.Plane {
	t.Helper()
	p, err := sketch.NewPlane(
		math.P3(ox, oy, oz),
		math.V3(xx, xy, xz).AsUnit(),
		math.V3(yx, yy, yz).AsUnit(),
	)
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	return p
}

// userCircleSketch adds a circle at the document's sketch-space centre + radius on the given plane.
func userCircleSketch(def *compdef.PartComponentDefinition, plane sketch.Plane, cx, cy, r float64) *sketch.Sketch {
	sk := def.Sketches().Add(plane)
	sk.Circles().AddByCenterRadius(math.P2(cx, cy), math.Scalar(r))
	return sk
}

// TestLoftUserCirclesIssue1495 verifies the reported scenario from issue #1495 ("two circles in
// different planes, fail to create a loft between them") on the user's verbatim demo.opd geometry:
//
//	Sketch2 — plane origin (0,0,5), xAxis (0,1,0), yAxis (-1,0,0), circle centre (1.3958,-1.8419) r=0.8760
//	Sketch4 — plane origin (0,0,0), xAxis (0,-1,0), yAxis (-1,0,0), circle centre (-2,-1)        r=0.5128
//
// Lofting the two circle profiles must yield one validated solid (an oblique circular frustum).
func TestLoftUserCirclesIssue1495(t *testing.T) {
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "demo.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)

	top := userCircleSketch(def, // Sketch2 @ z=5
		userPlane(t, 0, 0, 5, 0, 1, 0, -1, 0, 0), 1.3958273142764592, -1.8419164559524412, 0.8760172654982382)
	bottom := userCircleSketch(def, // Sketch4 @ z=0
		userPlane(t, 0, 0, 0, 0, -1, 0, -1, 0, 0), -2, -1, 0.512836453031296)

	l := NewLoftTool()
	s.SetPicker(&seqPicker{sels: []Selectable{
		ProfileHandle{Sketch: bottom, ProfileIndex: 0},
		ProfileHandle{Sketch: top, ProfileIndex: 0},
	}})
	s.StartTool(l)
	s.Click(10, 200) // bottom circle
	s.Click(10, 10)  // top circle
	if !l.CanCommit() {
		t.Fatalf("loft not ready with %d sections (issue #1495: two circles must be loftable)", l.SectionCount())
	}
	if err := s.OK(); err != nil {
		t.Fatalf("loft commit failed (issue #1495 regression): %v", err)
	}

	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies after loft, want 1 solid", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("lofted body is not a valid solid: %+v", r)
	}

	// Oblique circular frustum: parallel sections, so volume matches the straight frustum
	// V = (π h/3)(r0² + r0·r1 + r1²); tessellated circles undershoot the analytic value slightly.
	const r0, r1, h = 0.512836453031296, 0.8760172654982382, 5.0
	want := stdmath.Pi * h / 3 * (r0*r0 + r0*r1 + r1*r1)
	got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	if got <= 0 {
		t.Fatalf("lofted solid has non-positive volume %g", got)
	}
	if rel := stdmath.Abs(got-want) / want; rel > 0.05 {
		t.Errorf("loft volume = %g, want ≈%.3f (analytic frustum), relErr %.3f", got, want, rel)
	}
	t.Logf("issue #1495 loft OK: valid solid, volume=%.4f (analytic frustum ≈%.4f)", got, want)
}
