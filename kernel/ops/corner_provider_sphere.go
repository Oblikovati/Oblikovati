// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// railProvider is the RailLoop-path counterpart of [CornerBlendProvider] (ADR-0051): it recognizes
// and fills a [RailLoop] junction directly (Fits/Build take the loop and the model scale, not a
// CornerBlendRequest) so a fill provider depends only on geom+math, never on BlendArm/topo.Face. It
// will grow its own tier walk (mirroring resolveCornerBlend) once a second RailLoop provider exists.
type railProvider interface {
	Name() CornerBlendKind
	Fits(RailLoop) bool
	Build(RailLoop, Resolution) (CornerBlendPatch, Certificate, bool)
}

// analyticSphereProvider recognizes a trihedral corner where every arm carries the SAME fillet
// radius r: the sphere∩arm-cylinder end-section of each arm is then a great circle of a single
// corner sphere of radius r (both surfaces share radius r at that intersection). Recognition reads
// only the rails' geometry — never Side.Adjacent, which matters only to the later G1-fill tiers.
type analyticSphereProvider struct{}

var _ railProvider = analyticSphereProvider{}

// Name reports the provider's telemetry kind (see CornerBlendKind: never read by assembly).
func (analyticSphereProvider) Name() CornerBlendKind { return BlendKindSphere }

// Fits is the cheap classification: every side is an exact Arc3d, all radii agree, and all arc
// centers are coincident — a corner sphere's signature. The real admissibility gate is Build's
// certificate; Fits only decides whether it's worth trying.
func (analyticSphereProvider) Fits(loop RailLoop) bool {
	if loop.Valence() < 3 {
		return false
	}
	r, ok := railRadius(loop)
	if !ok {
		return false
	}
	_, ok = railArcsConcentric(loop, railFitTol(r))
	return ok
}

// Build recovers the common center/radius and emits the corner sphere patch, or declines (ok=false)
// so the tier walk moves on: a non-arc side, disagreeing radii/centers, or a degenerate radius all
// fail here even if a caller skipped Fits.
func (analyticSphereProvider) Build(loop RailLoop, scale Resolution) (CornerBlendPatch, Certificate, bool) {
	r, ok := railRadius(loop)
	if !ok {
		return CornerBlendPatch{}, Certificate{}, false
	}
	center, ok := railArcsConcentric(loop, railFitTol(r))
	if !ok {
		return CornerBlendPatch{}, Certificate{}, false
	}
	sph, err := geom.NewSphere(center, r)
	if err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	patch := CornerBlendPatch{Surface: sph, Loops: railLoopToFilletLoops(loop), Kind: BlendKindSphere}
	return patch, certifySphere(sph, loop, scale), true
}

// sphereFitRelTol is Fits' cheap classification tolerance, relative to a representative arc radius —
// Build's certifySphere (gated by scale.Weld()/seamAngularTol via Certificate.Valid) is the real,
// model-relative admissibility gate (ADR-0042); this one only decides whether Build is worth trying.
const sphereFitRelTol = 1e-6

// sphereFitFloorTol is a tiny absolute floor under sphereFitRelTol so a (never expected in practice)
// zero-radius rail doesn't collapse the tolerance to zero and reject everything on float noise.
const sphereFitFloorTol = 1e-9

// railFitTol scales sphereFitRelTol by a representative length (the candidate radius), floored so it
// never vanishes.
func railFitTol(length float64) float64 {
	return stdmath.Max(sphereFitRelTol*stdmath.Abs(length), sphereFitFloorTol)
}

// railRadius returns the common arc radius shared by every side's Curve, or ok=false when any side's
// Curve isn't a geom.Arc3d or its radius disagrees with the first side's beyond railFitTol.
func railRadius(loop RailLoop) (float64, bool) {
	if len(loop.Sides) == 0 {
		return 0, false
	}
	arc0, ok := loop.Sides[0].Curve.(geom.Arc3d)
	if !ok {
		return 0, false
	}
	for _, side := range loop.Sides[1:] {
		arc, ok := side.Curve.(geom.Arc3d)
		if !ok || stdmath.Abs(arc.Radius-arc0.Radius) > railFitTol(arc0.Radius) {
			return 0, false
		}
	}
	return arc0.Radius, true
}

// railArcsConcentric returns the common Arc3d.Center shared by every side within tol, or ok=false
// when any side's Curve isn't a geom.Arc3d or its center disagrees with the first side's.
func railArcsConcentric(loop RailLoop, tol float64) (math.Point3, bool) {
	arc0, ok := loop.Sides[0].Curve.(geom.Arc3d)
	if !ok {
		return math.Point3{}, false
	}
	for _, side := range loop.Sides[1:] {
		arc, ok := side.Curve.(geom.Arc3d)
		if !ok || arc0.Center.DistanceTo(arc.Center) > tol {
			return math.Point3{}, false
		}
	}
	return arc0.Center, true
}

// railLoopToFilletLoops traces the rails into ONE assembly-ready boundary loop: each side is sampled
// open (sampleCurve3Open, excluding its far endpoint) so consecutive sides concatenate without a
// duplicate point at the shared corner, mirroring boundaryRing/sampleRailOpen (corner_blend_obstacle.go)
// but over geom.Curve3 (Arc3d) rather than geom.BSplineCurve.
func railLoopToFilletLoops(loop RailLoop) []filletLoop {
	var pts []math.Point3
	var curves []geom.Curve3
	for _, side := range loop.Sides {
		for _, p := range sampleCurve3Open(side.Curve, false) {
			pts = append(pts, p)
			curves = append(curves, side.Curve)
		}
	}
	return []filletLoop{{pts: pts, curves: curves}}
}

