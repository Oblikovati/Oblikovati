// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/sketch"
	"github.com/Oblikovati/oblikovati/scene"
)

// RayPicker is the real headless hit-test: it casts a camera ray through the clicked
// pixel and finds the nearest face of the scene bodies (the same query the GPU
// ID-buffer answers in production) and the nearest origin work plane. It implements
// [Picker], so a test "clicks on" a modeled solid or a datum plane — screen coordinate
// → ray → face/plane — with no GPU.
type RayPicker struct {
	camera   scene.Camera
	bodies   func() []*topo.Body
	planes   func() []*feature.WorkPlane
	points   func() []*feature.WorkPoint
	axes     func() []*feature.WorkAxis
	sketches func() []*sketch.Sketch
}

// pickPixelRadius is how close (in pixels) the cursor must be to a datum point or axis to
// snap it — small explicit targets are picked by screen proximity, not by depth.
const pickPixelRadius = 8.0

// NewRayPicker builds a picker over a camera and a provider of the current scene
// bodies (e.g. the active part's SurfaceBodies).
func NewRayPicker(camera scene.Camera, bodies func() []*topo.Body) *RayPicker {
	return &RayPicker{camera: camera, bodies: bodies}
}

// WithPlanes adds a provider of selectable work planes (the part's origin planes), so
// the picker can resolve a click on a datum plane in empty space.
func (p *RayPicker) WithPlanes(planes func() []*feature.WorkPlane) *RayPicker {
	p.planes = planes
	return p
}

// WithSketches adds a provider of the part's (visible) sketches, so the picker can
// resolve a click inside a sketch profile region — what an extrude/revolve consumes.
func (p *RayPicker) WithSketches(sketches func() []*sketch.Sketch) *RayPicker {
	p.sketches = sketches
	return p
}

// WithPoints / WithAxes add providers of the part's datum points and axes, so a click on
// one snaps to it (the reference inputs for point/axis-driven work planes).
func (p *RayPicker) WithPoints(points func() []*feature.WorkPoint) *RayPicker {
	p.points = points
	return p
}

func (p *RayPicker) WithAxes(axes func() []*feature.WorkAxis) *RayPicker {
	p.axes = axes
	return p
}

// SetCamera updates the view used for picking.
func (p *RayPicker) SetCamera(c scene.Camera) { p.camera = c }

// Pick returns the nearest selectable under the pixel honoring the filter: a face hit
// (or its owning body) when a solid is in front, a sketch profile region the ray lands
// in, or the nearest origin work plane whose finite display square the ray crosses —
// whichever the filter accepts and is closest. Ties favor faces, then planes, then
// profiles (the append order), so a solid in front wins over the sketch on its face.
func (p *RayPicker) Pick(x, y float64, filter *SelectionFilter) (Selectable, bool) {
	origin, dir := p.camera.RayThrough(x, y)
	// Precise datum snaps win first: a point, then an axis, the cursor lands on is the
	// target the user is aiming at, even with a face behind it (Inventor's snap order).
	if pt, _ := p.nearestPoint(origin, dir); pt != nil && filter.Accepts(SelectWorkPoint) {
		return WorkPointHandle{Point: pt}, true
	}
	if ax, _ := p.nearestAxis(origin, dir); ax != nil && filter.Accepts(SelectWorkAxis) {
		return WorkAxisHandle{Axis: ax}, true
	}
	var cands []pickCandidate
	if face, body, t := p.nearestFace(origin, dir); face != nil {
		if sel, ok := facePick(face, body, filter); ok {
			cands = append(cands, pickCandidate{t, sel})
		}
	}
	if plane, t := p.nearestPlane(origin, dir); plane != nil && filter.Accepts(SelectWorkPlane) {
		cands = append(cands, pickCandidate{t, WorkPlaneHandle{Plane: plane}})
	}
	if sel, t, ok := p.nearestProfile(origin, dir, filter); ok {
		cands = append(cands, pickCandidate{t, sel})
	}
	return nearestCandidate(cands)
}

