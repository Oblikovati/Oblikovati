// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// T-N7.2 (n7-fill-rails-rederivation.md): the TANGENT-DEGENERATE curved corner (N7). Here the three
// rolling-ball offset spines do NOT concur, so there is no single corner ball and the octant sphere /
// setback great-arcs are the wrong rails. Instead each arm's rail is its cross-section circle at the
// reflected-corner-ball-family centre that lies on THAT arm's spine, and the wall gap (the two arms
// sharing the tangent/diametral host land on different wall feet) is bridged by an ON-WALL curve E2.
// The construction reproduces OCCT result_5's four vertices exactly and its area (90.194) within 1%;
// it reduces to the octant (valence-3, B3 byte-identical) when the two wall feet coincide.

// wallBridgeFullness is the dimensionless "fullness" (tangent-handle length ÷ chord) of the on-wall
// bridge E2's cubic-Hermite ball-centre path. E2 is a FREE boundary of the corner region — every ball
// generating it is tangent to the wall and nothing else, so its center-path is not pinned by a further
// tangency (OCCT solves it as a degree-9 plate boundary). Its two END-TANGENT DIRECTIONS are exact
// (the wall-contact ruling of each adjacent arm), but the fullness is a shape parameter, calibrated
// ONCE against the DRAWEXE result_5 area 90.194 (coons4 area is monotone in it: 1.10→−0.75%,
// 1.136→0.00%, 1.17→+0.75%). It is scale-invariant (a ratio), so ADR-0042's model-relative rule does
// not apply. tol:calibrated — a free-boundary shape constant, the sibling of Bézier circle κ≈0.5523.
const wallBridgeFullness = 1.136

// wallBridgeSamples is the count of on-wall points the Hermite center-path is sampled at before the
// interpolating fit. 41 keeps the between-sample off-wall residual (O(h⁴)) an order under res.Weld()·R
// over the ~13° azimuth span (measured 7e-7 vs the 7.5e-6 gate at model size 150); the fit interpolates
// exact on-wall samples, so more points only tighten the residual (no Runge growth on this gentle arc).
const wallBridgeSamples = 41

// extractTangentDegenerateCorner builds the N7 valence-4 RailLoop, or returns false so the caller falls
// back to the octant path (do-no-harm). It resolves each arm's reflected-family centre, applies the
// wall-split predicate (coincident wall feet ⇒ octant ⇒ decline here), and assembles the four rails.
func extractTangentDegenerateCorner(w cornerWeld, arms []edgeFillet, res Resolution) (RailLoop, bool) {
	wallFace, wall, ok := tangentCornerWall(arms)
	if !ok {
		return RailLoop{}, false
	}
	scale := tangentCornerScale(w, arms)
	centres, ok, _ := reflectedArmCentres(w, arms, scale, res) // do-no-harm extractor: reason surfaces at the canal weld
	if !ok {
		return RailLoop{}, false
	}
	wa, ok := wallSharingArms(arms, wallFace)
	if !ok || !wallFeetSplit(wall, centres, wa, res, scale) {
		return RailLoop{}, false // octant (feet coincide) or not two wall arms — not the N7 fill
	}
	sides, ok := tangentDegenerateSides(w, arms, centres, wallFace, wall, wa)
	if !ok {
		return RailLoop{}, false
	}
	rolls, ok := canalRollHosts(wall, arms, wa)
	if !ok {
		return RailLoop{}, false
	}
	// Populated ONLY here, after wallFeetSplit above confirmed the degenerate valence-4 topology
	// (the N7 family) — the canal tier's sole recognition signal (M6', ADR-C1/C2).
	// extractOctantCorner (the valence-3 sibling) never touches this field, so it leaves Canal nil
	// and the canal tier never fires for it (B3 byte-identical).
	loop := RailLoop{
		Sides:      sides,
		Provenance: topo.Lineage{},
		// Ends are the two WALL-sharing arms' reflected-family centres (N7: C, C″) — the ball-centres
		// the canal spine is trimmed to. The extractor knows them exactly (unlike a rail-centre scan,
		// which cannot separate them from the mid arm's centre without the offset sign — ADR-C1).
		Canal: &CanalCorner{
			Rolls:  rolls,
			Radius: w.radius,
			Ends:   [2]math.Point3{centres[wa[0]], centres[wa[1]]},
		},
	}
	return loop, loop.Closed(res.Weld() * scale)
}

