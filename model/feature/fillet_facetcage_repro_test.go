// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// faceMix reports a body's face-geometry composition — the signature that tells a clean analytic
// blend (few planes + cylinders/tori) from a facet-cage (a plane explosion).
func faceMix(b *topo.Body) (planes, cyls, tori, other int) {
	for _, f := range b.Faces() {
		switch f.Geometry().(type) {
		case geom.Plane:
			planes++
		case geom.Cylinder:
			cyls++
		case geom.Torus:
			tori++
		default:
			other++
		}
	}
	return
}

// TestFilletIntoExistingRoundIsHonest_Rampam1 is the regression for Discord item #1 (#1797) as
// rampam's screenshots show it: fillet the top rim of a cube (rounds it — the body now carries
// cylinder faces), THEN fillet the four vertical edges. Each vertical edge is planar-flanked but its
// TOP vertex runs into the top-rim cylinders — a fillet-meets-fillet corner the planar rolling-ball
// blend cannot close. This must be REJECTED with an honest, actionable reason that names the rounded
// cause — NOT the misleading "result is not a valid solid" (which shipped a facet-cage octagon on the
// nightly rampam tested), and it must leave the good top-rim fillet intact (no mutation).
func TestFilletIntoExistingRoundIsHonest_Rampam1(t *testing.T) {
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}},
		sketch.XYPlane(), span{near: 0, far: 4}, 0, "box")
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)

	top := topPerimeterKeysF(box)
	if len(top) != 4 {
		t.Fatalf("plain cube top edges = %d, want 4", len(top))
	}
	f1 := NewDressUpFeatures(fs).AddFillet(top, func() float64 { return 0.5 })
	fs.Recompute()
	if !f1.Health().OK() {
		t.Fatalf("top-rim fillet sick: %+v", f1.Health())
	}
	goodP, goodC, goodTr, _ := faceMix(fs.Result()[0]) // the intact rounded-top body

	vert := verticalEdgeKeysF(fs.Result()[0])
	if len(vert) != 4 {
		t.Fatalf("vertical edges on rounded body = %d, want 4", len(vert))
	}
	f2 := NewDressUpFeatures(fs).AddFillet(vert, func() float64 { return 0.5 })
	fs.Recompute()

	// It must not silently "succeed" into a bad solid.
	if f2.Health().OK() {
		t.Fatalf("filleting an edge that runs into an existing round must be rejected, but it reported healthy")
	}
	reason := f2.Health().Reason
	if strings.Contains(reason, "not a valid solid") {
		t.Errorf("misleading reason — should name the rounded/curved cause, got: %q", reason)
	}
	if !strings.Contains(strings.ToLower(reason), "round") && !strings.Contains(strings.ToLower(reason), "curved") {
		t.Errorf("reason should name the rounded/curved cause (so the user knows to fillet these first), got: %q", reason)
	}

	// The prior good fillet is preserved and valid — no facet-cage, no mutation.
	res := fs.Result()[0]
	if rep := ops.Validate(res); !rep.Valid || !res.IsSolid() {
		t.Fatalf("rejected fillet mutated the body into an invalid solid: %+v", rep)
	}
	if p, c, tr, _ := faceMix(res); p != goodP || c != goodC || tr != goodTr {
		t.Errorf("rejected fillet changed the body: faces planes/cyl/tori = %d/%d/%d, want %d/%d/%d (unchanged)",
			p, c, tr, goodP, goodC, goodTr)
	}
}
