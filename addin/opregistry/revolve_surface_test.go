// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"math"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	omath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// revolveReadyPart builds a part with a rectangle clear of the axis (radius 2..4, height 0..3 on
// XY), so it can be revolved about the origin Y axis without collapsing onto a pole.
func revolveReadyPart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	d, err := s.Workspace().Add(doc.Part, "t.obk", true)
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	def := compdef.NewPartComponentDefinition()
	d.SetContent(def)
	sk := def.Sketches().Add(sketch.XYPlane())
	at := func(x, y float64) *sketch.Point { return sk.Points().Add(omath.P2(x, y)) }
	corners := []*sketch.Point{at(2, 0), at(4, 0), at(4, 3), at(2, 3)}
	for i, c := range corners {
		sk.Lines().Add(c, corners[(i+1)%len(corners)])
	}
	def.Recompute()
	return s
}

// TestRevolveSurfaceOperation drives the revolve tool with operation:"surface" (Inventor's
// kSurfaceOperation, #1858): a rectangle (radius 2..4, height 3) revolved 180° about the Y axis
// builds one healthy OPEN sheet body — the surface of revolution, no start/end profile caps, not
// booleaned. Its area matches the analytic surface-of-revolution area θ·h·(r1+r2)+θ(r2²−r1²)
// (verified against the OCCT oracle), within the tessellation tolerance.
func TestRevolveSurfaceOperation(t *testing.T) {
	t.Parallel()
	s := revolveReadyPart(t)
	raw, err := applyMap(t, s, "revolve", map[string]any{
		"sketchIndex": 0, "angle": "180 deg", "axisRef": "origin/axis/y", "operation": "surface",
	})
	if err != nil {
		t.Fatalf("surface revolve: %v", err)
	}
	var res struct {
		Bodies  int  `json:"bodies"`
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !res.Healthy || res.Bodies != 1 {
		t.Fatalf("surface revolve result = %+v, want one healthy body", res)
	}
	b := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).SurfaceBodies().Item(0)
	if b.IsSolid() {
		t.Error("surface-operation revolve produced a SOLID body, want an open sheet")
	}

	const r1, r2, h, theta = 2.0, 4.0, 3.0, math.Pi // 180°
	want := theta*h*(r1+r2) + theta*(r2*r2-r1*r1)   // 30π ≈ 94.25 (OCCT-confirmed formula)
	got := query.BodyGeometryProperties(b, ops.DefaultQuality()).Area
	if got < 0.90*want || got > 1.02*want {
		t.Errorf("surface-of-revolution area = %.4f, want ≈ %.4f (analytic; faceted within tolerance)", got, want)
	}
}

// TestRevolveSurfaceUnknownOperationErrors: an unknown operation is still a clean error after
// adding "surface" to the revolve enum.
func TestRevolveSurfaceUnknownOperationErrors(t *testing.T) {
	t.Parallel()
	s := revolveReadyPart(t)
	if _, err := applyMap(t, s, "revolve", map[string]any{"sketchIndex": 0, "angle": "90 deg", "operation": "bogus"}); err == nil {
		t.Error("unknown operation should error")
	}
}
