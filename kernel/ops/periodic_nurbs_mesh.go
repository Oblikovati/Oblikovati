// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Covering-space periodic tessellation of a closed-in-u B-spline face (Oblikovati#1510).
//
// A B-spline barrel/loft that is closed (periodic) in u carries an artificial SEAM edge: OCC cuts the
// cylindrical (u,v) chart open at u=0≡1 so it becomes a unit square. When a trim — an imported radial
// bore mouth — straddles that seam, the seam-cut outer loop is no longer a simple polygon (the same 3D
// seam appears at both u=0 and u=1, with the straddling mouth grafted in as a notch), so the planar
// metric CDT (nurbsPcurveMesh) silently drops the crossing boundary constraints and meshes only part of
// the domain — leaving the face non-watertight and grossly under-enclosed.
//
// The fix triangulates the cylinder DIRECTLY (identifying u=0 with u=1) via a covering space: the face's
// true loops (rims + whole mouths, seam edges discarded) are replicated across three u-periods, a single
// constrained Delaunay covers them, and exactly one canonical period of triangles is kept (selected by
// triangle centroid + a material-region test). A mouth that straddles the seam is meshed WHOLE in some
// copy; the canonical "cut" never splits a triangle, and period-shifted boundary vertices coincide in 3D
// (S(u,v)=S(u+P,v)) so the seam welds shut with no special case. Exact 3D edge points are kept, so the
// face still stitches crack-free to its caps and bore wall.

// cylLoop is one closed boundary loop of a closed-in-u face: exact 3D edge points with a CONTINUOUS
// (unwrapped) parameter u and the matching v. A rim wraps a full period in u (uSpan≈P); a mouth does not.
type cylLoop struct {
	p3 []math.Point3
	u  []float64
	v  []float64
}

// periodicNurbsFaceMesh meshes a closed-in-u B-spline face via the covering-space periodic CDT. It
// returns (nil, false) — so splineFaceMesh falls back to nurbsPcurveMesh — unless the face is genuinely a
// singly-closed B-spline band (closed in u, open in v) with recoverable rim+mouth loops.
func periodicNurbsFaceMesh(f *topo.Face, q Quality) (*Mesh, bool) {
	s, ok := f.Geometry().(geom.BSplineSurface)
	if !ok {
		return nil, false
	}
	ulo, uhi := s.UDomain()
	period := uhi - ulo
	if period <= 0 || !surfaceClosedInU(s) || surfaceClosedInV(s) {
		return nil, false // only a singly-periodic (cylinder-like) chart unrolls this way
	}
	loops, ok := faceCylinderLoops(f, s, q, period)
	if !ok {
		return nil, false
	}
	rims, mouths := classifyCylinderLoops(loops, period)
	if len(rims) != 2 || !anyMouthStraddlesSeam(mouths, ulo, period) {
		// The covering CDT is only NEEDED when a trim straddles the seam (the planar seam-cut loop is
		// then non-simple). A closed B-spline whose trims clear the seam — a smooth duct, a bore away
		// from the seam — meshes fine and more finely through nurbsPcurveMesh, so defer to it.
		return nil, false
	}
	m := coveringPeriodicMesh(s, q, ulo, uhi, rims, mouths)
	return m, m != nil
}

// surfaceClosedInU reports whether S(ulo,v)=S(uhi,v) across v — the geometric test for a closed
// (periodic) chart, independent of any B-rep periodic flag (an imported B-spline's domain is [0,1], not
// [0,2π], so isPeriodic does not apply). The tolerance is model-scaled off the sampled surface extent.
func surfaceClosedInU(s geom.BSplineSurface) bool {
	ulo, uhi := s.UDomain()
	vlo, vhi := s.VDomain()
	var samples []math.Point3
	maxGap := 0.0
	for k := 0; k < 5; k++ {
		v := vlo + (vhi-vlo)*(float64(k)+0.5)/5
		a, b := s.PointAt(ulo, v), s.PointAt(uhi, v)
		samples = append(samples, a, b)
		if d := a.DistanceTo(b); d > maxGap {
			maxGap = d
		}
	}
	return maxGap < ResolutionForPoints(samples).Weld()
}

