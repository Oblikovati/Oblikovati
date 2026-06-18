// SPDX-License-Identifier: GPL-2.0-only

// Package scene is the viewport-side geometry the renderer and interaction need —
// pure Go, no GPU. It holds the camera and the screen↔world ray math that picking
// and navigation use; the GPU scene graph and draw lists build on it. Keeping this
// below the GPU line (ADR-0014) makes hit-testing and camera behavior unit-testable
// headlessly.
package scene

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Camera is a perspective viewport camera: an eye looking at a target, a vertical
// field of view, and the pixel viewport size. It produces world-space rays through
// screen pixels (origin at the top-left, y down — the convention window systems use).
type Camera struct {
	Eye    math.Point3
	Target math.Point3
	Up     math.Vector3
	FOV    float64 // vertical field of view, radians
	Width  int
	Height int
	// Orthographic selects a parallel projection (no perspective foreshortening) instead
	// of the FOV perspective. The ortho extent is sized from FOV at the target depth so
	// toggling modes keeps the model at the same on-screen scale. Set per view by the
	// ViewCube projection-mode menu.
	Orthographic bool
}

// NewCamera returns a camera with a 45° vertical FOV looking down −Z by default; set
// Eye/Target to frame the model.
func NewCamera(width, height int) Camera {
	return Camera{
		Eye:    math.P3(0, 0, 10),
		Target: math.P3(0, 0, 0),
		Up:     math.V3(0, 1, 0),
		FOV:    stdmath.Pi / 4,
		Width:  width,
		Height: height,
	}
}

// Forward returns the unit view direction (eye → target).
func (c Camera) Forward() math.Vector3 { return unit(c.Eye.VectorTo(c.Target)) }

// WorldPerPixel is the world-space size one screen pixel spans at the target depth —
// the scale picking/snapping use to turn a pixel tolerance into a world tolerance.
func (c Camera) WorldPerPixel() float64 {
	if c.Height <= 0 {
		return 0
	}
	dist := float64(c.Eye.DistanceTo(c.Target))
	return 2 * dist * stdmath.Tan(c.FOV/2) / float64(c.Height)
}

// RayThrough returns the world-space ray (origin, unit direction) through screen
// pixel (px, py). It is the inverse of the perspective projection used to render.
func (c Camera) RayThrough(px, py float64) (math.Point3, math.Vector3) {
	forward := unit(c.Eye.VectorTo(c.Target))
	right := unit(forward.Cross(c.Up))
	trueUp := right.Cross(forward) // already unit (forward,right orthonormal)

	aspect := float64(c.Width) / float64(c.Height)
	tanHalf := stdmath.Tan(c.FOV / 2)
	ndcX := (px/float64(c.Width)*2 - 1) * aspect * tanHalf
	ndcY := (1 - py/float64(c.Height)*2) * tanHalf

	dir := forward.Add(right.Scale(ndcX)).Add(trueUp.Scale(ndcY))
	return c.Eye, unit(dir)
}

// Navigation: orbit / pan / dolly produce a NEW camera (immutable, like the rest of
// the geometry value types). They are the math behind Inventor's Rotate / Pan / Zoom
// navigation, kept here below the GPU line so the head wires raw mouse input to them
// and they stay unit-testable.

const (
	// minDistance stops a zoom-in from collapsing the eye onto the target.
	minDistance = 1e-3
	// poleCos guards orbit pitch from flipping over the up pole (cos of ~5°): a pitch
	// that would bring the view within ~5° of the up axis is skipped.
	poleCos = 0.996
	// fitMargin leaves a little breathing room around the model when framing it.
	fitMargin = 1.1
)

// Fit frames the box in the viewport, keeping the current view direction — Inventor's
// Zoom All. The target moves to the box center and the eye backs off along the view
// direction far enough that the box's bounding sphere fits the narrower of the vertical/
// horizontal field of view (plus a small margin). An empty box (or zero height) is a no-op.
func (c Camera) Fit(box math.Box) Camera {
	if box.IsEmpty() || c.Height <= 0 {
		return c
	}
	forward := c.Forward()
	if forward == (math.Vector3{}) {
		forward = math.V3(0, 0, -1)
	}
	radius := stdmath.Max(float64(box.Diagonal().Length())/2, minDistance)
	aspect := float64(c.Width) / float64(c.Height)
	hfov := 2 * stdmath.Atan(stdmath.Tan(c.FOV/2)*aspect)
	dist := radius / stdmath.Sin(stdmath.Min(c.FOV, hfov)/2) * fitMargin
	c.Target = box.Center()
	c.Eye = c.Target.TranslateBy(forward.Scale(-dist))
	return c
}

// Home sets Inventor's default isometric "home" view — looking at the model from the
// front-top-right — and frames it. Up is reset to +Y. With nothing to frame it just
// reorients about the current target at the current distance.
func (c Camera) Home(box math.Box) Camera {
	c.Up = math.V3(0, 1, 0)
	iso := unit(math.V3(1, 1, 1)) // eye offset direction from the target
	if box.IsEmpty() {
		c.Eye = c.Target.TranslateBy(iso.Scale(float64(c.Eye.DistanceTo(c.Target))))
		return c
	}
	c.Target = box.Center()
	c.Eye = c.Target.TranslateBy(iso) // unit offset; Fit sets the real distance
	return c.Fit(box)
}

