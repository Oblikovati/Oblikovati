// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Seamed sphere-cap tessellation (the S6/S7 sphere-host "notch" forensics). An imported boss
// HEMISPHERE face carries a coplanar multi-arc rim (its footprint equator, subdivided by the blend's
// runout terminations) PLUS the sphere's parametric seam meridian as ONE edge traversed twice
// (pole→rim forward, rim→pole reversed). sphereCapFan requires the WHOLE outer boundary coplanar, so
// the seam samples disqualify it; sphereZoneCapFan requires the rim to be ONE full-circle edge, so
// the subdivided equator disqualifies it too; the face then fell through to spherePatchMesh, whose
// interior Steiner grid is clamped at patchGridCap per axis — for a full hemisphere (stereographic
// chart bbox 2R × 2R) that clamp floors the interior density far below PropertyQuality's own
// contract, and the face under-reported its area by ~6e-4 relative NO MATTER the chord tolerance
// (blend/simple S6/S7: 1061.197 at every swept tolerance vs the closed 2π·13² = 1061.858 — a flat,
// non-converging deficit long mis-read as a trim notch; sphere-notch-report.md). Routing the seamed
// cap to the same latitude-ring fan the plain cap uses restores a mesh that honours the quality it
// was asked for. The seam edge borders only this face (both uses are this loop's), so the fan's
// seam-free interior leaves no weld counterpart dangling.

// sphereSeamedCapFan meshes a sphere face whose outer loop is a coplanar rim chain plus one doubled
// seam edge running to an enclosed pole, by fanning latitude rings from the rim (kept exactly — the
// shared edge discretization every neighbour welds to) to that pole. ok=false for any other shape,
// so every existing sphere path is byte-identical.
func sphereSeamedCapFan(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	sph, ok := s.(geom.Sphere)
	if !ok || len(f.Loops()) != 1 {
		return nil, false
	}
	rim, axis, ok := recognizeSeamedCapRim(f, sph, q)
	if !ok {
		return nil, false
	}
	return buildSphereCap(sph, rim, axis, q), true
}

// recognizeSeamedCapRim runs the recognizer chain: exactly one doubled (opposite-sense) loop edge —
// the seam — whose far vertex sits on the cap pole, with the remaining uses chaining into a closed
// coplanar rim ring validated through the SAME capAxis the plain cap uses. ok=false on any miss, so
// the caller declines and the face keeps its existing mesh path.
func recognizeSeamedCapRim(f *topo.Face, sph geom.Sphere, q Quality) ([]math.Point3, math.Vector3, bool) {
	loop := outerLoopOf(f)
	if loop == nil {
		return nil, math.Vector3{}, false
	}
	seam, ok := loneDoubledLoopEdge(loop)
	if !ok {
		return nil, math.Vector3{}, false
	}
	rim, ok := seamlessRimRing(loop, seam, q)
	if !ok || len(rim) < 3 {
		return nil, math.Vector3{}, false
	}
	axis, ok := capAxis(sph, rim)
	if !ok || !seamEndsAtPole(seam, sph, axis) {
		return nil, math.Vector3{}, false
	}
	return rim, axis, true
}

// loneDoubledLoopEdge returns the loop's single edge that is used exactly TWICE, in OPPOSITE senses —
// the parametric seam. Zero doubled edges (a plain cap, sphereCapFan's shape), more than one, or a
// same-sense double (a degenerate slit, not a seam) all decline.
func loneDoubledLoopEdge(l *topo.Loop) (*topo.Edge, bool) {
	uses := map[*topo.Edge][]bool{}
	for _, u := range l.EdgeUses() {
		uses[u.Edge()] = append(uses[u.Edge()], u.Reversed())
	}
	var seam *topo.Edge
	for e, revs := range uses {
		if len(revs) == 1 {
			continue
		}
		if len(revs) != 2 || revs[0] == revs[1] || seam != nil {
			return nil, false
		}
		seam = e
	}
	return seam, seam != nil
}

// seamlessRimRing chains the loop's non-seam edge discretizations, in loop order, into one closed
// rim ring. The seam block starts and ends at the same rim vertex (out to the pole and back), so the
// kept chain stays contiguous across it; ok=false if any junction gap exceeds the ring's own weld —
// a shape this fan does not understand must decline, never bridge.
func seamlessRimRing(l *topo.Loop, seam *topo.Edge, q Quality) ([]math.Point3, bool) {
	var segs [][]math.Point3
	for _, u := range l.EdgeUses() {
		if u.Edge() == seam {
			continue
		}
		pts := discretizeEdge(u.Edge(), q)
		if u.Reversed() {
			pts = reverse3(pts)
		}
		segs = append(segs, pts)
	}
	return joinRingSegments(segs)
}

// joinRingSegments concatenates orientated segments into a closed ring, dropping each junction's
// duplicated shared point and the final closing duplicate. ok=false on a junction gap over the weld.
func joinRingSegments(segs [][]math.Point3) ([]math.Point3, bool) {
	var all []math.Point3
	for _, s := range segs {
		all = append(all, s...)
	}
	weld := ResolutionForPoints(all).Weld()
	var out []math.Point3
	for _, s := range segs {
		if len(out) > 0 {
			if out[len(out)-1].DistanceTo(s[0]) > math.Scalar(weld) {
				return nil, false // the kept chain is not contiguous — not a seam-split rim
			}
			s = s[1:]
		}
		out = append(out, s...)
	}
	if n := len(out); n > 1 && out[0].DistanceTo(out[n-1]) < math.Scalar(weld) {
		out = out[:n-1]
	}
	return out, true
}

// seamPoleTol is the model-relative distance (fraction of the sphere radius) within which the seam's
// far vertex must sit on the cap pole (centre + R·axis). A parametric seam ends exactly there; an
// unrelated doubled edge (a slit into the face interior) generally does not, and must decline.
const seamPoleTol = 1e-6

// seamEndsAtPole reports whether either seam vertex lies on the cap's enclosed pole.
func seamEndsAtPole(seam *topo.Edge, sph geom.Sphere, axis math.Vector3) bool {
	pole := sph.Center.TranslateBy(axis.Scale(math.Scalar(sph.Radius)))
	tol := math.Scalar(seamPoleTol * sph.Radius)
	return seam.StartVertex().Point().DistanceTo(pole) <= tol ||
		seam.EndVertex().Point().DistanceTo(pole) <= tol
}
