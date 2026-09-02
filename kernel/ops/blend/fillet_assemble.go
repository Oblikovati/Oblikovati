// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
)

// Assembling the fillet result: welding the solved blend faces back onto the host body (split out
// of fillet.go for #2217).
//
// The rebuild-candidate machinery here is a best-of-N composition — several bodies are built and
// one is chosen by a priority ladder — which the ground rules forbid ("dispatch is a classification
// that selects exactly one path"). #3400 deletes the ladder and #3401 replaces the composition;
// the code is kept together in one file so that work has one place to land.

// assemblePlanarFilletBody runs the planar fillet's runout guards, assembles the do-no-harm body, and
// certifies it — naming the #1797 corner-into-round cause when the build-then-certify result still fails.
func assemblePlanarFilletBody(body *topo.Body, edges []filletPick, fils []edgeFillet, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter, concave ConcaveFill) (*topo.Body, error) {
	if err := applyRunoutSetback(fils); err != nil {
		return nil, err // a runout flank rail is parallel to its far plane — no pierce (n-valent degeneracy)
	}
	if err := validateRunoutFans(fils); err != nil {
		return nil, err // n-valent analogue of #1800: reject a self-intersecting/over-radius runout before it silently drops to an open shell
	}
	var res *topo.Body
	if !blendsCarryRadiusTorus(blends) {
		// A mixed-radius torus corner's transient band ends are not weldable, so its baseline body is
		// never built: the setback pass either certifies the torus composition or the nil baseline
		// falls through to the honest decline below (never a garbage fallback).
		res = assembleFilletBody(body, fils, blends)
	}
	res = adoptCornerSetback(body, edges, fils, blends, miters, concave, res) // corner setback (dihedral+trihedral), do-no-harm floor
	if res == nil {
		return nil, fmt.Errorf("fillet: mixed-radius torus corner did not compose into a certified solid")
	}
	rep := validate.Validate(res)
	if rep.Valid && res.IsSolid() {
		return res, nil
	}
	// build-then-certify (#1797): the corner-into-round was BUILT, not rejected up front. Most such
	// junctions close into a valid solid (asymmetric round); only the symmetric equal-radius corner
	// still fails. Name that actionable cause instead of the generic invalid-solid message.
	if e, round := firstCornerIntoRound(edges); round != nil {
		return nil, cornerIntoRoundError(e, round)
	}
	return nil, fmt.Errorf("fillet: result is not a valid solid %v", rep.Issues)
}

// rebuildChoice names one do-no-harm candidate composition of the two independent local
// rebuilds (mid-span obstacle, ADR-4; double-interference runout, ADR-5).
type rebuildChoice int

const (
	chooseBoth     rebuildChoice = iota // obstacle + runout composed into one watertight solid
	chooseObstacle                      // only the obstacle rebuild improves; runout dropped
	chooseRunout                        // only the runout rebuild improves; obstacle dropped
	chooseBaseline                      // neither improves — the pre-rebuild fillet (do-no-harm)
)

// chooseRebuild picks the highest-priority rebuild composition whose assembled body clears the
// do-no-harm bar. {both} wins when the two rebuilds compose watertight; else the best single path
// (obstacle preferred — the older, more-proven path); else baseline. Splitting the ADR-4 verdict so
// a failing runout can never veto a passing obstacle rebuild (M2 whole-branch review, systemic minor).
func chooseRebuild(improved func(rebuildChoice) bool) rebuildChoice {
	for _, c := range []rebuildChoice{chooseBoth, chooseObstacle, chooseRunout} {
		if improved(c) {
			return c
		}
	}
	return chooseBaseline
}

// assembleFilletBody builds the do-no-harm candidate bodies (ADR-4/ADR-5, Option 1, 2026-07-14; split
// into independent obstacle/runout verdicts, M3 whole-branch review) and picks the highest-priority one
// that clears the bar: a local rebuild may FIRE on a body it cannot fully resolve (e.g. a second obstacle
// column it does not model, or a runout that opens the shell), producing a degraded shell. Gating the two
// verdicts independently means a failing runout can never veto a passing obstacle rebuild (and vice
// versa) — only a strict improvement over the baseline (no-rebuild) fillet is ever kept. That baseline
// fallback is the same green body as before ADR-4 (HolesContained is a tripwire, not folded into Valid).
func assembleFilletBody(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend) *topo.Body {
	cands := rebuildCandidates(body, fils, blends) // lazily assembled; chooseBaseline always present
	choice := chooseRebuild(func(c rebuildChoice) bool {
		b, ok := cands[c]
		return ok && obstacleImprovedSolid(b)
	})
	return cands[choice]
}

// rebuildCandidates lazily assembles the do-no-harm candidate bodies: the baseline (no local rebuild) is
// always built; {both}/{obstacle-only}/{runout-only} are built only when that composition's collectors
// actually handle an edge — so the overwhelmingly common body (no obstacle, no runout anywhere) costs a
// SINGLE assembleBody call, the same cost as before this split.
func rebuildCandidates(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend) map[rebuildChoice]*topo.Body {
	cands := map[rebuildChoice]*topo.Body{chooseBaseline: assembleFilletFaces(body, fils, blends, false, false)}
	if _, bothFired := filletResultFaces(body, fils, blends, true, true); !bothFired {
		return cands // neither local rebuild handled any edge: baseline is the only useful candidate
	}
	addRebuildCandidate(cands, chooseBoth, body, fils, blends, true, true)
	addRebuildCandidate(cands, chooseObstacle, body, fils, blends, true, false)
	addRebuildCandidate(cands, chooseRunout, body, fils, blends, false, true)
	return cands
}

// addRebuildCandidate assembles one composition (both, obstacle-only, or runout-only) and records
// it under choice only when that composition's collectors handled an edge on their own.
func addRebuildCandidate(cands map[rebuildChoice]*topo.Body, choice rebuildChoice, body *topo.Body,
	fils []edgeFillet, blends map[uint64]*cornerBlend, enableObstacles, enableRunout bool) {
	if _, fired := filletResultFaces(body, fils, blends, enableObstacles, enableRunout); fired {
		cands[choice] = assembleFilletFaces(body, fils, blends, enableObstacles, enableRunout)
	}
}

// assembleFilletFaces builds and assembles one rebuild composition's faces in a single call. It goes
// through assembleCornerBlendBody (not bare assembleBody) because the planar trihedral path emits the
// same absolute-winding-sensitive corner sphere patch the curved path does: orientFilletShell only
// unifies RELATIVE windings, so a VOID-side corner ball (K6/L4's concave pocket corner) landed wound
// so the sphere-patch mesher filled the 7/8 COMPLEMENT (Ω = 7π/2, area 274.35 vs OCCT's octant
// 39.2699 = 25π/2) at every quality — a +235 area / +522 (= 4πr³/3·mesh) volume mis-measure the 1%
// corpus deps absorbed silently (patchgridcap-report.md §region).
func assembleFilletFaces(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend, enableObstacles, enableRunout bool) *topo.Body {
	faces, _ := filletResultFaces(body, fils, blends, enableObstacles, enableRunout)
	return assembleCornerBlendBody(faces)
}

// obstacleImprovedSolid reports whether an obstacle-rebuilt body is a watertight, hole-contained solid —
// the bar the rebuild must clear to be kept over the baseline fillet.
func obstacleImprovedSolid(res *topo.Body) bool {
	r := validate.Validate(res)
	return r.Valid && res.IsSolid() && r.HolesContained
}
