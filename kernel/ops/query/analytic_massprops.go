// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Analytic mass properties (M48/C3, Oblikovati/Oblikovati#3453..3458). The kernel ground
// rules require volume, centroid, area and inertia to integrate the ANALYTIC B-rep, not a
// triangle mesh — "an oracle that gates a result must be more exact than the result it
// gates." Each closed body is integrated face by face by the divergence theorem: a volume
// integral ∫∫∫ f dV becomes a surface flux ∮∮ F·n dA (∇·F = f), and each face's flux is a
// parametric integral over its trimmed (u, v) region evaluated by Gauss–Legendre quadrature
// of the surface itself. This is a derived numeric computation of the analytic surface, NOT
// a tessellation read (the ground rules permit it explicitly). Faces the analytic path does
// not yet cover fall back, per body, to the tessellated path — a named, temporary migration
// seam, not a silent degrade.

// MassTerms are the density-independent divergence-theorem sums for one face or a whole body,
// taken about the ORIGIN: the enclosed volume, the three first moments ∫x_i dV, the six
// second moments ∫x_i x_j dV (the covariance that reduces to the inertia tensor), and the
// surface area. Volume/moments/covariance carry the outward-flux sign; area is unsigned.
type MassTerms struct {
	Vol           float64 // ∫∫∫ 1 dV
	Mx, My, Mz    float64 // ∫∫∫ x, y, z dV
	Cxx, Cyy, Czz float64 // ∫∫∫ x², y², z² dV
	Cxy, Cyz, Czx float64 // ∫∫∫ xy, yz, zx dV
	Ax, Ay, Az    float64 // ∮∮ N dA — the outward VECTOR area, zero over a closed surface
	Area          float64 // ∮∮ dA
}

// add returns the component-wise sum (used to accumulate cells, segments, loops and faces).
func (a MassTerms) add(b MassTerms) MassTerms {
	return MassTerms{
		Vol: a.Vol + b.Vol, Mx: a.Mx + b.Mx, My: a.My + b.My, Mz: a.Mz + b.Mz,
		Cxx: a.Cxx + b.Cxx, Cyy: a.Cyy + b.Cyy, Czz: a.Czz + b.Czz,
		Cxy: a.Cxy + b.Cxy, Cyz: a.Cyz + b.Cyz, Czx: a.Czx + b.Czx,
		Ax: a.Ax + b.Ax, Ay: a.Ay + b.Ay, Az: a.Az + b.Az,
		Area: a.Area + b.Area,
	}
}

// scale multiplies every component (used by quadrature weights and the region-orientation sign).
func (a MassTerms) scale(s float64) MassTerms {
	return MassTerms{
		Vol: a.Vol * s, Mx: a.Mx * s, My: a.My * s, Mz: a.Mz * s,
		Cxx: a.Cxx * s, Cyy: a.Cyy * s, Czz: a.Czz * s,
		Cxy: a.Cxy * s, Cyz: a.Cyz * s, Czx: a.Czx * s,
		Ax: a.Ax * s, Ay: a.Ay * s, Az: a.Az * s,
		Area: a.Area * s,
	}
}

// scaleFlux multiplies only the outward-flux (divergence) components by s, leaving the
// unsigned area untouched — this is where a face's outward sense (Face.Reversed ⇒ −1) applies.
func (a MassTerms) scaleFlux(s float64) MassTerms {
	out := a.scale(s)
	out.Area = a.Area
	return out
}

// measure is the area component — the term integrated from an unsigned integrand, so its sign
// reports the boundary traversal's orientation.
func (a MassTerms) measure() float64 { return a.Area }

// converged reports whether a coarse and a refined estimate agree to tolerance on every component
// (mixed absolute/relative, so a component that is legitimately ~0 does not stall the refinement).
func (a MassTerms) converged(r MassTerms) bool {
	c := [14]float64{a.Vol, a.Mx, a.My, a.Mz, a.Cxx, a.Cyy, a.Czz, a.Cxy, a.Cyz, a.Czx, a.Ax, a.Ay, a.Az, a.Area}
	d := [14]float64{r.Vol, r.Mx, r.My, r.Mz, r.Cxx, r.Cyy, r.Czz, r.Cxy, r.Cyz, r.Czx, r.Ax, r.Ay, r.Az, r.Area}
	return componentsConverged(c[:], d[:])
}