// pickCandidate is one ray hit: its forward parameter and the selectable it resolves to.
type pickCandidate struct {
	t   float64
	sel Selectable
}

// nearestCandidate returns the candidate with the smallest ray parameter (the first one
// on ties, preserving the face→plane→profile precedence), or false when there are none.
func nearestCandidate(cands []pickCandidate) (Selectable, bool) {
	best, bestT := -1, stdmath.Inf(1)
	for i, c := range cands {
		if c.t < bestT {
			best, bestT = i, c.t
		}
	}
	if best < 0 {
		return nil, false
	}
	return cands[best].sel, true
}

// nearestProfile returns the closest sketch profile region the ray lands inside (mapped
// through each visible sketch's plane), and the ray parameter — when the filter accepts
// profiles. ok is false when no sketches are provided, profiles aren't accepted, or the
// ray misses every region.
func (p *RayPicker) nearestProfile(origin math.Point3, dir math.Vector3, filter *SelectionFilter) (Selectable, float64, bool) {
	if p.sketches == nil || !filter.Accepts(SelectProfile) {
		return nil, stdmath.Inf(1), false
	}
	var hit Selectable
	best := stdmath.Inf(1)
	for _, sk := range p.sketches() {
		t, uv, ok := rayPlanePoint(origin, dir, sk.Plane())
		if !ok || t >= best {
			continue
		}
		if idx, found := profileAt(sk, uv); found {
			best, hit = t, ProfileHandle{Sketch: sk, ProfileIndex: idx}
		}
	}
	return hit, best, hit != nil
}

// profileAt returns the index of the first profile in sk whose region contains the
// sketch-plane point uv, or false when uv is outside every profile.
func profileAt(sk *sketch.Sketch, uv math.Point2) (int, bool) {
	profiles := sk.Profiles()
	for i := 0; i < profiles.Count(); i++ {
		if profiles.Item(i).Contains(uv) {
			return i, true
		}
	}
	return 0, false
}

// rayPlanePoint intersects a ray with an (infinite) sketch plane, returning the forward
// ray parameter and the hit point in sketch (2D) coordinates.
func rayPlanePoint(origin math.Point3, dir math.Vector3, plane sketch.Plane) (float64, math.Point2, bool) {
	n := plane.Normal().AsVector()
	denom := dir.Dot(n)
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return 0, math.Point2{}, false
	}
	t := origin.VectorTo(plane.Origin()).Dot(n) / denom
	if t <= 0 {
		return 0, math.Point2{}, false
	}
	return t, plane.ToSketch(origin.TranslateBy(dir.Scale(t))), true
}

// nearestFace returns the closest ray-hit face, its body, and the ray parameter.
func (p *RayPicker) nearestFace(origin math.Point3, dir math.Vector3) (*topo.Face, *topo.Body, float64) {
	var hitFace *topo.Face
	var hitBody *topo.Body
	best := stdmath.Inf(1)
	for _, b := range p.bodies() {
		if f, t, ok := ops.RayCastFaces(b, origin, dir, ops.DefaultQuality()); ok && t < best {
			best, hitFace, hitBody = t, f, b
		}
	}
	return hitFace, hitBody, best
}

// nearestPlane returns the closest origin work plane whose display square the ray
// crosses, and the ray parameter (or nil/Inf if none).
func (p *RayPicker) nearestPlane(origin math.Point3, dir math.Vector3) (*feature.WorkPlane, float64) {
	if p.planes == nil {
		return nil, stdmath.Inf(1)
	}
	var hit *feature.WorkPlane
	best := stdmath.Inf(1)
	for _, wp := range p.planes() {
		if t, ok := rayWorkPlane(origin, dir, wp); ok && t < best {
			best, hit = t, wp
		}
	}
	return hit, best
}

