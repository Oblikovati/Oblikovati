// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"sort"

	"oblikovati.org/kernel/geom"
)

// Co-refinement of a shared section against the RULED WALL's own trim (ADR-0062).
//
// A section shared by a planar face and a ruled wall has to be bounded ONCE, by both trims, before
// either side arranges it. clipSectionToFace already does the planar half, and says why: each side
// would otherwise clip the unbounded curve in its own chart and arrive at the same corner by two
// routes, leaving a T-junction the stitch cannot weld (#3459). The wall half was missing — a section
// that ran off the end of the wall was DECLINED — so a plane that wedges a corner off a cylinder, whose
// section ellipse leaves through the top rim, fell to CSG. That single decline is what every notched
// cylinder in the corpus was built on.
//
// It is solvespace's contract too: MakeIntersectionCurvesAgainst computes the shared curves first, and
// CopySurfacesTrimAgainst then trims BOTH shells against those same curves.

// clipSectionToWall bounds a section curve to the runs of it INSIDE a wall face's own trim, returning
// the exact arcs. ok=false when it does not cross the wall's boundary at all (nothing to share) or an
// arc cannot be rebuilt between its ends.
func clipSectionToWall(cv geom.Curve3, wf curvedFace) ([]geom.Curve3, bool) {
	cuts := sectionWallCuts(cv, wf)
	if len(cuts) < 2 {
		return nil, false
	}
	var out []geom.Curve3
	for _, run := range sectionRuns(cv, cuts) {
		mid := cv.PointAt(sectionParamAt(cv, run.lo+(run.hi-run.lo)/2))
		if !pointInCurvedFace(wf, mid) {
			continue
		}
		arc, got := geom.ConicArcBetween(cv, cv.PointAt(run.lo), cv.PointAt(sectionParamAt(cv, run.hi)), mid)
		if !got {
			return nil, false
		}
		out = append(out, arc)
	}
	return out, len(out) > 0
}

// sectionWallCuts is the sorted set of parameters at which the section meets the edges that actually
// BORDER the wall — its rims, an oblique elliptical edge, the sides of a notch an earlier cut left.
// Each incidence is solved on the wall's surface in closed form, the same solver the chart's
// frame×imprint crossings use, so the two agree on where the corner is.
//
// borderEdges, not every loop edge: a closed side is walked with a SLIT — a ruling out and straight
// back along the same curve — so the loop closes in the parameter plane, and the region lies on BOTH
// sides of it. A slit bounds nothing, so it must not cut the section either; cutting there split the
// one run inside the wall into two arcs meeting at an invented corner. It is the same rule
// pointInCurvedFace follows to decide the side (#3453 follow-up).
func sectionWallCuts(cv geom.Curve3, wf curvedFace) []float64 {
	res := geom.ResolutionForBox(faceLoopBox(wf))
	var cuts []float64
	for _, te := range borderEdges(wf) {
		pts, _ := geom.SectionCrossingCandidates(wf.surface, te.le.curve, cv, res)
		for _, p := range pts {
			if _, onEdge := curveParamWithin(te.le.curve, te.le.t0, te.le.t1, p, res); !onEdge {
				continue
			}
			if t, ok := geom.ConicParamAt(cv, p); ok {
				cuts = append(cuts, t)
			}
		}
	}
	sort.Float64s(cuts)
	return dedupedParams(cuts)
}