// integrandsAt evaluates every divergence-theorem integrand (and the area element |∂P/∂u ×
// ∂P/∂v|) at one surface parameter point. N = P_u × P_v is the UNNORMALIZED normal, so its
// magnitude is the area element and its components carry the flux. The fields chosen give
// ∇·F = the desired volume integrand: F=(x,y,z)/3→1, F=(x²/2,0,0)→x, F=(x³/3,0,0)→x²,
// F=(x²y/2,0,0)→xy (and cyclic).
func integrandsAt(s geom.Surface, u, v float64) MassTerms {
	p := s.PointAt(u, v)
	du, dv := s.DerivativesAt(u, v)
	n := du.Cross(dv)
	px, py, pz := float64(p.X), float64(p.Y), float64(p.Z)
	nx, ny, nz := float64(n.X), float64(n.Y), float64(n.Z)
	return MassTerms{
		Vol: (px*nx + py*ny + pz*nz) / 3,
		Mx:  px * px * nx / 2, My: py * py * ny / 2, Mz: pz * pz * nz / 2,
		Cxx: px * px * px * nx / 3, Cyy: py * py * py * ny / 3, Czz: pz * pz * pz * nz / 3,
		Cxy: px * px * py * nx / 2, Cyz: py * py * pz * ny / 2, Czx: pz * pz * px * nz / 2,
		Ax: nx, Ay: ny, Az: nz,
		Area: float64(n.Length()),
	}
}

// areaTerms are the surface (shell) moments over one face or a whole body's boundary, about the
// ORIGIN, every integrand AREA-weighted by the analytic element dA=|∂P/∂u × ∂P/∂v| du dv. They are
// the same quantities the mesh signature integrates over triangles (centroidalMoments/triangleSkew),
// computed analytically so the two paths are INTERCHANGEABLE for congruence (#3449). A shell integral
// does not carry an outward-flux sign: every face contributes its positive area weight.
type areaTerms struct {
	area          float64 // ∮ dA
	sx, sy, sz    float64 // ∮ x, y, z dA
	sxx, syy, szz float64 // ∮ x², y², z² dA
	sxy, sxz, syz float64 // ∮ xy, xz, yz dA
	sxyz          float64 // ∮ xyz dA — the surface third moment the signature reflects on
}

// add returns the component-wise sum.
func (a areaTerms) add(b areaTerms) areaTerms {
	return areaTerms{
		area: a.area + b.area, sx: a.sx + b.sx, sy: a.sy + b.sy, sz: a.sz + b.sz,
		sxx: a.sxx + b.sxx, syy: a.syy + b.syy, szz: a.szz + b.szz,
		sxy: a.sxy + b.sxy, sxz: a.sxz + b.sxz, syz: a.syz + b.syz,
		sxyz: a.sxyz + b.sxyz,
	}
}

// scale multiplies every component (quadrature weights and the loop-orientation sign).
func (a areaTerms) scale(s float64) areaTerms {
	return areaTerms{
		area: a.area * s, sx: a.sx * s, sy: a.sy * s, sz: a.sz * s,
		sxx: a.sxx * s, syy: a.syy * s, szz: a.szz * s,
		sxy: a.sxy * s, sxz: a.sxz * s, syz: a.syz * s,
		sxyz: a.sxyz * s,
	}
}

// measure is the area component, whose sign reports the boundary traversal's orientation.
func (a areaTerms) measure() float64 { return a.area }

// converged mirrors massTerms.converged over the surface-moment components.
func (a areaTerms) converged(r areaTerms) bool {
	c := [11]float64{a.area, a.sx, a.sy, a.sz, a.sxx, a.syy, a.szz, a.sxy, a.sxz, a.syz, a.sxyz}
	d := [11]float64{r.area, r.sx, r.sy, r.sz, r.sxx, r.syy, r.szz, r.sxy, r.sxz, r.syz, r.sxyz}
	return componentsConverged(c[:], d[:])
}

// componentsConverged reports whether every paired component agrees within the mixed
// absolute/relative adaptive-quadrature tolerance.
func componentsConverged(coarse, refined []float64) bool {
	for i := range coarse {
		if stdmath.Abs(coarse[i]-refined[i]) > quadAbsTol+quadRelTol*stdmath.Abs(refined[i]) {
			return false
		}
	}
	return true
}

// areaIntegrandsAt evaluates the surface-moment integrands at one parameter point: each is the
// monomial in position times the area element |∂P/∂u × ∂P/∂v| (so ∫∫_D g du dv = ∮ g dA).
func areaIntegrandsAt(s geom.Surface, u, v float64) areaTerms {
	p := s.PointAt(u, v)
	du, dv := s.DerivativesAt(u, v)
	nl := float64(du.Cross(dv).Length())
	px, py, pz := float64(p.X), float64(p.Y), float64(p.Z)
	return areaTerms{
		area: nl, sx: px * nl, sy: py * nl, sz: pz * nl,
		sxx: px * px * nl, syy: py * py * nl, szz: pz * pz * nl,
		sxy: px * py * nl, sxz: px * pz * nl, syz: py * pz * nl,
		sxyz: px * py * pz * nl,
	}
}

