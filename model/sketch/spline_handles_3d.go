// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// 3D spline tangency handles — the SketchSplineHandle3D side of M06-F11
// (Oblikovati/Oblikovati#626), mirroring spline_handles.go in model space.
// The handle's end is a real, constrainable Point3D; the 3D sampler honors
// active handles with per-span Hermite evaluation.

// SplineHandle3D is the tangency handle on one fit point of a 3D spline.
type SplineHandle3D struct {
	entityBase
	Spline   *Spline3D
	FitIndex int
	Anchor   *Point3D
	End      *Point3D
}

// TangentDirection returns the handle's unit direction (anchor → end); ok is
// false while the handle is degenerate.
func (h *SplineHandle3D) TangentDirection() (math.Vector3, bool) {
	v := h.Anchor.Position().VectorTo(h.End.Position())
	l := float64(v.Length())
	if l == 0 {
		return math.Vector3{}, false
	}
	return v.Scale(math.Scalar(1 / l)), true
}

// Weight is the tangent-magnitude multiplier relative to a third of the
// neighbor chord (weight 1 = the default activation length).
func (h *SplineHandle3D) Weight() float64 {
	natural := h.naturalLength()
	if natural == 0 {
		return 1
	}
	return float64(h.Anchor.Position().DistanceTo(h.End.Position())) / natural
}

// naturalLength is the weight-1 handle length (a third of the neighbor chord).
func (h *SplineHandle3D) naturalLength() float64 {
	n := len(h.Spline.Points)
	if n < 2 {
		return 0
	}
	prev, next := h.FitIndex-1, h.FitIndex+1
	if h.Spline.Closed {
		prev, next = (prev+n)%n, next%n
	} else {
		if prev < 0 {
			prev = h.FitIndex
		}
		if next >= n {
			next = h.FitIndex
		}
	}
	return float64(h.Spline.Points[prev].Position().DistanceTo(h.Spline.Points[next].Position())) / 3
}

// SetTangent points the handle along dir at the given weight (0 keeps the
// current weight).
func (h *SplineHandle3D) SetTangent(dir math.Vector3, weight float64) error {
	l := float64(dir.Length())
	if l == 0 {
		return fmt.Errorf("spline handle tangent needs a non-zero direction, got %v", dir)
	}
	if weight == 0 {
		weight = h.Weight()
	}
	span := h.naturalLength() * weight
	h.End.SetPosition(h.Anchor.Position().TranslateBy(dir.Scale(math.Scalar(span / l))))
	return nil
}

// ActivateSplineHandle3D creates (or returns) the handle on the spline's
// fitIndex-th defining point, starting at the natural tangent with weight 1.
func (s *Sketch3D) ActivateSplineHandle3D(sp *Spline3D, fitIndex int) (*SplineHandle3D, error) {
	if !sp.IsFitType() {
		return nil, fmt.Errorf("spline handles need a fit spline, entity %d is control-point", sp.id)
	}
	if fitIndex < 0 || fitIndex >= len(sp.Points) {
		return nil, fmt.Errorf("fit point index %d out of range (spline has %d points)", fitIndex, len(sp.Points))
	}
	if h, ok := sp.handleAt(fitIndex); ok {
		return h, nil
	}
	anchor := sp.Points[fitIndex]
	h := &SplineHandle3D{
		entityBase: newEntity(),
		Spline:     sp, FitIndex: fitIndex,
		Anchor: anchor,
		End:    s.newPoint3D(defaultHandleEnd3D(sp, fitIndex, anchor.Position())),
	}
	sp.attachHandle(h)
	s.addEntity3D(h)
	return h, nil
}

// DeactivateSplineHandle3D removes the handle; reports whether one existed.
func (s *Sketch3D) DeactivateSplineHandle3D(sp *Spline3D, fitIndex int) bool {
	h, ok := sp.handleAt(fitIndex)
	if !ok {
		return false
	}
	sp.detachHandle(fitIndex)
	s.removeEntity3D(h)
	s.removePoint3D(h.End)
	return true
}

