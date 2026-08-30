// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
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

// massTerms are the density-independent divergence-theorem sums for one face or a whole body,
// taken about the ORIGIN: the enclosed volume, the three first moments ∫x_i dV, the six
// second moments ∫x_i x_j dV (the covariance that reduces to the inertia tensor), and the
// surface area. Volume/moments/covariance carry the outward-flux sign; area is unsigned.
type massTerms struct {
	vol           float64 // ∫∫∫ 1 dV
	mx, my, mz    float64 // ∫∫∫ x, y, z dV
	cxx, cyy, czz float64 // ∫∫∫ x², y², z² dV
	cxy, cyz, czx float64 // ∫∫∫ xy, yz, zx dV
	area          float64 // ∮∮ dA
}

// add returns the component-wise sum (used to accumulate cells, segments, loops and faces).
func (a massTerms) add(b massTerms) massTerms {
	return massTerms{
		vol: a.vol + b.vol, mx: a.mx + b.mx, my: a.my + b.my, mz: a.mz + b.mz,
		cxx: a.cxx + b.cxx, cyy: a.cyy + b.cyy, czz: a.czz + b.czz,
		cxy: a.cxy + b.cxy, cyz: a.cyz + b.cyz, czx: a.czx + b.czx,
		area: a.area + b.area,
	}
}

// scale multiplies every component (used by quadrature weights and the region-orientation sign).
func (a massTerms) scale(s float64) massTerms {
	return massTerms{
		vol: a.vol * s, mx: a.mx * s, my: a.my * s, mz: a.mz * s,
		cxx: a.cxx * s, cyy: a.cyy * s, czz: a.czz * s,
		cxy: a.cxy * s, cyz: a.cyz * s, czx: a.czx * s,
		area: a.area * s,
	}
}

// scaleFlux multiplies only the outward-flux (divergence) components by s, leaving the
// unsigned area untouched — this is where a face's outward sense (Face.Reversed ⇒ −1) applies.
func (a massTerms) scaleFlux(s float64) massTerms {
	out := a.scale(s)
	out.area = a.area
	return out
}

// converged reports whether a coarse and a refined estimate agree to tolerance on every component
// (mixed absolute/relative, so a component that is legitimately ~0 does not stall the refinement).
func (a massTerms) converged(r massTerms) bool {
	c := [11]float64{a.vol, a.mx, a.my, a.mz, a.cxx, a.cyy, a.czz, a.cxy, a.cyz, a.czx, a.area}
	d := [11]float64{r.vol, r.mx, r.my, r.mz, r.cxx, r.cyy, r.czz, r.cxy, r.cyz, r.czx, r.area}
	return componentsConverged(c[:], d[:])
}

