// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
)

// Ruled-side arrangement support (M2 Phase-1, Oblikovati/Oblikovati#1375, generalised #1405). What remains
// of the original single-valued boundary walk after the general (u,v)-arrangement trimmer
// (curved_halfspace_uv_arrangement.go) replaced it: the few helpers the arrangement still leans on — the
// wrapping test that picks the band orientation convention, the parameter unwrap that keeps a re-emitted
// conic arc on one monotone run, and the edge-chain reversal.

// wrapsAllU reports whether the kept v-interval is non-empty at every azimuth (the band wraps the seam).
// orientLoops uses it to gate the top-rim reversal: a wrapping band's pure top-rim hi loop is reversed,
// a non-wrapping tongue's mixed loop is not.
func (c ruledUV) wrapsAllU() bool {
	if c.pinched && c.keepsInsideOther() {
		// The Steinmetz intersect (and a cut's tool-side bite) keeps the two LOBES, which touch every azimuth
		// (the kept v-interval collapses to a point at the pinch azimuths) but are two disconnected lobes, NOT
		// a band — so the band convention and the wrapping-tube emission must never fire. A cut/join keeps the
		// OUTSIDE instead: two saddle-rimmed bands that GENUINELY wrap the azimuth, so those fall through to the
		// natural test below and take the wrapping-band emission like every other ruled cut/join (#1403).
		return false
	}
	for i := 0; i < 720; i++ {
		u := 2 * stdmath.Pi * float64(i) / 720
		if c.solidMode {
			if !c.anyKeptVSolid(u) { // general curved∩curved: kept at this azimuth iff some v is inside (#1403)
				return false
			}
			continue
		}
		if _, _, ok := c.keptV(u); !ok {
			return false
		}
	}
	return true
}

// unwrapParamNear shifts x by whole turns so it lands within ±0.5 of ref — turning a wrapped [0,1)
// parameter sequence into a monotone run, so a re-emitted full-ellipse section arc passes through the
// interior point rather than the complementary arc.
func unwrapParamNear(ref, x float64) float64 {
	for x-ref > 0.5 {
		x--
	}
	for x-ref < -0.5 {
		x++
	}
	return x
}

// reverseEdgeChain reverses a chain of loop edges (order reversed, each reversed).
func reverseEdgeChain(chain []loopEdge) []loopEdge {
	out := make([]loopEdge, len(chain))
	for i, e := range chain {
		out[len(chain)-1-i] = reverseEdge(e)
	}
	return out
}
