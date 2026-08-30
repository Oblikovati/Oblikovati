// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
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

// faceTerms integrates one face over its trimmed uv region and applies the face's outward
// sense. A boundary-less face (a whole sphere or torus) spans its full parameter rectangle; a
// bounded face reduces every integrand to a Green's-theorem line integral over its uv loops.
func faceTerms(f *topo.Face) (massTerms, bool) {
	var region massTerms
	var ok bool
	if len(f.Loops()) == 0 {
		region, ok = fullDomainTerms(f.Geometry())
	} else {
		region, ok = greenTerms(f)
	}
	if !ok {
		return massTerms{}, false
	}
	eps := 1.0
	if f.Reversed() { // outward material side is opposite the surface normal
		eps = -1
	}
	return region.scaleFlux(eps), true
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
