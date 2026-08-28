// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"sort"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// Loft skinning — RAIL and GUIDE handling (M48 #2236 split of loft_skin.go). Warps the aligned sections
// toward the loft's rails, mapping curves, area-graph stops, and centreline guide: each guide re-tracks
// the section points toward the guide with a falloff, and area stops scale the section radii. Section
// alignment lives in loft_skin.go; the surface fit in loft_skin_fit.go.

// railGuide deforms the densified sections so the loft follows guide rails (the kLoftWithRails
// mode): each rail pulls the nearest point-track onto it, with a smooth angular falloff to the
// neighbouring tracks, so the loft bulges toward the rail BETWEEN sections while the end sections
// stay put (the rail's ends are snapped to the section corner it tracks). Rails are model-space
// polylines; multiple rails' pulls sum. A no-op without rails.
func railGuide(sections [][]math.Point3, rails [][]math.Point3) [][]math.Point3 {
	levels := len(sections)
	if len(rails) == 0 || levels < 2 {
		return sections
	}
	n := len(sections[0])
	// Every rail's pull is measured against the ORIGINAL sections and the pulls are summed before
	// being applied, so multiple rails don't cross-talk (a sequential apply would let one rail's
	// deformation skew the next rail's measured displacement).
	disp := make([][]math.Vector3, levels)
	for s := range disp {
		disp[s] = make([]math.Vector3, n)
	}
	for _, rail := range rails {
		if len(rail) < 2 {
			continue
		}
		samples := resamplePath(rail, levels)
		jr := nearestTrack(sections[0], samples[0])
		samples[0], samples[levels-1] = sections[0][jr], sections[levels-1][jr] // pin the ends
		for s := range levels {
			d := sections[s][jr].VectorTo(samples[s])
			for j := range n {
				disp[s][j] = disp[s][j].Add(d.Scale(railFalloff(j, jr, n)))
			}
		}
	}
	for s := range levels {
		for j := range n {
			sections[s][j] = sections[s][j].TranslateBy(disp[s][j])
		}
	}
	return sections
}

// mapAlign sets the point correspondence. With no map curves it falls back to the automatic
// minimum-twist alignment (alignSections). With explicit map curves (the MapPointCurves mode) the
// first curve gives one anchor point per section; each section is rotated so the vertex nearest its
// anchor becomes index 0, so the anchors correspond across sections — letting the user pin (or
// deliberately twist) the correspondence the auto-alignment would otherwise choose.
func mapAlign(sections [][]math.Point3, mapCurves [][]math.Point3) [][]math.Point3 {
	if len(mapCurves) == 0 || len(mapCurves[0]) == 0 {
		return alignSections(sections)
	}
	mc := mapCurves[0]
	out := make([][]math.Point3, len(sections))
	for i, sec := range sections {
		if i >= len(mc) {
			out[i] = sec
			continue
		}
		out[i] = rotateLoop(sec, nearestTrack(sec, mc[i]))
	}
	return out
}

// areaGraphScale resizes each densified section about its centroid so the loft's cross-sectional
// area follows the area graph (the kLoftWithAreaGraphSections mode): at longitudinal fraction t the
// section is scaled by sqrt(graph(t)), so its area becomes graph(t)× the blended area. The end
// sections (t=0,1) are pinned to scale 1 so the user's profiles are preserved. No-op without stops.
func areaGraphScale(sections [][]math.Point3, stops []types.LoftAreaStop) [][]math.Point3 {
	levels := len(sections)
	if len(stops) == 0 || levels < 2 {
		return sections
	}
	graph := pinnedAreaStops(stops)
	for s := 1; s < levels-1; s++ { // ends stay scale 1 (the real sections)
		k := stdmath.Sqrt(areaGraphAt(float64(s)/float64(levels-1), graph))
		if k == 1 {
			continue
		}
		c := centroidOf(sections[s])
		for j := range sections[s] {
			sections[s][j] = c.TranslateBy(c.VectorTo(sections[s][j]).Scale(k))
		}
	}
	return sections
}