// Facing returns a camera looking straight at target along the given plane normal —
// Inventor's view when you open a sketch. The eye sits on the normal at the current
// eye–target distance (kept on the side the eye is currently on, so the view does not
// flip to the back of the plane), with up as the new up vector. A degenerate current
// distance falls back to a unit standoff.
func (c Camera) Facing(target math.Point3, normal, up math.Vector3) Camera {
	dist := c.Eye.DistanceTo(c.Target)
	if dist < minDistance {
		dist = 1
	}
	n := unit(normal)
	if target.VectorTo(c.Eye).Dot(n) < 0 {
		n = n.Scale(-1)
	}
	c.Target = target
	c.Eye = c.Target.TranslateBy(n.Scale(dist))
	c.Up = unit(up)
	return c
}

// Lerp blends two cameras (eye/target/up) by t∈[0,1], keeping the destination's FOV and
// viewport size — the per-frame step of a camera transition (e.g. entering a sketch).
func Lerp(a, b Camera, t float64) Camera {
	return Camera{
		Eye:    lerpPoint(a.Eye, b.Eye, t),
		Target: lerpPoint(a.Target, b.Target, t),
		Up:     a.Up.Add(b.Up.Sub(a.Up).Scale(t)),
		FOV:    b.FOV,
		Width:  b.Width,
		Height: b.Height,
	}
}

// lerpPoint linearly interpolates two points.
func lerpPoint(a, b math.Point3, t float64) math.Point3 {
	return a.TranslateBy(a.VectorTo(b).Scale(t))
}

// Dolly moves the eye toward (factor<1) or away from (factor>1) the target along the
// view direction, scaling the eye–target distance — Inventor's Zoom. The target is
// fixed; the distance is clamped to minDistance.
func (c Camera) Dolly(factor float64) Camera {
	if factor <= 0 {
		return c
	}
	offset := c.Target.VectorTo(c.Eye)
	dist := offset.Length() * factor
	if dist < minDistance {
		dist = minDistance
	}
	c.Eye = c.Target.TranslateBy(unit(offset).Scale(dist))
	return c
}

// DollyToCursor zooms by factor toward the point under screen pixel (px, py), keeping that point
// fixed on screen — Inventor's default zoom-to-cursor (N2), as opposed to Dolly's view-centred zoom.
// The pivot is where the cursor ray meets the target plane; Eye and Target are scaled about it, so
// the view direction and up are unchanged and the eye–target distance scales by factor exactly as
// Dolly, while whatever is under the cursor stays put. A cursor ray parallel to the plane (or factor
// ≤ 0) falls back to Dolly. The minDistance floor is honoured by clamping factor, so the pivot stays
// fixed right up to the closest zoom.
//
// Example: cam = cam.DollyToCursor(0.9, mouseX, mouseY) // one wheel notch in toward the cursor
func (c Camera) DollyToCursor(factor, px, py float64) Camera {
	if factor <= 0 || c.Height <= 0 {
		return c
	}
	pivot, ok := c.cursorPlanePoint(px, py)
	if !ok {
		return c.Dolly(factor)
	}
	if dist := float64(c.Eye.DistanceTo(c.Target)); dist > 0 && dist*factor < minDistance {
		factor = minDistance / dist
	}
	c.Eye = pivot.TranslateBy(pivot.VectorTo(c.Eye).Scale(factor))
	c.Target = pivot.TranslateBy(pivot.VectorTo(c.Target).Scale(factor))
	return c
}

// zoomRectMinPixels is the smallest drag (in viewport pixels) ZoomToRect acts on; a smaller box
// (a click or a twitch) is treated as no rectangle and ignored.
const zoomRectMinPixels = 3

// ZoomToRect reframes the view to the screen rectangle (x0,y0)-(x1,y1) in viewport pixels — Inventor's
// Zoom Window / Zoom Area (N16). The rectangle's centre becomes the new view centre and the box
// expands to fill the viewport: its larger relative dimension fits exactly, so the whole box stays
// visible. The view direction and up are unchanged. A degenerate (near-zero) rectangle, or a centre
// ray parallel to the target plane, is a no-op.
//
// Example: cam = cam.ZoomToRect(x0, y0, x1, y1) // fit the dragged window
func (c Camera) ZoomToRect(x0, y0, x1, y1 float64) Camera {
	if c.Width <= 0 || c.Height <= 0 {
		return c
	}
	w, h := stdmath.Abs(x1-x0), stdmath.Abs(y1-y0)
	if w < zoomRectMinPixels || h < zoomRectMinPixels {
		return c
	}
	pivot, ok := c.cursorPlanePoint((x0+x1)/2, (y0+y1)/2)
	if !ok {
		return c
	}
	fwd := c.Forward()
	dist := float64(c.Eye.DistanceTo(c.Target))
	c.Target = pivot
	c.Eye = pivot.TranslateBy(fwd.Scale(-dist))
	return c.Dolly(stdmath.Max(w/float64(c.Width), h/float64(c.Height)))
}

