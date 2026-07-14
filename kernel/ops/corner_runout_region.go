// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// runoutRegion bundles one or more runoutImprints (and their solved imprintCuts) whose spine
// intervals overlap on the fillet cylinder's axis — i.e. they interfere in the SAME span of the
// fillet band and must be treated as one coupled hole, not independent runouts. S1's two bosses
// (one per host plane, both crossing the same short stretch of the shared edge) is the exact
// double-interference case this exists for (plan docs/superpowers/plans/2026-07-14-curved-
// runout-imprint-fillet.md, advisor pitfall 5): tiling two overlapping single-boss holes
// independently would produce self-intersecting patches, so Task 9's tiler needs the coupling
// made explicit here first. loEdge/hiEdge is the merged interval in the cylinder's axial
// parameter (see spineParam), spanning every constituent imprint's cut.
type runoutRegion struct {
	imprints       []runoutImprint
	cuts           []imprintCut
	loEdge, hiEdge float64
}

// detectRunoutRegions finds S1-shaped multi-crossing runouts: it runs detectRunouts to find each
// host plane's candidate imprint, solves each one's exact circle∩band cut (skipping any that
// don't solve — e.g. an ellipse/b-spline footprint, Task 9/12's scope per solveImprint), and
// clusters the results by their spine-axis overlap. A region with a single imprint is a plain,
// uncoupled runout (not this milestone's tiling target, but still returned rather than dropped —
// callers that only care about the double-interference case filter on len(imprints)>=2).
//
// Example: S1's r8 top boss and r6 front boss both dip into the shared edge's fillet band across
// roughly the same x-range, so their two imprints merge into one runoutRegion spanning both cuts.
func detectRunoutRegions(ef edgeFillet, res Resolution) []runoutRegion {
	imprints := detectRunouts(ef, res)
	clusters := make([]spineCluster, 0, len(imprints))
	for _, im := range imprints {
		cut, ok := solveImprint(im, res)
		if !ok {
			continue
		}
		lo, hi := spineInterval(cut, ef.cyl)
		clusters = append(clusters, spineCluster{im: im, cut: cut, lo: lo, hi: hi})
	}
	return mergeSpineClusters(clusters, res.Weld())
}

// spineCluster is one imprint/cut pair with its projected spine-axis interval — the unit
// mergeSpineClusters sorts and merges into runoutRegions.
type spineCluster struct {
	im     runoutImprint
	cut    imprintCut
	lo, hi float64
}

// spineParam is a point's signed position along the fillet cylinder's axis: the projection of
// (p - cyl.Origin) onto cyl.AxisDir, i.e. Cylinder.PointAt's v parameter. This is the shared ruler
// spineInterval and Task 9's tiler both measure runout extent against.
func spineParam(p math.Point3, cyl geom.Cylinder) float64 {
	return float64(cyl.Origin.VectorTo(p).Dot(cyl.AxisDir.AsVector()))
}

// spineInterval projects a cut's two crossing points onto the fillet spine and returns the
// resulting [lo,hi] interval (lo <= hi regardless of which crossing is axially first).
func spineInterval(cut imprintCut, cyl geom.Cylinder) (lo, hi float64) {
	a, b := spineParam(cut.pMinus, cyl), spineParam(cut.pPlus, cyl)
	if a > b {
		a, b = b, a
	}
	return a, b
}

// mergeSpineClusters sorts clusters by their interval's low edge and merges any whose spine
// intervals overlap, touch, or sit within one weld tolerance of each other into a single
// runoutRegion — the double-interference grouping detectRunoutRegions exists to compute.
// weld is the fillet's model-relative merge slack (res.Weld()): the two bosses' cuts are
// independently solved conics, so their spine intervals can disagree by ULP-level float noise
// even when they truly touch, and an exact `c.lo <= hiEdge` compare would nondeterministically
// split or join them on that noise. Intervals separated by MORE than weld stay disjoint,
// singleton regions.
func mergeSpineClusters(clusters []spineCluster, weld float64) []runoutRegion {
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].lo < clusters[j].lo })
	var out []runoutRegion
	for _, c := range clusters {
		if n := len(out); n > 0 && c.lo <= out[n-1].hiEdge+weld {
			out[n-1] = extendRegion(out[n-1], c)
			continue
		}
		out = append(out, runoutRegion{
			imprints: []runoutImprint{c.im}, cuts: []imprintCut{c.cut}, loEdge: c.lo, hiEdge: c.hi,
		})
	}
	return out
}

// extendRegion folds one more overlapping cluster into an already-open region, widening the
// merged interval to cover the new cluster too.
func extendRegion(r runoutRegion, c spineCluster) runoutRegion {
	r.imprints = append(r.imprints, c.im)
	r.cuts = append(r.cuts, c.cut)
	if c.hi > r.hiEdge {
		r.hiEdge = c.hi
	}
	return r
}
