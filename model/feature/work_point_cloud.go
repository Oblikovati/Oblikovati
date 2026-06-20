// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/math"
)

// Datum-cloud provenance for work points (M17-F06, #645): a work point placed on a scanned point
// keeps a live link to its cloud so it follows the cloud's placement, and persists the cloud id so
// the link is re-attached after a reopen — the point counterpart of the point-cloud-fit plane.

// PointFromCloudSource yields a model-space point tracked from a point cloud — a snapped scan point
// that moves with the cloud. SourceID is the cloud's id (persisted to re-attach the link after a
// load); Position re-derives the current model-space location, ok=false when the cloud is gone.
type PointFromCloudSource interface {
	SourceID() string
	Position() (math.Point3, bool)
}

// pointCloudPointDef is a work point anchored on a scanned point. It re-derives its position from
// the cloud on each recompute (so it follows the cloud), freezing the last good position when the
// source is unavailable (a lost reference, or a freshly loaded document before re-attachment).
type pointCloudPointDef struct {
	source  PointFromCloudSource
	cloudID string // persisted provenance link (equals source.SourceID() while linked)
	frozen  math.Point3
	hasPos  bool
}

func (d *pointCloudPointDef) kindName() string { return "point-cloud-point" }
func (d *pointCloudPointDef) refs() []WorkRef  { return nil }

func (d *pointCloudPointDef) eval(workResolver) (math.Point3, error) {
	if d.source != nil {
		if p, ok := d.source.Position(); ok {
			d.frozen, d.hasPos = p, true
		}
	}
	if !d.hasPos {
		return math.Point3{}, errors.New("point-cloud point: no source cloud and no prior position")
	}
	return d.frozen, nil
}

// CloudID returns the provenance link — the id of the cloud this point is anchored to.
func (d *pointCloudPointDef) CloudID() string { return d.cloudID }

// FrozenPosition returns the last good model-space position (the serialized fallback). The host
// uses it to reconstruct the cloud-local anchor when re-attaching the live source after a load.
func (d *pointCloudPointDef) FrozenPosition() math.Point3 { return d.frozen }

func (d *pointCloudPointDef) relink(src PointFromCloudSource) bool {
	if src.SourceID() != d.cloudID {
		return false
	}
	d.source = src
	return true
}

// AddByCloudPoint creates a work point anchored on the scan point yielded by src, recording its
// cloud as provenance and seeding the frozen position from the initial location.
func (c *WorkPoints) AddByCloudPoint(src PointFromCloudSource) *WorkPoint {
	d := &pointCloudPointDef{source: src, cloudID: src.SourceID()}
	if p, ok := src.Position(); ok {
		d.frozen, d.hasPos = p, true
	}
	return c.addUser(d)
}

// RelinkCloudPoints re-attaches live cloud sources to this part's point-cloud work points after a
// load. attach receives the cloud id and the frozen model position (so the host can reconstruct
// the cloud-local anchor) and returns a source. Returns how many were relinked.
func (c *WorkPoints) RelinkCloudPoints(attach func(cloudID string, frozen math.Point3) (PointFromCloudSource, bool)) int {
	n := 0
	for _, w := range c.items {
		d, ok := w.def.(*pointCloudPointDef)
		if !ok {
			continue
		}
		if src, found := attach(d.cloudID, d.frozen); found && d.relink(src) {
			n++
		}
	}
	return n
}
