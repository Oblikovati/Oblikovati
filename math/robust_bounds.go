// SPDX-License-Identifier: GPL-2.0-only

package math

// robustExtentMargin is how many "core sizes" beyond a point set's central extent a point may
// sit before it is treated as a stray and dropped. It is large so no legitimately spread set is
// ever clipped (a uniform run of points keeps its full extent), yet far smaller than the
// thousands-of-times-the-extent gap a genuine off-sheet outlier sits at — e.g. a georeferenced
// DWG's stray monument entities, which would otherwise shrink the whole drawing to a sub-pixel
// dot when Fit frames the full span (and inflate the viewport far plane).
const robustExtentMargin = 16

// robustMinPoints is the count below which the exact bounding box is used: too few points to
// tell a stray from real spread, and small geometry never has the outlier problem.
const robustMinPoints = 16

// RobustPointBox boxes the bulk of pts, excluding strays that sit far beyond the central extent.
// It is used to frame a drawing (Fit/Home) and to size the viewport far plane so a handful of
// entities lying kilometres away neither shrink the framed drawing to a dot nor blow the far
// plane out (which degenerates the skybox projection). With no strays every point falls inside
// the window and the result is their exact bounding box, so ordinary and legitimately sparse
// geometry is unaffected. The result is the empty box when pts is empty.
//
// Example:
//
//	box := RobustPointBox(samplePoints) // frames the drawing, ignoring off-sheet strays
func RobustPointBox(pts []Point3) Box {
	if len(pts) < robustMinPoints {
		return BoxFromPoints(pts...)
	}
	lo, hi, ok := robustWindow(pts)
	if !ok { // no meaningful spread (margin 0) — box everything exactly
		return BoxFromPoints(pts...)
	}
	box := EmptyBox()
	kept := 0
	for _, p := range pts {
		if insideWindow(p, lo, hi) {
			box = box.ExtendPoint(p)
			kept++
		}
	}
	if kept == 0 { // window collapsed (shouldn't happen) — never drop everything
		return BoxFromPoints(pts...)
	}
	return box
}

// robustWindow returns the per-axis bounds [median−margin, median+margin] that a point must fall
// within (on every axis) to count toward the box. The margin is robustExtentMargin times the
// LARGEST axis core size, so the window scales with the set's biggest dimension — a long, thin
// but legitimate drawing keeps all its points, while a point thousands of times that size away
// (a true stray) is excluded. ok is false when there is no spread to bound.
func robustWindow(pts []Point3) (lo, hi Point3, ok bool) {
	xs, ys, zs := axisValues(pts)
	margin := robustExtentMargin * max(coreSize(xs), max(coreSize(ys), coreSize(zs)))
	if margin <= 0 {
		return Point3{}, Point3{}, false
	}
	med := P3(Median(xs), Median(ys), Median(zs))
	d := V3(margin, margin, margin)
	return med.TranslateBy(d.Scale(-1)), med.TranslateBy(d), true
}

// insideWindow reports whether p lies within the per-axis window [lo,hi].
func insideWindow(p, lo, hi Point3) bool {
	return p.X >= lo.X && p.X <= hi.X &&
		p.Y >= lo.Y && p.Y <= hi.Y &&
		p.Z >= lo.Z && p.Z <= hi.Z
}

// axisValues splits points into per-axis coordinate slices (the form the percentile/median
// helpers consume).
func axisValues(pts []Point3) (xs, ys, zs []float64) {
	xs, ys, zs = make([]float64, len(pts)), make([]float64, len(pts)), make([]float64, len(pts))
	for i, p := range pts {
		xs[i], ys[i], zs[i] = float64(p.X), float64(p.Y), float64(p.Z)
	}
	return xs, ys, zs
}

// coreSize is the central extent of one axis (its 1st–99th percentile span) — the set's
// characteristic size on that axis, immune to the extreme tails a stray occupies.
func coreSize(v []float64) float64 {
	return Percentile(v, 0.99) - Percentile(v, 0.01)
}
