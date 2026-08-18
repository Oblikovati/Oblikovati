// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// BEND RELIEF (#2072). A bend that stops short of the material's edge has to be relieved: without
// a notch at each end, the flat material beside the flange is asked to stretch as the wall folds,
// and it tears. Every flange spanned its whole edge until width extents landed (#1958), which is
// why nothing here needed to cut anything before — and why it does now.
//
// The notch is cut BESIDE the flange, not into it: a slot of the styled width running away from
// the bend's end along the edge, and the styled depth into the parent. Cutting inside the span
// would eat the wall the relief exists to protect.

// ReliefSpec is the styled relief a bend cuts, resolved for one recompute: the shape, and the
// notch's width along the bend line and depth into the parent (cm). It reaches a feature through
// [Input] because the shape is not a number and so cannot ride on a parameter like the sizes do.
type ReliefSpec struct {
	Shape types.ReliefShape
	Width float64
	Depth float64
	// Remnant is the thinnest strip of parent material this relief may leave standing (#1959). A
	// notch that would leave less takes the sliver with it: a strip that thin tears off in handling
	// and is not what anyone drew.
	Remnant float64
}

// BendOptions overrides the style's bend properties for ONE feature (Inventor's BendOptions,
// #1959). A nil field defers to the style, which is what makes it an override rather than a
// restatement — a flange that sets only a relief depth keeps the style's shape and width.
type BendOptions struct {
	ReliefShape         *types.ReliefShape
	ReliefWidth         func() float64
	ReliefDepth         func() float64
	MinimumRemnant      func() float64
	Transition          types.BendTransition
	TransitionArcRadius func() float64
}

// resolve folds the overrides onto the style's relief, leaving anything unset as the style has it.
func (o *BendOptions) resolve(style ReliefSpec) ReliefSpec {
	if o == nil {
		return style
	}
	out := style
	if o.ReliefShape != nil {
		out.Shape = *o.ReliefShape
	}
	for _, f := range []struct {
		src func() float64
		dst *float64
	}{{o.ReliefWidth, &out.Width}, {o.ReliefDepth, &out.Depth}, {o.MinimumRemnant, &out.Remnant}} {
		if f.src != nil {
			*f.dst = f.src()
		}
	}
	return out
}

// cuts reports whether this relief removes material at all. A tear relief is the deliberate
// absence of a cut — the material is left to tear along the bend, which is a real (if rough)
// manufacturing choice, not a missing value.
func (r ReliefSpec) cuts() bool {
	return r.Shape != types.ReliefTear && r.Width > 0 && r.Depth > 0
}

// reliefEnds is where a bend needs relieving: the two ends of its span along the picked edge, and
// which of them stop short of the edge's own ends. An end that reaches the material's boundary has
// nothing to tear and takes no notch.
type reliefEnds struct {
	from, to               float64
	relieveFrom, relieveTo bool
}

// bendReliefEnds decides which ends of a bend spanning [from, to] of an edgeLength-long edge need
// relief. The tolerance keeps a span that reaches the edge within rounding from cutting a notch
// that would hang off the part.
func bendReliefEnds(from, to, edgeLength float64) reliefEnds {
	const tol = 1e-9
	return reliefEnds{
		from: from, to: to,
		relieveFrom: from > tol,
		relieveTo:   to < edgeLength-tol,
	}
}

