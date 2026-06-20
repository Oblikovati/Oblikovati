// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// Datum-cloud provenance for sketch points (M17-F06, #645): a sketch point placed on a scanned
// point keeps a link to its cloud so it re-projects onto the sketch plane when the cloud moves, and
// persists the link so a reopened document re-anchors it — the sketch counterpart of the
// point-cloud-fit plane and the anchored work point.

// CloudPointAnchor is a scan point a sketch point is anchored to. SourceID is the cloud's id and
// LocalAnchor the cloud-local 3D position — both persisted so the link is re-attached after a load;
// ModelPosition re-derives the current model-space location (ok=false when the cloud is gone), which
// the sketch projects onto its plane.
type CloudPointAnchor interface {
	SourceID() string
	LocalAnchor() math.Point3
	ModelPosition() (math.Point3, bool)
}

// cloudAnchoredPoint links a sketch point to a scan point. The cloud id and local anchor are stored
// independently of the live source so they survive a load (when source is nil until relinked).
type cloudAnchoredPoint struct {
	point   *Point
	cloudID string
	local   math.Point3
	source  CloudPointAnchor
}

// AddCloudAnchoredPoint adds a standalone sketch point anchored on the scan point described by
// anchor, projected onto the sketch plane, and records the cloud link (provenance). The point
// follows the cloud on UpdateProjections.
func (s *Sketch) AddCloudAnchoredPoint(anchor CloudPointAnchor) *Point {
	pos := s.plane.ToSketch(modelOrZero(anchor))
	p := s.points.Add(pos)
	s.cloudPts = append(s.cloudPts, &cloudAnchoredPoint{
		point: p, cloudID: anchor.SourceID(), local: anchor.LocalAnchor(), source: anchor,
	})
	return p
}

// updateCloudAnchors re-projects every cloud-anchored point from its (live) source onto the sketch
// plane; a point whose source is detached (loaded but not yet relinked, or lost) keeps its position.
func (s *Sketch) updateCloudAnchors() {
	for _, a := range s.cloudPts {
		if a.source == nil {
			continue
		}
		if pos, ok := a.source.ModelPosition(); ok {
			a.point.SetPosition(s.plane.ToSketch(pos))
		}
	}
}

// CloudAnchors returns the persisted provenance of each cloud-anchored point — the point's id, the
// source cloud's id, and the cloud-local anchor — for serialization.
func (s *Sketch) CloudAnchors() []CloudAnchorInfo {
	out := make([]CloudAnchorInfo, 0, len(s.cloudPts))
	for _, a := range s.cloudPts {
		out = append(out, CloudAnchorInfo{PointID: a.point.id, CloudID: a.cloudID, Local: a.local})
	}
	return out
}

// CloudAnchorInfo is the persisted form of one cloud-anchored point.
type CloudAnchorInfo struct {
	PointID ID
	CloudID string
	Local   math.Point3
}

// RestoreCloudAnchor re-creates a cloud-anchored link over an already-restored point (its source is
// re-attached later by RelinkCloudAnchors). It is the restore-side counterpart of
// AddCloudAnchoredPoint.
func (s *Sketch) RestoreCloudAnchor(point *Point, cloudID string, local math.Point3) {
	s.cloudPts = append(s.cloudPts, &cloudAnchoredPoint{point: point, cloudID: cloudID, local: local})
}

// RelinkCloudAnchors re-attaches live cloud sources to this sketch's cloud-anchored points after a
// load (matching by cloud id), then re-projects them. attach receives the cloud id and the local
// anchor so the host can rebuild the source. Returns how many were relinked.
func (s *Sketch) RelinkCloudAnchors(attach func(cloudID string, local math.Point3) (CloudPointAnchor, bool)) int {
	n := 0
	for _, a := range s.cloudPts {
		if src, ok := attach(a.cloudID, a.local); ok {
			a.source = src
			n++
		}
	}
	if n > 0 {
		s.updateCloudAnchors()
	}
	return n
}

// modelOrZero returns the anchor's current model position, or the origin when it cannot fit (the
// initial seed; the point re-projects once a live source is present).
func modelOrZero(a CloudPointAnchor) math.Point3 {
	if p, ok := a.ModelPosition(); ok {
		return p
	}
	return math.Point3{}
}
