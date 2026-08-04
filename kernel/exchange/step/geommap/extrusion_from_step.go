// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"fmt"

	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// profileConic is a SURFACE_OF_LINEAR_EXTRUSION's swept conic reduced to what the elliptical-cylinder
// elementarisation needs: the profile plane's center and orthonormal major/normal directions plus the
// two semi-axes (a==b for a circle).
type profileConic struct {
	center   math.Point3
	majorDir math.Vector3
	normal   math.Vector3
	semiA    float64
	semiB    float64
}

// linearExtrusionFromStep maps SURFACE_OF_LINEAR_EXTRUSION(name, swept_curve, extrusion_axis) to a
// right elliptical cylinder when the profile is a conic (ELLIPSE or CIRCLE). An oblique linear
// extrusion of a conic IS a right elliptical cylinder whose perpendicular cross-section is the
// profile's oblique projection along the sweep direction; the two projected semi-axes are conjugate
// semi-diameters of that section (geometry-math consult 2026-07-13, Rytz/2×2-eigen). A non-conic
// profile or a grazing/degenerate section returns ErrUnsupportedSurface so the face is skipped rather
// than mis-built — the STEP importer's honest fallback (issue: F6/T6/T7/U3/U4 open-shell imports).
func linearExtrusionFromStep(g *part21.EntityGraph, ent *part21.RawEntity, id int, scale float64) (geom.Surface, error) {
	pc, err := extrusionProfileConic(g, ent, id, scale)
	if err != nil {
		return nil, err
	}
	d, err := extrusionAxis(g, ent)
	if err != nil {
		return nil, err
	}
	u1, u2 := conjugateDiameters(pc, d)
	cyl, err := geom.NewEllipticalCylinderFromConjugate(pc.center, d, u1, u2)
	if err != nil {
		// A section grazing the sweep direction collapses to a line; skip honestly (a fatal error
		// here would abort the whole import, unlike ErrUnsupportedSurface which face.go catches).
		return nil, ErrUnsupportedSurface{Keyword: ent.Keyword, ID: id}
	}
	return cyl, nil
}

// extrusionProfileConic reads the swept curve (parameter 1) and, for a conic profile, returns its
// plane/axes/semi-axes. Any other profile kind is out of scope → ErrUnsupportedSurface.
func extrusionProfileConic(g *part21.EntityGraph, ent *part21.RawEntity, id int, scale float64) (profileConic, error) {
	ref, err := refParam(ent.Params, 1)
	if err != nil {
		return profileConic{}, fmt.Errorf("geommap: SURFACE_OF_LINEAR_EXTRUSION swept_curve: %w", err)
	}
	c, err := Curve(g, ref, scale)
	if err != nil {
		return profileConic{}, err
	}
	switch c.Kind {
	case CurveEllipse:
		e := c.Ellipse
		return profileConic{center: e.Center, majorDir: e.RefDir, normal: e.Normal, semiA: e.Major, semiB: e.Minor}, nil
	case CurveCircle:
		ci := c.Circle
		return profileConic{center: ci.Center, majorDir: ci.RefDir, normal: ci.Normal, semiA: ci.Radius, semiB: ci.Radius}, nil
	default:
		return profileConic{}, ErrUnsupportedSurface{Keyword: ent.Keyword, ID: id}
	}
}

// LinearExtrusionBSpline reports whether surface #id is a SURFACE_OF_LINEAR_EXTRUSION whose swept
// profile is a B-spline curve, returning the profile and unit sweep direction when so. It exists so
// the topology layer can build the extrusion as a bounded B-spline patch (it needs the backing face's
// extent, which geommap does not see) — see NewExtrudedBSplineSurface. ok is false for a conic profile,
// so the exact elliptical-cylinder path (linearExtrusionFromStep) is untouched: this only ADDS coverage
// for the B-spline profile the importer used to skip, dropping the swept side face and leaving an open
// shell (corpus resurvey 2026-07-24 §4: G3–H1 base bodies).
func LinearExtrusionBSpline(g *part21.EntityGraph, id int, scale float64) (geom.BSplineCurve, math.Vector3, bool, error) {
	ent, err := g.Lookup(id)
	if err != nil {
		return geom.BSplineCurve{}, math.Vector3{}, false, err
	}
	if ent.Keyword != "SURFACE_OF_LINEAR_EXTRUSION" {
		return geom.BSplineCurve{}, math.Vector3{}, false, nil
	}
	ref, err := refParam(ent.Params, 1)
	if err != nil {
		return geom.BSplineCurve{}, math.Vector3{}, false, err
	}
	c, err := Curve(g, ref, scale)
	if err != nil || c.Kind != CurveBSpline {
		return geom.BSplineCurve{}, math.Vector3{}, false, err
	}
	dir, err := extrusionAxis(g, ent)
	return c.BSpline, dir, err == nil, err
}

