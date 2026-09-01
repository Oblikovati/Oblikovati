// SPDX-License-Identifier: GPL-2.0-only

package geom

// Analytic special-casing of surfaces and curves is legitimate — OCCT does exactly this
// (GeomAdaptor_Surface::GetType / GeomAbs_SurfaceType classify a closed analytic set through
// one enum discriminator that every consumer switches on). The problem the audit (I6) names
// is that our special-casing was UNCHECKABLE: ~77 `case geom.X:` type switches spread across
// consumers, each deciding independently which kinds it handles and what its default does —
// often a silent zero value (the #1403 class). Kind() is that single discriminator: every
// surface/curve names its kind, so switches become enumerable data (a coverage test can probe
// one instance of each kind), pure translators become tables keyed on the kind, and a new
// kind that a consumer forgot fails that consumer's coverage test instead of falling into a
// silent default.

// SurfaceKind classifies the analytic and freeform surfaces. Add a value here AND its Kind()
// method AND a probe in the coverage test (kind_coverage_test.go) when a new surface lands —
// the coverage test enforces all three.
type SurfaceKind uint8

const (
	SurfacePlane SurfaceKind = iota
	SurfaceCylinder
	SurfaceSphere
	SurfaceCone
	SurfaceTorus
	SurfaceBSpline
	SurfaceEllipticalCylinder
	SurfaceEllipticalCone
	SurfaceOffset
	SurfaceThreadedCylinder
	// surfaceKindCount is the sentinel one past the last real kind; range over
	// [0, surfaceKindCount) to enumerate every SurfaceKind.
	surfaceKindCount
)

// String names the kind for error messages (CLAUDE.md: exception messages name the value).
func (k SurfaceKind) String() string {
	if int(k) >= len(surfaceKindNames) {
		return "SurfaceKind(?)"
	}
	return surfaceKindNames[k]
}

var surfaceKindNames = [...]string{
	SurfacePlane:              "Plane",
	SurfaceCylinder:           "Cylinder",
	SurfaceSphere:             "Sphere",
	SurfaceCone:               "Cone",
	SurfaceTorus:              "Torus",
	SurfaceBSpline:            "BSplineSurface",
	SurfaceEllipticalCylinder: "EllipticalCylinder",
	SurfaceEllipticalCone:     "EllipticalCone",
	SurfaceOffset:             "OffsetSurface",
	SurfaceThreadedCylinder:   "ThreadedCylinder",
}

// SurfaceKinds returns every SurfaceKind in declaration order — the enumerator external
// coverage tests (e.g. step/geommap) range over to prove they handle or explicitly declare
// each kind, so a new kind fails a stale consumer at CI.
func SurfaceKinds() []SurfaceKind {
	all := make([]SurfaceKind, 0, surfaceKindCount)
	for k := range surfaceKindCount {
		all = append(all, k)
	}
	return all
}

// KindedSurface is a Surface that names its analytic kind. Every surface in the kernel
// satisfies it (asserted below); consumers that special-case may switch on Kind() instead of
// a type assertion, and pure translators key a table on it.
type KindedSurface interface {
	Surface
	Kind() SurfaceKind
}

func (p Plane) Kind() SurfaceKind              { return SurfacePlane }
func (c Cylinder) Kind() SurfaceKind           { return SurfaceCylinder }
func (s Sphere) Kind() SurfaceKind             { return SurfaceSphere }
func (c Cone) Kind() SurfaceKind               { return SurfaceCone }
func (t Torus) Kind() SurfaceKind              { return SurfaceTorus }
func (s BSplineSurface) Kind() SurfaceKind     { return SurfaceBSpline }
func (c EllipticalCylinder) Kind() SurfaceKind { return SurfaceEllipticalCylinder }
func (c EllipticalCone) Kind() SurfaceKind     { return SurfaceEllipticalCone }
func (o OffsetSurface) Kind() SurfaceKind      { return SurfaceOffset }
func (t ThreadedCylinder) Kind() SurfaceKind   { return SurfaceThreadedCylinder }

