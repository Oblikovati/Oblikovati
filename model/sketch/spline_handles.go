// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Spline tangency handles (M06-F11, Oblikovati/Oblikovati#626): every fit
// point of an interpolation spline owns a latent handle; activating it adds a
// SplineHandle entity whose end point is real, constrainable sketch geometry.
// The handle prescribes the curve's tangent at that fit point — direction
// from the anchor→end segment, magnitude scaling the natural tangent — and
// the sampler honors it with per-span Hermite evaluation, so dragging or
// constraining the handle reshapes the curve.

// SplineHandle is the tangency handle on one fit point of a spline. Anchor is
// the spline's own fit point (shared pointer — moving the point moves the
// handle); End is the draggable tip.
type SplineHandle struct {
	entityBase
	Spline   *Spline
	FitIndex int
	Anchor   *Point
	End      *Point
}

// TangentDirection returns the handle's unit direction (anchor → end); ok is
// false while the handle is degenerate (end on top of the anchor).
func (h *SplineHandle) TangentDirection() (math.Vector2, bool) {
	v := h.Anchor.Position().VectorTo(h.End.Position())
	l := float64(v.Length())
	if l == 0 {
		return math.Vector2{}, false
	}
	return v.Scale(math.Scalar(1 / l)), true
}

// Weight is the handle's tangent-magnitude multiplier: the handle length
// relative to a third of the neighbor chord (the Bezier-handle convention the
// default activation uses, so a fresh handle has weight 1).
func (h *SplineHandle) Weight() float64 {
	natural := h.naturalLength()
	if natural == 0 {
		return 1
	}
	return float64(h.Anchor.Position().DistanceTo(h.End.Position())) / natural
}

// naturalLength is the weight-1 handle length: a third of the chord between
// the fit point's neighbors (or to the single neighbor at an open end).
func (h *SplineHandle) naturalLength() float64 {
	pts := splinePositions(h.Spline)
	n := len(pts)
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
	return float64(pts[prev].DistanceTo(pts[next])) / 3
}

// SplineHandles tracks the sketch's active handles.
type SplineHandles struct {
	s     *Sketch
	items []*SplineHandle
}

// Count returns the number of active handles; Item returns the i-th.
func (c *SplineHandles) Count() int               { return len(c.items) }
func (c *SplineHandles) Item(i int) *SplineHandle { return c.items[i] }

// Activate creates (or returns) the handle on the spline's fitIndex-th
// defining point. A fresh handle starts at the curve's natural tangent with
// weight 1. Only fit (interpolation) splines carry handles.
func (c *SplineHandles) Activate(sp *Spline, fitIndex int) (*SplineHandle, error) {
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
	h := &SplineHandle{
		entityBase: newEntity(),
		Spline:     sp, FitIndex: fitIndex,
		Anchor: anchor,
		End:    c.s.newPoint(defaultHandleEnd(sp, fitIndex, anchor.Position())),
	}
	sp.attachHandle(h)
	c.s.add(h)
	c.items = append(c.items, h)
	return h, nil
}

// Deactivate removes the handle from its spline and the sketch; the curve
// returns to its natural tangent there. Reports whether a handle existed.
func (c *SplineHandles) Deactivate(sp *Spline, fitIndex int) bool {
	h, ok := sp.handleAt(fitIndex)
	if !ok {
		return false
	}
	sp.detachHandle(fitIndex)
	c.s.removeEntity(h)
	c.s.removePoint(h.End)
	for i, x := range c.items {
		if x == h {
			c.items = append(c.items[:i], c.items[i+1:]...)
			break
		}
	}
	return true
}

