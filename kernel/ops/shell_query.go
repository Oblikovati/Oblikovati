// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Shell and body point queries (M07-F06/F07, Oblikovati/Oblikovati#629/#630):
// signed shell volume (the void/solid classifier), and inside/on/outside
// containment for shells and whole bodies.

// PointContainment is the inside/on/outside verdict of a point query. The
// api/types Containment enum is the wire spelling; this is the kernel value.
type PointContainment uint8

const (
	ContainOutside PointContainment = iota
	ContainInside
	ContainOn
)

// String returns a stable name for diagnostics.
func (c PointContainment) String() string {
	switch c {
	case ContainInside:
		return "inside"
	case ContainOn:
		return "on"
	default:
		return "outside"
	}
}

// shellMesh merges the shell's face tessellations (each face oriented to its
// material-outward shading normals, Reversed honored).
func shellMesh(s *topo.Shell, q Quality) *Mesh {
	mesh := &Mesh{}
	for _, f := range s.Faces() {
		mergeMesh(mesh, TessellateFace(f, q))
	}
	return mesh
}

// ShellSignedVolume returns the divergence-theorem volume of the region the
// shell bounds, signed by its material orientation: positive for an outer
// (material-enclosing) shell, NEGATIVE for an inner void shell, whose
// material-outward face normals point into the cavity.
//
// Example: if ops.ShellSignedVolume(sh, ops.DefaultQuality()) < 0 { /* cavity skin */ }
func ShellSignedVolume(s *topo.Shell, q Quality) float64 {
	mesh := shellMesh(s, q)
	vol := 0.0
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		ia, ib, ic := mesh.Indices[t], mesh.Indices[t+1], mesh.Indices[t+2]
		a, b, c := mesh.Positions[ia], mesh.Positions[ib], mesh.Positions[ic]
		if outwardRef(mesh, ia, ib, ic).Dot(a.VectorTo(b).Cross(a.VectorTo(c))) < 0 {
			b, c = c, b // align geometric winding to the outward shading normal
		}
		vol += float64(a.AsVector().Dot(b.AsVector().Cross(c.AsVector()))) / 6
	}
	return vol
}

// ShellIsVoid reports whether the shell is an inner void (cavity) skin.
func ShellIsVoid(s *topo.Shell, q Quality) bool {
	return s.IsClosed() && ShellSignedVolume(s, q) < 0
}

// ShellContainment classifies p against the region the shell bounds — ON within onTol of a trimmed
// face, else INSIDE by odd ray crossings — from the analytic B-rep, reading no tessellation (the
// Quality argument is now vestigial). M48/C3 #3428.
func ShellContainment(s *topo.Shell, p math.Point3, _ Quality, onTol float64) PointContainment {
	return fromBrepContainment(brep.ClassifyShellPoint(s, p, onTol))
}

// BodyContainment classifies p against the solid body — ON within onTol of any trimmed face, INSIDE
// by odd crossings over all faces (a point inside a cavity is outside the material) — from the
// analytic B-rep, reading no tessellation (the Quality argument is now vestigial). M48/C3 #3429.
func BodyContainment(b *topo.Body, p math.Point3, _ Quality, onTol float64) PointContainment {
	return fromBrepContainment(brep.ClassifyPointTol(b, p, onTol))
}

// fromBrepContainment maps brep's classifier tri-state to the ops/api PointContainment.
func fromBrepContainment(c brep.Containment) PointContainment {
	switch c {
	case brep.Inside:
		return ContainInside
	case brep.OnSurface:
		return ContainOn
	default:
		return ContainOutside
	}
}

// meshContainment is the shared mesh-level classifier: ON when p is within onTol of any facet, else
// INSIDE/OUTSIDE by the generalized winding number (pointInMesh). The winding number replaces the
// former skewed parity ray (#1317) — robust to grazing edges/vertices — and still gives the right
// cavity verdict: an outer shell contributes +1 and a void shell −1, so a point in a cavity sums to
// 0 (outside the material), exactly as the old crossing-cancellation intended.
func meshContainment(mesh *Mesh, p math.Point3, onTol float64) PointContainment {
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		a := mesh.Positions[mesh.Indices[t]]
		b := mesh.Positions[mesh.Indices[t+1]]
		c := mesh.Positions[mesh.Indices[t+2]]
		if pointTriangleDistance(p, a, b, c) <= onTol {
			return ContainOn
		}
	}
	if pointInMesh(mesh, p) {
		return ContainInside
	}
	return ContainOutside
}

