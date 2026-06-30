// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// lExtrude builds an L-shaped prism — a 2×2 square with its +X+Y unit square removed — extruded 1
// unit in Z, and returns it with the reference key of the single CONCAVE (reflex) vertical edge at
// (1,1). Its volume is 3 (area 3 × height 1); the concave edge's notch opens toward +X,+Y, so an
// outward chamfer fills that quadrant and an inward chamfer relieves the solid behind it.
func lExtrude(t *testing.T) (*topo.Body, []byte) {
	t.Helper()
	poly := []math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 1}, {X: 1, Y: 1}, {X: 1, Y: 2}, {X: 0, Y: 2}}
	body := buildPrism(poly, sketch.XYPlane(), span{near: 0, far: 1}, 0, "L")
	for _, e := range body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		at11 := func(p math.Point3) bool { return stdmath.Abs(p.X-1) < 1e-9 && stdmath.Abs(p.Y-1) < 1e-9 }
		if at11(a) && at11(b) && ops.ClassifyEdgeConvexity(e) == ops.EdgeConcave {
			return body, e.ReferenceKey()
		}
	}
	t.Fatal("no concave vertical edge at (1,1)")
	return nil, nil
}

// chamferedConcave runs an equal-distance chamfer on the L's concave edge with the given strategy
// and returns the resulting body, failing on a sick feature or an invalid solid.
func chamferedConcave(t *testing.T, d float64, strategy types.ChamferConcaveStrategy) *topo.Body {
	t.Helper()
	body, edge := lExtrude(t)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(body)
	ch := NewDressUpFeatures(fs).AddChamferConcave([][]byte{edge}, func() float64 { return d }, true, strategy)
	fs.Recompute()
	if !ch.Health().OK() {
		t.Fatalf("concave chamfer (%v) sick: %+v", strategy, ch.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("concave chamfer (%v) not a valid solid: %+v", strategy, r)
	}
	return res
}

// TestChamferConcaveOutwardFills is the regression for the inverted-chamfer bug: an internal
// (concave) edge chamfered outward must FILL the inside corner with a 45° gusset, adding ½·d²·L of
// material — not cut a malformed sliver as the convex-only path did.
func TestChamferConcaveOutwardFills(t *testing.T) {
	const d = 0.4
	res := chamferedConcave(t, d, types.ChamferConcaveOutward)
	want := 3 + 0.5*d*d // L volume 3 + triangular fill ½·d·d·length(1)
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, want) > 1e-6 {
		t.Errorf("outward concave chamfer volume = %g, want %g (notch should be filled)", got, want)
	}
}

// TestChamferConcaveInwardRelieves checks the inward strategy cuts a recessed relief groove out of
// the corner instead, removing ½·d²·L of material.
func TestChamferConcaveInwardRelieves(t *testing.T) {
	const d = 0.4
	res := chamferedConcave(t, d, types.ChamferConcaveInward)
	want := 3 - 0.5*d*d // L volume 3 − triangular relief ½·d·d·length(1)
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, want) > 1e-6 {
		t.Errorf("inward concave chamfer volume = %g, want %g (corner should be relieved)", got, want)
	}
}

// TestChamferConcaveOutwardIsDefault confirms the zero-value strategy (AddChamferCorners) fills
// outward — so an existing/default chamfer of a concave edge adds material, the chosen default.
func TestChamferConcaveOutwardIsDefault(t *testing.T) {
	const d = 0.4
	body, edge := lExtrude(t)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(body)
	ch := NewDressUpFeatures(fs).AddChamferCorners([][]byte{edge}, func() float64 { return d }, true)
	fs.Recompute()
	if !ch.Health().OK() {
		t.Fatalf("default concave chamfer sick: %+v", ch.Health())
	}
	res := fs.Result()[0]
	want := 3 + 0.5*d*d
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, want) > 1e-6 {
		t.Errorf("default concave chamfer volume = %g, want %g (default must fill outward)", got, want)
	}
}
