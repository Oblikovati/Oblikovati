// SPDX-License-Identifier: GPL-2.0-only

package scene

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
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

// pointRayDistance is the perpendicular distance from p to the line (origin, unit dir).
func pointRayDistance(p, origin math.Point3, dir math.Vector3) float64 {
	v := origin.VectorTo(p)
	along := v.Dot(dir)
	closest := origin.TranslateBy(dir.Scale(float64(along)))
	return float64(closest.DistanceTo(p))
}

// TestDollyToCursorKeepsPivotUnderCursor: zooming toward an off-centre pixel keeps the world point
// under it on that pixel's ray (zoom-to-cursor, N2) and scales the eye–target distance like Dolly.
func TestDollyToCursorKeepsPivotUnderCursor(t *testing.T) {
	c := NewCamera(800, 600)
	px, py := 650.0, 180.0 // off-centre
	pivot, ok := c.cursorPlanePoint(px, py)
	if !ok {
		t.Fatal("cursorPlanePoint failed for an on-screen pixel")
	}
	z := c.DollyToCursor(0.5, px, py)
	if d := float64(z.Eye.DistanceTo(z.Target)); stdmath.Abs(d-5) > 1e-9 {
		t.Errorf("zoom-to-cursor distance = %v, want 5 (factor 0.5 of 10)", d)
	}
	if !z.Forward().IsEqualTo(c.Forward(), 1e-9) {
		t.Error("zoom-to-cursor must keep the view direction fixed")
	}
	o, dir := z.RayThrough(px, py)
	if dd := pointRayDistance(pivot, o, dir); dd > 1e-6 {
		t.Errorf("pivot is %g off the cursor ray after zoom (should stay under the cursor)", dd)
	}
}

// TestDollyToCursorCenterEqualsDolly: at the centre pixel the pivot is the target, so zoom-to-cursor
// reduces exactly to the view-centred Dolly.
func TestDollyToCursorCenterEqualsDolly(t *testing.T) {
	c := NewCamera(800, 600)
	z := c.DollyToCursor(0.5, 400, 300)
	d := c.Dolly(0.5)
	if !z.Eye.IsEqualTo(d.Eye, 1e-9) || !z.Target.IsEqualTo(d.Target, 1e-9) {
		t.Errorf("centre zoom-to-cursor = (eye %v, target %v), want Dolly (eye %v, target %v)", z.Eye, z.Target, d.Eye, d.Target)
	}
}

// TestDollyToCursorClampsMinDistance: a hard zoom-in clamps to minDistance like Dolly.
func TestDollyToCursorClampsMinDistance(t *testing.T) {
	c := NewCamera(800, 600)
	z := c.DollyToCursor(1e-12, 650, 180)
	if d := float64(z.Eye.DistanceTo(z.Target)); stdmath.Abs(d-minDistance) > 1e-9 {
		t.Errorf("clamped zoom-to-cursor distance = %v, want %v", d, minDistance)
	}
}

// TestZoomToRectCentersAndFills: zooming to a centred half-size box keeps the view direction, recentres
// on the box centre, and halves the eye–target distance (the box's relative size sets the zoom).
func TestZoomToRectCentersAndFills(t *testing.T) {
	c := NewCamera(800, 600)
	before, _ := c.cursorPlanePoint(400, 300) // box centre = viewport centre → the current target depth
	z := c.ZoomToRect(200, 150, 600, 450)     // a 400×300 box (half W, half H) centred
	if !z.Forward().IsEqualTo(c.Forward(), 1e-9) {
		t.Error("ZoomToRect must keep the view direction")
	}
	if !z.Target.IsEqualTo(before, 1e-6) {
		t.Errorf("recentre target = %v, want the box-centre world point %v", z.Target, before)
	}
	if d := float64(z.Eye.DistanceTo(z.Target)); stdmath.Abs(d-5) > 1e-6 {
		t.Errorf("zoom distance = %v, want 5 (half of 10: the box is half the viewport)", d)
	}
}

// TestZoomToRectOffCentreRecenters: an off-centre box moves the target to that box's centre.
func TestZoomToRectOffCentreRecenters(t *testing.T) {
	c := NewCamera(800, 600)
	pivot, _ := c.cursorPlanePoint(650, 180)
	z := c.ZoomToRect(600, 130, 700, 230) // centre (650,180)
	if !z.Target.IsEqualTo(pivot, 1e-6) {
		t.Errorf("off-centre zoom target = %v, want %v", z.Target, pivot)
	}
}

