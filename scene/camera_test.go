// SPDX-License-Identifier: GPL-2.0-only

package scene

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

func TestCenterRayPointsAtTarget(t *testing.T) {
	c := NewCamera(800, 600) // eye (0,0,10) → target origin, looking −Z
	origin, dir := c.RayThrough(400, 300)
	if !origin.IsEqualTo(math.P3(0, 0, 10), 1e-9) {
		t.Errorf("ray origin = %v, want the eye (0,0,10)", origin)
	}
	// The center pixel's ray points straight at the target: −Z.
	if !dir.IsEqualTo(math.V3(0, 0, -1), 1e-9) {
		t.Errorf("center ray dir = %v, want (0,0,-1)", dir)
	}
}

func TestOffCenterRaysSpread(t *testing.T) {
	c := NewCamera(800, 600)
	_, right := c.RayThrough(800, 300) // far-right pixel
	_, left := c.RayThrough(0, 300)    // far-left pixel
	// Right pixel ray leans +X, left leans −X (mirror), both still forward (−Z).
	if right.X <= 0 || left.X >= 0 {
		t.Errorf("horizontal spread wrong: rightX=%v leftX=%v", right.X, left.X)
	}
	if stdmath.Abs(right.X+left.X) > 1e-9 {
		t.Error("left/right rays should be mirror images in X")
	}
	_, top := c.RayThrough(400, 0) // top pixel → leans +Y (y-down screen)
	if top.Y <= 0 {
		t.Errorf("top pixel ray Y = %v, want > 0", top.Y)
	}
	if !dirIsUnit(right) || !dirIsUnit(top) {
		t.Error("ray directions should be unit length")
	}
}

func dirIsUnit(v math.Vector3) bool { return stdmath.Abs(v.Length()-1) < 1e-9 }

func TestDollyScalesDistanceKeepingDirection(t *testing.T) {
	c := NewCamera(800, 600) // eye (0,0,10), target origin
	in := c.Dolly(0.5)
	if d := float64(in.Eye.DistanceTo(in.Target)); stdmath.Abs(d-5) > 1e-9 {
		t.Errorf("zoom-in distance = %v, want 5", d)
	}
	if !in.Target.IsEqualTo(c.Target, 1e-9) || !in.Forward().IsEqualTo(c.Forward(), 1e-9) {
		t.Error("dolly must keep the target and view direction fixed")
	}
	out := c.Dolly(2)
	if d := float64(out.Eye.DistanceTo(out.Target)); stdmath.Abs(d-20) > 1e-9 {
		t.Errorf("zoom-out distance = %v, want 20", d)
	}
	// Zooming in hard clamps to minDistance rather than collapsing onto the target.
	if d := float64(c.Dolly(1e-12).Eye.DistanceTo(c.Target)); stdmath.Abs(d-minDistance) > 1e-12 {
		t.Errorf("clamped distance = %v, want %v", d, minDistance)
	}
}

func TestOrbitPreservesDistanceAndYawsAroundUp(t *testing.T) {
	c := NewCamera(800, 600) // eye (0,0,10), up +Y
	dist := float64(c.Eye.DistanceTo(c.Target))

	q := c.Orbit(stdmath.Pi/2, 0) // quarter turn about +Y: (0,0,10) → (10,0,0)
	if !q.Eye.IsEqualTo(math.P3(10, 0, 0), 1e-9) {
		t.Errorf("eye after 90° yaw = %v, want (10,0,0)", q.Eye)
	}
	if d := float64(q.Eye.DistanceTo(q.Target)); stdmath.Abs(d-dist) > 1e-9 {
		t.Errorf("orbit changed distance: %v, want %v", d, dist)
	}
	// A full turn returns to the start.
	if !c.Orbit(2*stdmath.Pi, 0).Eye.IsEqualTo(c.Eye, 1e-9) {
		t.Error("full 360° yaw should return to the original eye")
	}
	// A pitch that would flip over the up pole is skipped (eye unchanged).
	if !c.Orbit(0, stdmath.Pi/2).Eye.IsEqualTo(c.Eye, 1e-9) {
		t.Error("near-pole pitch should be clamped (no change)")
	}
}

func TestFitFramesBoxKeepingDirection(t *testing.T) {
	c := NewCamera(800, 600) // looks −Z
	box := math.NewBox(math.P3(2, 2, 2), math.P3(6, 4, 8))

	f := c.Fit(box)
	if !f.Target.IsEqualTo(box.Center(), 1e-9) {
		t.Errorf("target = %v, want box center %v", f.Target, box.Center())
	}
	if !f.Forward().IsEqualTo(c.Forward(), 1e-9) {
		t.Error("Fit must keep the view direction")
	}
	for _, corner := range box.Corners() {
		if !insideFrustum(f, corner) {
			t.Errorf("corner %v not inside the framed view", corner)
		}
	}
	if c.Fit(math.EmptyBox()) != c {
		t.Error("Fit of an empty box should be a no-op")
	}
}

func TestHomeSetsIsometricAndFrames(t *testing.T) {
	c := NewCamera(800, 600)
	box := math.NewBox(math.P3(0, 0, 0), math.P3(4, 3, 5))

	h := c.Home(box)
	if !h.Up.IsEqualTo(math.V3(0, 1, 0), 1e-9) {
		t.Error("home should reset Up to +Y")
	}
	if !h.Target.IsEqualTo(box.Center(), 1e-9) {
		t.Error("home should target the box center")
	}
	if !h.Forward().IsEqualTo(unit(math.V3(-1, -1, -1)), 1e-9) {
		t.Errorf("home forward = %v, want the iso diagonal (−1,−1,−1)", h.Forward())
	}
	for _, corner := range box.Corners() {
		if !insideFrustum(h, corner) {
			t.Errorf("corner %v not framed by the home view", corner)
		}
	}
}

