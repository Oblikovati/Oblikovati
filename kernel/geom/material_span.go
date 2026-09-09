// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// How thick the MATERIAL a boundary surface bounds is, measured from the surface itself.
//
// A body's thinness used to be read off its axis-aligned bounding box, which describes the
// COORDINATE FRAME as much as the body: measured on the RING corpus pair, the same 1e-10 drill is
// 2e-10 thick along Z and 3.4043798713069418 thick after a 37 degree turn about (1,1,0), so the
// boolean's size classification refused it in one orientation and built it in the other (#3524).
// Everything here answers from the surfaces' own geometry, so the answer turns with the body.
//
// Each function is one-sided in the same way SurfacesApart is: a value means PROVEN, and ok=false
// means "no closed form here", never "the material is thick".

// BoundarySurface is a surface carrying the sense that makes its normal point OUT of the material —
// a topological face reduced to what a question about material needs. Callers hand one over instead
// of switching on a geometry kind themselves; the kernel ground rules keep those switches here.
type BoundarySurface struct {
	Surface Surface
	// Reversed marks a face whose outward (material) side is opposite its surface normal, the
	// sense topo.Face.Reversed reports.
	Reversed bool
}

// MaterialSlab is a PLANAR boundary expressed in the canonical frame of its own direction: the two
// opposite faces of one slab report the SAME Dir, so a caller can group planar faces by Dir in one
// sort instead of testing each face against every other.
//
// A group that holds both material sides brackets material along Dir. How WIDE that material is, is
// the caller's question and not this type's: the two nearest opposed faces are a LOCAL gap, which on
// a non-convex body can be the space between two unrelated regions (kernel/ops/boolean measures the
// body's whole extent along Dir instead — see slabGroupWidth there).
type MaterialSlab struct {
	// Dir is the plane normal with a canonical sign, so a normal and its negation agree.
	Dir math.UnitVector3
	// Offset is the plane's signed distance from the world origin along Dir.
	Offset float64
	// MaterialAbove reports that the material lies on the +Dir side of Offset (the outward normal
	// points along −Dir).
	MaterialAbove bool
}

// AsMaterialSlab reports a planar boundary surface in canonical slab form; ok=false for every other
// surface kind.
//
// Example:
//
//	s, ok := geom.AsMaterialSlab(geom.BoundarySurface{Surface: f.Geometry(), Reversed: f.Reversed()})
func AsMaterialSlab(b BoundarySurface) (MaterialSlab, bool) {
	p, ok := b.Surface.(Plane)
	if !ok {
		return MaterialSlab{}, false
	}
	dir, flipped, err := canonicalDirection(outwardNormal(p.Normal(), b.Reversed))
	if err != nil {
		return MaterialSlab{}, false
	}
	offset := float64(p.Origin.AsVector().Dot(dir.AsVector()))
	return MaterialSlab{Dir: dir, Offset: offset, MaterialAbove: flipped}, true
}

// SameDirection reports whether two slabs lie on parallel planes — the test that collects the faces
// of one slab into a group after a sort by Dir.
func (s MaterialSlab) SameDirection(o MaterialSlab) bool {
	return parallelDirs(s.Dir.AsVector(), o.Dir.AsVector())
}

// EnclosedSpan is the thickness of the material a CLOSED boundary surface wraps: a full cylinder or
// sphere of radius r encloses a rod or ball 2r across, a torus its tube 2·MinorRadius.
//
// The caller must have established that the face trims the WHOLE wrap. A fillet's 30 degree strip
// lies on a cylinder of small radius without the body being anywhere near that thin.
//
// ok=false for a surface that encloses nothing measurable — a plane, a b-spline, and a cone, whose
// enclosed material runs to a point at the apex so its thickness is a property of the trim rather
// than of the surface — and for one whose outward normal points INTO the wrap, where what is
// enclosed is a void (a bore) and not material at all.
func EnclosedSpan(b BoundarySurface) (float64, bool) {
	if b.Reversed {
		return 0, false
	}
	switch s := b.Surface.(type) {
	case Cylinder:
		return 2 * s.Radius, true
	case Sphere:
		return 2 * s.Radius, true
	case Torus:
		return 2 * s.MinorRadius, true
	}
	return 0, false
}