var (
	_ KindedSurface = Plane{}
	_ KindedSurface = Cylinder{}
	_ KindedSurface = Sphere{}
	_ KindedSurface = Cone{}
	_ KindedSurface = Torus{}
	_ KindedSurface = BSplineSurface{}
	_ KindedSurface = EllipticalCylinder{}
	_ KindedSurface = EllipticalCone{}
	_ KindedSurface = OffsetSurface{}
	_ KindedSurface = ThreadedCylinder{}
)

// CurveKind classifies the analytic and freeform 3D curves. Add a value here AND its Kind()
// method AND a probe in the coverage test when a new curve lands.
type CurveKind uint8

const (
	CurveLine CurveKind = iota
	CurveLineSegment
	CurvePolyline
	CurveCircle
	CurveArc
	CurveEllipse
	CurveEllipticalArc
	CurveHyperbolicArc
	CurveParabola
	CurveBSpline
	CurveHelix
	CurveVariableHelix
	CurveSpiric
	CurveTorusCyl
	CurveRuledQuadric
	// curveKindCount is the sentinel one past the last real kind.
	curveKindCount
)

// String names the kind for error messages.
func (k CurveKind) String() string {
	if int(k) >= len(curveKindNames) {
		return "CurveKind(?)"
	}
	return curveKindNames[k]
}

var curveKindNames = [...]string{
	CurveLine:          "Line",
	CurveLineSegment:   "LineSegment",
	CurvePolyline:      "Polyline",
	CurveCircle:        "Circle",
	CurveArc:           "Arc3d",
	CurveEllipse:       "EllipseFull",
	CurveEllipticalArc: "EllipticalArc",
	CurveHyperbolicArc: "HyperbolicArc",
	CurveParabola:      "Parabola",
	CurveBSpline:       "BSplineCurve",
	CurveHelix:         "Helix3d",
	CurveVariableHelix: "VariableHelix3d",
	CurveSpiric:        "SpiricArc",
	CurveTorusCyl:      "TorusCylinderArc",
	CurveRuledQuadric:  "RuledQuadricArc",
}

// CurveKinds returns every CurveKind in declaration order (see SurfaceKinds).
func CurveKinds() []CurveKind {
	all := make([]CurveKind, 0, curveKindCount)
	for k := range curveKindCount {
		all = append(all, k)
	}
	return all
}

// KindedCurve is a Curve3 that names its analytic kind.
type KindedCurve interface {
	Curve3
	Kind() CurveKind
}

func (l Line) Kind() CurveKind             { return CurveLine }
func (s LineSegment) Kind() CurveKind      { return CurveLineSegment }
func (p Polyline) Kind() CurveKind         { return CurvePolyline }
func (c Circle) Kind() CurveKind           { return CurveCircle }
func (a Arc3d) Kind() CurveKind            { return CurveArc }
func (e EllipseFull) Kind() CurveKind      { return CurveEllipse }
func (e EllipticalArc) Kind() CurveKind    { return CurveEllipticalArc }
func (h HyperbolicArc) Kind() CurveKind    { return CurveHyperbolicArc }
func (p Parabola) Kind() CurveKind         { return CurveParabola }
func (c BSplineCurve) Kind() CurveKind     { return CurveBSpline }
func (h Helix3d) Kind() CurveKind          { return CurveHelix }
func (h VariableHelix3d) Kind() CurveKind  { return CurveVariableHelix }
func (s SpiricArc) Kind() CurveKind        { return CurveSpiric }
func (a TorusCylinderArc) Kind() CurveKind { return CurveTorusCyl }
func (a RuledQuadricArc) Kind() CurveKind  { return CurveRuledQuadric }

var (
	_ KindedCurve = Line{}
	_ KindedCurve = LineSegment{}
	_ KindedCurve = Polyline{}
	_ KindedCurve = Circle{}
	_ KindedCurve = Arc3d{}
	_ KindedCurve = EllipseFull{}
	_ KindedCurve = EllipticalArc{}
	_ KindedCurve = HyperbolicArc{}
	_ KindedCurve = Parabola{}
	_ KindedCurve = BSplineCurve{}
	_ KindedCurve = Helix3d{}
	_ KindedCurve = VariableHelix3d{}
	_ KindedCurve = SpiricArc{}
	_ KindedCurve = TorusCylinderArc{}
	_ KindedCurve = RuledQuadricArc{}
)
