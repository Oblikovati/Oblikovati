// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
)

// The fillet pipeline: what runs between a caller's picks and an assembled body (split out of
// fillet.go for #2217).
//
// One pass resolves the picks, solves the corners, computes a blend per edge and assembles the
// result. The widened retry above it is a SECOND, differently-scoped attempt at the same pipeline
// rather than a classified strategy of its own — a known ground-rule violation tracked by #3412
// and #3413, recorded here so the shape is visible where it lives.

func filletEdgesCornerRec(body *topo.Body, picks []EdgeFilletRadii, corner CornerStrategy, concave ConcaveFill, rec *diag.Recorder) (*topo.Body, error) {
	// Try the caller's OWN selection first, completely unwidened — byte-identical to the pipeline
	// this function ran before pick propagation existed. Every existing capability (a lone rim/arc
	// pick, ADR-0050 P6's deliberate open-run SUBSET of a longer chain, a single curved-neighbour
	// edge's specific decline, an ordinary planar corner build or its own #1797 build-then-certify
	// check) already resolves — successfully or with its own actionable cause — on the raw picks, so
	// trying them first can never regress a currently-passing case.
	res, rawErr := runFilletPipeline(body, picks, corner, concave, rec)
	if rawErr == nil {
		return res, nil
	}
	if res, err := tryWidenedFilletPipeline(body, picks, corner, concave, rec); err == nil {
		return res, nil
	}
	return nil, rawErr // prefer the caller's OWN selection's cause over a synthetic widened one
}

// tryWidenedFilletPipeline is filletEdgesCornerRec's OCCT-parity fallback: a pick seeds its whole
// tangent-continuous spine (PerformElement, ChFi3d_Builder::PerformElement), the single-click case
// D8/F2 exemplify (one edge of an 8-edge loop; DRAWEXE fillets all 18 faces of the loop) — but ONLY
// once the raw selection has already failed to resolve on its own (see filletEdgesCornerRec):
// widening unconditionally BEFORE that attempt regressed TestFilletOpenCurvedTangentStripe (P6's
// open 3-edge subset of an 8-edge closed loop propagated to the WHOLE loop), TestFilletEdgesRoutesArc
// (a lone arc-cap pick chained onto the box's own perimeter before loneArcPick saw it as a
// singleton), and TestFilletOnCurvedSeamRejectsClearly / TestFilletIntoExistingRoundRejectedHonestly_1797
// (a single already-curved or already-rounded-into edge widened into a multi-edge corner solve that
// fails with an unrelated internal cause instead of its own specific, actionable one) — all
// bisected against dee581df, the last verified commit, which passed every one of them.
func tryWidenedFilletPipeline(body *topo.Body, picks []EdgeFilletRadii, corner CornerStrategy, concave ConcaveFill, rec *diag.Recorder) (*topo.Body, error) {
	widened := expandPicksAlongTangentSpines(body, picks)
	return runFilletPipeline(body, widened, corner, concave, rec)
}

// runFilletPipeline is the fillet dispatch + build pipeline for picks exactly as given: the three
// closed-form curved routes (lone rim, lone arc, curved tangent chain), falling through to the
// generic planar corner solve.
func runFilletPipeline(body *topo.Body, picks []EdgeFilletRadii, corner CornerStrategy, concave ConcaveFill, rec *diag.Recorder) (*topo.Body, error) {
	if b, err, ok := dispatchCurvedPick(body, picks); ok {
		return b, err
	}
	edges, err := resolveFilletPicks(body, picks)
	if err != nil {
		return nil, err
	}
	switch corner {
	case CornerRound:
		edges = roundThirdEdges(edges) // fillet the third edge at constant radius → 3-edge sphere
	case CornerSetback:
		edges = setbackThirdEdges(edges) // taper the third edge (r→0 run-out) → smooth set-back sphere
	}
	return filletResolvedEdges(body, edges, concave, rec)
}

// dispatchCurvedPick tries the three closed-form curved routes — a lone rim pick, a lone arc pick,
// or a caller-selected curved tangent chain — against picks exactly as given. handled=false when
// none matches, so the caller can widen and retry rather than treating "no curved shape yet" as an
// error.
func dispatchCurvedPick(body *topo.Body, picks []EdgeFilletRadii) (result *topo.Body, err error, handled bool) {
	if rim := loneRimPick(body, picks); rim != nil {
		b, e := FilletCylinderRim(body, rim.Key, rim.R0) // a circular cylinder/cap rim → toroidal band
		return b, e, true
	}
	if arc := loneArcPick(body, picks); arc != nil {
		b, e := FilletCylinderArc(body, arc.Key, arc.R0) // a cylinder/cap arc → torus + setback end-caps
		return b, e, true
	}
	if chain, r, closed, ok := curvedTangentChain(body, picks); ok {
		// A closed mixed tangent loop (#1797) rounds as one continuous stripe; a contiguous open run
		// (ADR-0050 P6) rounds the same way but terminates in a flat setback cap at each end.
		b, e := filletTangentStripe(body, chain, closed, r)
		return b, e, true
	}
	return nil, nil, false
}

// filletResolvedEdges solves the corners and edge fillets of an already-resolved pick list and
// assembles the validated result body. Round/setback corners have already been reduced to 3-edge
// sphere blends by augmenting the third edge, so the corner solver only ever sees miters and blends.
func filletResolvedEdges(body *topo.Body, edges []filletPick, concave ConcaveFill, rec *diag.Recorder) (*topo.Body, error) {
	if err := validateFilletRadii(edges, concave); err != nil {
		return nil, err // #1800: reject an over-large radius before it self-intersects
	}
	blends, miters, err := computeCorners(body, edges)
	if err != nil {
		return nil, err
	}
	fils, err := computeFillets(body, edges, blends, miters, concave, rec)
	if err != nil {
		return nil, err
	}
	if curvedArmFils(fils) {
		return weldCurvedArmOrFloor(body, fils, blends, miters) // M5 Slice A weld or the do-no-harm floor
	}
	return assemblePlanarFilletBody(body, edges, fils, blends, miters, concave)
}