// insideFrustum reports whether p lies in front of c and within both half-FOVs — the
// check that Fit/Home actually frame every model corner.
func insideFrustum(c Camera, p math.Point3) bool {
	fwd := c.Forward()
	right := unit(fwd.Cross(c.Up))
	up := right.Cross(fwd)
	v := c.Eye.VectorTo(p)
	along := float64(v.Dot(fwd))
	if along <= 0 {
		return false
	}
	aspect := float64(c.Width) / float64(c.Height)
	hfov := 2 * stdmath.Atan(stdmath.Tan(c.FOV/2)*aspect)
	vAng := stdmath.Atan2(stdmath.Abs(float64(v.Dot(up))), along)
	hAng := stdmath.Atan2(stdmath.Abs(float64(v.Dot(right))), along)
	return vAng <= c.FOV/2 && hAng <= hfov/2
}

func TestPanShiftsEyeAndTargetTogether(t *testing.T) {
	c := NewCamera(800, 600)
	p := c.Pan(100, 0) // drag right
	if !p.Forward().IsEqualTo(c.Forward(), 1e-9) {
		t.Error("pan must not change the view direction")
	}
	if d := float64(p.Eye.DistanceTo(p.Target)); stdmath.Abs(d-float64(c.Eye.DistanceTo(c.Target))) > 1e-9 {
		t.Error("pan must keep the eye–target distance")
	}
	// Dragging right (+x screen) slides the camera −X (content follows the cursor).
	if p.Target.X >= 0 {
		t.Errorf("drag-right target X = %v, want < 0", p.Target.X)
	}
	// Eye and target move by the same vector (rigid slide).
	if !c.Eye.VectorTo(p.Eye).IsEqualTo(c.Target.VectorTo(p.Target), 1e-9) {
		t.Error("eye and target must move by the same vector")
	}
}

func TestForwardAndDefaults(t *testing.T) {
	c := NewCamera(640, 480)
	if c.Width != 640 || c.Height != 480 || c.FOV <= 0 {
		t.Error("NewCamera defaults wrong")
	}
	// Default camera at (0,0,10) → (0,0,0) looks along −Z.
	if !c.Forward().IsEqualTo(math.V3(0, 0, -1), 1e-9) {
		t.Errorf("Forward = %v, want (0,0,-1)", c.Forward())
	}
	// A degenerate (eye==target) direction yields the zero vector, not NaN.
	d := NewCamera(10, 10)
	d.Target = d.Eye
	if d.Forward() != math.V3(0, 0, 0) {
		t.Errorf("degenerate Forward = %v, want zero", d.Forward())
	}
}

func TestFacingLooksAlongPlaneNormalAtPreservedDistance(t *testing.T) {
	c := NewCamera(100, 100) // eye (0,0,10), target origin, up +Y, distance 10
	// Face a plane whose normal is +Y and up is +Z (an XZ-like plane through origin).
	f := c.Facing(math.P3(0, 0, 0), math.V3(0, 1, 0), math.V3(0, 0, 1))
	if d := float64(f.Eye.DistanceTo(f.Target)); stdmath.Abs(d-10) > 1e-9 {
		t.Errorf("facing distance = %v, want 10 (preserved)", d)
	}
	if dot := stdmath.Abs(float64(f.Forward().Dot(math.V3(0, 1, 0)))); dot < 1-1e-9 {
		t.Errorf("view not parallel to the plane normal: |fwd·n| = %v", dot)
	}
	if !f.Up.IsEqualTo(math.V3(0, 0, 1), 1e-9) {
		t.Errorf("facing up = %v, want +Z", f.Up)
	}
}

func TestFacingKeepsEyeOnViewerSide(t *testing.T) {
	c := NewCamera(100, 100) // eye on +Z
	// Even with a normal pointing away (−Z), the eye stays on the viewer's (+Z) side.
	f := c.Facing(math.P3(0, 0, 0), math.V3(0, 0, -1), math.V3(0, 1, 0))
	if f.Eye.Z <= 0 {
		t.Errorf("facing flipped the eye to the back of the plane: eye = %v", f.Eye)
	}
}

func TestLerpBlendsEndpoints(t *testing.T) {
	a := NewCamera(100, 100)
	b := a
	b.Eye = math.P3(10, 0, 0)
	mid := Lerp(a, b, 0.5)
	if stdmath.Abs(float64(mid.Eye.X)-5) > 1e-9 {
		t.Errorf("lerp midpoint eye.X = %v, want 5", mid.Eye.X)
	}
	if !Lerp(a, b, 1).Eye.IsEqualTo(b.Eye, 1e-9) || !Lerp(a, b, 0).Eye.IsEqualTo(a.Eye, 1e-9) {
		t.Error("lerp endpoints should equal the inputs")
	}
}

func TestWorldPerPixelScalesWithDistance(t *testing.T) {
	c := NewCamera(100, 200) // height 200
	near := c.WorldPerPixel()
	c.Eye = math.P3(0, 0, 20) // twice the distance → twice the world-per-pixel
	if far := c.WorldPerPixel(); stdmath.Abs(far-2*near) > 1e-9 {
		t.Errorf("world-per-pixel did not scale with distance: near %v far %v", near, far)
	}
	c.Height = 0
	if c.WorldPerPixel() != 0 {
		t.Error("zero height should give zero world-per-pixel")
	}
}
