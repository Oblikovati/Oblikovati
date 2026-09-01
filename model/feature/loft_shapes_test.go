// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Loft SHAPE matrix (regular loft, Free condition — Slice 1). Each future setting slice
// (tangency conditions, rails, centerline, area, point-map) adds its own rows × these shapes.
// Lofts are finicky, so every shape is built and the result asserted a valid manifold solid.

// circleOn returns a sketch with one circle (radius r) centered on the plane.
func circleOn(plane sketch.Plane, r float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(plane)
	s.Circles().AddByCenterRadius(math.P2(0, 0), math.Scalar(r))
	return s
}

// rotatedSquareOn returns a centered square (half-size h) rotated by ang radians.
func rotatedSquareOn(plane sketch.Plane, h, ang float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(plane)
	c := stdmath.Cos(ang)
	sn := stdmath.Sin(ang)
	rot := func(x, y float64) *sketch.Point {
		return s.Points().Add(math.P2(math.Scalar(x*c-y*sn), math.Scalar(x*sn+y*c)))
	}
	p0, p1, p2, p3 := rot(-h, -h), rot(h, -h), rot(h, h), rot(-h, h)
	s.Lines().Add(p0, p1)
	s.Lines().Add(p1, p2)
	s.Lines().Add(p2, p3)
	s.Lines().Add(p3, p0)
	return s
}

// splineBlobOn returns a sketch with one closed spline through pts — an organic section.
func splineBlobOn(plane sketch.Plane, pts [][2]float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(plane)
	pp := make([]math.Point2, len(pts))
	for i, p := range pts {
		pp[i] = math.P2(math.Scalar(p[0]), math.Scalar(p[1]))
	}
	s.Splines().AddByPoints(pp, true)
	return s
}

// loftSolid builds a loft of the sections and asserts it is a single valid solid, returning it.
func loftSolid(t *testing.T, sections []LoftSection, closed bool) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil)
	pf := NewLoftFeatures(fs).Add(sections, closed, ops.NewBody)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("loft went sick: %+v", pf.Health())
	}
	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("loft produced %d bodies, want 1", len(bodies))
	}
	if r := ops.Validate(bodies[0]); !r.Valid || !bodies[0].IsSolid() {
		t.Fatalf("lofted body is not a valid solid: valid=%v solid=%v issues=%v", r.Valid, bodies[0].IsSolid(), capIssues3(r.Issues))
	}
	return bodies[0]
}

func capIssues3(s []string) []string {
	if len(s) > 3 {
		return s[:3]
	}
	return s
}

func sec(s *sketch.Sketch) LoftSection { return LoftSection{Sketch: s, ProfileIndex: 0} }

func TestLoftShapeCylindrical(t *testing.T) {
	t.Parallel()
	// circle→circle = a cone frustum. V = πh/3·(R²+Rr+r²).
	body := loftSolid(t, []LoftSection{sec(circleOn(sketch.XYPlane(), 2)), sec(circleOn(planeAtZ(4), 1))}, false)
	want := stdmath.Pi * 4 / 3 * (4 + 2 + 1)
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, want) > 0.03 {
		t.Errorf("cone-frustum volume = %g, want ≈%g", v, want)
	}
}

func TestLoftShapeMismatchedCount(t *testing.T) {
	t.Parallel()
	// square (4 pts) → circle (faceted) — different vertex counts; correspondence must cope.
	loftSolid(t, []LoftSection{sec(centeredSquareOn(sketch.XYPlane(), 2)), sec(circleOn(planeAtZ(4), 1.5))}, false)
}

func TestLoftShapeTwistedSquares(t *testing.T) {
	t.Parallel()
	// square → square rotated 45°: the correspondence must untwist (no self-intersection),
	// giving a valid solid. Volume ≈ a prism of the square area × height (twist preserves it).
	body := loftSolid(t, []LoftSection{sec(centeredSquareOn(sketch.XYPlane(), 2)), sec(rotatedSquareOn(planeAtZ(4), 2, stdmath.Pi/4))}, false)
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v <= 0 {
		t.Fatalf("twisted loft volume %g <= 0", v)
	}
}

func TestLoftShapeMultiSectionBulges(t *testing.T) {
	t.Parallel()
	// 3 circles whose MIDDLE is shifted +X: the spline blend must curve through it, so the
	// body's +X extent exceeds the straight-loft bound (the end circles reach only x=+2).
	// shift the middle section sketch's plane origin +X by 3 (a banana).
	midPlane, _ := sketch.NewPlane(math.P3(3, 0, 2), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	mid := circleOn(midPlane, 2)
	body := loftSolid(t, []LoftSection{sec(circleOn(sketch.XYPlane(), 2)), sec(mid), sec(circleOn(planeAtZ(4), 2))}, false)
	maxX := body.RangeBox().Max.X
	if float64(maxX) < 3.5 {
		t.Errorf("multi-section loft did not bulge to the offset middle: max x = %.3f, want > 3.5 (straight would be 2)", float64(maxX))
	}
}

// annulusOn returns a sketch with concentric outer/inner circles and the index of the annular
// (holed) profile — a hollow section.
func annulusOn(plane sketch.Plane, outerR, innerR float64) (*sketch.Sketch, int) {
	s := sketch.NewSketches().Add(plane)
	s.Circles().AddByCenterRadius(math.P2(0, 0), math.Scalar(outerR))
	s.Circles().AddByCenterRadius(math.P2(0, 0), math.Scalar(innerR))
	for i := 0; i < s.Profiles().Count(); i++ {
		if len(s.Profiles().Item(i).InnerLoops()) > 0 {
			return s, i
		}
	}
	return s, 0
}

func TestLoftShapePipe(t *testing.T) {
	t.Parallel()
	// annulus → annulus = a tapered hollow pipe (tube). The outer loops skin the body, the
	// inner loops skin a bore that is cut out. Volume = outer cone frustum − inner cone frustum.
	sb, ib := annulusOn(sketch.XYPlane(), 2.0, 1.5)
	st, it := annulusOn(planeAtZ(4), 1.4, 1.0)
	body := loftSolid(t, []LoftSection{{Sketch: sb, ProfileIndex: ib}, {Sketch: st, ProfileIndex: it}}, false)
	cone := func(rr, r, h float64) float64 { return stdmath.Pi * h / 3 * (rr*rr + rr*r + r*r) }
	want := cone(2.0, 1.4, 4) - cone(1.5, 1.0, 4)
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, want) > 0.05 {
		t.Errorf("lofted pipe volume = %g, want ≈%g (hollow tube)", v, want)
	}
}

func TestLoftShapeOrganicSpline(t *testing.T) {
	t.Parallel()
	// Organic closed-spline sections (a rounded blob → a different blob) lofted.
	blob := [][2]float64{{2, 0}, {1.2, 1.6}, {-1, 1.8}, {-2, 0}, {-1, -1.6}, {1.2, -1.6}}
	blob2 := [][2]float64{{1.4, 0}, {0.8, 1.0}, {-0.7, 1.2}, {-1.4, 0}, {-0.7, -1.0}, {0.8, -1.0}}
	loftSolid(t, []LoftSection{sec(splineBlobOn(sketch.XYPlane(), blob)), sec(splineBlobOn(planeAtZ(4), blob2))}, false)
}