// AnalyticGeometryProperties integrates the body's volume, area and centroid over its analytic
// B-rep faces. ok is false when a face is not yet analytically integrable (e.g. a trimmed NURBS
// whose uv boundary cannot be reconstructed), so the caller can fall back to the mesh path.
//
// Example: gp, ok := query.AnalyticGeometryProperties(cyl) // ok ⇒ gp.Volume == πr²h exactly
func AnalyticGeometryProperties(b *topo.Body) (GeometryProperties, bool) {
	t, ok := AnalyticBodyTerms(b)
	if !ok {
		return GeometryProperties{}, false
	}
	return GeometryFromTerms(t), true
}

// analyticInertia integrates the body's inertia tensor (about its centroid, per unit density)
// over its analytic B-rep faces. ok mirrors AnalyticGeometryProperties.
func analyticInertia(b *topo.Body) (InertiaTensor, bool) {
	t, ok := AnalyticBodyTerms(b)
	if !ok {
		return InertiaTensor{}, false
	}
	return inertiaFromTerms(t), true
}

// AnalyticFaceArea returns one face's area by the analytic surface integral ∫∫ |∂P/∂u × ∂P/∂v|
// over its trimmed uv region — the exact polygon area for a planar face, the surface integral for
// a curved one. ok is false when the face is not analytically integrable (fall back to the mesh).
//
// Example: a, ok := query.AnalyticFaceArea(cylinderSideFace) // ok ⇒ a == 2πrh
func AnalyticFaceArea(f *topo.Face) (float64, bool) {
	t, ok := FaceTerms(f)
	if !ok {
		return 0, false
	}
	return t.Area, true
}

// AnalyticShellVolume integrates the SIGNED volume the shell bounds over its analytic faces
// (M48/C3 #3482). The sign is the shell's material orientation and comes out of the divergence
// theorem for free: an outer shell's material-outward normals point away from the region it
// encloses (positive flux), a void shell's point into the cavity (negative). ok is false when a
// face is not analytically integrable, so the caller falls back to the mesh sum as one unit.
//
// Example: v, ok := query.AnalyticShellVolume(cavitySkin) // ok ⇒ v < 0
func AnalyticShellVolume(s *topo.Shell) (float64, bool) {
	if s == nil {
		return 0, false
	}
	var total MassTerms
	for _, f := range s.Faces() {
		ft, ok := FaceTerms(f)
		if !ok {
			return 0, false
		}
		total = total.add(ft)
	}
	return total.Vol, true
}

// AnalyticBodyTerms sums every face's divergence-theorem contribution. A non-solid body has no
// enclosed volume to integrate; any face the analytic path cannot cover forces a whole-body
// fallback so the result never mixes analytic and mesh contributions.
func AnalyticBodyTerms(b *topo.Body) (MassTerms, bool) {
	if b == nil || !b.IsSolid() {
		return MassTerms{}, false
	}
	var total MassTerms
	for _, f := range b.Faces() {
		ft, ok := FaceTerms(f)
		if !ok {
			return MassTerms{}, false
		}
		total = total.add(ft)
	}
	if !vectorAreaCloses(b, total) {
		return MassTerms{}, false
	}
	return total, true
}

// vectorAreaCloses checks the divergence theorem's own precondition on the assembled body: the
// outward vector area ∮∮ N dA of a CLOSED surface is exactly zero, whatever the shape. A face
// integrated over the wrong region, with a flipped orientation, or omitted, leaves a residual — so
// this is a post-condition that turns a wrong analytic answer into a DECLINE, and the tessellated
// fallback then measures the body instead. It costs three more integrands and catches the whole
// class (M48/C3, Oblikovati/Oblikovati#3453).
//
// It is deliberately NOT widened by the body's achieved boundary tolerance, though that number now
// exists (AchievedBoundarySlack). Widening it was tried and reverted: the residual a marched
// boundary produces and the residual a genuinely mis-taken face region produces are the SAME size on
// the same bodies, so the widened gate admitted a crossing-cylinder cut whose volume was 9.8% wrong
// while its boundary noise accounted for the residual. The tolerance is not the axis to move; the
// approximate boundary is (#3489).
func vectorAreaCloses(_ *topo.Body, t MassTerms) bool {
	if t.Area <= 0 {
		return false
	}
	residual := stdmath.Sqrt(t.Ax*t.Ax + t.Ay*t.Ay + t.Az*t.Az)
	return residual <= vectorAreaClosureTol*t.Area
}