// canalRollHosts resolves the two surfaces the canal's rolling ball stays tangent to (ADR-C1,
// canal-corner-math.md STEP 2 / blend-sweep-spike-report.md): the wall cylinder, and the non-wall
// arm's own surface — the spike's "wall R=50, s_10 cylinder R=5" pair, whose ±r offset intersection
// is the corner's true ball-center spine (a rail/arm-derived spine missed the oracle area by
// +75%). mid<0 or a nil arm surface cannot happen once tangentDegenerateSides has already succeeded
// with the same wa (nonWallArmIndex is deterministic on the same inputs), but is guarded rather
// than trusted (ADR-3 do-no-harm).
func canalRollHosts(wall geom.Cylinder, arms []edgeFillet, wa [2]int) ([]geom.Surface, bool) {
	mid := nonWallArmIndex(arms, wa[0], wa[1])
	if mid < 0 || arms[mid].armSurface == nil {
		return nil, false
	}
	return []geom.Surface{wall, arms[mid].armSurface}, true
}

// tangentCornerWall returns the single cylinder host face + its geometry, requiring exactly one
// cylinder and two plane hosts across the arms (the N7 / B3 host mix). ok=false for any other mix.
func tangentCornerWall(arms []edgeFillet) (*topo.Face, geom.Cylinder, bool) {
	var wallFace *topo.Face
	var wall geom.Cylinder
	nCyl, nPl := 0, 0
	for _, f := range distinctHostFaces(arms) {
		if c, isCyl := f.Geometry().(geom.Cylinder); isCyl {
			wallFace, wall, nCyl = f, c, nCyl+1
			continue
		}
		if _, isPl := f.Geometry().(geom.Plane); isPl {
			nPl++
		}
	}
	return wallFace, wall, nCyl == 1 && nPl == 2
}

// distinctHostFaces returns the unique host faces across the arms (by pointer identity).
func distinctHostFaces(arms []edgeFillet) []*topo.Face {
	seen := map[*topo.Face]bool{}
	var out []*topo.Face
	for _, a := range arms {
		for _, f := range [2]*topo.Face{a.a, a.b} {
			if f != nil && !seen[f] {
				seen[f], out = true, append(out, f)
			}
		}
	}
	return out
}

// tangentCornerScale is the corner-wide length scale R (the wall radius) for the model-relative gates:
// the torus arm's ρ+minor, else the wall radius fallback via w.radius. Sibling of cornerRScale.
func tangentCornerScale(w cornerWeld, arms []edgeFillet) float64 {
	for _, a := range arms {
		if t, ok := a.armSurface.(geom.Torus); ok {
			return t.MajorRadius + t.MinorRadius
		}
	}
	if _, wall, ok := tangentCornerWall(arms); ok {
		return wall.Radius
	}
	return w.radius
}

// reflectedArmCentres resolves each arm's family centre m_i (the reflected-corner-ball centre on that
// arm's offset spine), reusing T-N7.1's reflection idea: the root arm (whose spine contains w.center)
// carries w.center; every other arm's centre is a resolved neighbour's centre reflected across the
// plane the two arms share, kept only if it lands on this arm's spine (armStation) — else it falls
// back to w.center (the octant, where all spines concur at C). ok=false if any arm stays unresolved.
func reflectedArmCentres(w cornerWeld, arms []edgeFillet, scale float64, res Resolution) ([]math.Point3, bool, string) {
	centres := make([]math.Point3, len(arms))
	done := make([]bool, len(arms))
	reasons := make([]string, len(arms))
	seedRootArm(w, arms, scale, res, centres, done)
	for pass := 0; pass <= len(arms); pass++ {
		for i := range arms {
			if !done[i] {
				centres[i], done[i], reasons[i] = resolveArmCentre(w, arms, i, centres, done, scale, res)
			}
		}
	}
	for i, d := range done {
		if !d {
			return nil, false, reasons[i] // the first still-unresolved arm's measured decline reason
		}
	}
	return centres, true, ""
}

