// SPDX-License-Identifier: GPL-2.0-only

package geom_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// torusDistance is how far p sits from the torus surface.
func torusDistance(t geom.Torus, p math.Point3) float64 {
	d := t.Center.VectorTo(p)
	axial := float64(d.Dot(t.AxisDir.AsVector()))
	radial := float64(d.Sub(t.AxisDir.AsVector().Scale(math.Scalar(axial))).Length())
	return stdmath.Abs(stdmath.Hypot(radial-t.MajorRadius, axial) - t.MinorRadius)
}

// TestTorusPlaneSectionLiesOnBoth pins the spiric section (ADR-0061 stage 3): every point of every curve
// it returns lies on BOTH the torus and the cutting plane, whatever shape the section takes.
func TestTorusPlaneSectionLiesOnBoth(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	for _, tc := range []struct {
		name string
		o    math.Point3
		d    math.Vector3
		want int
	}{
		{"axis-parallel through the hole", math.P3(1, 0, 0), math.V3(1, 0, 0), 2},
		{"axis-parallel through one wall", math.P3(6, 0, 0), math.V3(1, 0, 0), 2},
		{"oblique", math.P3(0, 0, 1), math.V3(0.3, 0, 1), 2},
	} {
		pl, err := geom.NewPlane(tc.o, tc.d)
		if err != nil {
			t.Fatalf("%s: plane: %v", tc.name, err)
		}
		curves, ok := geom.TorusPlaneSection(tor, pl)
		if !ok {
			t.Errorf("%s: the section declined", tc.name)
			continue
		}
		if len(curves) != tc.want {
			t.Errorf("%s: %d curve(s), want %d", tc.name, len(curves), tc.want)
		}
		n := pl.Normal()
		for ci, cv := range curves {
			for k := 0; k <= 40; k++ {
				p := cv.PointAt(float64(k) / 40)
				if d := stdmath.Abs(float64(pl.Origin.VectorTo(p).Dot(n))); d > 1e-9 {
					t.Errorf("%s curve%d: a point is %g off the PLANE", tc.name, ci, d)
					break
				}
				if d := torusDistance(tor, p); d > 1e-9 {
					t.Errorf("%s curve%d: a point is %g off the TORUS", tc.name, ci, d)
					break
				}
			}
		}
	}
}

// TestTorusPlaneSectionClosesItsOvals: where the plane reaches only part of the tube, the two branches
// meet at the ends of that part, so the pair closes into one oval.
func TestTorusPlaneSectionClosesItsOvals(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	pl, err := geom.NewPlane(math.P3(6, 0, 0), math.V3(1, 0, 0)) // cuts the outer wall only
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	curves, ok := geom.TorusPlaneSection(tor, pl)
	if !ok || len(curves) != 2 {
		t.Fatalf("section ok=%v n=%d, want two branches", ok, len(curves))
	}
	// branch +1 runs v0→v1, branch −1 runs v1→v0, so each one's end meets the other's start.
	for _, pair := range [][2]int{{0, 1}, {1, 0}} {
		a, b := curves[pair[0]].PointAt(1), curves[pair[1]].PointAt(0)
		if d := float64(a.DistanceTo(b)); d > 1e-9 {
			t.Errorf("branch %d ends %g from where branch %d starts; the oval must close", pair[0], d, pair[1])
		}
	}
}

// TestTorusPlaneSectionDeclinesAPurelyAxialNormal: a plane perpendicular to the axis has no radial
// component in its normal, so the azimuth is not single-valued in the tube angle and the branch form
// does not apply — that plane is the two-circle case the conic path takes instead.
func TestTorusPlaneSectionDeclinesAPurelyAxialNormal(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	pl, err := geom.NewPlane(math.P3(0, 0, 0.5), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	if _, ok := geom.TorusPlaneSection(tor, pl); ok {
		t.Error("a purely axial cut normal leaves u multi-valued in v; the spiric form must decline it")
	}
}
