// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	"sort"

	stdmath "math"

	dset "oblikovati.org/kernel/ops/internal/disjoint"
	"oblikovati.org/kernel/ops/tessellate"
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
// Example: groups := query.GroupIdenticalBodies(bodies, query.IdenticalBodiesOptions{MatchReflection: true}, query.DefaultQuality())
func GroupIdenticalBodies(bodies []*topo.Body, opt IdenticalBodiesOptions, q Quality) [][]int {
	tol := opt.Tolerance
	if tol <= 0 {
		tol = 1e-6
	}
	sigs := make([]BodySignature, len(bodies))
	for i, b := range bodies {
		sigs[i] = SignatureOf(b, q)
	}
	parent := make([]int, len(bodies))
	for i := range parent {
		parent[i] = i
	}
	for i := range bodies {
		for j := i + 1; j < len(bodies); j++ {
			if sigs[i].matches(sigs[j], tol, opt) {
				dset.Union(parent, i, j)
			}
		}
	}
	return indexGroups(parent)
}

// BodySignature is the rigid-motion invariant fingerprint. Volume, area and the sorted principal
// SURFACE second moments are rotation/translation invariant. The skew is the signed surface third
// moment ∮xyz dA that flips under a coordinate-plane mirror (the reflection discriminator); it is
// computed in the WORLD frame (centroid-relative) and so is NOT itself rotation-invariant, so it is
// compared ONLY when MatchReflection is off (see matches), where the reference semantics assume
// axis-aligned/near-symmetric bodies; a truly rotation-invariant chirality pseudoscalar is out of
// scope for #3449. Both the analytic and the mesh path integrate these SAME surface quantities
// (surfaceCentroidalMoments vs centroidalMoments), so the two fingerprints are interchangeable —
// two congruent bodies match even when one takes the analytic path and the other the mesh fallback.
type BodySignature struct {
	Volume, Area           float64
	Moments                [3]float64 // sorted principal second moments about the centroid
	Skew                   float64    // signed third moment — flips under reflection (world frame)
	Faces, Edges, Vertices int
}

// SignatureOf computes the fingerprint from the body's ANALYTIC mass properties (#3449): the
// kernel ground rules require an oracle that gates a result to be more exact than the result it
// gates, so volume, area, centroid, the principal second moments and the reflection skew all
// integrate the analytic B-rep. A solid the quadrature cannot reconstruct (analyticBodyTerms
// declines) falls back to the whole-body tessellated path, so congruence stays robust; two
// congruent bodies always share a path (integrability is itself a rigid-motion invariant), so the
// analytic and mesh fingerprints are never cross-compared.
func SignatureOf(b *topo.Body, q Quality) BodySignature {
	if sig, ok := AnalyticSignature(b); ok {
		return sig
	}
	return meshSignature(b, q)
}

// AnalyticSignature builds the fingerprint from one analytic integration of the body. The volume,
// area and centroid come from the divergence-flux terms; the principal moments and skew come from
// the SURFACE (shell) moments — the SAME quantities the mesh path integrates over triangles
// (centroidalMoments/triangleSkew), so the analytic and mesh fingerprints are INTERCHANGEABLE and a
// body built one way (→ analytic) matches its congruent twin built another (→ mesh) (#3449). It
// declines (ok=false) exactly when the analytic path declines, so the caller can fall back.
func AnalyticSignature(b *topo.Body) (BodySignature, bool) {
	mt, okM := AnalyticBodyTerms(b)
	at, okA := AnalyticAreaTerms(b)
	if !okM || !okA {
		return BodySignature{}, false
	}
	gp := GeometryFromTerms(mt)
	moments, skew := SurfaceCentroidalMoments(at, gp.Centroid)
	return BodySignature{
		Volume: gp.Volume, Area: gp.Area, Moments: moments, Skew: skew,
		Faces: len(b.Faces()), Edges: len(b.Edges()), Vertices: len(b.Vertices()),
	}, true
}