// pinnedAreaStops returns the stops sorted by T with implicit scale-1 endpoints added at T=0/1 (so
// the graph pins to the unscaled end sections unless the user overrode those positions).
func pinnedAreaStops(stops []types.LoftAreaStop) []types.LoftAreaStop {
	out := append([]types.LoftAreaStop(nil), stops...)
	has := func(t float64) bool {
		for _, s := range out {
			if stdmath.Abs(s.T-t) < 1e-9 {
				return true
			}
		}
		return false
	}
	if !has(0) {
		out = append(out, types.LoftAreaStop{T: 0, Scale: 1})
	}
	if !has(1) {
		out = append(out, types.LoftAreaStop{T: 1, Scale: 1})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].T < out[j].T })
	return out
}

// areaGraphAt linearly interpolates the area scale at fraction t over the (sorted, end-pinned)
// stops; clamped to the first/last stop outside their range.
func areaGraphAt(t float64, sorted []types.LoftAreaStop) float64 {
	if t <= sorted[0].T {
		return sorted[0].Scale
	}
	last := sorted[len(sorted)-1]
	if t >= last.T {
		return last.Scale
	}
	for i := 1; i < len(sorted); i++ {
		if t <= sorted[i].T {
			a, b := sorted[i-1], sorted[i]
			f := (t - a.T) / (b.T - a.T)
			return a.Scale + f*(b.Scale-a.Scale)
		}
	}
	return last.Scale
}

// centerlineGuide bends the loft's spine (the kLoftWithCenterline mode): each densified
// sub-section is translated bodily so its centroid lies on the centerline curve, so the loft
// follows a curved path while keeping its cross-sections. Unlike a rail (which pulls one side with
// falloff), the whole section moves. The centerline's ends are pinned to the end-section
// centroids. A no-op without a centerline.
func centerlineGuide(sections [][]math.Point3, centerline []math.Point3) [][]math.Point3 {
	levels := len(sections)
	if len(centerline) < 2 || levels < 2 {
		return sections
	}
	samples := resamplePath(centerline, levels)
	samples[0], samples[levels-1] = centroidOf(sections[0]), centroidOf(sections[levels-1]) // pin ends
	for s := range levels {
		d := centroidOf(sections[s]).VectorTo(samples[s])
		for j := range sections[s] {
			sections[s][j] = sections[s][j].TranslateBy(d)
		}
	}
	return sections
}

// nearestTrack returns the index of the section point closest to p.
func nearestTrack(section []math.Point3, p math.Point3) int {
	best, bestD := 0, stdmath.Inf(1)
	for j, q := range section {
		if d := float64(q.DistanceTo(p)); d < bestD {
			best, bestD = j, d
		}
	}
	return best
}

// railFalloff weights a rail's pull on track j: 1 at the rail's track jr, falling on a raised
// cosine to 0 at the diametrically opposite track (a localized, smooth bulge).
func railFalloff(j, jr, n int) float64 {
	d := j - jr
	if d < 0 {
		d = -d
	}
	if n-d < d {
		d = n - d
	}
	frac := float64(d) / (float64(n) / 2)
	if frac >= 1 {
		return 0
	}
	return 0.5 * (1 + stdmath.Cos(stdmath.Pi*frac))
}

// resamplePath returns count points spaced evenly by arc length along an OPEN polyline (endpoints
// included). A zero-length path yields count copies of its first point.
func resamplePath(poly []math.Point3, count int) []math.Point3 {
	if count < 2 || len(poly) < 2 {
		return poly
	}
	segLen := make([]float64, len(poly)-1)
	total := 0.0
	for i := range segLen {
		segLen[i] = float64(poly[i].DistanceTo(poly[i+1]))
		total += segLen[i]
	}
	out := make([]math.Point3, count)
	if total == 0 {
		for k := range out {
			out[k] = poly[0]
		}
		return out
	}
	out[0], out[count-1] = poly[0], poly[len(poly)-1]
	step, seg, acc := total/float64(count-1), 0, 0.0
	for k := 1; k < count-1; k++ {
		target := step * float64(k)
		for seg < len(segLen)-1 && acc+segLen[seg] < target {
			acc += segLen[seg]
			seg++
		}
		f := 0.0
		if segLen[seg] > 0 {
			f = (target - acc) / segLen[seg]
		}
		out[k] = poly[seg].TranslateBy(poly[seg].VectorTo(poly[seg+1]).Scale(f))
	}
	return out
}
