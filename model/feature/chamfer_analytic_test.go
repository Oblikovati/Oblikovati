// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// extrudedCylinderTopRim builds an analytic cylinder (radius r, height h) by extruding a circle and
// returns the engine plus the reference key of its TOP circular rim edge.
func extrudedCylinderTopRim(t *testing.T, r, h float64) (*PartFeatures, []byte) {
	t.Helper()
	fs := NewPartFeatures(nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Circles().AddByCenterRadius(math.P2(0, 0), r)
	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return h })
	fs.Recompute()
	body := fs.Result()[0]
	for _, e := range body.Edges() {
		if c, ok := e.Geometry().(geom.Circle); ok && stdmath.Abs(float64(c.Center.Z)-h) < 1e-6 {
			return fs, e.ReferenceKey()
		}
	}
	t.Fatal("no top rim circle edge on the analytic cylinder")
	return nil, nil
}

func bodyConeCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cone); ok {
			n++
		}
	}
	return n
}

// TestAnalyticChamferOfCylinderRimIsACone is the #127 fix: chamfering the rim of an extruded circle
// (an analytic cylinder) yields a TRUE conical chamfer — one geom.Cone
// face on a valid watertight solid — not the non-manifold/faceted result the rim used to give.
func TestAnalyticChamferOfCylinderRimIsACone(t *testing.T) {
	t.Parallel()
	const r, h, d = 5.0, 10.0, 2.0
	fs, rim := extrudedCylinderTopRim(t, r, h)
	pf := NewDressUpFeatures(fs).AddChamfer([][]byte{rim}, func() float64 { return d })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("analytic chamfer sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("chamfered cylinder is not a valid solid: %+v", r.Issues)
	}
	if n := bodyConeCount(body); n != 1 {
		t.Fatalf("chamfered rim has %d cone faces, want 1 (a true conical chamfer, #127)", n)
	}
	full := stdmath.Pi * r * r * h
	removed := stdmath.Pi*r*r*d - stdmath.Pi*d*(3*r*r-3*r*d+d*d)/3
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, full-removed) > 0.03 {
		t.Errorf("chamfered cylinder volume = %g, want ≈%g", got, full-removed)
	}
}

// TestAnalyticChamferLeavesBoxChamferAlone pins that the analytic-cylinder fast path does NOT
// disturb an ordinary straight-edge chamfer: a box edge isn't a cylinder rim,
// so it falls through to the general wedge-cut and stays a valid solid with no spurious cone face.
func TestAnalyticChamferLeavesBoxChamferAlone(t *testing.T) {
	t.Parallel()
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var edge []byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			edge = e.ReferenceKey()
			break
		}
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	pf := NewDressUpFeatures(fs).AddChamfer([][]byte{edge}, func() float64 { return 0.5 })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("box chamfer sick with the gate on: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("box chamfer not a valid solid: %+v", r.Issues)
	}
	if n := bodyConeCount(body); n != 0 {
		t.Errorf("box chamfer produced %d cone faces, want 0 (straight bevel)", n)
	}
}