// pointTriangleDistance is the exact distance from p to triangle abc
// (Ericson, Real-Time Collision Detection §5.1.5 — region tests on the
// barycentric Voronoi regions).
func pointTriangleDistance(p, a, b, c math.Point3) float64 {
	closest := closestPointOnTriangle(p, a, b, c)
	return float64(p.DistanceTo(closest))
}

// triDots carries the six dot products and frame the Voronoi region tests share.
type triDots struct {
	a, b, c                math.Point3
	ab, ac                 math.Vector3
	d1, d2, d3, d4, d5, d6 float64
}

func newTriDots(p, a, b, c math.Point3) triDots {
	ab, ac := a.VectorTo(b), a.VectorTo(c)
	ap, bp, cp := a.VectorTo(p), b.VectorTo(p), c.VectorTo(p)
	return triDots{
		a: a, b: b, c: c, ab: ab, ac: ac,
		d1: float64(ab.Dot(ap)), d2: float64(ac.Dot(ap)),
		d3: float64(ab.Dot(bp)), d4: float64(ac.Dot(bp)),
		d5: float64(ab.Dot(cp)), d6: float64(ac.Dot(cp)),
	}
}

func closestPointOnTriangle(p, a, b, c math.Point3) math.Point3 {
	d := newTriDots(p, a, b, c)
	if v, hit := d.vertexRegion(); hit {
		return v
	}
	if v, hit := d.edgeRegion(); hit {
		return v
	}
	return d.faceRegion()
}

// vertexRegion returns the closest point when p projects beyond a corner.
func (d triDots) vertexRegion() (math.Point3, bool) {
	if d.d1 <= 0 && d.d2 <= 0 {
		return d.a, true
	}
	if d.d3 >= 0 && d.d4 <= d.d3 {
		return d.b, true
	}
	if d.d6 >= 0 && d.d5 <= d.d6 {
		return d.c, true
	}
	return math.Point3{}, false
}

// edgeRegion returns the closest point when p projects onto an edge band.
func (d triDots) edgeRegion() (math.Point3, bool) {
	if v, hit := d.abRegion(); hit {
		return v, true
	}
	if v, hit := d.acRegion(); hit {
		return v, true
	}
	return d.bcRegion()
}

func (d triDots) abRegion() (math.Point3, bool) {
	if vc := d.d1*d.d4 - d.d3*d.d2; vc <= 0 && d.d1 >= 0 && d.d3 <= 0 && d.d1-d.d3 > 0 {
		t := math.Scalar(d.d1 / (d.d1 - d.d3))
		return d.a.TranslateBy(d.ab.Scale(t)), true
	}
	return math.Point3{}, false
}

func (d triDots) acRegion() (math.Point3, bool) {
	if vb := d.d5*d.d2 - d.d1*d.d6; vb <= 0 && d.d2 >= 0 && d.d6 <= 0 && d.d2-d.d6 > 0 {
		t := math.Scalar(d.d2 / (d.d2 - d.d6))
		return d.a.TranslateBy(d.ac.Scale(t)), true
	}
	return math.Point3{}, false
}

func (d triDots) bcRegion() (math.Point3, bool) {
	if va := d.d3*d.d6 - d.d5*d.d4; va <= 0 && d.d4-d.d3 >= 0 && d.d5-d.d6 >= 0 && (d.d4-d.d3)+(d.d5-d.d6) > 0 {
		t := math.Scalar((d.d4 - d.d3) / ((d.d4 - d.d3) + (d.d5 - d.d6)))
		return d.b.TranslateBy(d.b.VectorTo(d.c).Scale(t)), true
	}
	return math.Point3{}, false
}

// faceRegion projects into the triangle's interior barycentrically.
func (d triDots) faceRegion() math.Point3 {
	va := d.d3*d.d6 - d.d5*d.d4
	vb := d.d5*d.d2 - d.d1*d.d6
	vc := d.d1*d.d4 - d.d3*d.d2
	denom := va + vb + vc
	if denom == 0 {
		return d.a // degenerate sliver triangle: any vertex is as close as it gets
	}
	v, w := math.Scalar(vb/denom), math.Scalar(vc/denom)
	return d.a.TranslateBy(d.ab.Scale(v).Add(d.ac.Scale(w)))
}
