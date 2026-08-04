// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// The canal-aware HOST retrims (M6' C4 W3b, architecture: canal-armweld-architecture.md §"W3 addendum —
// per-host bite-loop composition"). Every CORNER roll host of the canal (N7: the cylinder WALL R=50, the
// two planes, and the s_10 boss R=5) is re-clipped by ONE generalized canalHostBite (fillet_curved_canal_
// bite.go), which composes each host's inner bite from the arms' already-built host rails (shared-edge
// identity) bridged by the host's canal foot-locus when it has one — superseding W3's foot-locus-only
// retrimCanalHost, which could bridge only the wall and honest-declined because the foot-locus ENDPOINTS
// (the arm-rail junctions W0/W1, ~37u interior to the wall band) are not on the host's own loop. W3b's
// insight: it is the arm rails' OUTER ends that anchor on the host loop, not the foot-locus endpoints.
//
// Every FAR-RUNOUT bitten host (the y=0 cut, the exit caps) is spliced by the EXISTING farArcsBiting/
// farRunoutFace path VERBATIM (canalFarOrPassthrough) — the same leaf functions the single-ball
// curvedHostFace calls, so the two produce byte-identical far faces (ADR: verbatim reuse).

// canalHostFaces retrims every body face for the canal corner (the canal analogue of curvedHostFaces): a
// face any arm rolls on, or that carries a canal foot-locus, is a CORNER host and is re-clipped by
// canalHostBite; every far-runout bitten host is spliced by the EXISTING farArcsBiting/farRunoutFace path
// VERBATIM; any face neither touches passes through transformFace unchanged. Declines with a diagnostic
// reason (carrying the offending host + junction/anchor gap) on any retrim failure. rolls is the
// CanalCorner.Rolls payload (rolls[0]=wall, rolls[1]=s_10 surface) that tags the foot-loci to their hosts.
func canalHostFaces(body *topo.Body, w cornerWeld, boundaries canalBoundaries, bundles []canalArmBundle, rolls []geom.Surface, res Resolution) ([]filletFace, string) {
	tol := res.Weld() * w.radius
	out := make([]filletFace, 0, len(body.Faces()))
	for _, f := range body.Faces() {
		ff, reason := canalHostFace(f, boundaries, bundles, rolls, w, res, tol)
		if reason != "" {
			return nil, reason
		}
		out = append(out, ff)
	}
	return out, ""
}

// canalHostFace routes one host face to its treatment: a CORNER host — one that any arm rolls on (it has
// collected arm rails) OR that a canal foot-locus lies on (rolls identity) — is re-clipped by the
// generalized canalHostBite; every other face takes the verbatim far-runout / pass-through path. The
// predicate (rails OR foot-locus) covers all four N7 corner hosts uniformly: the wall (2 rails + feet[0]),
// each plane (2 rails, no foot-locus), and the s_10 boss (0 rails + feet[1]).
func canalHostFace(f *topo.Face, boundaries canalBoundaries, bundles []canalArmBundle, rolls []geom.Surface, w cornerWeld, res Resolution, tol float64) (filletFace, string) {
	_, hasBridge := footLocusForHost(f, boundaries, rolls, tol)
	if len(armRailsOnHost(f, bundles)) > 0 || hasBridge {
		return canalHostBite(f, bundles, boundaries, rolls, w, res)
	}
	return canalImprintFace(f, bundles, tol)
}

// farBundles projects the per-arm canal bundles onto the far-only armRails view the verbatim far-runout
// leaf functions (farArcsBiting / farRunoutFace) consume — so the far-runout branch stays byte-identical
// to the single-ball path (it reads only .far).
func farBundles(bundles []canalArmBundle) []armRails {
	out := make([]armRails, len(bundles))
	for i, b := range bundles {
		out[i] = armRails{far: b.far}
	}
	return out
}

// The far-end host imprints (the notch faces the arm termini bite) are retrimmed by canalImprintFace
// (fillet_curved_canal_imprint.go, F2), which splices each arm's terminal — or its extension edge +
// terminal when the wall-side foot runs off the loop (derivation §5) — and passes untouched faces through
// transformFace. It supersedes the W3 armRails-only far-runout branch (which could not carry the extension
// edges); farBundles remains the far-only projection the shared-edge identity tests read.
