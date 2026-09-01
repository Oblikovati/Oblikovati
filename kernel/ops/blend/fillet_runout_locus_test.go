// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestContactLocusRailLiesOnItsOwnBandLoft proves the railB fix at its root: on the real S1 flank
// bands (surf-rst — the synthesised locus), the contact-locus rail must lie ON the band's own canal
// loft everywhere, because contactLocusRail now builds the loft's OWN boundary row
// (geom.CanalFootLocusRail) instead of the degree-1 polyline through the 7 node feet whose chords
// sagged off the patch (S1 4.157e-05·diag, S9 7.927e-04·diag — railb-locus-report.md).
func TestContactLocusRailLiesOnItsOwnBandLoft(t *testing.T) {
	t.Parallel()
	tl := s1Tiling(t)
	for name, band := range map[string]*runoutBand{"left": tl.left, "right": tl.right} {
		rc := band.payload()
		surf, err := geom.LoftCanalStations(rc.Centers, rc.FeetA, rc.FeetB, tl.cyl.Radius, 1e-6)
		if err != nil {
			t.Fatalf("%s flank: LoftCanalStations: %v", name, err)
		}
		worst := 0.0
		lo, hi := band.railB.Domain()
		for i := 0; i <= 100; i++ {
			v := lo + (hi-lo)*float64(i)/100
			if d := float64(band.railB.PointAt(v).DistanceTo(surf.PointAt(1, v))); d > worst {
				worst = d
			}
		}
		if worst > 1e-9 {
			t.Errorf("%s flank: railB leaves its own band loft by %.3e (want <= 1e-9) — "+
				"the locus is off the very patch it bounds (the polyline-chord defect)", name, worst)
		}
	}
}

// TestContactLocusRailInterpolatesStationContacts proves the rail passes THROUGH every solved
// station contact (no smoothing): each station's exact footB sits on the rail at that station's
// chord-length parameter over the centres — the parameterisation the loft shares.
func TestContactLocusRailInterpolatesStationContacts(t *testing.T) {
	t.Parallel()
	tl := s1Tiling(t)
	for name, band := range map[string]*runoutBand{"left": tl.left, "right": tl.right} {
		params := centreChordParams(band.stations)
		worst := 0.0
		for j, st := range band.stations {
			if d := float64(band.railB.PointAt(params[j]).DistanceTo(st.footB)); d > worst {
				worst = d
			}
		}
		if worst > 1e-9 {
			t.Errorf("%s flank: railB misses a solved contact by %.3e (want <= 1e-9) — "+
				"the locus no longer interpolates the exact tangency feet", name, worst)
		}
	}
}

// centreChordParams is the loft's chord-length parameterisation over the station centres
// (P&T §9.2.1, the same rule geom's alphaParams(…, 1) applies), recomputed here so the test
// asserts the SHARED parameterisation rather than trusting the implementation under test.
func centreChordParams(sts []runoutStation) []float64 {
	cum := make([]float64, len(sts))
	for k := 1; k < len(sts); k++ {
		cum[k] = cum[k-1] + float64(sts[k-1].centre.DistanceTo(sts[k].centre))
	}
	out := make([]float64, len(sts))
	for k := range out {
		out[k] = cum[k] / cum[len(sts)-1]
	}
	out[len(out)-1] = 1
	return out
}

// TestPlainContactDetourCarriesLocusSubSpans is the consumer-side wiring gate: the ONE-boss plain
// face's notch detour must hand every locus segment its own sub-span of the interpolated rail (the
// same TrimmedCurve3 value the patch's loop offers), never a bare chord — a nil there would make
// the edge catalog record a nil-vs-curve adoption on every locus edge (the ratchet's floor is 2)
// and ship the patch's offer under an orientation the notch never checked.
func TestPlainContactDetourCarriesLocusSubSpans(t *testing.T) {
	t.Parallel()
	band, from, to := syntheticLocusBand(t)
	tl := setbackTiling{mid: band, weld: 1e-9,
		bCutLo: curveStart(band.railB), bCutHi: curveEnd(band.railB)}
	segs, ok := plainContactDetour(tl)(from, to)
	if !ok {
		t.Fatalf("plainContactDetour declined")
	}
	if len(segs) != ringSegSamples+2 {
		t.Fatalf("got %d segs, want %d (entry + %d locus samples + exit)", len(segs), ringSegSamples+2, ringSegSamples)
	}
	for i := 1; i <= ringSegSamples; i++ {
		assertSegCarriesOwnSubSpan(t, segs, i)
	}
}

