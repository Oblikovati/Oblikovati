// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// line3 builds a degree-1 B-spline through 3 points with uniform knots, so PointAt(0,0.5,1) hits the
// three points exactly — a clean grid curve for network tests.
func line3(t *testing.T, p0, p1, p2 math.Point3) BSplineCurve {
	t.Helper()
	c, err := NewBSplineCurve(1, []math.Point3{p0, p1, p2}, []float64{1, 1, 1}, []float64{0, 0, 0.5, 1, 1})
	if err != nil {
		t.Fatalf("line3: %v", err)
	}
	return c
}

// onSurface reports the distance from p to its nearest point on s (via the on-surface inverse).
func onSurface(s Surface, p math.Point3) float64 {
	u, v := s.ParamAt(p)
	return float64(p.DistanceTo(s.PointAt(u, v)))
}

// TestNetworkSurfacePassesThroughFlatCurves: a 3×3 grid of lines in z=0 yields the flat plane — every
// curve sample lies on the network.
func TestNetworkSurfacePassesThroughFlatCurves(t *testing.T) {
	t.Parallel()
	pt := func(x, y float64) math.Point3 { return math.P3(math.Scalar(x), math.Scalar(y), 0) }
	uCurves := []BSplineCurve{ // along x at y=0,1,2
		line3(t, pt(0, 0), pt(1, 0), pt(2, 0)),
		line3(t, pt(0, 1), pt(1, 1), pt(2, 1)),
		line3(t, pt(0, 2), pt(1, 2), pt(2, 2)),
	}
	vCurves := []BSplineCurve{ // along y at x=0,1,2
		line3(t, pt(0, 0), pt(0, 1), pt(0, 2)),
		line3(t, pt(1, 0), pt(1, 1), pt(1, 2)),
		line3(t, pt(2, 0), pt(2, 1), pt(2, 2)),
	}
	net, err := NetworkSurface(uCurves, vCurves)
	if err != nil {
		t.Fatalf("NetworkSurface: %v", err)
	}
	for _, c := range append(append([]BSplineCurve{}, uCurves...), vCurves...) {
		for i := 0; i <= 8; i++ {
			if d := onSurface(net, c.PointAt(float64(i)/8)); d > 1e-6 {
				t.Fatalf("curve sample off the network: dist %g", d)
			}
		}
	}
}

// TestNetworkSurfaceInterpolatesNodes: a curved (saddle) grid yields a surface passing through every
// grid node exactly — the interpolation guarantee.
func TestNetworkSurfaceInterpolatesNodes(t *testing.T) {
	t.Parallel()
	z := func(a, b int) float64 { return 0.3 * float64((a-1)*(b-1)) } // saddle
	pt := func(a, b int) math.Point3 {
		return math.P3(math.Scalar(a), math.Scalar(b), math.Scalar(z(a, b)))
	}
	uCurves := make([]BSplineCurve, 3) // along a (u) at v-station b
	vCurves := make([]BSplineCurve, 3) // along b (v) at u-station a
	for b := range 3 {
		uCurves[b] = line3(t, pt(0, b), pt(1, b), pt(2, b))
	}
	for a := range 3 {
		vCurves[a] = line3(t, pt(a, 0), pt(a, 1), pt(a, 2))
	}
	net, err := NetworkSurface(uCurves, vCurves)
	if err != nil {
		t.Fatalf("NetworkSurface: %v", err)
	}
	for a := range 3 {
		for b := range 3 {
			if d := onSurface(net, pt(a, b)); d > 1e-6 {
				t.Errorf("grid node (%d,%d) off the network: dist %g", a, b, d)
			}
		}
	}
}

// TestNetworkSurfaceRejectsNonIntersecting: v-curves lifted off the u-curves' plane do not form a
// grid — a clear error (not a silent bad surface).
func TestNetworkSurfaceRejectsNonIntersecting(t *testing.T) {
	t.Parallel()
	flat := func(x, y, zoff float64) math.Point3 {
		return math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(zoff))
	}
	uCurves := []BSplineCurve{
		line3(t, flat(0, 0, 0), flat(1, 0, 0), flat(2, 0, 0)),
		line3(t, flat(0, 2, 0), flat(1, 2, 0), flat(2, 2, 0)),
	}
	vCurves := []BSplineCurve{ // lifted +5 in z: never meet the u-curves
		line3(t, flat(0, 0, 5), flat(0, 1, 5), flat(0, 2, 5)),
		line3(t, flat(2, 0, 5), flat(2, 1, 5), flat(2, 2, 5)),
	}
	if _, err := NetworkSurface(uCurves, vCurves); err == nil {
		t.Error("non-intersecting curves should error")
	}
}

func TestNetworkSurfaceNeedsTwoEachWay(t *testing.T) {
	t.Parallel()
	c := line3(t, math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0))
	if _, err := NetworkSurface([]BSplineCurve{c}, []BSplineCurve{c, c}); err == nil {
		t.Error("a single u-curve should error")
	}
}
