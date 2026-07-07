// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Marcher is the general constant-radius blend solver (ADR-0050 Phase 4, method B): the rolling-ball
// CENTRE curve is the intersection of the two support offsets, and the blend surface is the canal of
// radius r about it. For primitive support pairs that intersection is exact (a line ⇒ a cylinder
// blend, a circle ⇒ a torus blend) via our analytic SSI; otherwise the centre curve is marched and
// the canal is fit as a B-spline (approx.go). It is injected into blend.Builder by ops, which
// supplies the in-solid predicate and the model-scaled resolution (mirrors OCCT BRepBlend_Walking).
type Marcher struct {
	Inside func(math.Point3) bool // reports whether a point is inside the target solid
	Res    geom.Resolution        // model-relative coincidence scale (from the body bbox)
}

// March builds a blend segment per spine edge and returns the run. A segment that cannot be built
// (no valid seed section — the radius exceeds the local curvature bound) yields the localized status
// and stops the run there, following OCCT's partial-result contract.
func (m *Marcher) March(sp *Spine, sec SectionFunctional) Result {
	var out []BlendSegment
	for i := 0; i < sp.NbEdges(); i++ {
		faces := sp.SupportFaces(i)
		if len(faces) != 2 {
			return Result{Segments: out, Status: StatusWalkingFailed}
		}
		seg, st := m.segment(sp, i, faces[0].Geometry(), faces[1].Geometry(), sec)
		if st != StatusOk {
			return Result{Segments: out, Status: st}
		}
		out = append(out, seg)
	}
	return Result{Segments: out, Status: StatusOk}
}

// segment builds one blend segment over spine edge i between supports a and b.
func (m *Marcher) segment(sp *Spine, i int, a, b geom.Surface, sec SectionFunctional) (BlendSegment, ErrorStatus) {
	r := sec.Extent(midSpineParam(sp, i))
	anchor, ok := m.expectedCentre(sp.PointAt(midSpineParam(sp, i)), a, b, r)
	if !ok {
		return BlendSegment{}, StatusStartSectionFailed // radius exceeds the local curvature bound here
	}
	centre, ok := m.centreCurve(a, b, r, anchor)
	if !ok {
		return BlendSegment{}, StatusStartSectionFailed
	}
	surf, ok := analyticBlendSurface(centre, r)
	if !ok {
		lo, hi := centre.Domain()
		var st ErrorStatus
		if surf, st = fitCanal(centre, a, b, r, lo, hi, m.Res.Weld(), m.Inside); st != StatusOk {
			return BlendSegment{}, st
		}
	}
	first, last := sp.EdgeSpineRange(i)
	return BlendSegment{
		Surface:    surf,
		OnS1:       FaceInterference{Face: sp.SupportFaces(i)[0]},
		OnS2:       FaceInterference{Face: sp.SupportFaces(i)[1]},
		FirstSpine: first,
		LastSpine:  last,
	}, StatusOk
}

// centreCurve returns the rolling-ball centre curve between primitive supports a and b at radius r:
// it intersects the analytic offsets for each of the four inside/outside sign combinations and
// returns the branch passing through the anchor (the analytic ball centre at the guide station). The
// anchor — not domain sampling — selects the branch, so it is robust for an UNBOUNDED centre curve
// (a plane∩plane blend's centre is an infinite line, whose domain midpoint is meaningless): #1797's
// straight segments failed the old domain-sampled validation for exactly this reason.
func (m *Marcher) centreCurve(a, b geom.Surface, r float64, anchor math.Point3) (geom.Curve3, bool) {
	best, bestErr := geom.Curve3(nil), stdmath.Inf(1)
	for _, sa := range [2]float64{-r, r} {
		oa, oka := offsetPrimitive(a, sa)
		if !oka {
			continue
		}
		for _, sb := range [2]float64{-r, r} {
			ob, okb := offsetPrimitive(b, sb)
			if !okb {
				continue
			}
			curves, handled := geom.IntersectSurfacesAnalytic(oa, ob, m.Res)
			if !handled {
				continue
			}
			for _, cv := range curves {
				if d := distPointToCurve(cv, anchor); d < bestErr {
					best, bestErr = cv, d
				}
			}
		}
	}
	if best == nil || bestErr > m.Res.Weld() {
		return nil, false
	}
	return best, true
}

