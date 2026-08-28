// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Inventor .ipt part translator — the REVOLVE feature builder (M48 #2231 split of part.go). Detecting a
// revolve profile + axis from the decoded sketches or the node graph, and building the revolve (kernel or
// profile path).

// revolveBinds reports whether RevolveProfile finds a valid closed profile + axis in this sketch
// set — the gate that decides whether a revolve can be built from it at all.
func revolveBinds(sketches []ipt.Sketch) bool {
	_, ok := ipt.RevolveProfile(sketches)
	return ok
}

// verticalLine reports whether a decoded line is vertical (a revolve centreline is drawn upright).
func verticalLine(l ipt.Line) bool   { return math.Abs(l.A.X-l.B.X) < 1e-4 }
func horizontalLine(l ipt.Line) bool { return math.Abs(l.A.Y-l.B.Y) < 1e-4 }

// axisAlignedLine reports whether a line runs along X or Y — the orientations a revolve centreline
// takes in this corpus (a shaft turned about a vertical or a horizontal axis). An oblique axis is
// not recognised (none observed), so it declines to the mesh rather than guess.
func axisAlignedLine(l ipt.Line) bool { return verticalLine(l) || horizontalLine(l) }

// revolveAxisIndex returns the centreline's index in s.Lines. It prefers the isolated line
// RevolveAxisLine finds (both endpoints shared with no other line — the clean shaft encoding); when
// that is ambiguous — a real INTERNAL edge whose ends land on other edges' interiors also reads as
// isolated (PressureRoller's x=0.8 splitter) — it falls back to the single VERTICAL CONSTRUCTION
// line, which a revolve draws as its centreline. ok=false when neither names one unambiguous axis.
func revolveAxisIndex(s ipt.Sketch) (int, bool) {
	if ai, ok := ipt.RevolveAxisLine(s); ok && ai < len(s.Lines) && axisAlignedLine(s.Lines[ai]) {
		return ai, true
	}
	axis, n := -1, 0
	for i, l := range s.Lines {
		if s.LineIsConstruction(i) && axisAlignedLine(l) {
			axis, n = i, n+1
		}
	}
	return axis, n == 1
}

// graphRevolveCandidate reports whether the node-graph sketches hold a plausible revolve the ipt
// line-ring gate can't confirm: a vertical centreline inside a many-line profile. Such a profile
// (a shaft with a keyway/notch, or a stepped roller split by an internal edge) is a valid region the
// KERNEL arranges but ipt.isClosedRing (a naive head-to-tail line walk) rejects. When this holds,
// extractSketches keeps the graph so buildRevolve's kernel fallback can try it.
func graphRevolveCandidate(sketches []ipt.Sketch) bool {
	for _, s := range sketches {
		if !s.Resolved || len(s.Lines) < 4 {
			continue
		}
		if _, ok := revolveAxisIndex(s); ok {
			return true
		}
	}
	return false
}

// profileOneSideOfAxis reports whether every profile-line endpoint lies on one side of the
// axis-aligned centreline at axisIdx — a valid solid of revolution never crosses its axis. The signed
// distance is measured perpendicular to the axis: across X for a vertical centreline, across Y for a
// horizontal one (a shaft turned about the X axis, e.g. a bushing). The ipt equivalent is unexported.
func profileOneSideOfAxis(lines []ipt.Line, axisIdx int) bool {
	ax := lines[axisIdx]
	vertical := verticalLine(ax)
	ref := ax.A.X
	if !vertical {
		ref = ax.A.Y
	}
	sign := 0
	for i, l := range lines {
		if i == axisIdx {
			continue
		}
		for _, p := range []ipt.Point2D{l.A, l.B} {
			d := p.X - ref
			if !vertical {
				d = p.Y - ref
			}
			if math.Abs(d) < 1e-6 {
				continue
			}
			s := 1
			if d < 0 {
				s = -1
			}
			if sign == 0 {
				sign = s
			} else if sign != s {
				return false
			}
		}
	}
	return sign != 0
}

// tryKernelRevolve builds a revolve the ipt gate declined by trusting the KERNEL's arranged profiles
// instead of the line-ring walk. It fires only after RevolveProfile fails, so no currently-building
// revolve reaches it. Guards keep it honest: an unambiguous vertical centreline (see
// revolveAxisIndex), at least one CLOSED arranged profile, and the profile geometry wholly one side
// of the axis. Every closed profile one side of the axis is revolved and unioned (a stepped roller's
// internal edge splits one region into two, both part of the same solid). Returns the ids of the
// features it added (empty when no sketch qualifies) so the caller can drop them if they don't close
// to a solid.
// A preferred sketch index >= 0 (from ipt.RevolveProfileSketch — the profile the Revolution feature
// actually names) revolves EXACTLY that sketch: a machined part's cut profiles are never revolved by
// mistake. If the named profile can't form a revolve here, it declines rather than guess another
// sketch. preferred < 0 keeps the scan (used on the incidence line set, whose indices don't map to
// the node graph the reference is keyed on).
func tryKernelRevolve(def *compdef.PartComponentDefinition, seg []byte, placed []placedSketch, emitted []emittedSketch, preferred int, axis revolveAxisRef) []feature.ID {
	if preferred >= 0 {
		if preferred < len(emitted) {
			return revolveSketchAt(def, seg, placed, emitted, preferred, axis)
		}
		return nil
	}
	for i := range emitted {
		if ids := revolveSketchAt(def, seg, placed, emitted, i, axis); ids != nil {
			return ids
		}
	}
	return nil
}

