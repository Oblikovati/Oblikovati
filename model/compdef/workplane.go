// SPDX-License-Identifier: GPL-2.0-only

package compdef

import "github.com/Oblikovati/oblikovati/model/sketch"

// WorkPlane is a part datum plane — one of the three origin planes (XY/XZ/YZ) or, in
// future, a user-created work plane. It hosts sketches and is selectable in the browser
// and the 3D view. DisplaySize is the half-extent at which the (infinite) plane is
// drawn and hit-tested as a finite square so it can be picked in the viewport.
type WorkPlane struct {
	name        string
	plane       sketch.Plane
	displaySize float64
}

// newWorkPlane constructs a named datum plane with a display half-extent.
func newWorkPlane(name string, plane sketch.Plane, displaySize float64) *WorkPlane {
	return &WorkPlane{name: name, plane: plane, displaySize: displaySize}
}

// Name returns the plane's display name (e.g. "XY Plane").
func (w *WorkPlane) Name() string { return w.name }

// Plane returns the underlying sketch plane (origin + in-plane axes), the host a new
// sketch is created on.
func (w *WorkPlane) Plane() sketch.Plane { return w.plane }

// DisplaySize is the half-edge length of the square the plane is drawn/picked as.
func (w *WorkPlane) DisplaySize() float64 { return w.displaySize }

// defaultOriginPlaneSize is the half-extent the origin planes display at.
const defaultOriginPlaneSize = 5.0

// newOriginPlanes builds the three standard origin work planes of a fresh part.
func newOriginPlanes() []*WorkPlane {
	return []*WorkPlane{
		newWorkPlane("XY Plane", sketch.XYPlane(), defaultOriginPlaneSize),
		newWorkPlane("XZ Plane", sketch.XZPlane(), defaultOriginPlaneSize),
		newWorkPlane("YZ Plane", sketch.YZPlane(), defaultOriginPlaneSize),
	}
}

// OriginPlanes returns the part's origin datum planes (XY/XZ/YZ), the Origin folder's
// contents in the browser and the default sketch hosts.
func (d *PartComponentDefinition) OriginPlanes() []*WorkPlane {
	return append([]*WorkPlane(nil), d.origin...)
}

// WorkPlaneByName returns the origin plane with the given name, or false.
func (d *PartComponentDefinition) WorkPlaneByName(name string) (*WorkPlane, bool) {
	for _, w := range d.origin {
		if w.name == name {
			return w, true
		}
	}
	return nil, false
}