// SurfaceCentroidalMoments is the analytic twin of centroidalMoments: it shifts the origin shell
// moments to the centroid, forms the surface second-moment (shell inertia) matrix, diagonalizes it
// to the sorted principal moments, and returns the centroid-relative surface skew ∮xyz dA. It
// matches centroidalMoments term for term — including the inertia sign convention on the products.
func SurfaceCentroidalMoments(t areaTerms, centroid math.Point3) ([3]float64, float64) {
	dx, dy, dz := float64(centroid.X), float64(centroid.Y), float64(centroid.Z)
	sxx, syy, szz, sxy, sxz, syz := shellSecondMoments(t, dx, dy, dz)
	eig := symmetricEigenvalues3(syy+szz, sxx+szz, sxx+syy, -sxy, -sxz, -syz)
	sort.Float64s(eig[:])
	return eig, shellSkew(t, dx, dy, dz)
}

// shellSecondMoments shifts the origin surface moments to the centroid (parallel-axis for a shell),
// returning the six centroid-relative products ∮(x−dx)², … , ∮(x−dx)(y−dy) dA.
func shellSecondMoments(t areaTerms, dx, dy, dz float64) (sxx, syy, szz, sxy, sxz, syz float64) {
	sxx = t.sxx - 2*dx*t.sx + dx*dx*t.area
	syy = t.syy - 2*dy*t.sy + dy*dy*t.area
	szz = t.szz - 2*dz*t.sz + dz*dz*t.area
	sxy = t.sxy - dx*t.sy - dy*t.sx + dx*dy*t.area
	sxz = t.sxz - dx*t.sz - dz*t.sx + dx*dz*t.area
	syz = t.syz - dy*t.sz - dz*t.sy + dy*dz*t.area
	return
}

// shellSkew is the centroid-relative surface third moment ∮(x−dx)(y−dy)(z−dz) dA — the reflection
// discriminator, matching triangleSkew's ∮xyz over the mesh.
func shellSkew(t areaTerms, dx, dy, dz float64) float64 {
	return t.sxyz - dz*t.sxy - dy*t.sxz - dx*t.syz +
		dx*dy*t.sz + dx*dz*t.sy + dy*dz*t.sx - dx*dy*dz*t.area
}

// meshSignature is the tessellated fallback: the same invariants integrated over the triangle mesh
// for bodies the analytic path does not yet cover (a named, temporary migration seam).
func meshSignature(b *topo.Body, q Quality) BodySignature {
	mesh, _ := tessellate.TessellateBody(b, q)
	props := MeshGeometryProperties(mesh)
	moments, skew := CentroidalMoments(mesh, props)
	return BodySignature{
		Volume: props.Volume, Area: props.Area, Moments: moments, Skew: skew,
		Faces: len(b.Faces()), Edges: len(b.Edges()), Vertices: len(b.Vertices()),
	}
}

// CentroidalMoments integrates the surface second moments (eigenvalues,
// sorted) and a signed third moment about the centroid. Quadrature is EXACT
// per polynomial degree — edge midpoints for the quadratics (degree-2 exact),
// the Strang 4-point rule for the cubic skew — so the invariants do not
// depend on how a planar face happened to triangulate (a rotated copy's
// earcut may pick different diagonals).
func CentroidalMoments(mesh *Mesh, props GeometryProperties) ([3]float64, float64) {
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
func (s BodySignature) matches(o BodySignature, tol float64, opt IdenticalBodiesOptions) bool {
	if opt.MatchTopology && (s.Faces != o.Faces || s.Edges != o.Edges || s.Vertices != o.Vertices) {
		return false
	}
	if !RelClose(s.Volume, o.Volume, tol) || !RelClose(s.Area, o.Area, tol) {
		return false
	}
	scale := s.Moments[2]
	for i := range s.Moments {
		if stdmath.Abs(s.Moments[i]-o.Moments[i]) > tol*stdmath.Max(scale, 1e-12) {
			return false
		}
	}
	if !opt.MatchReflection && !skewClose(s.Skew, o.Skew, tol, scale) {
		return false
	}
	return true
}

func RelClose(a, b, tol float64) bool {
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
		r := dset.Find(parent, i)
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