// surfaceClosedInV is the v-direction analogue; a doubly-closed chart (a torus) needs the torus path, not
// this one, so periodicNurbsFaceMesh defers when both directions close.
func surfaceClosedInV(s geom.BSplineSurface) bool {
	ulo, uhi := s.UDomain()
	vlo, vhi := s.VDomain()
	var samples []math.Point3
	maxGap := 0.0
	for k := 0; k < 5; k++ {
		u := ulo + (uhi-ulo)*(float64(k)+0.5)/5
		a, b := s.PointAt(u, vlo), s.PointAt(u, vhi)
		samples = append(samples, a, b)
		if d := a.DistanceTo(b); d > maxGap {
			maxGap = d
		}
	}
	return maxGap < ResolutionForPoints(samples).Weld()
}

// faceCylinderLoops recovers the face's true boundary loops on the cylinder: it discards the artificial
// SEAM edges (an edge used twice by one face is a seam), traces the remaining oriented edge polylines into
// closed loops by welded endpoints (so a mouth split by the seam rejoins into one loop), and marches each
// loop's continuous (unwrapped) (u,v). ok=false if a loop cannot be traced or marched.
func faceCylinderLoops(f *topo.Face, s geom.BSplineSurface, q Quality, period float64) ([]cylLoop, bool) {
	rings, ok := traceClosedRings(nonSeamEdgePolylines(f, q))
	if !ok {
		return nil, false
	}
	loops := make([]cylLoop, 0, len(rings))
	for _, r := range rings {
		loops = append(loops, marchLoopUV(s, r, period))
	}
	return loops, true
}

// nonSeamEdgePolylines returns the face's edges as oriented 3D polylines with the SEAM edges removed (an
// edge a closed-in-u face borders on both sides — used twice — is its seam). What remains are the rim and
// mouth boundaries on the cylinder.
func nonSeamEdgePolylines(f *topo.Face, q Quality) [][]math.Point3 {
	use := map[uint64]int{}
	for _, l := range f.Loops() {
		for _, eu := range l.EdgeUses() {
			use[eu.Edge().ID()]++
		}
	}
	var segs [][]math.Point3
	for _, l := range f.Loops() {
		for _, eu := range l.EdgeUses() {
			if use[eu.Edge().ID()] >= 2 {
				continue // seam edge
			}
			pts := discretizeEdge(eu.Edge(), q)
			if eu.Reversed() {
				pts = reverse3(pts)
			}
			if len(pts) >= 2 {
				segs = append(segs, pts)
			}
		}
	}
	return segs
}

// traceClosedRings chains oriented polyline segments head-to-tail by welded endpoints into closed rings.
// After the seam edges are dropped, the remaining edges of a closed-in-u face form a disjoint union of
// simple cycles (each rim, each mouth), so every welded endpoint has exactly one outgoing segment.
// ok=false if a chain dead-ends (an unexpected open boundary) or does not close.
func traceClosedRings(segs [][]math.Point3) ([][]math.Point3, bool) {
	if len(segs) == 0 {
		return nil, false
	}
	grid := weldGrid(segs)
	key := func(p math.Point3) [3]int64 { return quantizePoint(p, grid) }
	from := map[[3]int64]int{}
	for i, s := range segs {
		from[key(s[0])] = i
	}
	used := make([]bool, len(segs))
	var rings [][]math.Point3
	for start := range segs {
		if used[start] {
			continue
		}
		ring, ok := traceOneRing(segs, from, key, used, start)
		if !ok {
			return nil, false
		}
		rings = append(rings, ring)
	}
	return rings, true
}

