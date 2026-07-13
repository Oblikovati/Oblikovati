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