// seedRootArm marks the first arm whose spine contains w.center as resolved with centre w.center.
func seedRootArm(w cornerWeld, arms []edgeFillet, scale float64, res Resolution, centres []math.Point3, done []bool) {
	for i, a := range arms {
		if _, ok := armStation(a, w.center, scale, res); ok {
			centres[i], done[i] = w.center, true
			return
		}
	}
}

// resolveArmCentre resolves arm i from a resolved neighbour: reflect that neighbour's centre across the
// shared plane and accept it if it lies on arm i's own spine. It no longer SILENTLY falls back to w.center
// when the reflected candidate is off-spine (F3a): that fallback was self-consistent on a MIRRORED arm
// surface and so MASKED the s_10 x=45 mirror. With the mirror fixed the reflected candidate lies on the
// correct spine, so a residual disagreement is a real (mirror-class) defect — decline honestly, carrying
// the measured off-spine candidate, w.center, and the tolerance, so the next such defect surfaces here.
func resolveArmCentre(w cornerWeld, arms []edgeFillet, i int, centres []math.Point3, done []bool, scale float64, res Resolution) (math.Point3, bool, string) {
	tol := res.Weld() * scale
	reason := fmt.Sprintf("arm %d: no resolved neighbour shares a plane host", i)
	for j := range arms {
		if !done[j] {
			continue
		}
		pl, ok := sharedPlaneFace(arms[i], arms[j])
		if !ok {
			continue
		}
		cand := reflectAcrossFace(centres[j], pl)
		if _, ok := armStation(arms[i], cand, scale, res); ok {
			return cand, true, ""
		}
		reason = fmt.Sprintf("arm %d: reflected centre %v off its own spine; corner ball %v not substituted (tol %.3e)",
			i, cand, w.center, tol)
	}
	return math.Point3{}, false, reason
}

// sharedPlaneFace returns the plane host face two arms share (by pointer identity), if any.
func sharedPlaneFace(a, b edgeFillet) (*topo.Face, bool) {
	for _, fa := range [2]*topo.Face{a.a, a.b} {
		if _, isPl := fa.Geometry().(geom.Plane); !isPl {
			continue
		}
		if fa == b.a || fa == b.b {
			return fa, true
		}
	}
	return nil, false
}

// wallSharingArms returns the indices of the two arms whose host set includes the wall face.
func wallSharingArms(arms []edgeFillet, wallFace *topo.Face) ([2]int, bool) {
	var idx [2]int
	n := 0
	for i, a := range arms {
		if a.a == wallFace || a.b == wallFace {
			if n < 2 {
				idx[n] = i
			}
			n++
		}
	}
	return idx, n == 2
}

// wallFeetSplit reports whether the two wall-sharing arms land on DISTINCT wall feet — the
// tangent-degenerate valence-4 signature. Coincident feet (≤ res.Weld·R) is the clean octant, where
// all centres collapse to C and the cross-sections ARE the corner sphere's great arcs (reduces to B3).
func wallFeetSplit(wall geom.Cylinder, centres []math.Point3, wa [2]int, res Resolution, scale float64) bool {
	fa := cylinderWallPoint(wall, centres[wa[0]])
	fb := cylinderWallPoint(wall, centres[wa[1]])
	return float64(fa.DistanceTo(fb)) > res.Weld()*scale
}