// traceOneRing walks the segment chain from start, following each segment's end to the segment that begins
// there, until it returns to start (a closed ring) — appending each segment minus its shared join point.
// ok=false on an open chain (no continuation) or a tangle (a vertex with more than one continuation).
func traceOneRing(segs [][]math.Point3, from map[[3]int64]int, key func(math.Point3) [3]int64, used []bool, start int) ([]math.Point3, bool) {
	var ring []math.Point3
	for cur, steps := start, 0; steps <= len(segs); steps++ {
		used[cur] = true
		seg := segs[cur]
		ring = append(ring, seg[:len(seg)-1]...)
		next, ok := from[key(seg[len(seg)-1])]
		if !ok {
			return nil, false
		}
		if next == start {
			return ring, true
		}
		if used[next] {
			return nil, false
		}
		cur = next
	}
	return nil, false
}

// marchLoopUV assigns each ring point its (u,v) by a global closest-point inversion, then unwraps the u
// sequence into CONTINUOUS values so a rim's u advances monotonically around the cylinder and the period
// jump at the seam is removed. A seeded march was tried first but ParamNear reflects at the seam (it finds
// a local closest point on the near side), folding a rim's u back on itself; the global inversion + unwrap
// tracks the wrap correctly.
func marchLoopUV(s geom.BSplineSurface, ring []math.Point3, period float64) cylLoop {
	us := make([]float64, len(ring))
	vs := make([]float64, len(ring))
	for i := range ring {
		us[i], vs[i] = s.ParamAt(ring[i])
	}
	return cylLoop{p3: ring, u: unwrapPeriod(us, period), v: vs}
}

// unwrapPeriod makes a periodic parameter contiguous by folding each step into (−P/2, P/2] and
// accumulating — like cumulativeUnwrap but for an arbitrary period P (a [0,1] B-spline domain, not 2π).
func unwrapPeriod(a []float64, period float64) []float64 {
	out := make([]float64, len(a))
	out[0] = a[0]
	half := period / 2
	for i := 1; i < len(a); i++ {
		d := a[i] - a[i-1]
		for d > half {
			d -= period
		}
		for d <= -half {
			d += period
		}
		out[i] = out[i-1] + d
	}
	return out
}

// classifyCylinderLoops splits the loops into rims (those that wrap a full period in u — the band's
// bottom and top boundary) and mouths (localized interior holes). A rim's unwrapped u travels ≈one full
// period around the cycle; a mouth's returns to its start.
func classifyCylinderLoops(loops []cylLoop, period float64) (rims, mouths []cylLoop) {
	for _, l := range loops {
		if loopWrapsPeriod(l, period) {
			rims = append(rims, l)
		} else {
			mouths = append(mouths, l)
		}
	}
	return rims, mouths
}

// loopWrapsPeriod reports whether a loop encircles the cylinder once: its unwrapped u advances by about a
// full period from first to last sample (a rim) rather than returning near its start (a mouth).
func loopWrapsPeriod(l cylLoop, period float64) bool {
	net := stdmath.Abs(l.u[len(l.u)-1] - l.u[0])
	// The ring drops its closing duplicate, so a wrapping rim's last sample sits one step short of a full
	// period; half a period cleanly separates a wrap from any plausible mouth.
	return net > 0.5*period
}

// anyMouthStraddlesSeam reports whether some mouth crosses a seam line (ulo + k·period): its unwrapped u
// range spans a period boundary. Only then is the covering CDT needed — otherwise the planar seam-cut path
// handles the face. A mouth exactly clear of the seam stays on the planar path.
func anyMouthStraddlesSeam(mouths []cylLoop, ulo, period float64) bool {
	for _, m := range mouths {
		umin, umax := m.u[0], m.u[0]
		for _, u := range m.u {
			umin, umax = stdmath.Min(umin, u), stdmath.Max(umax, u)
		}
		if stdmath.Floor((umin-ulo)/period) != stdmath.Floor((umax-ulo)/period) {
			return true
		}
	}
	return false
}