// AchievedBoundarySlack is the largest error the body's own boundary approximation can produce in a
// quantity integrated over its faces — the vector-area residual and the volume alike. It is the sum
// over edges of twice the edge's achieved tolerance times its length: both faces meeting on an edge
// invert its points onto THEIR OWN surface, so a point d off the true curve lands up to d away on
// each, and the two faces disagree about that boundary by up to 2d along its length.
//
// It is zero for an all-analytic body, which is what keeps the exact case held to the exact standard.
//
// Example: if rel := math.Abs(got-want) / want; rel > query.AchievedBoundarySlack(b)/want { /* real */ }
func AchievedBoundarySlack(b *topo.Body) float64 {
	slack := 0.0
	for _, e := range b.Edges() {
		tol := e.Tolerance()
		if tol <= 0 {
			continue
		}
		lo, hi := e.Geometry().Domain()
		slack += 2 * tol * geom.CurveLength3(e.Geometry(), lo, hi)
	}
	return slack
}

// vectorAreaClosureTol is how much of the total area the vector-area residual may be. The adaptive
// quadrature converges to ~1e-11 relative, so this leaves five orders of margin for accumulation
// while still catching a single mis-oriented face, whose residual is twice that face's own area.
const vectorAreaClosureTol = 1e-6 // tol:numeric — relative closure of the outward vector area

// AnalyticAreaTerms sums every face's surface (shell) moments over the body's boundary. It declines
// exactly when analyticBodyTerms would (a face the analytic path cannot reconstruct), so the
// congruence signature falls back to the mesh path as one unit rather than mixing sources.
func AnalyticAreaTerms(b *topo.Body) (areaTerms, bool) {
	if b == nil || !b.IsSolid() {
		return areaTerms{}, false
	}
	var total areaTerms
	for _, f := range b.Faces() {
		ft, ok := areaFaceTerms(f)
		if !ok {
			return areaTerms{}, false
		}
		total = total.add(ft)
	}
	return total, true
}

// FaceTerms integrates one face's flux terms over its trimmed uv region and applies the face's
// outward sense (Face.Reversed ⇒ the material side is opposite the surface normal).
func FaceTerms(f *topo.Face) (MassTerms, bool) {
	s := f.Geometry()
	region, ok := faceRegion(f, func(u, v float64) MassTerms { return integrandsAt(s, u, v) })
	if !ok {
		return MassTerms{}, false
	}
	eps := 1.0
	if f.Reversed() { // outward material side is opposite the surface normal
		eps = -1
	}
	return region.scaleFlux(eps), true
}

// areaFaceTerms integrates one face's surface moments over its trimmed uv region. Unlike faceTerms
// it applies NO outward-flux sign: a shell integral is unsigned, so every face — outer wall, cap or
// bore wall — contributes its positive area weight (Face.Reversed does not flip a surface moment).
func areaFaceTerms(f *topo.Face) (areaTerms, bool) {
	s := f.Geometry()
	return faceRegion(f, func(u, v float64) areaTerms { return areaIntegrandsAt(s, u, v) })
}

// GeometryFromTerms turns origin sums into volume, area and centroid. The centroid is the first
// moment over the (signed) volume, so it is orientation-independent; the reported volume is the
// magnitude (a solid encloses positive volume regardless of parametrization handedness).
func GeometryFromTerms(t MassTerms) GeometryProperties {
	centroid := math.P3(0, 0, 0)
	if t.Vol != 0 {
		centroid = math.P3(math.Scalar(t.Mx/t.Vol), math.Scalar(t.My/t.Vol), math.Scalar(t.Mz/t.Vol))
	}
	return GeometryProperties{Volume: probe.AbsFloat(t.Vol), Area: t.Area, Centroid: centroid}
}

// inertiaFromTerms reduces the origin covariance to the inertia tensor about the centroid. The
// covariance and volume share the flux sign, so both are normalized to the positive orientation
// before the reduction (mirroring meshInertia), keeping ∫x_i x_j dV physically positive.
func inertiaFromTerms(t MassTerms) InertiaTensor {
	if t.Vol == 0 {
		return InertiaTensor{}
	}
	sgn := 1.0
	if t.Vol < 0 {
		sgn = -1
	}
	cov := mat3{
		{sgn * t.Cxx, sgn * t.Cxy, sgn * t.Czx},
		{sgn * t.Cxy, sgn * t.Cyy, sgn * t.Cyz},
		{sgn * t.Czx, sgn * t.Cyz, sgn * t.Czz},
	}
	d := math.P3(math.Scalar(t.Mx/t.Vol), math.Scalar(t.My/t.Vol), math.Scalar(t.Mz/t.Vol))
	return inertiaFromCovariance(cov, sgn*t.Vol, d)
}
