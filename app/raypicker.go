// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/scene"
)

// RayPicker is the real headless hit-test: it casts a camera ray through the clicked
// pixel and finds the nearest face of the scene bodies (the same query the GPU
// ID-buffer answers in production) and the nearest origin work plane. It implements
// [Picker], so a test "clicks on" a modeled solid or a datum plane — screen coordinate
// → ray → face/plane — with no GPU.
type RayPicker struct {
	camera scene.Camera
	bodies func() []*topo.Body
	planes func() []*feature.WorkPlane
}

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

// SetCamera updates the view used for picking.
func (p *RayPicker) SetCamera(c scene.Camera) { p.camera = c }

// Pick returns the nearest selectable under the pixel honoring the filter: a face hit
// (or its owning body) when a solid is in front, otherwise the nearest origin work
// plane whose finite display square the ray crosses.
func (p *RayPicker) Pick(x, y float64, filter *SelectionFilter) (Selectable, bool) {
	origin, dir := p.camera.RayThrough(x, y)
	face, body, faceT := p.nearestFace(origin, dir)
	plane, planeT := p.nearestPlane(origin, dir)
	faceSel, faceOK := facePick(face, body, filter)
	planeOK := plane != nil && filter.Accepts(SelectWorkPlane)
	switch {
	case faceOK && (!planeOK || faceT <= planeT):
		return faceSel, true
	case planeOK:
		return WorkPlaneHandle{Plane: plane}, true
	default:
		return nil, false
	}
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
		return FaceHandle{Face: face}, true
	case filter.Accepts(SelectBody):
		return BodyHandle{Body: body}, true
	default:
		return nil, false
	}
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
