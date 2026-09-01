// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Trimmed curved-face tessellation — the QUALITY / DEFLECTION POLICY (M48 #2224 split of
// tessellate_trim.go). The fallback path for a face whose trim can't be reduced to a conforming iso
// rectangle: sample the whole (clamped) UV domain, but choose the grid lines adaptively so the mesh
// resolves curvature wherever it occurs — unioned over several isoparms, not a single mid-line (#1412).

// fullDomainGridMesh samples the surface over its whole (clamped) UV domain — the fallback
// for a face whose trim can't be reduced to a conforming iso rectangle.
func fullDomainGridMesh(s geom.Surface, q Quality) *Mesh {
	uLo, uHi := clampSpan(s.UDomain())
	vLo, vHi := clampSpan(s.VDomain())
	us := unionIsoparmParams(s, uLo, uHi, vLo, vHi, true, q)
	vs := unionIsoparmParams(s, uLo, uHi, vLo, vHi, false, q)
	return closedDomainMesh(s, us, vs) // watertight on a closed surface (periodic seam + poles); else == gridMesh
}

// isoparmSampleFractions are the fixed-direction positions at which unionIsoparmParams samples the
// varying direction — the ends, quarters and middle, enough to catch a curvature feature anywhere
// across the domain that a single mid-line would miss.
var isoparmSampleFractions = []float64{0, 0.25, 0.5, 0.75, 1}

// unionIsoparmParams returns the adaptive parameter breakpoints of the VARYING direction, unioned over
// SEVERAL isoparms of the fixed direction instead of a single mid-line — so the grid resolves curvature
// wherever it occurs in the domain (a torus tube, an off-centre bump), not only where the mid-line
// happens to look (#1412). A uniformly-curved surface yields the same breakpoints on every isoparm, so
// the union equals the old single-line result and nothing is densified (no flat-face regression).
func unionIsoparmParams(s geom.Surface, uLo, uHi, vLo, vHi float64, alongU bool, q Quality) []float64 {
	lo, hi, fixedLo, fixedHi := uLo, uHi, vLo, vHi
	if !alongU {
		lo, hi, fixedLo, fixedHi = vLo, vHi, uLo, uHi
	}
	var merged []float64
	for _, frac := range isoparmSampleFractions {
		fixed := fixedLo + frac*(fixedHi-fixedLo)
		eval := func(t float64) math.Point3 {
			if alongU {
				return s.PointAt(t, fixed)
			}
			return s.PointAt(fixed, t)
		}
		merged = mergeSortedParams(merged, adaptiveParams(eval, lo, hi, q.Tol(), q.AngleTol()))
	}
	return merged
}

// mergeSortedParams merges two ascending parameter slices into one, dropping a value within a tiny
// fraction of the span of one already kept (so unioning many isoparms does not pile near-coincident
// grid lines that would make sliver cells).
func mergeSortedParams(a, b []float64) []float64 {
	if len(a) == 0 {
		return b
	}
	out := append([]float64(nil), a...)
	out = append(out, b...)
	sort.Float64s(out)
	span := out[len(out)-1] - out[0]
	eps := span * 1e-9 // tol:numeric (relative to the parameter span)
	deduped := out[:1]
	for _, v := range out[1:] {
		if v-deduped[len(deduped)-1] > eps {
			deduped = append(deduped, v)
		}
	}
	return deduped
}
