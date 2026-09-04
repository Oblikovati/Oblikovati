// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// polarVLimit asks the GEOMETRY whether a surface's v limit is a pole — a point the whole azimuth
// collapses to — rather than asking what kind of surface it is. Only a pole gives a cap's contour a
// line to close against. A torus's v limit is a PERIOD: it closes onto itself, so a contour built there
// would measure a region that does not exist (ADR-0062).
func TestPolarVLimitIsAskedOfTheGeometry(t *testing.T) {
	t.Parallel()
	sphere, err := geom.NewSphere(math.P3(1, 2, 3), 4)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	v, ok := polarVLimit(sphere)
	if !ok {
		t.Fatal("a sphere's v limit is its pole; polarVLimit refused it")
	}
	if _, vHi := sphere.VDomain(); v != vHi {
		t.Errorf("polarVLimit = %g, want the high v limit %g", v, vHi)
	}
	// And the point it names really is one point.
	uLo, uHi := sphere.UDomain()
	if d := float64(sphere.PointAt(uLo, v).DistanceTo(sphere.PointAt((uLo+uHi)/2, v))); d > 1e-9 {
		t.Errorf("the named limit spreads %g: it is not a pole", d)
	}

	torus, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	if _, ok := polarVLimit(torus); ok {
		t.Error("a torus's v limit is a period, not a pole; polarVLimit accepted it")
	}
}

// A cylinder's v runs to infinity, so it has no limit to close against either.
func TestPolarVLimitRefusesAnUnboundedAxis(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	if _, vHi := cyl.VDomain(); !stdmath.IsInf(vHi, 0) {
		t.Skipf("this cylinder's v is bounded (%g); the test needs an unbounded one", vHi)
	}
	if _, ok := polarVLimit(cyl); ok {
		t.Error("an unbounded v has no limit to close a contour against")
	}
}
