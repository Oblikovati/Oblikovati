// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"sort"

	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Identical-body grouping (M07-F05, Oblikovati/Oblikovati#628): the reference
// TransientBRep.GetIdenticalBodies. Bodies compare by their rigid-motion
// INVARIANT signature — volume, surface area and the sorted principal moments
// about the centroid — which a translation or rotation cannot change; a
// reflection cannot either, so MatchReflection=false additionally compares the
// signed moment skew (the centroidal third moment), which flips under
// reflection. MatchTopology additionally requires equal face/edge/vertex
// counts. Signature equality is necessary, not sufficient: two genuinely
// different shapes with identical invariants (a measure-zero coincidence)
// would group together; exact congruence testing is the M25 oracle's domain.

// IdenticalBodiesOptions mirrors the reference NameValueMap options.
type IdenticalBodiesOptions struct {
	Tolerance       float64 // relative; 0 → 1e-6 (the reference default)
	MatchTopology   bool
	MatchReflection bool // default true in the reference: mirrored bodies match
}

// GroupIdenticalBodies partitions the bodies into groups of identical ones;
// each group lists indices into the input slice.
//
// Example: groups := ops.GroupIdenticalBodies(bodies, ops.IdenticalBodiesOptions{MatchReflection: true}, ops.DefaultQuality())
func GroupIdenticalBodies(bodies []*topo.Body, opt IdenticalBodiesOptions, q Quality) [][]int {
	tol := opt.Tolerance
	if tol <= 0 {
		tol = 1e-6
	}
	sigs := make([]bodySignature, len(bodies))
	for i, b := range bodies {
		sigs[i] = signatureOf(b, q)
	}
	parent := make([]int, len(bodies))
	for i := range parent {
		parent[i] = i
	}
	for i := range bodies {
		for j := i + 1; j < len(bodies); j++ {
			if sigs[i].matches(sigs[j], tol, opt) {
				union(parent, i, j)
			}
		}
	}
	return indexGroups(parent)
}

// bodySignature is the rigid-motion invariant fingerprint.
type bodySignature struct {
	volume, area           float64
	moments                [3]float64 // sorted principal second moments about the centroid
	skew                   float64    // signed third moment — flips under reflection
	faces, edges, vertices int
}

// signatureOf computes the fingerprint from the body's tessellation.
func signatureOf(b *topo.Body, q Quality) bodySignature {
	mesh, _ := TessellateBody(b, q)
	props := meshGeometryProperties(mesh)
	moments, skew := centroidalMoments(mesh, props)
	return bodySignature{
		volume: props.Volume, area: props.Area, moments: moments, skew: skew,
		faces: len(b.Faces()), edges: len(b.Edges()), vertices: len(b.Vertices()),
	}
}

// centroidalMoments integrates the surface second moments (eigenvalues,
// sorted) and a signed third moment about the centroid. Quadrature is EXACT
// per polynomial degree — edge midpoints for the quadratics (degree-2 exact),
// the Strang 4-point rule for the cubic skew — so the invariants do not
// depend on how a planar face happened to triangulate (a rotated copy's
// earcut may pick different diagonals).
func centroidalMoments(mesh *Mesh, props GeometryProperties) ([3]float64, float64) {
	var ixx, iyy, izz, ixy, ixz, iyz, skew float64
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		a := mesh.Positions[mesh.Indices[t]]
		b := mesh.Positions[mesh.Indices[t+1]]
		c := mesh.Positions[mesh.Indices[t+2]]
		area := float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length()) / 2
		for _, qp := range midpointRule(a, b, c, props.Centroid) {
			w := area / 3
			ixx += w * (qp[1]*qp[1] + qp[2]*qp[2])
			iyy += w * (qp[0]*qp[0] + qp[2]*qp[2])
			izz += w * (qp[0]*qp[0] + qp[1]*qp[1])
			ixy += w * qp[0] * qp[1]
			ixz += w * qp[0] * qp[2]
			iyz += w * qp[1] * qp[2]
		}
		skew += area * triangleSkew(a, b, c, props.Centroid)
	}
	eig := symmetricEigenvalues3(ixx, iyy, izz, -ixy, -ixz, -iyz)
	sort.Float64s(eig[:])
	return eig, skew
}