// revolveSketchAt revolves the single sketch at index i if it forms a valid revolve (an unambiguous
// axis-aligned centreline, at least one closed arranged profile, all geometry one side of the axis),
// returning the added feature ids or nil when it does not qualify. When the geometric heuristic can't
// name the centreline but the feature's decoded axis reference (axis) can, the collinear profile edge
// it points at is used — the case where the axis is an ordinary edge, neither isolated nor
// construction (CapstainMotorCap turns about its y=0 top edge).
func revolveSketchAt(def *compdef.PartComponentDefinition, seg []byte, placed []placedSketch, emitted []emittedSketch, i int, axis revolveAxisRef) []feature.ID {
	if emitted[i].sk == nil || i >= len(placed) {
		return nil
	}
	s := placed[i].geom
	ai, ok := revolveAxisIndex(s)
	if !ok && axis.ok {
		ai, ok = axisLineFromReference(s, axis)
	}
	if !ok || ai >= len(emitted[i].lines) || emitted[i].lines[ai] == nil {
		return nil
	}
	closed := closedProfileIndices(emitted[i].sk)
	if len(closed) == 0 || !profileOneSideOfAxis(s.Lines, ai) {
		return nil
	}
	// A HORIZONTAL centreline forces a full turn: the partial-angle decode (soleSweepAngle) is
	// unreliable about a horizontal axis — it mis-reads a profile chamfer's angle as the sweep
	// (CapstainFrontBody's 125° is line[2]'s chamfer, not the extent), building a wrong pie-slice.
	// A wrong full turn instead over-fills and is caught by the mesh gate; a mis-sized partial is
	// not. Vertical-axis revolves keep the decoded angle (the 270° fixture depends on it).
	angle := revolveAngleFn(seg)
	if horizontalLine(s.Lines[ai]) {
		angle = nil
	}
	return revolveClosedProfiles(def, emitted[i], ai, closed, angle)
}

// closedProfileIndices returns the indices of the sketch's closed arranged profiles.
func closedProfileIndices(sk *sketch.Sketch) []int {
	var out []int
	profs := sk.Profiles()
	for p := 0; p < profs.Count(); p++ {
		if profs.Item(p).IsClosed() {
			out = append(out, p)
		}
	}
	return out
}

// revolveClosedProfiles turns each closed profile about the axis, the first starting the body and
// the rest joining it (adjacent regions of one solid of revolution). Returns the added feature ids.
func revolveClosedProfiles(def *compdef.PartComponentDefinition, e emittedSketch, axisLine int, closed []int, angle func() float64) []feature.ID {
	var ids []feature.ID
	for k, pi := range closed {
		op := ops.NewBody
		if k > 0 {
			op = ops.Join
		}
		f := feature.NewRevolveFeatures(def.Features()).AddAboutCenterlineLine(e.sk, pi, e.sk, e.lines[axisLine], angle, op)
		ids = append(ids, f.ID())
	}
	return ids
}

// revolveAngleFn returns the swept-angle accessor for a partial revolve, or nil for a full turn.
func revolveAngleFn(seg []byte) func() float64 {
	if a, ok := ipt.RevolveAngle(seg); ok {
		return func() float64 { return a }
	}
	return nil
}

// buildRevolve builds the revolve over the already-emitted sketches — it emits no geometry
// itself. It picks the profile + axis by the binding RevolveProfile validates (a CLOSED,
// one-sided profile about an unambiguous centreline, which may live in a different sketch than
// the profile) and revolves about that line. When no such profile/axis is found, or the chosen
// sketch/line came out empty, it builds nothing and returns a note; the emitted sketches remain
// for inspection. angle is nil ⇒ full 360°.
func buildRevolve(def *compdef.PartComponentDefinition, seg []byte, placed []placedSketch, emitted []emittedSketch) (bool, []string) {
	geoms := make([]ipt.Sketch, len(placed))
	for i := range placed {
		geoms[i] = placed[i].geom
	}
	b, ok := ipt.RevolveProfile(geoms)
	if !ok {
		// The line-ring gate declined. A profile with a notch/keyway (a curve whose end lands on
		// another edge's interior) is a valid closed region the kernel arranges but that walk can't
		// close — try the kernel's own profile before falling back to the mesh. Keep the result ONLY
		// if it closes to a solid: an open/self-intersecting revolve means a mis-decoded profile or
		// axis, and the faithful display mesh is better than a wrong parametric body (a pre-2023
		// PressureRoller copy built a half-open partial roller that way).
		if ids := tryKernelRevolve(def, seg, placed, emitted, -1, revolveAxisRef{}); len(ids) > 0 {
			def.Recompute()
			if firstBodyIsSolid(def) {
				return true, nil
			}
			for _, id := range ids {
				def.Features().Remove(id)
			}
			return false, []string{"revolve: kernel profile did not close to a solid — imported body used"}
		}
		return false, []string{"revolve: no unambiguous closed profile + axis — sketches emitted, revolve not built"}
	}
	if b.ProfileSketch >= len(emitted) || emitted[b.ProfileSketch].sk == nil {
		return false, []string{"revolve: chosen profile sketch is empty — revolve not built"}
	}
	axis := emitted[b.AxisSketch]
	if b.AxisLine >= len(axis.lines) || axis.lines[b.AxisLine] == nil {
		return false, []string{"revolve: axis line not emitted — revolve not built"}
	}
	var angle func() float64 // nil ⇒ full 360°
	if a, ok := ipt.RevolveAngle(seg); ok {
		a := a
		angle = func() float64 { return a }
	}
	feature.NewRevolveFeatures(def.Features()).AddAboutCenterlineLine(
		emitted[b.ProfileSketch].sk, 0, axis.sk, axis.lines[b.AxisLine], angle, ops.NewBody)
	return true, nil
}