// cutBendRelief removes the styled relief notches from bodies at the ends of a bend that stop
// short of the edge. It returns the bodies unchanged when the relief cuts nothing.
func cutBendRelief(bodies []*topo.Body, edge *topo.Edge, ends reliefEnds, spec ReliefSpec,
	thickness, edgeLength float64, flip bool, feat string) ([]*topo.Body, error) {
	if !spec.cuts() || (!ends.relieveFrom && !ends.relieveTo) {
		return bodies, nil
	}
	frame, err := reliefFrameOf(edge, flip)
	if err != nil {
		return nil, err
	}
	out := bodies
	for _, n := range reliefNotches(ends, spec, edgeLength) {
		tool := reliefTool(frame, n, spec, thickness, feat)
		if out, err = cutFrom(out, tool); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// reliefNotch is one notch's span along the bend line: [start, end] in edge coordinates.
type reliefNotch struct{ start, end float64 }

// reliefNotches places a notch beside each end that needs relieving, running AWAY from the bend so
// the cut never eats into the wall it protects. edgeLength lets a notch swallow a remnant too thin
// to survive handling (#1959).
func reliefNotches(ends reliefEnds, spec ReliefSpec, edgeLength float64) []reliefNotch {
	var out []reliefNotch
	if ends.relieveFrom {
		out = append(out, swallowRemnant(reliefNotch{ends.from - spec.Width, ends.from}, spec, 0, edgeLength))
	}
	if ends.relieveTo {
		out = append(out, swallowRemnant(reliefNotch{ends.to, ends.to + spec.Width}, spec, 0, edgeLength))
	}
	return out
}

// swallowRemnant widens a notch to the material's edge when the strip it would otherwise leave is
// thinner than the minimum remnant. Leaving the sliver is worse than removing it: it tears off in
// handling, and a part that arrives with a torn edge is not the part that was drawn.
func swallowRemnant(n reliefNotch, spec ReliefSpec, low, high float64) reliefNotch {
	// A zero or negative remnant leaves every strip standing, which falls out of the comparisons
	// below rather than needing a guard of its own.
	if n.start-low < spec.Remnant {
		n.start = low
	}
	if high-n.end < spec.Remnant {
		n.end = high
	}
	return n
}

// reliefFrame is the edge's own frame: where it starts, which way it runs, and the plane of the
// parent face the notch is cut in.
type reliefFrame struct {
	origin math.Point3
	along  math.UnitVector3
	out    math.UnitVector3 // away from the sheet, so −out is into it
	up     math.UnitVector3 // the parent face's outward normal
}

// reliefFrameOf resolves the edge's frame, reusing the flange's own so a notch and the wall it
// relieves cannot disagree about which way the material lies.
func reliefFrameOf(edge *topo.Edge, flip bool) (reliefFrame, error) {
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	along, err := math.UnitVector3FromVector(v0.VectorTo(v1))
	if err != nil {
		return reliefFrame{}, err
	}
	up, out, err := flangeFrame(edge, v0.Midpoint(v1), along, flip)
	if err != nil {
		return reliefFrame{}, err
	}
	return reliefFrame{origin: v0, along: along, out: out, up: up}, nil
}

// reliefTool builds one notch's cutting solid: the notch rectangle in the parent face's plane,
// extruded clean through the material both ways so the cut cannot leave a skin behind.
func reliefTool(fr reliefFrame, n reliefNotch, spec ReliefSpec, thickness float64, feat string) *topo.Body {
	plane := planeFromFrame(fr.origin, fr.up, fr.along)
	into := -1.0 // −out is into the sheet
	if plane.YAxis().AsVector().Dot(fr.out.AsVector()) < 0 {
		into = 1.0 // the plane's Y already points inward
	}
	poly := reliefProfile(n, spec, into)
	// Overshoot the material both ways: the notch is a through cut, and a tool that stopped on
	// the faces would leave coincident surfaces for the boolean to reconcile.
	pad := thickness
	return buildPrism(poly, plane, span{near: -thickness - pad, far: pad}, 0, feat+"/relief")
}

// reliefProfile is the notch outline in the parent face's plane: a rectangle of the relief's
// width and depth, with a rounded inner end for a round relief. u runs along the bend line and v
// into the material.
func reliefProfile(n reliefNotch, spec ReliefSpec, into float64) []math.Point2 {
	depth := spec.Depth * into
	if spec.Shape != types.ReliefRound {
		return ensureCCW2([]math.Point2{
			math.P2(n.start, 0), math.P2(n.end, 0), math.P2(n.end, depth), math.P2(n.start, depth),
		})
	}
	return ensureCCW2(roundedNotchProfile(n, spec.Depth, into))
}

// roundedNotchProfile is the straight notch with its inner end replaced by a half-round of the
// notch's own half-width — the filleted relief, which leaves no inside corner for a crack to start
// in. A notch shallower than its half-width has no straight run left, so the round is all there is.
func roundedNotchProfile(n reliefNotch, depth, into float64) []math.Point2 {
	radius := (n.end - n.start) / 2
	centreU, straight := (n.start+n.end)/2, stdmath.Max(0, depth-radius)
	pts := []math.Point2{math.P2(n.start, 0), math.P2(n.end, 0), math.P2(n.end, straight*into)}
	const arcSteps = 8
	for k := 1; k < arcSteps; k++ { // the far side's corner points are already in the list
		phi := stdmath.Pi * float64(k) / arcSteps
		pts = append(pts, math.P2(centreU+radius*stdmath.Cos(phi), (straight+radius*stdmath.Sin(phi))*into))
	}
	return append(pts, math.P2(n.start, straight*into))
}

// cutFrom subtracts a tool from every running body, dropping any that the cut consumes entirely.
func cutFrom(bodies []*topo.Body, tool *topo.Body) ([]*topo.Body, error) {
	out := make([]*topo.Body, 0, len(bodies))
	for _, b := range bodies {
		res, err := ops.Boolean(ops.Cut, b, tool)
		if err != nil {
			return nil, err
		}
		if res != nil && len(res.Faces()) > 0 {
			out = append(out, res)
		}
	}
	return out, nil
}
