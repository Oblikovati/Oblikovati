// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The constant-thickness CROSS-SECTION a folded sheet-metal wall is extruded from.
//
// A flange is one bend and one straight run. A hem can be several: a double hem folds twice, and a
// teardrop curls past half a turn and then runs back to the sheet (#1956). So the section is traced
// as a CHAIN of (bend, run) steps along the material's CENTRELINE, with the two surfaces half a
// thickness either side. Tracing the centreline is what lets a step curl the other way — a double
// hem's second fold must stack its leg ON TOP of the first, not curl back down into the parent —
// because then the two surfaces simply swap which one is inside the bend, with no bookkeeping.
//
// The whole section lives in the plane spanned by out (along the sheet, away from the edge) and up
// (the parent face's outward normal), so the frame is just an angle in that plane. A flange is
// written here as a one-step chain rather than kept as a second implementation of the same arc.

// bendRun is one step of a folded section: a bend through Angle at inside radius Radius, followed
// by a straight Run. Either part may be zero — a rolled hem is all bend, and a zero-angle step is a
// plain straight run. A NEGATIVE angle curls the opposite way (see the double hem). A run is never
// negative: material that starts behind the edge is a SHIFTED section, not a backwards run, which
// would double the band back over the bend that follows it (#1957).
type bendRun struct {
	Angle  float64
	Radius float64
	Run    float64
}

// bandTracer walks the centreline of the folded material through the section plane.
type bandTracer struct {
	out, up math.Vector3
	at      math.Vector3 // centreline position
	theta   float64      // heading, measured from out toward up
	half    float64      // half the material thickness
}

// dir / left are the tracer's heading and the direction a positive bend curls toward.
func (b bandTracer) dir() math.Vector3  { return b.rot(b.out) }
func (b bandTracer) left() math.Vector3 { return b.rot(b.up) }

// rot turns a vector of the section plane by the tracer's heading (out toward up).
func (b bandTracer) rot(v math.Vector3) math.Vector3 {
	c, s := stdmath.Cos(b.theta), stdmath.Sin(b.theta)
	a, d := v.Dot(b.out), v.Dot(b.up)
	return b.out.Scale(float64(a)*c - float64(d)*s).Add(b.up.Scale(float64(a)*s + float64(d)*c))
}

// surfaces returns the material's two surface points at the current centreline position.
func (b bandTracer) surfaces() (l, r math.Vector3) {
	off := b.left().Scale(b.half)
	return b.at.Add(off), b.at.Add(off.Scale(-1))
}

// bandPolygon returns the closed 2D section of a chain of bends and runs, projected by proj. The
// material leaves the edge along out with up as the parent face's outward normal, so the section
// starts flush with the parent sheet's own cross-section and a positive bend curls up over it.
func bandPolygon(steps []bendRun, out, up math.Vector3, thickness float64,
	proj func(math.Vector3) math.Point2) []math.Point2 {
	b := &bandTracer{out: out, up: up, at: up.Scale(-thickness / 2), half: thickness / 2}
	l, r := b.surfaces()
	sideL, sideR := []math.Point2{proj(l)}, []math.Point2{proj(r)}
	for _, s := range steps {
		ls, rs := b.step(s, proj)
		sideL, sideR = append(sideL, ls...), append(sideR, rs...)
	}
	poly := append([]math.Point2(nil), sideL...)
	for k := len(sideR) - 1; k >= 0; k-- {
		poly = append(poly, sideR[k])
	}
	return ensureCCW2(poly)
}

// step advances the tracer through one bend and its straight run, returning the surface points it
// sweeps (the starting pair is already in the caller's lists).
func (b *bandTracer) step(s bendRun, proj func(math.Vector3) math.Point2) (sideL, sideR []math.Point2) {
	if s.Angle != 0 && s.Radius > 0 {
		sideL, sideR = b.bend(s, proj)
	}
	if s.Run <= 0 {
		return sideL, sideR
	}
	b.at = b.at.Add(b.dir().Scale(s.Run))
	l, r := b.surfaces()
	return append(sideL, proj(l)), append(sideR, proj(r))
}

// bend sweeps the tracer through one bend, faceting the arc at flangeFacetStep. The centre sits
// Radius+half to the left for a positive angle and the same distance to the right for a negative
// one, so the INSIDE radius is Radius whichever way the material curls.
func (b *bandTracer) bend(s bendRun, proj func(math.Vector3) math.Point2) (sideL, sideR []math.Point2) {
	sign, sweep := 1.0, s.Angle
	if s.Angle < 0 {
		sign, sweep = -1, -s.Angle
	}
	reach := sign * (s.Radius + b.half)
	centre := b.at.Add(b.left().Scale(reach))
	start, steps := b.theta, int(stdmath.Max(2, stdmath.Round(sweep/flangeFacetStep)))
	for k := 1; k <= steps; k++ {
		b.theta = start + sign*sweep*float64(k)/float64(steps)
		b.at = centre.Add(b.left().Scale(-reach))
		l, r := b.surfaces()
		sideL, sideR = append(sideL, proj(l)), append(sideR, proj(r))
	}
	b.theta = start + s.Angle
	b.at = centre.Add(b.left().Scale(-reach))
	return sideL, sideR
}
