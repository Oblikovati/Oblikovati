// SPDX-License-Identifier: GPL-2.0-only

package scene

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// centeredCamera looks from +Z down −Z at the origin with a 90° FOV and a square viewport, so the
// half-angle is 45° and the on-axis frustum math is easy to reason about (half-width ≈ dist).
func centeredCamera() Camera {
	c := NewCamera(400, 400)
	c.Eye = math.P3(0, 0, 10)
	c.Target = math.P3(0, 0, 0)
	c.Up = math.V3(0, 1, 0)
	c.FOV = stdmath.Pi / 2 // 90° vertical
	return c
}

func boxAt(cx, cy, cz, half float64) math.Box {
	c := math.P3(math.Scalar(cx), math.Scalar(cy), math.Scalar(cz))
	d := math.V3(math.Scalar(half), math.Scalar(half), math.Scalar(half))
	return math.NewBox(c.TranslateBy(d.Negate()), c.TranslateBy(d))
}

func TestFrustumKeepsOnAxisBoxAndDropsOffScreen(t *testing.T) {
	f := centeredCamera().Frustum()
	cases := []struct {
		name string
		box  math.Box
		want bool
	}{
		{"in front, on axis", boxAt(0, 0, 0, 1), true},
		{"behind the eye", boxAt(0, 0, 20, 1), false},
		{"far to the right (off-screen)", boxAt(100, 0, 0, 1), false},
		{"far above (off-screen)", boxAt(0, 100, 0, 1), false},
		{"just inside the right edge", boxAt(8, 0, 0, 1), true}, // at z=0 half-width ≈ 10 (dist 10, 45°)
		{"empty box", math.EmptyBox(), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := f.IntersectsBox(c.box); got != c.want {
				t.Errorf("IntersectsBox = %v, want %v", got, c.want)
			}
		})
	}
}

// TestFrustumNeverDropsAVisiblePoint is the safety property: any point that projects inside the
// viewport must be kept (no false negatives → no popping). It samples a grid on the target plane,
// keeps those that project on-screen, and checks each is kept by the frustum.
func TestFrustumNeverDropsAVisiblePoint(t *testing.T) {
	cam := centeredCamera()
	f := cam.Frustum()
	for ix := -9; ix <= 9; ix++ {
		for iy := -9; iy <= 9; iy++ {
			p := math.P3(math.Scalar(ix), math.Scalar(iy), 0)
			px, py := project(cam, p)
			if px < 0 || px > float64(cam.Width) || py < 0 || py > float64(cam.Height) {
				continue // off-screen: the frustum may drop it
			}
			if !f.IntersectsBox(math.NewBox(p, p)) {
				t.Errorf("point %v projects on-screen at (%.0f,%.0f) but was culled", p, px, py)
			}
		}
	}
}

func TestFrustumOrthographicCulls(t *testing.T) {
	cam := centeredCamera()
	cam.Orthographic = true
	f := cam.Frustum()
	if !f.IntersectsBox(boxAt(0, 0, 0, 1)) {
		t.Error("ortho frustum should keep an on-axis box")
	}
	if f.IntersectsBox(boxAt(100, 0, 0, 1)) {
		t.Error("ortho frustum should cull a box well outside the lateral extent")
	}
	if f.IntersectsBox(boxAt(0, 0, 20, 1)) {
		t.Error("ortho frustum should cull a box behind the eye")
	}
}

// project maps a world point to a viewport pixel via the same NDC math RayThrough inverts, so the
// "is it on screen" oracle in the safety test is independent of the frustum under test.
func project(c Camera, p math.Point3) (px, py float64) {
	fwd := unit(c.Eye.VectorTo(c.Target))
	right := unit(fwd.Cross(c.Up))
	up := right.Cross(fwd)
	v := c.Eye.VectorTo(p)
	z := float64(v.Dot(fwd))
	tanHalf := stdmath.Tan(c.FOV / 2)
	ndcX := float64(v.Dot(right)) / z / (tanHalf * c.aspect())
	ndcY := float64(v.Dot(up)) / z / tanHalf
	px = (ndcX + 1) / 2 * float64(c.Width)
	py = (1 - ndcY) / 2 * float64(c.Height)
	return px, py
}