// defaultHandleEnd3D places a fresh handle along the natural fitted tangent.
func defaultHandleEnd3D(sp *Spline3D, fitIndex int, anchor math.Point3) math.Point3 {
	dir := math.V3(1, 0, 0)
	pts := make([]math.Point3, len(sp.Points))
	for i, p := range sp.Points {
		pts[i] = p.Position()
	}
	if curve, ubar, err := fitCurve3DFor(pts, sp.Closed); err == nil {
		t := curve.TangentAt(ubar[fitIndex])
		if l := float64(t.Length()); l > 0 {
			dir = t.Scale(math.Scalar(1 / l))
		}
	}
	h := SplineHandle3D{Spline: sp, FitIndex: fitIndex}
	return anchor.TranslateBy(dir.Scale(math.Scalar(h.naturalLength())))
}

// handleAt / attachHandle / detachHandle / Handles mirror the 2D spline's.
func (sp *Spline3D) handleAt(fitIndex int) (*SplineHandle3D, bool) {
	h, ok := sp.handles[fitIndex]
	return h, ok
}

func (sp *Spline3D) attachHandle(h *SplineHandle3D) {
	if sp.handles == nil {
		sp.handles = map[int]*SplineHandle3D{}
	}
	sp.handles[h.FitIndex] = h
}

func (sp *Spline3D) detachHandle(fitIndex int) { delete(sp.handles, fitIndex) }

// Handles returns the spline's active handles in fit-point order.
func (sp *Spline3D) Handles() []*SplineHandle3D {
	out := make([]*SplineHandle3D, 0, len(sp.handles))
	for i := range sp.Points {
		if h, ok := sp.handles[i]; ok {
			out = append(out, h)
		}
	}
	return out
}

// sampleHandledSpline3D mirrors sampleHandledSpline: Hermite spans whose end
// derivatives come from the fitted curve, overridden by active handles.
func sampleHandledSpline3D(sp *Spline3D, perSpan int) []math.Point3 {
	pts := make([]math.Point3, len(sp.Points))
	for i, p := range sp.Points {
		pts[i] = p.Position()
	}
	curve, ubar, err := fitCurve3DFor(pts, sp.Closed)
	if err != nil {
		return pts
	}
	derivs := make([]math.Vector3, len(pts))
	for i := range pts {
		natural := curve.TangentAt(ubar[i])
		derivs[i] = natural
		if h, ok := sp.handles[i]; ok {
			if dir, valid := h.TangentDirection(); valid {
				derivs[i] = dir.Scale(math.Scalar(h.Weight() * float64(natural.Length())))
			}
		}
	}
	return hermiteChain3D(pts, derivs, ubar, sp.Closed, perSpan)
}

// hermiteChain3D evaluates the cubic Hermite chain through pts with the given
// per-point derivatives at the fit parameters.
func hermiteChain3D(pts []math.Point3, derivs []math.Vector3, ubar []float64, closed bool, perSpan int) []math.Point3 {
	out := make([]math.Point3, 0, (len(ubar)-1)*perSpan+1)
	for k := 0; k+1 < len(ubar); k++ {
		du := math.Scalar(ubar[k+1] - ubar[k])
		p0, p1 := pts[k%len(pts)], pts[(k+1)%len(pts)]
		m0, m1 := derivs[k%len(pts)].Scale(du), derivs[(k+1)%len(pts)].Scale(du)
		out = append(out, p0)
		for j := 1; j < perSpan; j++ {
			out = append(out, hermitePoint3D(p0, p1, m0, m1, float64(j)/float64(perSpan)))
		}
	}
	if !closed {
		out = append(out, pts[len(pts)-1])
	}
	return out
}

// hermitePoint3D evaluates the cubic Hermite span (p0, m0) → (p1, m1) at s∈[0,1].
func hermitePoint3D(p0, p1 math.Point3, m0, m1 math.Vector3, s float64) math.Point3 {
	s2, s3 := s*s, s*s*s
	h00 := math.Scalar(2*s3 - 3*s2 + 1)
	h10 := math.Scalar(s3 - 2*s2 + s)
	h01 := math.Scalar(-2*s3 + 3*s2)
	h11 := math.Scalar(s3 - s2)
	v := p0.AsVector().Scale(h00).Add(m0.Scale(h10)).Add(p1.AsVector().Scale(h01)).Add(m1.Scale(h11))
	return v.AsPoint()
}
