// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Loft RAILS matrix (Slice 3, kLoftWithRails): guide curves pull the loft's outer surface to
// follow them between the sections. A rail touches the end sections and bulges/necks in between;
// its pull is localized (the opposite side of the section is unaffected) and the end sections stay.

// railThroughX returns a rail (provider) on the +X side of a radius-2 loft (z 0→4) that passes
// through x=midX at the mid height — a single guide curve.
func railThroughX(midX float64) func() []math.Point3 {
	return func() []math.Point3 {
		return []math.Point3{math.P3(2, 0, 0), math.P3(math.Scalar(midX), 0, 2), math.P3(2, 0, 4)}
	}
}

// railedCircles lofts two equal circles (r=2, z 0 and 4) under the given rails, asserting a valid
// solid and returning it.
func railedCircles(t *testing.T, rails []func() []math.Point3) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil)
	secs := []LoftSection{sec(circleOn(sketch.XYPlane(), 2)), sec(circleOn(planeAtZ(4), 2))}
	pf := NewLoftFeatures(fs).AddGuided(secs, false, ops.NewBody, LoftEnd{}, LoftEnd{}, LoftGuideSet{Rails: rails})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("railed loft went sick: %+v", pf.Health())
	}
	b := fs.Result()[0]
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("railed loft not a valid solid: valid=%v solid=%v issues=%v", r.Valid, b.IsSolid(), capIssues3(r.Issues))
	}
	return b
}

// TestLoftRailBulgesToRail: a rail bulging to x=3.5 pulls the loft's +X side out to follow it —
// the body's max x reaches the rail (a ruled cylinder would stay at 2).
func TestLoftRailBulgesToRail(t *testing.T) {
	t.Parallel()
	b := railedCircles(t, []func() []math.Point3{railThroughX(3.5)})
	if maxX := float64(b.RangeBox().Max.X); maxX < 3.4 || maxX > 3.6 {
		t.Errorf("loft did not follow the bulging rail: max x = %.3f, want ≈3.5", maxX)
	}
}

// TestLoftRailIsLocalized: a +X rail bulge leaves the −X side of the loft untouched (the falloff
// is localized around the rail's track).
func TestLoftRailIsLocalized(t *testing.T) {
	t.Parallel()
	b := railedCircles(t, []func() []math.Point3{railThroughX(3.5)})
	if minX := float64(b.RangeBox().Min.X); minX < -2.05 || minX > -1.95 {
		t.Errorf("rail bulge disturbed the opposite side: min x = %.3f, want ≈-2.0", minX)
	}
}

// TestLoftRailNecksReducesVolume: a rail necking to x=1.0 pulls that side in, so the railed loft
// holds less than the un-railed cylinder.
func TestLoftRailNecksReducesVolume(t *testing.T) {
	t.Parallel()
	free := ops.BodyGeometryProperties(railedCircles(t, nil), ops.DefaultQuality()).Volume
	neck := ops.BodyGeometryProperties(railedCircles(t, []func() []math.Point3{railThroughX(1.0)}), ops.DefaultQuality()).Volume
	if neck >= free {
		t.Errorf("necking rail did not reduce volume: necked %.3f, free %.3f", neck, free)
	}
}

// TestLoftRailKeepsEnds: a rail does not move the end sections — the body still spans exactly the
// section heights (z 0→4) and the end caps keep radius 2 (max x at the ends is unchanged).
func TestLoftRailKeepsEnds(t *testing.T) {
	t.Parallel()
	b := railedCircles(t, []func() []math.Point3{railThroughX(3.5)})
	bb := b.RangeBox()
	if z0, z1 := float64(bb.Min.Z), float64(bb.Max.Z); z0 < -1e-6 || z0 > 1e-6 || z1 < 4-1e-6 || z1 > 4+1e-6 {
		t.Errorf("rail moved the loft ends in z: span [%.4f,%.4f], want [0,4]", z0, z1)
	}
}

// TestLoftRailRoundTrip: a railed loft's guide polyline survives a recipe save/restore (so a
// reopened .obk rebuilds the bulged loft, not the ruled one).
func TestLoftRailRoundTrip(t *testing.T) {
	t.Parallel()
	bottom := circleOn(sketch.XYPlane(), 2)
	top := circleOn(planeAtZ(4), 2)
	idx := sketchList{sks: []*sketch.Sketch{bottom, top}}
	fs := NewPartFeatures(nil)
	NewLoftFeatures(fs).AddGuided(
		[]LoftSection{{Sketch: bottom, ProfileIndex: 0}, {Sketch: top, ProfileIndex: 0}},
		false, ops.NewBody, LoftEnd{}, LoftEnd{},
		LoftGuideSet{Rails: []func() []math.Point3{railThroughX(3.5)}},
	)
	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	got := fresh.Item(0).Definition().(*LoftFeature).Definition()
	if len(got.Rails) != 1 {
		t.Fatalf("rail round-trip: got %d rails, want 1", len(got.Rails))
	}
	pts := got.Rails[0]()
	if len(pts) != 3 || float64(pts[1].X) < 3.4 || float64(pts[1].X) > 3.6 {
		t.Errorf("rail polyline round-trip lost the bulge: %v", pts)
	}
	if got.LoftType() != "rails" {
		t.Errorf("LoftType = %q, want \"rails\"", got.LoftType())
	}
}

// TestLoftRailTwoOpposite: two rails on opposite sides (+X out, +Y out) both bulge their side.
func TestLoftRailTwoOpposite(t *testing.T) {
	t.Parallel()
	railY := func() []math.Point3 {
		return []math.Point3{math.P3(0, 2, 0), math.P3(0, 3.5, 2), math.P3(0, 2, 4)}
	}
	b := railedCircles(t, []func() []math.Point3{railThroughX(3.5), railY})
	bb := b.RangeBox()
	if float64(bb.Max.X) < 3.4 || float64(bb.Max.Y) < 3.4 {
		t.Errorf("two rails did not both bulge: max x=%.3f y=%.3f, want both ≈3.5", float64(bb.Max.X), float64(bb.Max.Y))
	}
}
