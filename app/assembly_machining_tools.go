// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// Assembly Revolve and Hole machining tools (#766), alongside the Extrude in
// assembly_sketch_features.go. Revolve spins a picked sketch profile about an axis into the
// participants; Hole drills a parametric bore (no sketch). Both add an assembly feature that
// recomputes across every component. Inputs come from the generic tool-param dialog.

// signedAxis is one of the six signed coordinate axes a Revolve/Hole can use, by display name.
type signedAxis struct {
	name string
	v    [3]float64
}

var signedAxes = []signedAxis{
	{"+X", [3]float64{1, 0, 0}}, {"-X", [3]float64{-1, 0, 0}},
	{"+Y", [3]float64{0, 1, 0}}, {"-Y", [3]float64{0, -1, 0}},
	{"+Z", [3]float64{0, 0, 1}}, {"-Z", [3]float64{0, 0, -1}},
}

// signedAxisNames are the chooser labels; signedAxisDir resolves an index to its unit direction
// (the literals are unit by construction, so the error is unreachable).
func signedAxisNames() []string {
	out := make([]string, len(signedAxes))
	for i, a := range signedAxes {
		out[i] = a.name
	}
	return out
}

func signedAxisDir(i int) math.UnitVector3 {
	d, _ := math.NewUnitVector3(signedAxes[i].v[0], signedAxes[i].v[1], signedAxes[i].v[2])
	return d
}

// --- Revolve --------------------------------------------------------------

// AssemblyRevolveTool spins a picked assembly-sketch profile about a coordinate axis through the
// origin into a machining feature on every participant.
type AssemblyRevolveTool struct {
	profile   *ProfileHandle
	angle     float64 // radians
	operation int     // index into assemblyExtrudeOps
	axisIndex int     // index into signedAxes
}

// NewAssemblyRevolveTool returns a full-turn Cut about +Y by default.
func NewAssemblyRevolveTool() *AssemblyRevolveTool {
	return &AssemblyRevolveTool{angle: FullTurn, axisIndex: 2}
}
func (t *AssemblyRevolveTool) Name() string { return "Revolve" }
func (t *AssemblyRevolveTool) Prompt(*Session) string {
	return "Pick a sketch profile, set the axis, angle and operation, then OK."
}
func (t *AssemblyRevolveTool) Start(*Session) { /* no setup; the tool waits for a profile pick */ }
func (t *AssemblyRevolveTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(ProfileHandle); ok {
		t.profile = &h
	}
}
func (t *AssemblyRevolveTool) Cancel(*Session) { t.profile = nil }
func (t *AssemblyRevolveTool) CanCommit() bool {
	return t.profile != nil && t.angle > 0 && t.angle <= FullTurn+1e-9
}

func (t *AssemblyRevolveTool) Commit(s *Session) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	if t.profile == nil {
		return errors.New("revolve: pick a sketch profile first")
	}
	axis := feature.NewDatumAxis(math.P3(0, 0, 0), signedAxisDir(t.axisIndex))
	a := t.angle
	af := asm.AddFeature(feature.NewAssemblyRevolveFeature(t.profile.Sketch, t.profile.ProfileIndex, axis, assemblyExtrudeOps[t.operation], func() float64 { return a }))
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	s.recordEdit(asm, "Revolve") // the feature program persists + undoes (#785)
	return nil
}

func (t *AssemblyRevolveTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{
			{"Angle (deg)", func() float64 { return degFromRad(t.angle) }, func(d float64) { t.angle = radFromDeg(d) }},
		},
		Choices: []ChoiceParam{
			{Label: "Axis", Options: signedAxisNames(), Get: func() int { return t.axisIndex }, Set: func(i int) { t.axisIndex = i }},
			{Label: "Operation", Options: []string{"Cut", "Join", "Intersect", "New body"}, Get: func() int { return t.operation }, Set: func(i int) { t.operation = i }},
		},
	}
}

// --- Hole -----------------------------------------------------------------

// AssemblyHoleTool drills a parametric cylindrical bore (centre, axis, diameter, depth) through
// every participant — a through-feature placed by coordinate, no sketch needed.
type AssemblyHoleTool struct {
	cx, cy, cz float64
	axisIndex  int // index into signedAxes (the drill direction)
	diameter   float64
	depth      float64
}

// NewAssemblyHoleTool returns a 1-diameter, 1-deep hole drilling down (-Z) by default.
func NewAssemblyHoleTool() *AssemblyHoleTool {
	return &AssemblyHoleTool{axisIndex: 5, diameter: 1, depth: 1}
}
func (t *AssemblyHoleTool) Name() string { return "Hole" }
func (t *AssemblyHoleTool) Prompt(*Session) string {
	return "Set the centre, axis, diameter and depth, then OK."
}

// The hole tool takes only numeric input (centre, axis, diameter, depth), so it has no
// selection lifecycle: nothing to set up, pick, or undo on cancel.
func (t *AssemblyHoleTool) Start(*Session)            {}
func (t *AssemblyHoleTool) Pick(*Session, Selectable) {}
func (t *AssemblyHoleTool) Cancel(*Session)           {}
func (t *AssemblyHoleTool) CanCommit() bool           { return t.diameter > 0 && t.depth > 0 }

func (t *AssemblyHoleTool) Commit(s *Session) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	hole, err := feature.NewAssemblyHoleFeature(math.P3(t.cx, t.cy, t.cz), signedAxisDir(t.axisIndex), t.diameter, t.depth)
	if err != nil {
		return err
	}
	af := asm.AddFeature(hole)
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	s.recordEdit(asm, "Hole") // the feature program persists + undoes (#785)
	return nil
}

func (t *AssemblyHoleTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{
			{"Centre X", func() float64 { return t.cx }, func(v float64) { t.cx = v }},
			{"Centre Y", func() float64 { return t.cy }, func(v float64) { t.cy = v }},
			{"Centre Z", func() float64 { return t.cz }, func(v float64) { t.cz = v }},
			{"Diameter", func() float64 { return t.diameter }, func(v float64) { t.diameter = v }},
			{"Depth", func() float64 { return t.depth }, func(v float64) { t.depth = v }},
		},
		Choices: []ChoiceParam{
			{Label: "Axis", Options: signedAxisNames(), Get: func() int { return t.axisIndex }, Set: func(i int) { t.axisIndex = i }},
		},
	}
}
