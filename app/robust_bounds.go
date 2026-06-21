// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/math"

// robustExtentMargin is how many "core sizes" beyond a drawing's central extent a point may
// sit before it is treated as a stray and dropped from the framed bounds. It is large so no
// legitimately spread drawing is ever clipped (a uniform run of points keeps its full extent),
// yet far smaller than the thousands-of-times-the-extent gap a genuine off-sheet outlier sits
// at — e.g. a georeferenced DWG's stray monument entities, which would otherwise shrink the
// whole drawing to a sub-pixel dot when Fit frames the full span.
const robustExtentMargin = 16

// robustMinPoints is the count below which framing just uses the exact bounding box: too few
// points to tell a stray from real spread, and small geometry never has the outlier problem.
const robustMinPoints = 16

// robustPointBox frames the bulk of pts, excluding strays that sit far beyond the central
// extent so Fit/Home frame the main drawing instead of shrinking it because a handful of
// entities lie kilometres away. With no strays every point falls inside the window and the
// result is their exact bounding box, so ordinary and legitimately sparse geometry is framed
// exactly as before. The result is the empty box when pts is empty.
func robustPointBox(pts []math.Point3) math.Box {
	if len(pts) < robustMinPoints {
		return math.BoxFromPoints(pts...)
	}
	lo, hi, ok := robustWindow(pts)
	if !ok { // no meaningful spread (margin 0) — frame everything exactly
		return math.BoxFromPoints(pts...)
	}
	box := math.EmptyBox()
	kept := 0
	for _, p := range pts {
		if insideWindow(p, lo, hi) {
			box = box.ExtendPoint(p)
			kept++
		}
	}
	if kept == 0 { // window collapsed (shouldn't happen) — never drop everything
		return math.BoxFromPoints(pts...)
	}
	return box
}

// robustWindow returns the per-axis bounds [median−margin, median+margin] that a point must
// fall within (on every axis) to count toward the framed box. The margin is robustExtentMargin
// times the LARGEST axis core size, so the window scales with the drawing's biggest dimension —
// a long, thin but legitimate drawing keeps all its points, while a point thousands of times
// that size away (a true stray) is excluded. ok is false when there is no spread to bound.
func robustWindow(pts []math.Point3) (lo, hi math.Point3, ok bool) {
	xs, ys, zs := axisValues(pts)
	margin := robustExtentMargin * maxF(coreSize(xs), maxF(coreSize(ys), coreSize(zs)))
	if margin <= 0 {
		return math.Point3{}, math.Point3{}, false
	}
	med := math.P3(math.Median(xs), math.Median(ys), math.Median(zs))
	d := math.V3(margin, margin, margin)
	return med.TranslateBy(d.Scale(-1)), med.TranslateBy(d), true
}

// insideWindow reports whether p lies within the per-axis window [lo,hi].
func insideWindow(p, lo, hi math.Point3) bool {
	return p.X >= lo.X && p.X <= hi.X &&
		p.Y >= lo.Y && p.Y <= hi.Y &&
		p.Z >= lo.Z && p.Z <= hi.Z
}

// axisValues splits points into per-axis coordinate slices (the form the percentile/median
// helpers consume).
func axisValues(pts []math.Point3) (xs, ys, zs []float64) {
	xs, ys, zs = make([]float64, len(pts)), make([]float64, len(pts)), make([]float64, len(pts))
	for i, p := range pts {
		xs[i], ys[i], zs[i] = float64(p.X), float64(p.Y), float64(p.Z)
	}
	return xs, ys, zs
}

// coreSize is the central extent of one axis (its 1st–99th percentile span) — the drawing's
// characteristic size on that axis, immune to the extreme tails a stray entity occupies.
func coreSize(v []float64) float64 {
	return math.Percentile(v, 0.99) - math.Percentile(v, 0.01)
}