// SetTangent points the handle's end along dir (need not be unit) at the
// given weight (0 keeps the current weight; a degenerate dir errors).
func (h *SplineHandle) SetTangent(dir math.Vector2, weight float64) error {
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

// defaultHandleEnd places a fresh handle along the spline's natural tangent
// at weight 1; a degenerate fit falls back to +X.
func defaultHandleEnd(sp *Spline, fitIndex int, anchor math.Point2) math.Point2 {
	dir := math.V2(1, 0)
	if curve, ubar, err := fitCurveFor(splinePositions(sp), sp.Closed, fitParameterization(sp.FitMethod)); err == nil {
		t := curve.TangentAt(ubar[fitIndex])
		if l := float64(t.Length()); l > 0 {
			dir = t.Scale(math.Scalar(1 / l))
		}
	}
	h := SplineHandle{Spline: sp, FitIndex: fitIndex}
	return anchor.TranslateBy(dir.Scale(math.Scalar(h.naturalLength())))
}

// handleAt / attachHandle / detachHandle manage a spline's handle map.
func (sp *Spline) handleAt(fitIndex int) (*SplineHandle, bool) {
	h, ok := sp.handles[fitIndex]
	return h, ok
}

func (sp *Spline) attachHandle(h *SplineHandle) {
	if sp.handles == nil {
		sp.handles = map[int]*SplineHandle{}
	}
	sp.handles[h.FitIndex] = h
}

func (sp *Spline) detachHandle(fitIndex int) { delete(sp.handles, fitIndex) }

// Handles returns the spline's active handles in fit-point order.
func (sp *Spline) Handles() []*SplineHandle {
	out := make([]*SplineHandle, 0, len(sp.handles))
	for i := range sp.Points {
		if h, ok := sp.handles[i]; ok {
			out = append(out, h)
		}
	}
	return out
}

// sampleHandledSpline samples a fit spline whose handles override the natural
// tangent: each span is a cubic Hermite between consecutive fit points whose
// end derivatives are the natural curve derivative — replaced, where a handle
// is active, by the handle direction scaled to weight × the natural
// magnitude. With no active handles it matches sampleFitSpline.
func sampleHandledSpline(sp *Spline, perSpan int) []math.Point2 {
	pts := splinePositions(sp)
	curve, ubar, err := fitCurveFor(pts, sp.Closed, fitParameterization(sp.FitMethod))
	if err != nil {
		return pts
	}
	derivs := handleDerivatives(sp, curve, ubar)
	out := make([]math.Point2, 0, (len(ubar)-1)*perSpan+1)
	for k := 0; k+1 < len(ubar); k++ {
		du := ubar[k+1] - ubar[k]
		p0, p1 := pts[k%len(pts)], pts[(k+1)%len(pts)]
		m0, m1 := derivs[k%len(pts)].Scale(math.Scalar(du)), derivs[(k+1)%len(pts)].Scale(math.Scalar(du))
		out = append(out, p0)
		for j := 1; j < perSpan; j++ {
			out = append(out, hermitePoint(p0, p1, m0, m1, float64(j)/float64(perSpan)))
		}
	}
	if !sp.Closed {
		out = append(out, pts[len(pts)-1])
	}
	return out
}

// handleDerivatives returns dP/du at every fit point: the fitted curve's
// natural derivative, overridden by active handles (direction from the
// handle, magnitude = weight × natural).
func handleDerivatives(sp *Spline, curve geom.BSplineCurve2d, ubar []float64) []math.Vector2 {
	out := make([]math.Vector2, len(sp.Points))
	for i := range sp.Points {
		natural := curve.TangentAt(ubar[i])
		out[i] = natural
		h, ok := sp.handles[i]
		if !ok {
			continue
		}
		dir, valid := h.TangentDirection()
		if !valid {
			continue
		}
		out[i] = dir.Scale(math.Scalar(h.Weight() * float64(natural.Length())))
	}
	return out
}

// hermitePoint evaluates the cubic Hermite span (p0, m0) → (p1, m1) at s∈[0,1].
func hermitePoint(p0, p1 math.Point2, m0, m1 math.Vector2, s float64) math.Point2 {
	s2, s3 := s*s, s*s*s
	h00 := math.Scalar(2*s3 - 3*s2 + 1)
	h10 := math.Scalar(s3 - 2*s2 + s)
	h01 := math.Scalar(-2*s3 + 3*s2)
	h11 := math.Scalar(s3 - s2)
	v := p0.AsVector().Scale(h00).Add(m0.Scale(h10)).Add(p1.AsVector().Scale(h01)).Add(m1.Scale(h11))
	return v.AsPoint()
}

// hasActiveHandles reports whether any handle shapes this spline.
func (sp *Spline) hasActiveHandles() bool { return len(sp.handles) > 0 }