// integrandsAt evaluates every divergence-theorem integrand (and the area element |∂P/∂u ×
// ∂P/∂v|) at one surface parameter point. N = P_u × P_v is the UNNORMALIZED normal, so its
// magnitude is the area element and its components carry the flux. The fields chosen give
// ∇·F = the desired volume integrand: F=(x,y,z)/3→1, F=(x²/2,0,0)→x, F=(x³/3,0,0)→x²,
// F=(x²y/2,0,0)→xy (and cyclic).
func integrandsAt(s geom.Surface, u, v float64) massTerms {
	p := s.PointAt(u, v)
	du, dv := s.DerivativesAt(u, v)
	n := du.Cross(dv)
	px, py, pz := float64(p.X), float64(p.Y), float64(p.Z)
	nx, ny, nz := float64(n.X), float64(n.Y), float64(n.Z)
	return massTerms{
		vol: (px*nx + py*ny + pz*nz) / 3,
		mx:  px * px * nx / 2, my: py * py * ny / 2, mz: pz * pz * nz / 2,
		cxx: px * px * px * nx / 3, cyy: py * py * py * ny / 3, czz: pz * pz * pz * nz / 3,
		cxy: px * px * py * nx / 2, cyz: py * py * pz * ny / 2, czx: pz * pz * px * nz / 2,
		area: float64(n.Length()),
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
// Example: gp, ok := ops.AnalyticGeometryProperties(cyl) // ok ⇒ gp.Volume == πr²h exactly
func AnalyticGeometryProperties(b *topo.Body) (GeometryProperties, bool) {
	t, ok := analyticBodyTerms(b)
	if !ok {
		return GeometryProperties{}, false
	}
	return geometryFromTerms(t), true
}

// AnalyticInertia integrates the body's inertia tensor (about its centroid, per unit density)
// over its analytic B-rep faces. ok mirrors AnalyticGeometryProperties.
func AnalyticInertia(b *topo.Body) (InertiaTensor, bool) {
	t, ok := analyticBodyTerms(b)
	if !ok {
		return InertiaTensor{}, false
	}
	return inertiaFromTerms(t), true
}

// AnalyticFaceArea returns one face's area by the analytic surface integral ∫∫ |∂P/∂u × ∂P/∂v|
// over its trimmed uv region — the exact polygon area for a planar face, the surface integral for
// a curved one. ok is false when the face is not analytically integrable (fall back to the mesh).
//
// Example: a, ok := ops.AnalyticFaceArea(cylinderSideFace) // ok ⇒ a == 2πrh
func AnalyticFaceArea(f *topo.Face) (float64, bool) {
	t, ok := faceTerms(f)
	if !ok {
		return 0, false
	}
	return t.area, true
}

// analyticBodyTerms sums every face's divergence-theorem contribution. A non-solid body has no
// enclosed volume to integrate; any face the analytic path cannot cover forces a whole-body
// fallback so the result never mixes analytic and mesh contributions.
func analyticBodyTerms(b *topo.Body) (massTerms, bool) {
	if b == nil || !b.IsSolid() {
		return massTerms{}, false
	}
	var total massTerms
	for _, f := range b.Faces() {
		ft, ok := faceTerms(f)
		if !ok {
			return massTerms{}, false
		}
		total = total.add(ft)
	}
	return total, true
}

// analyticAreaTerms sums every face's surface (shell) moments over the body's boundary. It declines
// exactly when analyticBodyTerms would (a face the analytic path cannot reconstruct), so the
// congruence signature falls back to the mesh path as one unit rather than mixing sources.
func analyticAreaTerms(b *topo.Body) (areaTerms, bool) {
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

// faceTerms integrates one face's flux terms over its trimmed uv region and applies the face's
// outward sense (Face.Reversed ⇒ the material side is opposite the surface normal).
func faceTerms(f *topo.Face) (massTerms, bool) {
	s := f.Geometry()
	region, ok := faceRegion(f, func(u, v float64) massTerms { return integrandsAt(s, u, v) })
	if !ok {
		return massTerms{}, false
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

// geometryFromTerms turns origin sums into volume, area and centroid. The centroid is the first
// moment over the (signed) volume, so it is orientation-independent; the reported volume is the
// magnitude (a solid encloses positive volume regardless of parametrization handedness).
func geometryFromTerms(t massTerms) GeometryProperties {
	centroid := math.P3(0, 0, 0)
	if t.vol != 0 {
		centroid = math.P3(math.Scalar(t.mx/t.vol), math.Scalar(t.my/t.vol), math.Scalar(t.mz/t.vol))
	}
	return GeometryProperties{Volume: absFloat(t.vol), Area: t.area, Centroid: centroid}
}

// inertiaFromTerms reduces the origin covariance to the inertia tensor about the centroid. The
// covariance and volume share the flux sign, so both are normalized to the positive orientation
// before the reduction (mirroring meshInertia), keeping ∫x_i x_j dV physically positive.
func inertiaFromTerms(t massTerms) InertiaTensor {
	if t.vol == 0 {
		return InertiaTensor{}
	}
	sgn := 1.0
	if t.vol < 0 {
		sgn = -1
	}
	cov := mat3{
		{sgn * t.cxx, sgn * t.cxy, sgn * t.czx},
		{sgn * t.cxy, sgn * t.cyy, sgn * t.cyz},
		{sgn * t.czx, sgn * t.cyz, sgn * t.czz},
	}
	d := math.P3(math.Scalar(t.mx/t.vol), math.Scalar(t.my/t.vol), math.Scalar(t.mz/t.vol))
	return inertiaFromCovariance(cov, sgn*t.vol, d)
}