// NewExtrudedBSplineSurface represents a linear extrusion of a NURBS profile C(u) along the unit
// direction d over the sweep range v∈[lo,hi] as the EXACT v-degree-1 tensor NURBS surface
// S(u,v)=C(u)+v·d (Piegl & Tiller, "The NURBS Book" §8.3, extruded-surface construction: two control
// rows, the profile and the profile translated by the sweep, weights repeated). The STEP surface is
// infinite; the caller derives [lo,hi] from the backing face's extent along d so the bounded patch
// covers the trimmed face (topomap.extrusionSweepRange). lo<hi is required.
func NewExtrudedBSplineSurface(profile geom.BSplineCurve, d math.Vector3, lo, hi float64) (geom.Surface, error) {
	// NaN-REJECTING form, deliberate: every comparison with NaN is false, so `!(a > b)` fires on a
	// NaN operand while `a <= b` stays silent and lets it through. Do not "simplify" (sonar go:S1940).
	if !(hi > lo) {
		return nil, fmt.Errorf("geommap: extrusion sweep range [%g,%g] is not increasing", lo, hi)
	}
	n := len(profile.Ctrl)
	ctrl := make([][]math.Point3, n)
	weights := make([][]float64, n)
	for i, p := range profile.Ctrl {
		ctrl[i] = []math.Point3{p.TranslateBy(d.Scale(lo)), p.TranslateBy(d.Scale(hi))}
		weights[i] = []float64{profileWeight(profile, i), profileWeight(profile, i)}
	}
	return geom.NewBSplineSurface(profile.Degree, 1, ctrl, weights, profile.Knots, []float64{lo, lo, hi, hi})
}

// profileWeight returns control point i's weight, defaulting to 1 for a non-rational profile whose
// weight slice was left empty.
func profileWeight(profile geom.BSplineCurve, i int) float64 {
	if i < len(profile.Weights) {
		return profile.Weights[i]
	}
	return 1
}

// extrusionAxis reads the extrusion VECTOR (parameter 2) and returns its orientation as a unit
// vector — the sweep direction. The VECTOR's magnitude is irrelevant to the infinite cylinder's axis.
func extrusionAxis(g *part21.EntityGraph, ent *part21.RawEntity) (math.Vector3, error) {
	ref, err := refParam(ent.Params, 2)
	if err != nil {
		return math.Vector3{}, fmt.Errorf("geommap: SURFACE_OF_LINEAR_EXTRUSION axis: %w", err)
	}
	vec, err := g.Lookup(ref)
	if err != nil {
		return math.Vector3{}, err
	}
	if vec.Keyword != "VECTOR" {
		return math.Vector3{}, fmt.Errorf("geommap: extrusion axis #%d is %s, want VECTOR", ref, vec.Keyword)
	}
	dirRef, err := refParam(vec.Params, 1)
	if err != nil {
		return math.Vector3{}, fmt.Errorf("geommap: VECTOR orientation: %w", err)
	}
	dir, err := Direction(g, dirRef)
	if err != nil {
		return math.Vector3{}, err
	}
	return unitVec(dir), nil
}

// conjugateDiameters returns the two conjugate semi-diameters of the profile conic's oblique
// projection along the unit sweep direction d: the major axis (length semiA) and the minor axis
// (length semiB, along normal×major), each rejected perpendicular to d.
func conjugateDiameters(pc profileConic, d math.Vector3) (u1, u2 math.Vector3) {
	m := unitVec(pc.majorDir)
	n := unitVec(pc.normal)
	w := n.Cross(m) // minor-axis direction (unit: n ⟂ m by the placement's Gram-Schmidt)
	u1 = rejectFrom(m, d).Scale(pc.semiA)
	u2 = rejectFrom(w, d).Scale(pc.semiB)
	return u1, u2
}

// rejectFrom removes the component of v along the unit direction d (orthogonal projection onto the
// plane perpendicular to d).
func rejectFrom(v, d math.Vector3) math.Vector3 { return v.Sub(d.Scale(v.Dot(d))) }

// unitVec normalizes a nonzero vector (callers pass DIRECTION/placement vectors, guaranteed nonzero).
func unitVec(v math.Vector3) math.Vector3 { return v.Scale(1 / v.Length()) }
