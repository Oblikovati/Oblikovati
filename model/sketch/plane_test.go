// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati/math"
)

const tol = 1e-9

func TestStandardPlaneMapping(t *testing.T) {
	// On the XY plane, sketch (u,v) maps to model (u,v,0).
	xy := XYPlane()
	got := xy.ToModel(math.P2(3, 4))
	if !got.IsEqualTo(math.P3(3, 4, 0), tol) {
		t.Errorf("XY ToModel(3,4) = %v, want (3,4,0)", got)
	}
	// On the XZ plane, sketch (u,v) maps to model (u,0,v).
	xz := XZPlane()
	if g := xz.ToModel(math.P2(3, 4)); !g.IsEqualTo(math.P3(3, 0, 4), tol) {
		t.Errorf("XZ ToModel(3,4) = %v, want (3,0,4)", g)
	}
	// On the YZ plane, sketch (u,v) maps to model (0,u,v).
	yz := YZPlane()
	if g := yz.ToModel(math.P2(3, 4)); !g.IsEqualTo(math.P3(0, 3, 4), tol) {
		t.Errorf("YZ ToModel(3,4) = %v, want (0,3,4)", g)
	}
}

func TestMappingRoundTrips(t *testing.T) {
	origin := math.P3(10, -5, 2)
	x, _ := math.NewUnitVector3(0, 1, 0)
	y, _ := math.NewUnitVector3(0, 0, 1)
	pl, err := NewPlane(origin, x, y)
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	for _, p := range []math.Point2{math.P2(0, 0), math.P2(2.5, 7), math.P2(-3, 1.25)} {
		back := pl.ToSketch(pl.ToModel(p))
		if !back.IsEqualTo(p, tol) {
			t.Errorf("round trip of %v = %v", p, back)
		}
	}
	// The normal is x cross y.
	if n := pl.Normal(); !n.AsVector().IsEqualTo(math.V3(1, 0, 0), tol) {
		t.Errorf("normal = %v, want (1,0,0)", n)
	}
	if !pl.Origin().IsEqualTo(origin, tol) {
		t.Errorf("Origin = %v, want %v", pl.Origin(), origin)
	}
	if !pl.XAxis().AsVector().IsEqualTo(math.V3(0, 1, 0), tol) || !pl.YAxis().AsVector().IsEqualTo(math.V3(0, 0, 1), tol) {
		t.Errorf("axes = %v,%v", pl.XAxis(), pl.YAxis())
	}
}

func TestToSketchProjectsOffPlanePoint(t *testing.T) {
	xy := XYPlane()
	// A point above the plane projects onto it, dropping z.
	if g := xy.ToSketch(math.P3(3, 4, 99)); !g.IsEqualTo(math.P2(3, 4), tol) {
		t.Errorf("ToSketch dropped-z = %v, want (3,4)", g)
	}
}

func TestNewPlaneRejectsNonPerpendicularAxes(t *testing.T) {
	x, _ := math.NewUnitVector3(1, 0, 0)
	skew, _ := math.NewUnitVector3(1, 1, 0)
	if _, err := NewPlane(math.P3(0, 0, 0), x, skew); err == nil {
		t.Error("NewPlane accepted non-perpendicular axes")
	}
}