// cursorPlanePoint returns the world point where the ray through pixel (px, py) meets the plane
// through the target perpendicular to the view direction — the focal-depth point under the cursor.
// ok is false when the ray is parallel to that plane or points away from it.
func (c Camera) cursorPlanePoint(px, py float64) (math.Point3, bool) {
	origin, dir := c.RayThrough(px, py)
	forward := c.Forward()
	denom := float64(dir.Dot(forward))
	if stdmath.Abs(denom) < 1e-9 {
		return math.Point3{}, false
	}
	t := float64(origin.VectorTo(c.Target).Dot(forward)) / denom
	if t <= 0 {
		return math.Point3{}, false
	}
	return origin.TranslateBy(dir.Scale(t)), true
}

// Orbit rotates the eye around the fixed target — a turntable: yaw about the up axis,
// then pitch about the current right axis (Inventor's Rotate). The eye–target distance
// is preserved; a pitch that would flip over the up pole is skipped.
func (c Camera) Orbit(yaw, pitch float64) Camera {
	up := unit(c.Up)
	offset := rotateAboutAxis(c.Target.VectorTo(c.Eye), up, yaw)
	forward := unit(offset.Scale(-1)) // eye → target after the yaw
	right := unit(forward.Cross(up))
	if pitched := rotateAboutAxis(offset, right, pitch); absDot(unit(pitched), up) < poleCos {
		offset = pitched
	}
	c.Eye = c.Target.TranslateBy(offset)
	return c
}

// SetPivotUnderCursor recenters the orbit on the world point under screen pixel (px,py) — at the
// current focal depth (the cursor ray ∩ target plane) — keeping the view direction and eye–target
// distance. That point becomes the orbit centre (the target) and is brought to the view centre;
// nothing rotates or scales, so subsequent orbits pivot about it. This is Inventor's "set a new
// pivot" in Free Orbit (N9). A degenerate cursor ray is a no-op.
//
// Example: cam = cam.SetPivotUnderCursor(mouseX, mouseY) // click to re-centre the orbit
func (c Camera) SetPivotUnderCursor(px, py float64) Camera {
	p, ok := c.cursorPlanePoint(px, py)
	if !ok {
		return c
	}
	dist := float64(c.Eye.DistanceTo(c.Target))
	if dist < minDistance {
		dist = 1
	}
	fwd := c.Forward()
	c.Target = p
	c.Eye = p.TranslateBy(fwd.Scale(-dist))
	return c
}

// Roll twists the view about the forward (eye→target) axis — the spin a Free-Orbit ring perimeter
// drag produces (#913 N8). Eye and Target are unchanged; only Up rotates, so the model appears to
// turn in-plane about the view centre. A degenerate view direction is a no-op.
//
// Example: cam = cam.Roll(0.1) // twist the view ~5.7°
func (c Camera) Roll(angle float64) Camera {
	forward := c.Forward()
	if forward == (math.Vector3{}) {
		return c
	}
	c.Up = unit(rotateAboutAxis(c.Up, forward, angle))
	return c
}

// Pan slides the eye and target together in the view plane so the point under the
// cursor tracks the drag (Inventor's Pan). dxPixels/dyPixels are screen-space deltas
// (y down); the world step per pixel is derived from the target distance and FOV.
func (c Camera) Pan(dxPixels, dyPixels float64) Camera {
	if c.Height <= 0 {
		return c
	}
	forward := unit(c.Eye.VectorTo(c.Target))
	right := unit(forward.Cross(c.Up))
	trueUp := right.Cross(forward)
	dist := c.Eye.VectorTo(c.Target).Length()
	worldPerPixel := 2 * dist * stdmath.Tan(c.FOV/2) / float64(c.Height)
	move := right.Scale(-dxPixels * worldPerPixel).Add(trueUp.Scale(dyPixels * worldPerPixel))
	c.Eye = c.Eye.TranslateBy(move)
	c.Target = c.Target.TranslateBy(move)
	return c
}

// rotateAboutAxis rotates v about a unit axis by angle (Rodrigues' rotation formula).
func rotateAboutAxis(v, axis math.Vector3, angle float64) math.Vector3 {
	cos, sin := stdmath.Cos(angle), stdmath.Sin(angle)
	return v.Scale(cos).
		Add(axis.Cross(v).Scale(sin)).
		Add(axis.Scale(axis.Dot(v) * (1 - cos)))
}

// absDot is |a·b|, used to measure how close two directions are to (anti)parallel.
func absDot(a, b math.Vector3) float64 { return stdmath.Abs(a.Dot(b)) }

// unit normalizes a vector (zero stays zero).
func unit(v math.Vector3) math.Vector3 {
	if l := v.Length(); l > math.DefaultTolerance {
		return v.Scale(1 / l)
	}
	return math.V3(0, 0, 0)
}