// facePick wraps a face hit as the handle the filter wants (face or owning body),
// reporting ok=false when there is no face or the filter admits neither kind.
func facePick(face *topo.Face, body *topo.Body, filter *SelectionFilter) (Selectable, bool) {
	switch {
	case face == nil:
		return nil, false
	case filter.Accepts(SelectFace):
		return FaceHandle{Face: face, Body: body}, true
	case filter.Accepts(SelectBody):
		return BodyHandle{Body: body}, true
	default:
		return nil, false
	}
}

// nearestPoint returns the closest datum point within the pixel-snap radius of the ray,
// and the forward ray parameter (or nil/Inf if none).
func (p *RayPicker) nearestPoint(origin math.Point3, dir math.Vector3) (*feature.WorkPoint, float64) {
	if p.points == nil {
		return nil, stdmath.Inf(1)
	}
	tol := pickPixelRadius * p.camera.WorldPerPixel()
	var hit *feature.WorkPoint
	best := stdmath.Inf(1)
	for _, wp := range p.points() {
		t := origin.VectorTo(wp.Point()).Dot(dir)
		if t <= 0 || t >= best {
			continue
		}
		if origin.TranslateBy(dir.Scale(t)).DistanceTo(wp.Point()) <= tol {
			best, hit = t, wp
		}
	}
	return hit, best
}

// nearestAxis returns the closest datum axis within the pixel-snap radius of the ray, and
// the forward ray parameter at the closest approach (or nil/Inf if none).
func (p *RayPicker) nearestAxis(origin math.Point3, dir math.Vector3) (*feature.WorkAxis, float64) {
	if p.axes == nil {
		return nil, stdmath.Inf(1)
	}
	tol := pickPixelRadius * p.camera.WorldPerPixel()
	var hit *feature.WorkAxis
	best := stdmath.Inf(1)
	for _, ax := range p.axes() {
		if t, d, ok := rayAxisDistance(origin, dir, ax); ok && t < best && d <= tol {
			best, hit = t, ax
		}
	}
	return hit, best
}

// rayAxisDistance returns the forward ray parameter and world distance at the closest
// approach between the camera ray (origin, unit dir) and an infinite work axis. ok is
// false when the ray is parallel to the axis or the closest approach is behind the camera.
func rayAxisDistance(origin math.Point3, dir math.Vector3, ax *feature.WorkAxis) (float64, float64, bool) {
	u := ax.Direction().AsVector()
	w0 := ax.Origin().VectorTo(origin) // O − A
	b := dir.Dot(u)
	denom := 1 - b*b
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return 0, 0, false
	}
	d, e := dir.Dot(w0), u.Dot(w0)
	sc := (b*e - d) / denom // ray parameter at closest approach
	if sc <= 0 {
		return 0, 0, false
	}
	tc := (e - b*d) / denom // axis parameter at closest approach
	rayPoint := origin.TranslateBy(dir.Scale(sc))
	axisPoint := ax.Origin().TranslateBy(u.Scale(tc))
	return sc, rayPoint.DistanceTo(axisPoint), true
}

// rayWorkPlane intersects a ray with a work plane and reports the forward parameter
// when the hit lies within the plane's finite display square.
func rayWorkPlane(origin math.Point3, dir math.Vector3, wp *feature.WorkPlane) (float64, bool) {
	plane := wp.Plane()
	n := plane.Normal().AsVector()
	denom := dir.Dot(n)
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return 0, false
	}
	t := origin.VectorTo(plane.Origin()).Dot(n) / denom
	if t <= 0 {
		return 0, false
	}
	uv := plane.ToSketch(origin.TranslateBy(dir.Scale(t)))
	if stdmath.Abs(uv.X) > wp.DisplaySize() || stdmath.Abs(uv.Y) > wp.DisplaySize() {
		return 0, false
	}
	return t, true
}
