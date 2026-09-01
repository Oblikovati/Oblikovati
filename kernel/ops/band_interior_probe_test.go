// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"
)

// The wrapping-band probe (bandInteriorUV, Oblikovati/Oblikovati#2247, #3447). A face whose loops WRAP the
// parameter seam — a bore wall, a cone band, a cylinder tunnel — is not a closed polygon in the plane, so
// the even-odd grid cannot be asked about it and the per-face boolean certificate has to be handed a
// CONSTRUCTED point instead. The construction that stepped 5% of a boundary chord inward along that
// chord's normal put the probe a hair off the boundary, where it could sit outside the true trim while the
// loops' sampled polygon still called it inside: the gate then disproved a correct body from a point it
// had guessed, and a five-face blind hole fell to a 1830-face faceted rescue.
//
// The invariant is therefore not "the probe is inside" — the bad probe passed that — but "the probe is
// DEEP inside", far enough that two independent classifiers cannot legitimately disagree about it. These
// tests hold that line on the construction directly, where the boolean corpus gates only its consequences.

// bandProbeTestVLo/VHi are the reference band's rim stations in the parameter that CLOSES; it runs one
// full turn in the parameter that wraps, as a bore wall does.
const (
	bandProbeTestVLo = 2.0
	bandProbeTestVHi = 8.0
)

// bandProbeLoops is a bore wall in parameter space: two rim loops at v = lo and v = hi, each running a
// full turn in u without returning to its start, which is exactly what makes loopsWrapASeam true.
func bandProbeLoops(vLo, vHi float64, samplesPerRim int) []faceLoop {
	return []faceLoop{bandProbeRim(vLo, samplesPerRim), bandProbeRim(vHi, samplesPerRim)}
}

// bandProbeRim is one full-turn rim at a fixed v, sampled in u over [0, 2π].
func bandProbeRim(v float64, n int) faceLoop {
	samples := make([]arcSample, n)
	for i := range samples {
		u := 2 * stdmath.Pi * float64(i) / float64(n-1)
		samples[i] = arcSample{t: float64(i), u: u, v: v}
	}
	return faceLoop{edges: []loopEdge{{samples: samples}}, netU: 2 * stdmath.Pi}
}

// TestWrappingBandProbeSitsMidBand: the probe must land halfway between the two rims, which is the
// deepest point of the band at its station — not a fraction of a sampling chord off one rim, where the
// depth shrinks with every extra sample the discretizer takes.
func TestWrappingBandProbeSitsMidBand(t *testing.T) {
	t.Parallel()
	loops := bandProbeLoops(bandProbeTestVLo, bandProbeTestVHi, 64)
	_, v, ok := bandInteriorUV(loops)
	if !ok {
		t.Fatal("bandInteriorUV declined a plain two-rim bore wall")
	}
	want := (bandProbeTestVLo + bandProbeTestVHi) / 2
	if stdmath.Abs(v-want) > 1e-9 {
		t.Errorf("probe v=%.9f, want the mid-band %.9f — a probe near a rim is one the trim and the "+
			"sampled polygon may disagree about (#2247)", v, want)
	}
}

// TestWrappingBandProbeDepthIsIndependentOfSampling is the property the old chord-step construction could
// not have: refining the boundary discretization must not push the probe towards the boundary. The step
// was a fraction of a CHORD, so doubling the samples halved the depth; the mid-band midpoint does not move.
func TestWrappingBandProbeDepthIsIndependentOfSampling(t *testing.T) {
	t.Parallel()
	var first float64
	for i, n := range []int{16, 64, 256} {
		_, v, ok := bandInteriorUV(bandProbeLoops(bandProbeTestVLo, bandProbeTestVHi, n))
		if !ok {
			t.Fatalf("%d samples per rim: bandInteriorUV declined", n)
		}
		depth := stdmath.Min(v-bandProbeTestVLo, bandProbeTestVHi-v)
		if i == 0 {
			first = depth
			continue
		}
		if stdmath.Abs(depth-first) > 1e-9 {
			t.Errorf("%d samples per rim: probe depth %.9f, want %.9f — depth must not follow the "+
				"discretization (#2247)", n, depth, first)
		}
	}
}

// TestWrappingBandProbeDeclinesAThicknesslessStation: a band with both rims at the same v has no interior,
// and the probe must say so rather than return a point on the boundary itself.
func TestWrappingBandProbeDeclinesAThicknesslessStation(t *testing.T) {
	t.Parallel()
	if _, _, ok := bandInteriorUV(bandProbeLoops(bandProbeTestVLo, bandProbeTestVLo, 64)); ok {
		t.Error("bandInteriorUV claimed an interior point on a band of zero thickness")
	}
}
