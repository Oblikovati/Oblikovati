// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/math"
)

// Including a 2D sketch's geometry into a 3D sketch (Inventor "Include Geometry" with a
// planar sketch as the source) reuses the same source seam as part-edge include: a 2D
// entity is adapted into a [PointSource]/[CurveSource] whose 3D position is the 2D point
// mapped through the source sketch's host plane. The adapters re-resolve the entity by id
// on every read, so an edit to the source 2D sketch flows through recompute, and a deleted
// source entity reports lost (ok=false) so the include freezes rather than dangles.

// sketch2DPointSource adapts a 2D sketch point into a PointSource: its 3D position is the
// point's sketch-space position lifted through the host plane.
type sketch2DPointSource struct {
	src   *Sketch
	plane Plane
	entID ID
	sid   string
}

// NewSketch2DPointSource adapts the point entID of the 2D sketch src into a PointSource
// suitable for [Sketch3D.IncludePoint3D]. The point is resolved by id on each read, so the
// include tracks edits to the source sketch and freezes if the point is deleted.
func NewSketch2DPointSource(src *Sketch, entID ID) PointSource {
	return &sketch2DPointSource{
		src:   src,
		plane: src.Plane(),
		entID: entID,
		sid:   fmt.Sprintf("sk2d:%d:%d", src.ID(), entID),
	}
}

func (s *sketch2DPointSource) SourceID() string { return s.sid }

// Position re-resolves the source point and lifts it into model space; ok=false once the
// point no longer resolves (it was deleted).
func (s *sketch2DPointSource) Position() (math.Point3, bool) {
	p, ok := s.src.PointByID(s.entID)
	if !ok {
		return math.Point3{}, false
	}
	return s.plane.ToModel(p.Position()), true
}

// sketch2DCurveSource adapts a 2D sketch curve entity into a CurveSource: its polyline is
// the entity's natural-order samples lifted through the host plane.
type sketch2DCurveSource struct {
	src   *Sketch
	plane Plane
	entID ID
	sid   string
}

// NewSketch2DCurveSource adapts the curve entID of the 2D sketch src into a CurveSource
// suitable for [Sketch3D.IncludeCurve3D]. The curve is resolved and re-sampled on each
// read, so the include tracks edits to the source sketch and freezes if it is deleted.
func NewSketch2DCurveSource(src *Sketch, entID ID) CurveSource {
	return &sketch2DCurveSource{
		src:   src,
		plane: src.Plane(),
		entID: entID,
		sid:   fmt.Sprintf("sk2d:%d:%d", src.ID(), entID),
	}
}

func (s *sketch2DCurveSource) SourceID() string { return s.sid }

// SamplePoints re-resolves the source curve, samples it in natural order, and lifts each
// point into model space; ok=false once the curve no longer resolves (it was deleted).
func (s *sketch2DCurveSource) SamplePoints() ([]math.Point3, bool) {
	e, ok := s.src.EntityByID(s.entID)
	if !ok {
		return nil, false
	}
	pts := naturalPolyline(e)
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[i] = s.plane.ToModel(p)
	}
	return out, true
}