// expectedCentre returns the exact rolling-ball centre at a guide station: the point receded from the
// spine point mid by r along the interior bisector of the two supports (offDir = −(nA+nB)/(1+nA·nB),
// the same convex-fillet offset ops uses), with each support normal oriented outward via Inside. It
// doubles as the seed-section existence check — ok=false when the centre admits no exposed section of
// radius r (the radius exceeds the local curvature bound, OCCT's StartSolFailure).
func (m *Marcher) expectedCentre(mid math.Point3, a, b geom.Surface, r float64) (math.Point3, bool) {
	nA, nB := m.outwardNormal(a, mid), m.outwardNormal(b, mid)
	denom := 1 + nA.Dot(nB)
	if denom < 1e-9 { // supports face opposite ways (a ~180° edge) — no rolling-ball corner
		return math.Point3{}, false
	}
	offDir := nA.Add(nB).Scale(-r / denom)
	c := mid.TranslateBy(offDir)
	if _, ok := sectionAt(c, a, b, r, m.Res.Weld(), m.Inside); !ok {
		return math.Point3{}, false
	}
	return c, true
}

// outwardNormal returns support s's unit normal at the foot of p, flipped to point OUT of the solid
// (a small step along it leaves the material). The sign is what makes offDir recede into the solid.
func (m *Marcher) outwardNormal(s geom.Surface, p math.Point3) math.Vector3 {
	u, v := s.ParamAt(p)
	n := unitVec(s.NormalAt(u, v))
	if m.Inside(p.TranslateBy(n.Scale(10 * m.Res.Weld()))) {
		return n.Scale(-1)
	}
	return n
}

// distPointToCurve is the distance from p to curve cv — closed-form for an unbounded line, else the
// minimum over a dense domain sampling (all other analytic centre curves — circle, ellipse — are
// bounded). It ranks the offset-intersection branches so centreCurve picks the one through the anchor.
func distPointToCurve(cv geom.Curve3, p math.Point3) float64 {
	lo, hi := cv.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		o := cv.PointAt(0)
		d := unitVec(cv.TangentAt(0))
		w := o.VectorTo(p)
		return float64(w.Sub(d.Scale(w.Dot(d))).Length())
	}
	best := stdmath.Inf(1)
	const n = 64
	for i := 0; i <= n; i++ {
		if d := float64(cv.PointAt(lo + (hi-lo)*float64(i)/n).DistanceTo(p)); d < best {
			best = d
		}
	}
	return best
}

// offsetPrimitive returns the exact offset of a primitive support by signed distance d — the plane
// shifted d along its normal, or the coaxial cylinder of radius R+d — or ok=false for a non-primitive
// (the general path uses geom.OffsetSurface + the SSI tracer instead).
func offsetPrimitive(s geom.Surface, d float64) (geom.Surface, bool) {
	switch t := s.(type) {
	case geom.Plane:
		p, err := geom.NewPlane(t.Origin.TranslateBy(t.Normal().Scale(math.Scalar(d))), t.Normal())
		return p, err == nil
	case geom.Cylinder:
		if t.Radius+d <= 0 {
			return nil, false
		}
		cyl, err := geom.NewCylinderWithRef(t.Origin, t.AxisDir.AsVector(), t.Ref.AsVector(), t.Radius+d)
		return cyl, err == nil
	}
	return nil, false
}

// analyticBlendSurface returns the exact blend surface for a primitive centre curve: the canal of
// radius r about a straight centre is a cylinder, about a circular centre a torus. Any other centre
// (an ellipse or a marched NURBS curve) has no primitive canal and returns ok=false (approx.go fits
// it as a B-spline).
func analyticBlendSurface(centre geom.Curve3, r float64) (geom.Surface, bool) {
	switch c := centre.(type) {
	case geom.Line:
		cyl, err := geom.NewCylinder(c.Origin, c.Dir.AsVector(), r)
		return cyl, err == nil
	case geom.LineSegment:
		cyl, err := geom.NewCylinder(c.StartPoint, c.StartPoint.VectorTo(c.EndPoint), r)
		return cyl, err == nil
	case geom.Circle:
		tor, err := geom.NewTorus(c.Center, c.Normal.AsVector(), c.Radius, r)
		return tor, err == nil
	}
	return nil, false
}

// midSpineParam is the spine abscissa at the middle of edge i (where the section radius is sampled).
func midSpineParam(sp *Spine, i int) float64 {
	lo, hi := sp.EdgeSpineRange(i)
	return (lo + hi) / 2
}