// TestZoomToRectIgnoresDegenerate: a near-zero box (a click) is a no-op.
func TestZoomToRectIgnoresDegenerate(t *testing.T) {
	c := NewCamera(800, 600)
	if z := c.ZoomToRect(400, 300, 401, 301); z != c {
		t.Error("a degenerate rectangle should leave the camera unchanged")
	}
}

// TestSetPivotUnderCursorRecenters: clicking off-centre to set the orbit pivot brings that point to
// the view centre (the new target) while keeping the view direction and eye–target distance.
func TestSetPivotUnderCursorRecenters(t *testing.T) {
	c := NewCamera(800, 600)
	dist0 := float64(c.Eye.DistanceTo(c.Target))
	pivot, _ := c.cursorPlanePoint(650, 180)
	p := c.SetPivotUnderCursor(650, 180)
	if !p.Target.IsEqualTo(pivot, 1e-6) {
		t.Errorf("new target = %v, want the clicked world point %v", p.Target, pivot)
	}
	if !p.Forward().IsEqualTo(c.Forward(), 1e-9) {
		t.Error("set-pivot must keep the view direction")
	}
	if d := float64(p.Eye.DistanceTo(p.Target)); stdmath.Abs(d-dist0) > 1e-6 {
		t.Errorf("set-pivot distance = %v, want %v (unchanged)", d, dist0)
	}
	// The new pivot is the target, so it projects to the view centre.
	o, dir := p.RayThrough(400, 300)
	if dd := pointRayDistance(p.Target, o, dir); dd > 1e-6 {
		t.Errorf("pivot is %g off the centre ray (should be centred)", dd)
	}
}

// TestRollRotatesUpAboutView: roll spins the up vector about the forward axis without moving the
// eye or target; a quarter turn takes +Y up to ±X (in the screen plane).
func TestRollRotatesUpAboutView(t *testing.T) {
	c := NewCamera(800, 600) // eye (0,0,10) → target origin, forward −Z, up +Y
	r := c.Roll(stdmath.Pi / 2)
	if !r.Eye.IsEqualTo(c.Eye, 1e-9) || !r.Target.IsEqualTo(c.Target, 1e-9) {
		t.Error("roll must not move the eye or target")
	}
	if stdmath.Abs(float64(r.Up.Y)) > 1e-9 || stdmath.Abs(stdmath.Abs(float64(r.Up.X))-1) > 1e-9 {
		t.Errorf("rolled up = %v, want ±X (a quarter turn of +Y about −Z)", r.Up)
	}
	if !r.Up.IsEqualTo(unit(r.Up), 1e-9) {
		t.Error("rolled up should stay unit length")
	}
}

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

// TestOrbitConstrainedYawsAboutWorldUpAndLevels: constrained orbit yaws about the world vertical and
// re-levels the view (removing roll), unlike Orbit which yaws about the camera's own up.
func TestOrbitConstrainedYawsAboutWorldUpAndLevels(t *testing.T) {
	c := NewCamera(800, 600).Roll(0.6) // a rolled view: camera up is no longer world +Y
	worldUp := math.V3(0, 1, 0)

	q := c.OrbitConstrained(stdmath.Pi/2, 0, worldUp) // quarter turn about world +Y: (0,0,10)→(10,0,0)
	if !q.Eye.IsEqualTo(math.P3(10, 0, 0), 1e-9) {
		t.Errorf("eye after constrained 90° yaw = %v, want (10,0,0)", q.Eye)
	}
	if !q.Up.IsEqualTo(worldUp, 1e-9) {
		t.Errorf("constrained orbit up = %v, want world +Y (re-levelled, no roll)", q.Up)
	}
	if d := float64(q.Eye.DistanceTo(q.Target)); stdmath.Abs(d-10) > 1e-9 {
		t.Errorf("constrained orbit changed distance: %v, want 10", d)
	}
	// A near-pole pitch is skipped; a degenerate worldUp falls back to Orbit.
	if !c.OrbitConstrained(0, stdmath.Pi/2, worldUp).Eye.IsEqualTo(c.Eye, 1e-9) {
		t.Error("near-pole constrained pitch should be clamped")
	}
	if c.OrbitConstrained(0.3, 0, math.V3(0, 0, 0)) != c.Orbit(0.3, 0) {
		t.Error("degenerate worldUp should fall back to Orbit")
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
