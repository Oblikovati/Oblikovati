// SPDX-License-Identifier: GPL-2.0-only

package sketch

// EntityKind names what a sketch entity IS — the stable identifier its
// persistence codec is registered under (#1624, audit I1). The string values
// are the .obk recipe vocabulary (ADR-0020) and must never change once
// shipped; they intentionally stay decoupled from the wire enum
// api/types.SketchEntityKind, which follows API SemVer instead — the router
// maps between the two at its boundary.
//
// 2D and 3D entities share one vocabulary (a Line3D is still a "line"); the
// dimension is carried by which codec registry the kind is looked up in.
type EntityKind string

// The persisted entity kinds. A kind used by both dimensions (line, circle,
// arc, ellipse, ellipticalArc, spline, fixedSpline, equationCurve) has one
// constant and two codecs.
const (
	LineKind               EntityKind = "line"
	CircleKind             EntityKind = "circle"
	ArcKind                EntityKind = "arc"
	EllipseKind            EntityKind = "ellipse"
	EllipticalArcKind      EntityKind = "ellipticalArc"
	SplineKind             EntityKind = "spline"
	ControlPointSplineKind EntityKind = "controlPointSpline" // 3D only: a non-fit Spline3D
	FixedSplineKind        EntityKind = "fixedSpline"
	OffsetSplineKind       EntityKind = "offsetSpline"
	EquationCurveKind      EntityKind = "equationCurve"
	BlockInstanceKind      EntityKind = "blockInstance"
	ImageKind              EntityKind = "image"
	TextKind               EntityKind = "text"
	FillRegionKind         EntityKind = "fillRegion"
	ProjectedPointKind     EntityKind = "projectedPoint"
	ProjectedCurveKind     EntityKind = "projectedCurve"
	HelicalKind            EntityKind = "helical"
)

// Kind identifies each 2D entity for codec dispatch (and, over the API, for
// enumeration). Each concrete type declares its own — like definingPoints in
// drag.go — so no consumer needs a type switch.
func (l *Line) Kind() EntityKind           { return LineKind }
func (c *Circle) Kind() EntityKind         { return CircleKind }
func (a *Arc) Kind() EntityKind            { return ArcKind }
func (e *Ellipse) Kind() EntityKind        { return EllipseKind }
func (e *EllipticalArc) Kind() EntityKind  { return EllipticalArcKind }
func (s *Spline) Kind() EntityKind         { return SplineKind }
func (f *FixedSpline) Kind() EntityKind    { return FixedSplineKind }
func (o *OffsetSpline) Kind() EntityKind   { return OffsetSplineKind }
func (e *EquationCurve) Kind() EntityKind  { return EquationCurveKind }
func (b *BlockInstance) Kind() EntityKind  { return BlockInstanceKind }
func (i *SketchImage) Kind() EntityKind    { return ImageKind }
func (t *TextBox) Kind() EntityKind        { return TextKind }
func (f *FillRegion) Kind() EntityKind     { return FillRegionKind }
func (p *ProjectedPoint) Kind() EntityKind { return ProjectedPointKind }
func (c *ProjectedCurve) Kind() EntityKind { return ProjectedCurveKind }

// Kind identifies each 3D entity. A Spline3D's kind depends on its fit flag —
// the two spellings restore through different factory arguments, exactly as
// the retired serialize switch encoded them.
func (l *Line3D) Kind() EntityKind          { return LineKind }
func (c *Circle3D) Kind() EntityKind        { return CircleKind }
func (a *Arc3D) Kind() EntityKind           { return ArcKind }
func (e *Ellipse3D) Kind() EntityKind       { return EllipseKind }
func (e *EllipticalArc3D) Kind() EntityKind { return EllipticalArcKind }
func (f *FixedSpline3D) Kind() EntityKind   { return FixedSplineKind }
func (e *EquationCurve3D) Kind() EntityKind { return EquationCurveKind }
func (h *HelicalCurve3D) Kind() EntityKind  { return HelicalKind }

func (s *Spline3D) Kind() EntityKind {
	if s.fit {
		return SplineKind
	}
	return ControlPointSplineKind
}

// kindedEntity is the dispatch seam the codec registries key on. It stays
// unexported until the full ShapedEntity capability lands (#1624 step 2).
type kindedEntity interface{ Kind() EntityKind }
