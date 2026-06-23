// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Projecting the part's datum geometry — the origin centre point, the origin/user work axes
// and planes — into a sketch needs associative handles like the B-rep edge/vertex sources
// (reference_source.go), but resolved through the part's [feature.WorkGeometry] by
// [feature.WorkRef] instead of by topology key. They re-resolve on every read so a moved
// user datum re-projects, and report lost when the reference no longer resolves. They
// structurally satisfy sketch.PointSource / sketch.CurveSource without this package importing
// the sketch seam types (#1262).

// referenceLineHalfSpan is the default half-length (model units) a projected work axis or
// plane-intersection reference line extends from its anchor when the part carries no geometry
// yet to size the line against (e.g. an origin axis on an empty part).
const referenceLineHalfSpan = 10.0

// WorkPointRefSource adapts a datum work point (by WorkRef) to a sketch point source: it
// re-resolves the point through the part's work geometry and yields its position.
type WorkPointRefSource struct {
	ref  feature.WorkRef
	work func() *feature.WorkGeometry
}

// NewWorkPointRefSource binds an associative point source to the datum point with reference
// ref on part. The closure (not a bound value) is used so it always sees the current work
// geometry.
func NewWorkPointRefSource(part *PartComponentDefinition, ref feature.WorkRef) WorkPointRefSource {
	return WorkPointRefSource{ref: ref, work: func() *feature.WorkGeometry { return part.WorkGeometry() }}
}

// SourceID returns the datum reference (its stable cross-recompute identity).
func (s WorkPointRefSource) SourceID() string { return string(s.ref) }

// Position re-resolves the datum point by reference; ok=false when it no longer resolves.
func (s WorkPointRefSource) Position() (math.Point3, bool) {
	w, ok := s.work().WorkPointByRef(s.ref)
	if !ok {
		return math.Point3{}, false
	}
	return w.Point(), true
}

// WorkAxisRefSource adapts a datum work axis (by WorkRef) to a sketch curve source: it
// samples the axis as a segment centred on its origin, so projecting it (the orthogonal map
// onto the sketch plane in ProjectCurve) yields the projected reference line — or a point
// when the axis is perpendicular to the sketch plane, matching Inventor.
type WorkAxisRefSource struct {
	ref  feature.WorkRef
	work func() *feature.WorkGeometry
	span func() float64
}

// NewWorkAxisRefSource binds an associative curve source to the datum axis with reference ref.
func NewWorkAxisRefSource(part *PartComponentDefinition, ref feature.WorkRef) WorkAxisRefSource {
	return WorkAxisRefSource{
		ref:  ref,
		work: func() *feature.WorkGeometry { return part.WorkGeometry() },
		span: part.referenceLineHalfSpan,
	}
}

// SourceID returns the datum axis reference.
func (s WorkAxisRefSource) SourceID() string { return string(s.ref) }

// SamplePoints re-resolves the axis and returns its two endpoints at ±span from the origin;
// ok=false when the axis no longer resolves.
func (s WorkAxisRefSource) SamplePoints() ([]math.Point3, bool) {
	a, ok := s.work().AxisByRef(s.ref)
	if !ok {
		return nil, false
	}
	step := a.Direction().AsVector().Scale(s.span())
	return []math.Point3{a.Origin().TranslateBy(step.Scale(-1)), a.Origin().TranslateBy(step)}, true
}

// WorkPlaneRefSource adapts a datum work plane (by WorkRef) to a sketch curve source: it
// returns the line where the work plane meets the target sketch plane — Inventor projects a
// work plane as that intersection line. ok=false when the planes are parallel (no line) or
// the reference is lost. The target sketch plane is fixed at construction, matching how a
// projected curve binds to one sketch plane (model/sketch projection.go).
type WorkPlaneRefSource struct {
	ref    feature.WorkRef
	target sketch.Plane
	work   func() *feature.WorkGeometry
	span   func() float64
}

// NewWorkPlaneRefSource binds an associative curve source to the datum plane with reference
// ref, projected onto the target sketch plane.
func NewWorkPlaneRefSource(part *PartComponentDefinition, ref feature.WorkRef, target sketch.Plane) WorkPlaneRefSource {
	return WorkPlaneRefSource{
		ref:    ref,
		target: target,
		work:   func() *feature.WorkGeometry { return part.WorkGeometry() },
		span:   part.referenceLineHalfSpan,
	}
}

// SourceID returns the datum plane reference.
func (s WorkPlaneRefSource) SourceID() string { return string(s.ref) }

// SamplePoints re-resolves the work plane and returns the two endpoints of its intersection
// line with the target sketch plane, at ±span from the line's anchor point; ok=false when
// the planes are parallel or the reference is lost.
func (s WorkPlaneRefSource) SamplePoints() ([]math.Point3, bool) {
	wp, err := s.work().WorkPlaneByRef(s.ref)
	if err != nil {
		return nil, false
	}
	at, dir, err := feature.PlaneIntersectionLine(wp.Plane(), s.target)
	if err != nil {
		return nil, false
	}
	step := dir.AsVector().Scale(s.span())
	return []math.Point3{at.TranslateBy(step.Scale(-1)), at.TranslateBy(step)}, true
}

// referenceLineHalfSpan sizes a projected work-axis / plane-intersection reference line: half
// the model range-box diagonal when the part has geometry, else the default span so an origin
// datum on an empty part still projects a visible line.
func (d *PartComponentDefinition) referenceLineHalfSpan() float64 {
	box := d.RangeBox()
	if box.IsEmpty() {
		return referenceLineHalfSpan
	}
	if half := box.Diagonal().Length() / 2; half > referenceLineHalfSpan {
		return half
	}
	return referenceLineHalfSpan
}

// WorkPlaneIntersectsSketch reports whether the datum plane ref meets the target sketch plane
// in a line — false when they are parallel (no reference line to project). Callers probe this
// before projecting a work plane so a parallel pick is skipped rather than producing a dead,
// geometry-less reference curve (#1262).
func (d *PartComponentDefinition) WorkPlaneIntersectsSketch(ref feature.WorkRef, target sketch.Plane) bool {
	_, ok := NewWorkPlaneRefSource(d, ref, target).SamplePoints()
	return ok
}

// WorkPointKeyResolves / WorkAxisKeyResolves / WorkPlaneKeyResolves report whether a WorkRef
// currently binds to a datum point/axis/plane — the router uses them to classify a projected
// reference (mirroring EdgeKeyResolves for B-rep references).
func (d *PartComponentDefinition) WorkPointKeyResolves(ref string) bool {
	_, ok := d.work.WorkPointByRef(feature.WorkRef(ref))
	return ok
}

// WorkAxisKeyResolves reports whether a WorkRef currently binds to a datum axis.
func (d *PartComponentDefinition) WorkAxisKeyResolves(ref string) bool {
	_, ok := d.work.AxisByRef(feature.WorkRef(ref))
	return ok
}

// WorkPlaneKeyResolves reports whether a WorkRef currently binds to a datum plane.
func (d *PartComponentDefinition) WorkPlaneKeyResolves(ref string) bool {
	_, err := d.work.WorkPlaneByRef(feature.WorkRef(ref))
	return err == nil
}