// midpointRule returns the three edge midpoints in centroid-relative
// coordinates — the degree-2-exact triangle quadrature points (weight ⅓ each).
func midpointRule(a, b, c math.Point3, centroid math.Point3) [3][3]float64 {
	rel := func(p, q math.Point3) [3]float64 {
		return [3]float64{
			float64(p.X+q.X)/2 - float64(centroid.X),
			float64(p.Y+q.Y)/2 - float64(centroid.Y),
			float64(p.Z+q.Z)/2 - float64(centroid.Z),
		}
	}
	return [3][3]float64{rel(a, b), rel(b, c), rel(c, a)}
}

// triangleSkew integrates xyz (centroid-relative) over the triangle with the
// Strang degree-3 rule: centroid weight −27/48, the three (3/5, 1/5, 1/5)
// points 25/48 each.
func triangleSkew(a, b, c, centroid math.Point3) float64 {
	at := func(wa, wb, wc float64) float64 {
		x := wa*float64(a.X) + wb*float64(b.X) + wc*float64(c.X) - float64(centroid.X)
		y := wa*float64(a.Y) + wb*float64(b.Y) + wc*float64(c.Y) - float64(centroid.Y)
		z := wa*float64(a.Z) + wb*float64(b.Z) + wc*float64(c.Z) - float64(centroid.Z)
		return x * y * z
	}
	const third = 1.0 / 3
	sum := -27.0 / 48 * at(third, third, third)
	sum += 25.0 / 48 * (at(0.6, 0.2, 0.2) + at(0.2, 0.6, 0.2) + at(0.2, 0.2, 0.6))
	return sum
}

// matches compares two signatures under the options.
func (s bodySignature) matches(o bodySignature, tol float64, opt IdenticalBodiesOptions) bool {
	if opt.MatchTopology && (s.faces != o.faces || s.edges != o.edges || s.vertices != o.vertices) {
		return false
	}
	if !relClose(s.volume, o.volume, tol) || !relClose(s.area, o.area, tol) {
		return false
	}
	scale := s.moments[2]
	for i := range s.moments {
		if stdmath.Abs(s.moments[i]-o.moments[i]) > tol*stdmath.Max(scale, 1e-12) {
			return false
		}
	}
	if !opt.MatchReflection && !skewClose(s.skew, o.skew, tol, scale) {
		return false
	}
	return true
}

func relClose(a, b, tol float64) bool {
	scale := stdmath.Max(stdmath.Abs(a), stdmath.Abs(b))
	if scale == 0 {
		return true
	}
	return stdmath.Abs(a-b)/scale <= tol
}

// skewClose compares the signed skew (sign included — a mirror flips it).
func skewClose(a, b, tol, scale float64) bool {
	return stdmath.Abs(a-b) <= tol*stdmath.Max(scale, 1e-12)
}

// symmetricEigenvalues3 solves the symmetric 3×3 eigenvalue problem
// (analytic trigonometric method; Smith 1961).
func symmetricEigenvalues3(a11, a22, a33, a12, a13, a23 float64) [3]float64 {
	p1 := a12*a12 + a13*a13 + a23*a23
	q := (a11 + a22 + a33) / 3
	if p1 == 0 {
		return [3]float64{a11, a22, a33}
	}
	p2 := (a11-q)*(a11-q) + (a22-q)*(a22-q) + (a33-q)*(a33-q) + 2*p1
	p := stdmath.Sqrt(p2 / 6)
	det := (a11-q)*((a22-q)*(a33-q)-a23*a23) -
		a12*(a12*(a33-q)-a23*a13) +
		a13*(a12*a23-(a22-q)*a13)
	r := det / (2 * p * p * p)
	r = stdmath.Max(-1, stdmath.Min(1, r))
	phi := stdmath.Acos(r) / 3
	e1 := q + 2*p*stdmath.Cos(phi)
	e3 := q + 2*p*stdmath.Cos(phi+2*stdmath.Pi/3)
	return [3]float64{e1, 3*q - e1 - e3, e3}
}

// indexGroups buckets indices by union-find root, ordered by first member.
func indexGroups(parent []int) [][]int {
	order := []int{}
	members := map[int][]int{}
	for i := range parent {
		r := find(parent, i)
		if _, ok := members[r]; !ok {
			order = append(order, r)
		}
		members[r] = append(members[r], i)
	}
	out := make([][]int, len(order))
	for i, r := range order {
		out[i] = members[r]
	}
	return out
}
