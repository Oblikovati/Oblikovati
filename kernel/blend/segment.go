// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CommonPoint marks an endpoint of a blend segment — a point common to the blend and one support
// face's boundary, and shared with the adjacent segment or corner. It carries the spine parameter
// where it sits, the 3D point and tangent, and (when it lands on one) the body vertex or edge it
// coincides with. Mirrors OCCT ChFiDS_CommonPoint (the four per-segment start/end points).
type CommonPoint struct {
	SpineParam float64
	Point      math.Point3
	Tangent    math.Vector3
	OnVertex   *topo.Vertex // non-nil ⇒ the point is a body vertex
	OnEdge     *topo.Edge   // non-nil ⇒ the point lies on this edge at EdgeParam
	EdgeParam  float64
	Tolerance  float64
}

// FaceInterference is the trimmed 3D curve where the blend surface meets one support face — the
// boundary the segment is trimmed against on that side, with the parameter range of the curve the
// segment spans. Mirrors OCCT ChFiDS_FaceInterference. (We hold the curve directly rather than an
// index into a surface pool, which OCCT needs only for its transient bookkeeping.)
type FaceInterference struct {
	Face        *topo.Face
	Curve       geom.Curve3 // blend ∩ face, in 3D
	First, Last float64     // parameter span along Curve
}

// BlendSegment is one stretch of a fillet or chamfer along the spine: the blend surface, its
// interference with each of the two support faces, and the four common points bounding it
// (start/end × side 1/2), plus the spine-parameter range it covers. This is our ChFiDS_SurfData —
// the marcher (Phase 4) emits a run of these, and ops trims and stitches them into the result body.
type BlendSegment struct {
	Surface               geom.Surface
	OnS1, OnS2            FaceInterference // interference with support face 1 and 2
	Start1, Start2        CommonPoint      // the segment's First endpoint on side 1 / side 2
	End1, End2            CommonPoint      // the segment's Last endpoint on side 1 / side 2
	FirstSpine, LastSpine float64          // spine-parameter range this segment covers
}

// SpineSpan is the guide-parameter range the segment covers, for stitching consecutive segments.
func (s BlendSegment) SpineSpan() (first, last float64) { return s.FirstSpine, s.LastSpine }
