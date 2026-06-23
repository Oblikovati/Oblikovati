// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Surface deviation / gap analysis (M36-F14): sample a surface on a (u,v) grid and report the SIGNED
// distance from each sample to a reference — another surface, or a point set (an imported master or a
// scan cloud). The signed field drives a colour deviation map; the summary (min/max/RMS, out-of-
// tolerance count) is the numeric gate Class-A work checks a panel against its master and its
// neighbours with. Sign is positive on the reference's outward-normal side (surface case) or along
// the sampled surface's own normal (point-set case).

// DeviationSample is one grid sample: its (u,v), model-space point, and signed distance to the
// reference (the nearest reference point, signed by side).
type DeviationSample struct {
	U, V     float64
	Point    math.Point3
	Distance float64
}

// DeviationReport is a deviation map plus its summary statistics. Min/Max are signed extremes; AbsMax
// is the largest magnitude; RMS is the root-mean-square distance. UCount×VCount is the grid.
type DeviationReport struct {
	Samples        []DeviationSample
	Min, Max       float64
	AbsMax, RMS    float64
	UCount, VCount int
}

// OutOfTolerance counts samples whose |distance| exceeds tol — the out-of-band points a map highlights.
func (r DeviationReport) OutOfTolerance(tol float64) int {
	n := 0
	for _, s := range r.Samples {
		if stdmath.Abs(s.Distance) > tol {
			n++
		}
	}
	return n
}

// SurfaceDeviationToSurface samples src on a uCount×vCount grid and reports the signed distance from
// each sample to the target surface (nearest point via the target's on-surface inverse, signed by the
// target's outward normal). Deviation of an exact offset copy equals the offset.
func SurfaceDeviationToSurface(src, target geom.Surface, uCount, vCount int) DeviationReport {
	return gridDeviation(src, uCount, vCount, func(p math.Point3) float64 {
		tu, tv := target.ParamAt(p)
		near := target.PointAt(tu, tv)
		v := near.VectorTo(p)
		d := float64(v.Length())
		if v.Dot(target.NormalAt(tu, tv)) < 0 {
			d = -d
		}
		return d
	})
}

// SurfaceDeviationToPoints samples src and reports the signed distance from each sample to the
// nearest point in pts (a scan cloud / imported master), signed along src's own normal at the sample.
// Returns an empty report when pts is empty.
func SurfaceDeviationToPoints(src geom.Surface, pts []math.Point3, uCount, vCount int) DeviationReport {
	if len(pts) == 0 {
		return DeviationReport{UCount: uCount, VCount: vCount}
	}
	return gridDeviationAt(src, uCount, vCount, func(p math.Point3, u, v float64) float64 {
		near, best := pts[0], stdmath.Inf(1)
		for _, q := range pts {
			if d := float64(p.DistanceTo(q)); d < best {
				best, near = d, q
			}
		}
		signed := best
		if near.VectorTo(p).Dot(src.NormalAt(u, v)) < 0 {
			signed = -signed
		}
		return signed
	})
}

// gridDeviation samples src's [0,1]² grid and applies dist(point) at each sample.
func gridDeviation(src geom.Surface, uCount, vCount int, dist func(math.Point3) float64) DeviationReport {
	return gridDeviationAt(src, uCount, vCount, func(p math.Point3, _, _ float64) float64 { return dist(p) })
}

// gridDeviationAt samples src's [0,1]² grid and applies dist(point,u,v), assembling the report.
func gridDeviationAt(src geom.Surface, uCount, vCount int, dist func(math.Point3, float64, float64) float64) DeviationReport {
	if uCount < 2 {
		uCount = 2
	}
	if vCount < 2 {
		vCount = 2
	}
	r := DeviationReport{UCount: uCount, VCount: vCount, Min: stdmath.Inf(1), Max: stdmath.Inf(-1)}
	var sumSq float64
	for i := 0; i < uCount; i++ {
		u := float64(i) / float64(uCount-1)
		for j := 0; j < vCount; j++ {
			v := float64(j) / float64(vCount-1)
			p := src.PointAt(u, v)
			d := dist(p, u, v)
			r.Samples = append(r.Samples, DeviationSample{U: u, V: v, Point: p, Distance: d})
			r.Min, r.Max = stdmath.Min(r.Min, d), stdmath.Max(r.Max, d)
			if a := stdmath.Abs(d); a > r.AbsMax {
				r.AbsMax = a
			}
			sumSq += d * d
		}
	}
	if n := len(r.Samples); n > 0 {
		r.RMS = stdmath.Sqrt(sumSq / float64(n))
	}
	return r
}