// assertSegCarriesOwnSubSpan checks locus seg i carries a curve spanning exactly its own two
// endpoints, with its mid-station ON the band's rail (not the chord).
func assertSegCarriesOwnSubSpan(t *testing.T, segs []notchSeg, i int) {
	t.Helper()
	seg := segs[i]
	if seg.curve == nil {
		t.Fatalf("locus seg %d carries no curve — the notch still declines the boundary it shares with the patch", i)
	}
	lo, hi := seg.curve.Domain()
	next := segs[i+1].pt
	if d := float64(seg.curve.PointAt(lo).DistanceTo(seg.pt)); d > 1e-12 {
		t.Errorf("locus seg %d start: curve endpoint %v is %.3e from the segment's own %v", i, seg.curve.PointAt(lo), d, seg.pt)
	}
	if d := float64(seg.curve.PointAt(hi).DistanceTo(next)); d > 1e-12 {
		t.Errorf("locus seg %d end: curve endpoint %v is %.3e from the segment's own %v", i, seg.curve.PointAt(hi), d, next)
	}
	mid := seg.curve.PointAt(lo + (hi-lo)/2)
	chord := seg.pt.Lerp(next, 0.5)
	if float64(mid.DistanceTo(chord)) < 1e-12 {
		t.Errorf("locus seg %d mid-station equals its chord midpoint — the sub-span is a chord, not the rail", i)
	}
}

// syntheticLocusBand builds a minimal surf-rst band whose contacts walk a parabola in the z=0
// plane (curved, so a chord is distinguishable from the rail), plus detour entry/exit points.
func syntheticLocusBand(t *testing.T) (*runoutBand, math.Point3, math.Point3) {
	t.Helper()
	const n, r = 73, 2.0
	sts := make([]runoutStation, n)
	for i := range n {
		x := -3 + 6*float64(i)/float64(n-1)
		y := 0.1 * x * x
		sts[i] = runoutStation{
			s:      float64(i),
			centre: math.P3(math.Scalar(x), math.Scalar(y), r),
			footA:  math.P3(math.Scalar(x), math.Scalar(y-r), r),
			footB:  math.P3(math.Scalar(x), math.Scalar(y), 0),
		}
	}
	rail, ok := contactLocusRail(sts)
	if !ok {
		t.Fatalf("contactLocusRail declined the synthetic band")
	}
	band := &runoutBand{stations: sts, railB: rail}
	return band, math.P3(-5, 0, 0), math.P3(5, 0, 0)
}

// TestReverseCurve3MirrorsTheInterpolatedLocus guards the direction seam orientedLocus depends on:
// reversing the (degree-3, non-uniform-knot) locus must trace the SAME curve by value backwards —
// reversed(t) == forward(lo+hi−t) — so a host notch entering from the far corner reads the same
// locus the patch does. (The degree-1 branch keeps its exact control-point reversal.)
func TestReverseCurve3MirrorsTheInterpolatedLocus(t *testing.T) {
	t.Parallel()
	band, _, _ := syntheticLocusBand(t)
	rev := reverseCurve3(band.railB)
	lo, hi := band.railB.Domain()
	worst := 0.0
	for i := 0; i <= 100; i++ {
		f := float64(i) / 100
		d := float64(rev.PointAt(lo + f*(hi-lo)).DistanceTo(band.railB.PointAt(hi - f*(hi-lo))))
		if d > worst {
			worst = d
		}
	}
	if worst > 1e-12 {
		t.Errorf("reversed locus departs the mirrored forward locus by %.3e (want <= 1e-12) — "+
			"the two traversal directions no longer read one curve by value", worst)
	}
	if stdmath.Abs(float64(curveStart(rev).DistanceTo(curveEnd(band.railB)))) > 1e-12 {
		t.Errorf("reversed locus does not start at the forward locus's end")
	}
}
