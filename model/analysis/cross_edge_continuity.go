// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cross-edge continuity measurement (M36-F13) — the quantitative companion to the visual
// interrogation overlay (F12) and the numeric acceptance oracle for the continuity-construction
// features (match/blend/bridge/extend). Given two surfaces meeting along a shared edge, it samples
// the edge and reports, at each sample, the positional gap (G0), the angle between the surface
// normals (G1), and the difference in normal curvature across the boundary (G2). Built purely on
// the existing surface evaluators (geom.SurfaceCurvatures et al.), so it is deterministic.

// EdgeParam maps a normalized edge parameter t ∈ [0,1] to a surface's (u, v) coordinates — how the
// shared edge is traced on one of the two surfaces.
type EdgeParam func(t float64) (u, v float64)

// ContinuitySample is the cross-edge deviation at one point along the shared edge.
type ContinuitySample struct {
	T         float64 // normalized edge parameter
	Gap       float64 // G0: positional gap between the two surface points (length)
	NormalDeg float64 // G1: angle between the two surface normals, folded to [0,90] (degrees)
	CurvDiff  float64 // G2: absolute normal-curvature difference in the cross-boundary direction (1/length)
	CurvPct   float64 // G2: that difference as a percent of the larger curvature
}

// ContinuityReport is the along-edge continuity measurement and its aggregates. The samples carry
// the colored along-edge plot the UI draws; the Max/Avg fields are the pass/fail numbers.
type ContinuityReport struct {
	Samples                    []ContinuitySample
	MaxGap, AvgGap             float64
	MaxNormalDeg, AvgNormalDeg float64
	MaxCurvDiff, AvgCurvDiff   float64
	MaxCurvPct, AvgCurvPct     float64
}

// CrossEdgeContinuity samples the shared edge (samples ≥ 2 points) and reports the G0/G1/G2
// deviation between surfaces a and b along it. ea and eb trace the same edge on each surface.
//
// Example: rep := CrossEdgeContinuity(faceA, faceB, edgeOnA, edgeOnB, 25); if rep.MaxNormalDeg < 0.1 { tangent }.
func CrossEdgeContinuity(a, b geom.Surface, ea, eb EdgeParam, samples int) ContinuityReport {
	if samples < 2 {
		samples = 2
	}
	rep := ContinuityReport{Samples: make([]ContinuitySample, samples)}
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(samples-1)
		rep.Samples[i] = continuityAt(a, b, ea, eb, t)
	}
	aggregate(&rep)
	return rep
}

// continuityAt measures the three deviations at one edge parameter.
func continuityAt(a, b geom.Surface, ea, eb EdgeParam, t float64) ContinuitySample {
	ua, va := ea(t)
	ub, vb := eb(t)
	pa, pb := a.PointAt(ua, va), b.PointAt(ub, vb)
	na, nb := a.NormalAt(ua, va), b.NormalAt(ub, vb)
	te := edgeTangent(a, ea, t)
	ka := normalCurvatureCross(a, ua, va, na, te)
	kb := alignCurvature(normalCurvatureCross(b, ub, vb, nb, te), na, nb)
	diff := stdmath.Abs(ka - kb)
	return ContinuitySample{
		T:         t,
		Gap:       float64(pa.DistanceTo(pb)),
		NormalDeg: foldedAngleDeg(na, nb),
		CurvDiff:  diff,
		CurvPct:   percentOf(diff, ka, kb),
	}
}

// edgeTangent estimates the unit edge tangent at t from a central finite difference of a's edge
// trace (clamped at the ends).
func edgeTangent(a geom.Surface, ea EdgeParam, t float64) math.Vector3 {
	const h = 1e-4
	t0, t1 := stdmath.Max(0, t-h), stdmath.Min(1, t+h)
	u0, v0 := ea(t0)
	u1, v1 := ea(t1)
	return normalize(a.PointAt(u0, v0).VectorTo(a.PointAt(u1, v1)))
}

// normalCurvatureCross returns the surface's normal curvature in the cross-boundary direction
// (perpendicular to the edge within the tangent plane), via Euler's formula on the principal
// curvatures.
func normalCurvatureCross(s geom.Surface, u, v float64, n, edgeTan math.Vector3) float64 {
	cross := normalize(n.Cross(edgeTan)) // tangent-plane direction perpendicular to the edge
	maxDir, kMax, kMin := geom.SurfaceCurvatures(s, u, v)
	if float64(maxDir.Length()) < 1e-12 {
		return kMax // degenerate frame: principal curvatures are equal (or zero)
	}
	cosA := float64(normalize(maxDir).Dot(cross))
	cos2 := cosA * cosA
	return kMax*cos2 + kMin*(1-cos2) // Euler: κ(α) = kMax·cos²α + kMin·sin²α
}

// alignCurvature flips kb's sign when the two surface normals oppose, so both curvatures are
// expressed with respect to the same physical bending sense before differencing.
func alignCurvature(kb float64, na, nb math.Vector3) float64 {
	if na.Dot(nb) < 0 {
		return -kb
	}
	return kb
}

// foldedAngleDeg returns the angle between two normals in degrees, folded to [0, 90] so an opposite
// (but parallel-plane) orientation reads as 0 — tangent-plane agreement, not normal-vector equality.
func foldedAngleDeg(na, nb math.Vector3) float64 {
	deg := float64(normalize(na).AngleTo(normalize(nb))) * 180 / stdmath.Pi
	if deg > 90 {
		deg = 180 - deg
	}
	return deg
}

// percentOf expresses a curvature difference as a percent of the larger magnitude (0 when both are
// ~flat).
func percentOf(diff, ka, kb float64) float64 {
	denom := stdmath.Max(stdmath.Abs(ka), stdmath.Abs(kb))
	if denom < 1e-12 {
		return 0
	}
	return diff / denom * 100
}

// aggregate fills the report's max/avg fields from its samples.
func aggregate(rep *ContinuityReport) {
	n := float64(len(rep.Samples))
	for _, s := range rep.Samples {
		rep.MaxGap = stdmath.Max(rep.MaxGap, s.Gap)
		rep.MaxNormalDeg = stdmath.Max(rep.MaxNormalDeg, s.NormalDeg)
		rep.MaxCurvDiff = stdmath.Max(rep.MaxCurvDiff, s.CurvDiff)
		rep.MaxCurvPct = stdmath.Max(rep.MaxCurvPct, s.CurvPct)
		rep.AvgGap += s.Gap / n
		rep.AvgNormalDeg += s.NormalDeg / n
		rep.AvgCurvDiff += s.CurvDiff / n
		rep.AvgCurvPct += s.CurvPct / n
	}
}
