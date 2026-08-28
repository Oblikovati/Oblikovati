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
	ControlPointSplineKind EntityKind = "controlPointSpline" // a non-fit spline, 2D or 3D
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

// The enumeration-only kinds: entities that are never persisted as recipe rows
// (points persist in the recipe's Points table, handles inside their spline,
// the surface-derived 3D curves rebind on recompute) but still name themselves
// over the API. Spellings match the wire enums by construction.
const (
	PointKind                 EntityKind = "point"
	SplineHandleKind          EntityKind = "splineHandle"
	IncludedPointKind         EntityKind = "includedPoint"
	IncludedCurveKind         EntityKind = "includedCurve"
	IntersectionCurveKind     EntityKind = "intersection"
	SilhouetteCurveKind       EntityKind = "silhouette"
	ProjectToSurfaceCurveKind EntityKind = "projectToSurface"
	OnFaceCurveKind           EntityKind = "onFace"
	OffsetCurveKind           EntityKind = "offset"
)

// Kind identifies each 2D entity for codec dispatch (and, over the API, for
// enumeration). Each concrete type declares its own — like definingPoints in
// drag.go — so no consumer needs a type switch.
func (p *Point) Kind() EntityKind          { return PointKind }
func (l *Line) Kind() EntityKind           { return LineKind }
func (c *Circle) Kind() EntityKind         { return CircleKind }
func (a *Arc) Kind() EntityKind            { return ArcKind }
func (e *Ellipse) Kind() EntityKind        { return EllipseKind }
func (e *EllipticalArc) Kind() EntityKind  { return EllipticalArcKind }
func (h *SplineHandle) Kind() EntityKind   { return SplineHandleKind }
func (f *FixedSpline) Kind() EntityKind    { return FixedSplineKind }
func (o *OffsetSpline) Kind() EntityKind   { return OffsetSplineKind }
func (e *EquationCurve) Kind() EntityKind  { return EquationCurveKind }
func (b *BlockInstance) Kind() EntityKind  { return BlockInstanceKind }
func (i *SketchImage) Kind() EntityKind    { return ImageKind }
func (t *TextBox) Kind() EntityKind        { return TextKind }
func (f *FillRegion) Kind() EntityKind     { return FillRegionKind }
func (p *ProjectedPoint) Kind() EntityKind { return ProjectedPointKind }

// A spline's kind depends on its fit flag in either dimension: interpolation
// splines are "spline", approximating ones "controlPointSpline" — the wire
// spelling (#150), now also the persisted one (the decoder reads the persisted
// Fit flag, so either spelling restores).
func (s *Spline) Kind() EntityKind   { return splineKindFor(s.fit) }
func (s *Spline3D) Kind() EntityKind { return splineKindFor(s.fit) }

func splineKindFor(fit bool) EntityKind {
	if fit {
		return SplineKind
	}
	return ControlPointSplineKind
}

// Kind identifies each 3D entity.
func (p *Point3D) Kind() EntityKind         { return PointKind }
func (l *Line3D) Kind() EntityKind          { return LineKind }
func (c *Circle3D) Kind() EntityKind        { return CircleKind }
func (a *Arc3D) Kind() EntityKind           { return ArcKind }
func (e *Ellipse3D) Kind() EntityKind       { return EllipseKind }
func (e *EllipticalArc3D) Kind() EntityKind { return EllipticalArcKind }
func (h *SplineHandle3D) Kind() EntityKind  { return SplineHandleKind }
func (f *FixedSpline3D) Kind() EntityKind   { return FixedSplineKind }
func (e *EquationCurve3D) Kind() EntityKind { return EquationCurveKind }
func (h *HelicalCurve3D) Kind() EntityKind  { return HelicalKind }

// Kind identifies the reference/derived 3D geometry (enumeration-only kinds).
func (p *IncludedPoint3D) Kind() EntityKind         { return IncludedPointKind }
func (c *IncludedCurve3D) Kind() EntityKind         { return IncludedCurveKind }
func (c *IntersectionCurve3D) Kind() EntityKind     { return IntersectionCurveKind }
func (c *SilhouetteCurve3D) Kind() EntityKind       { return SilhouetteCurveKind }
func (c *ProjectToSurfaceCurve3D) Kind() EntityKind { return ProjectToSurfaceCurveKind }
func (c *OnFaceCurve3D) Kind() EntityKind           { return OnFaceCurveKind }
func (c *OffsetCurve3) Kind() EntityKind            { return OffsetCurveKind }

// kindedEntity is the dispatch seam the codec registries key on. It stays
// unexported until the full ShapedEntity capability lands (#1624 step 2).
type kindedEntity interface{ Kind() EntityKind }