// sampleCurve3Open returns ringSegSamples points along c (reversed if rev), EXCLUDING the far endpoint,
// so segments from consecutive calls concatenate without duplicating a shared corner. The density is the
// one ring granularity every blend boundary shares (ringSegSamples), so a host/wall/patch that tiles the
// SAME curve welds point-for-point. Generic over geom.Curve3 — sampleRailOpen (corner_blend_obstacle.go)
// is its geom.BSplineCurve-only sibling.
func sampleCurve3Open(c geom.Curve3, rev bool) []math.Point3 {
	return sampleCurveN(c, ringSegSamples, rev)
}

// sampleCurveN is sampleCurve3Open at an explicit chord count n (≥1) — the density lever the intact-boss
// torus rim uses to sample its large host arcs finely enough that the doubly-curved band lofts within
// tolerance (rimSubArcChordCount, fillet_setback_close.go), while every ruled boss/blend stays at
// ringSegSamples. Excludes the far endpoint, like sampleCurve3Open, so consecutive spans concatenate.
func sampleCurveN(c geom.Curve3, n int, rev bool) []math.Point3 {
	lo, hi := c.Domain()
	pts := make([]math.Point3, n)
	for i := 0; i < n; i++ {
		pts[i] = c.PointAt(openSampleParam(lo, hi, i, n, rev))
	}
	return pts
}

// sampleCurve3OpenTrimmed returns the SAME ringSegSamples points sampleCurve3Open returns (they MUST be
// byte-identical — the watertight shared-vertex weld between a canal face and its neighbour keys on the
// point, ADR-C4-2/F3) AND, per sample i, the base curve RESTRICTED to that sample's little sub-span
// [t_i, t_{i+1}] (a geom.TrimmedCurve3). Every NORMAL kernel edge carries its curve trimmed to exactly
// that edge (curveSpan==vGap); a canal boundary sub-edge must too, else the shared tessellator
// (sampleEdgeCurve) sweeps the WHOLE rail once per sub-edge, self-overlapping the loop → the cylinder
// mesher tiles exactly half and the NURBS patch mesher folds+diverges (N7 defect, 2000× slowdown;
// .superpowers/sdd/n7-tessellation-diagnosis.md). The far endpoint is excluded from pts (open sampling,
// like sampleCurve3Open) but IS the Hi of the last sub-curve, so consecutive sides still concatenate
// without duplicating the shared corner.
func sampleCurve3OpenTrimmed(c geom.Curve3, rev bool) ([]math.Point3, []geom.Curve3) {
	lo, hi := c.Domain()
	pts := make([]math.Point3, ringSegSamples)
	curves := make([]geom.Curve3, ringSegSamples)
	for i := 0; i < ringSegSamples; i++ {
		a := openSampleParam(lo, hi, i, ringSegSamples, rev)   // this sub-edge's own start vertex param
		b := openSampleParam(lo, hi, i+1, ringSegSamples, rev) // its end vertex param (far endpoint at i+1==n)
		pts[i] = c.PointAt(a)
		curves[i] = geom.TrimmedCurve3{Base: c, Lo: a, Hi: b}
	}
	return pts, curves
}

// openSampleParam is the base-curve parameter of open-sample k∈[0,n]: the SAME lo+f·(hi-lo) mapping
// sampleCurveN uses (f=k/n, reversed to 1-k/n), factored out so sampleCurve3OpenTrimmed's pts stay
// byte-identical to sampleCurve3Open's. k==n yields the excluded far endpoint (hi forward, lo reversed).
func openSampleParam(lo, hi float64, k, n int, rev bool) float64 {
	f := float64(k) / float64(n)
	if rev {
		f = 1 - f
	}
	return lo + f*(hi-lo)
}

// certifySphere proves the sphere patch (ADR-3): Closed/WeldsArms are structural (the rail loop
// closes; every sampled rail point actually lies on the sphere within scale.Weld()); NoFold is always
// true (a sphere's parametrization never folds); MaxAngleDev is exactly 0 — the great-circle rails lie
// EXACTLY on the sphere, so the sphere IS the surface, not a fit to it.
func certifySphere(sph geom.Sphere, loop RailLoop, scale Resolution) Certificate {
	weld := scale.Weld()
	maxDev, weldsArms := sphereRailDeviation(sph, loop, weld)
	return Certificate{
		Closed: loop.Closed(weld), WeldsArms: weldsArms, NoFold: true,
		MaxDev: maxDev, MaxAngleDev: 0,
	}
}

// sphereRailDeviation samples every side's rail and returns the max |dist(pt,Center) − Radius| and
// whether every sample stays within weld of the sphere (Certificate.WeldsArms's proof).
func sphereRailDeviation(sph geom.Sphere, loop RailLoop, weld float64) (maxDev float64, weldsArms bool) {
	weldsArms = true
	for _, side := range loop.Sides {
		for _, p := range sampleCurve3Open(side.Curve, false) {
			dev := stdmath.Abs(p.DistanceTo(sph.Center) - sph.Radius)
			maxDev = stdmath.Max(maxDev, dev)
			weldsArms = weldsArms && dev <= weld
		}
	}
	return maxDev, weldsArms
}
