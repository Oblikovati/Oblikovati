// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	stdmath "math"

	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Circle recovery from a view's projection (M14-F04 PBI-144, #391). A hole table tabulates
// drilled holes, which are extrude-cut features — and a cut hole's B-rep rim is NOT an analytic
// geom.Circle edge, so scanning body.Edges() (what centre marks do) misses it. The hidden-line
// projection, however, tessellates every edge into segments tagged by source edge; a hole rim's
// segments form a closed, equidistant loop. So we recover holes by circle-fitting those loops,
// which works for cut holes and primitive cylinders alike.

// projectedCircle is a circular edge recovered from a view's projection: its centre on the view
// plane and radius, both in model centimetres (the projection's units).
type projectedCircle struct {
	center math.Point2
	radius float64
}

const (
	circleFitSectors = 6    // angular bins a loop must all hit to count as a full circle (not an arc)
	circleFitTolFrac = 0.06 // max radius standard-deviation / mean for a loop to read as circular
)

// circlesFromProjection clusters the body's projected edge segments into connected loops (segments
// sharing an endpoint) and fits a circle to each closed, equidistant, fully-revolved loop. It
// clusters by connectivity rather than by source edge because a sketch extrude-cut FACETS a hole
// rim into many separate straight edges — so grouping by edge key would split one rim into dozens
// of one-segment groups. Connectivity reunites the faceted rim, recovering the hole.
func circlesFromProjection(body *topo.Body, basis hlr.View) []projectedCircle {
	segs := hlr.Project(body, basis, ops.DefaultQuality())
	clusters := clusterConnectedSegments(segs)
	var out []projectedCircle
	for _, pts := range clusters {
		if c, ok := fitCircle(pts); ok {
			out = append(out, c)
		}
	}
	return out
}

// quantizeKey snaps a projected point to a grid so segments that share an endpoint (up to
// floating-point noise) hash equal and so connect in the union-find.
func quantizeKey(p math.Point2) [2]int64 {
	const grid = 1e4 // 1/grid cm ≈ 1 µm resolution — finer than any tessellation chord
	return [2]int64{int64(stdmath.Round(float64(p.X) * grid)), int64(stdmath.Round(float64(p.Y) * grid))}
}

// endpointUnionFind groups quantized projected points into connected sets.
type endpointUnionFind struct{ parent map[[2]int64][2]int64 }

func newEndpointUnionFind() *endpointUnionFind {
	return &endpointUnionFind{parent: map[[2]int64][2]int64{}}
}

// find returns the representative of k's set, inserting k as its own root on first sight.
func (u *endpointUnionFind) find(k [2]int64) [2]int64 {
	p, ok := u.parent[k]
	if !ok {
		u.parent[k] = k
		return k
	}
	if p != k {
		u.parent[k] = u.find(p)
	}
	return u.parent[k]
}

// union merges the sets of a and b.
func (u *endpointUnionFind) union(a, b [2]int64) { u.parent[u.find(a)] = u.find(b) }

// clusterConnectedSegments groups segment endpoints into connected components via union-find on
// shared (quantized) endpoints, returning each component's points. A hole rim — even faceted into
// many edges — is one connected component; the outer silhouette is another.
func clusterConnectedSegments(segs []hlr.Segment) [][]math.Point2 {
	uf := newEndpointUnionFind()
	pointOf := map[[2]int64]math.Point2{}
	for _, s := range segs {
		ka, kb := quantizeKey(s.A), quantizeKey(s.B)
		pointOf[ka], pointOf[kb] = s.A, s.B
		uf.union(ka, kb)
	}
	groups := map[[2]int64][]math.Point2{}
	for k, p := range pointOf {
		groups[uf.find(k)] = append(groups[uf.find(k)], p)
	}
	out := make([][]math.Point2, 0, len(groups))
	for _, pts := range groups {
		out = append(out, pts)
	}
	return out
}

// fitCircle fits a circle to a loop of projected points, returning it only when the points lie at a
// near-constant radius from their centroid and revolve fully around it (so a straight edge or a
// partial arc is rejected). The centroid is the centre; the mean radius is the radius.
func fitCircle(pts []math.Point2) (projectedCircle, bool) {
	if len(pts) < circleFitSectors {
		return projectedCircle{}, false
	}
	cx, cy := centroid(pts)
	mean, stddev := radiusStats(pts, cx, cy)
	if mean < 1e-6 || stddev/mean > circleFitTolFrac {
		return projectedCircle{}, false
	}
	if !revolvesFully(pts, cx, cy) {
		return projectedCircle{}, false
	}
	return projectedCircle{center: math.P2(math.Scalar(cx), math.Scalar(cy)), radius: mean}, true
}

// centroid returns the mean position of the points.
func centroid(pts []math.Point2) (cx, cy float64) {
	for _, p := range pts {
		cx, cy = cx+float64(p.X), cy+float64(p.Y)
	}
	n := float64(len(pts))
	return cx / n, cy / n
}

// radiusStats returns the mean and standard deviation of the points' distances from (cx, cy).
func radiusStats(pts []math.Point2, cx, cy float64) (mean, stddev float64) {
	var sum, sumSq float64
	for _, p := range pts {
		r := stdmath.Hypot(float64(p.X)-cx, float64(p.Y)-cy)
		sum, sumSq = sum+r, sumSq+r*r
	}
	n := float64(len(pts))
	mean = sum / n
	variance := sumSq/n - mean*mean
	if variance < 0 {
		variance = 0 // guard floating-point noise below zero
	}
	return mean, stdmath.Sqrt(variance)
}

// revolvesFully reports whether the points hit every angular sector around (cx, cy), so a full
// circle passes but a partial arc (which leaves sectors empty) does not.
func revolvesFully(pts []math.Point2, cx, cy float64) bool {
	var hit [circleFitSectors]bool
	for _, p := range pts {
		ang := stdmath.Atan2(float64(p.Y)-cy, float64(p.X)-cx) + stdmath.Pi
		idx := int(ang / (2 * stdmath.Pi) * circleFitSectors)
		if idx >= circleFitSectors {
			idx = circleFitSectors - 1
		}
		hit[idx] = true
	}
	for _, h := range hit {
		if !h {
			return false
		}
	}
	return true
}
