// SPDX-License-Identifier: GPL-2.0-only

package compdef

import "github.com/Oblikovati/oblikovati/model/feature"

// The part's datum geometry — the origin coordinate system (XY/XZ/YZ planes, X/Y/Z
// axes, center point) plus user-created work planes/axes/points — lives in one
// [feature.WorkGeometry], matching how Inventor structures a ComponentDefinition's
// WorkPlanes/WorkAxes/WorkPoints collections (the origin elements are grounded
// coordinate-system members). The part delegates to it.

// WorkGeometry returns the part's construction-geometry frame.
func (d *PartComponentDefinition) WorkGeometry() *feature.WorkGeometry { return d.work }

// WorkPlanes/WorkAxes/WorkPoints/UserCoordinateSystems expose the datum collections.
func (d *PartComponentDefinition) WorkPlanes() *feature.WorkPlanes { return d.work.WorkPlanes() }
func (d *PartComponentDefinition) WorkAxes() *feature.WorkAxes     { return d.work.WorkAxes() }
func (d *PartComponentDefinition) WorkPoints() *feature.WorkPoints { return d.work.WorkPoints() }
func (d *PartComponentDefinition) UserCoordinateSystems() *feature.UserCoordinateSystems {
	return d.work.UserCoordinateSystems()
}

// OriginPlanes returns the part's origin datum planes (XY/XZ/YZ), the Origin folder's
// contents in the browser and the default sketch hosts.
func (d *PartComponentDefinition) OriginPlanes() []*feature.WorkPlane {
	return d.work.OriginPlanes()
}

// WorkPlaneByName returns the work plane with the given name (e.g. "XY Plane"), or false.
func (d *PartComponentDefinition) WorkPlaneByName(name string) (*feature.WorkPlane, bool) {
	planes := d.work.WorkPlanes()
	for i := 0; i < planes.Count(); i++ {
		if w := planes.Item(i); w.Name() == name {
			return w, true
		}
	}
	return nil, false
}
