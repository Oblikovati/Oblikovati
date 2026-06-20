// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Associative point-cloud-fit work plane (M17-F06, #645): a work plane derived by best-fitting a
// scanned point cloud, which carries its provenance — the source cloud's id — so it re-fits when
// the cloud moves and the link round-trips in the document. The model/feature package reaches the
// cloud only through PlaneFitSource, never the pointcloud package (the seam discipline used by the
// projection PointSource).

// PlaneFitSource yields a best-fit plane frame from external data (a point cloud's current points).
// SourceID is the cloud's stable id, persisted so the link can be re-attached after a load.
// FitFrame returns ok=false when the source cannot fit a plane right now (too few points, lost
// reference), so the def freezes its last good fit instead of jumping.
type PlaneFitSource interface {
	SourceID() string
	FitFrame() (origin math.Point3, xAxis, yAxis math.UnitVector3, ok bool)
}

// pointCloudFitPlaneDef is a work plane associatively fit to a point cloud. On each recompute it
// re-fits to the cloud's current points (so the plane follows the cloud); when the source is
// unavailable — a lost reference, or a freshly loaded document before the source is re-attached —
// it falls back to the last good fit (the frozen frame, which is what persists).
type pointCloudFitPlaneDef struct {
	source  PlaneFitSource
	cloudID string // persisted provenance link (equals source.SourceID() while linked)
	// frozen frame: the last successful fit, the fallback and the serialized form.
	origin math.Point3
	x, y   math.UnitVector3
	hasFit bool
}

func (d *pointCloudFitPlaneDef) kindName() string { return "point-cloud-fit" }
func (d *pointCloudFitPlaneDef) refs() []WorkRef  { return nil }

// eval re-fits from the live source when attached, updating the frozen frame; otherwise it returns
// the frozen frame, erroring only when there is neither a source nor a prior fit.
func (d *pointCloudFitPlaneDef) eval(workResolver) (sketch.Plane, error) {
	if d.source != nil {
		if o, x, y, ok := d.source.FitFrame(); ok {
			d.origin, d.x, d.y, d.hasFit = o, x, y, true
		}
	}
	if !d.hasFit {
		return sketch.Plane{}, errors.New("point-cloud-fit plane: no source cloud to fit and no prior fit")
	}
	return sketch.NewPlane(d.origin, d.x, d.y)
}

// CloudID returns the provenance link — the id of the cloud this plane is fit to.
func (d *pointCloudFitPlaneDef) CloudID() string { return d.cloudID }

// relink attaches a live source whose id matches this def's cloud, restoring associativity after a
// load. It is a no-op when the source's id does not match.
func (d *pointCloudFitPlaneDef) relink(src PlaneFitSource) bool {
	if src.SourceID() != d.cloudID {
		return false
	}
	d.source = src
	return true
}

// AddByPointCloudFit creates a work plane fit to the cloud identified by src.SourceID(), recording
// that cloud as its provenance. The initial fit (if the source can provide one) seeds the frozen
// frame so the plane has geometry even before the first recompute.
func (c *WorkPlanes) AddByPointCloudFit(src PlaneFitSource) *WorkPlane {
	d := &pointCloudFitPlaneDef{source: src, cloudID: src.SourceID()}
	if o, x, y, ok := src.FitFrame(); ok {
		d.origin, d.x, d.y, d.hasFit = o, x, y, true
	}
	return c.addUser(d)
}

// RelinkCloudFits re-attaches live sources to this part's point-cloud-fit planes after a load,
// asking attach for a source by cloud id. It is the post-load wiring that restores associativity
// (the link itself round-trips; the live source object does not). Returns how many were relinked.
func (c *WorkPlanes) RelinkCloudFits(attach func(cloudID string) (PlaneFitSource, bool)) int {
	n := 0
	for _, w := range c.items {
		d, ok := w.def.(*pointCloudFitPlaneDef)
		if !ok {
			continue
		}
		if src, found := attach(d.cloudID); found && d.relink(src) {
			n++
		}
	}
	return n
}