// OpposedSpan is the thickness of the material two boundary surfaces of the SAME class hold between
// them, for the pairs whose separation is constant in closed form: coaxial cylinders (a tube wall)
// and concentric spheres (a shell). ok=false for every other pair, and for a pair whose outward
// normals do not point AWAY from one another — what lies between those is a void, not material.
//
// Cones are deliberately absent although SurfacesApart proves their separation: which side of a
// coaxial equal-angle cone pair carries the material is not decided by the apex offset alone.
//
// Example:
//
//	if wall, ok := geom.OpposedSpan(outer, bore); ok { /* the tube wall is `wall` thick */ }
func OpposedSpan(a, b BoundarySurface) (float64, bool) {
	switch x := a.Surface.(type) {
	case Cylinder:
		y, ok := b.Surface.(Cylinder)
		return coaxialCylinderSpan(x, a.Reversed, y, b.Reversed, ok)
	case Sphere:
		y, ok := b.Surface.(Sphere)
		return concentricSphereSpan(x, a.Reversed, y, b.Reversed, ok)
	}
	return 0, false
}

// coaxialCylinderSpan is a tube wall: two cylinders on ONE axis line, the wider one facing out and
// the narrower facing in, are |Δr| of material apart everywhere.
func coaxialCylinderSpan(a Cylinder, aRev bool, b Cylinder, bRev, isCylinder bool) (float64, bool) {
	if !isCylinder || !parallelDirs(a.AxisDir.AsVector(), b.AxisDir.AsVector()) {
		return 0, false
	}
	if !onAxisLine(a.Origin, a.AxisDir, b.Origin) {
		return 0, false
	}
	return nestedShellSpan(a.Radius, aRev, b.Radius, bRev)
}

// concentricSphereSpan is a spherical shell: two spheres about ONE centre, the wider facing out and
// the narrower facing in.
func concentricSphereSpan(a Sphere, aRev bool, b Sphere, bRev, isSphere bool) (float64, bool) {
	if !isSphere {
		return 0, false
	}
	d := float64(a.Center.VectorTo(b.Center).Length())
	if d > ResolutionForSize(stdmath.Max(a.Radius, b.Radius)).Weld() {
		return 0, false
	}
	return nestedShellSpan(a.Radius, aRev, b.Radius, bRev)
}

// nestedShellSpan is the wall of two nested radial surfaces: |Δr|, but only when the outer one holds
// the material inside it (facing out) and the inner one holds it outside (facing in). Equal radii are
// the same surface twice and bound nothing.
func nestedShellSpan(ra float64, aRev bool, rb float64, bRev bool) (float64, bool) {
	outerReversed, innerReversed := aRev, bRev
	if rb > ra {
		outerReversed, innerReversed = bRev, aRev
	}
	if ra == rb || outerReversed || !innerReversed {
		return 0, false
	}
	return stdmath.Abs(ra - rb), true
}

// outwardNormal turns a surface normal and a face sense into the direction pointing OUT of material.
func outwardNormal(n math.Vector3, reversed bool) math.Vector3 {
	if reversed {
		return n.Scale(-1)
	}
	return n
}

// canonicalDirection gives a direction and its negation the SAME representative — the first non-zero
// component is made positive — and reports whether it had to flip. Negation is exact in IEEE
// arithmetic, and normalisation divides by a length that is identical for v and −v, so two exactly
// opposite face normals canonicalise to the identical unit vector rather than to near neighbours.
func canonicalDirection(v math.Vector3) (math.UnitVector3, bool, error) {
	u, err := math.UnitVector3FromVector(v)
	if err != nil {
		return math.UnitVector3{}, false, err
	}
	if !leadsNegative(u) {
		return u, false, nil
	}
	return u.Negate(), true, nil
}

// leadsNegative reports whether a direction's first non-zero component is negative — the total order
// that picks one of the two senses, with no tolerance so the choice is byte-identical across runs.
func leadsNegative(u math.UnitVector3) bool {
	if u.X() != 0 {
		return u.X() < 0
	}
	if u.Y() != 0 {
		return u.Y() < 0
	}
	return u.Z() < 0
}
