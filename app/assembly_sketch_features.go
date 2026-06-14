// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/feature"
)

// Assembly sketched features (#766): once a sketch is authored in assembly space (the sketch
// environment is content-agnostic), an assembly Extrude runs its profile against every
// participant body as a machining feature — adding or cutting material across the components. The
// profile is picked in the viewport (a ProfileHandle); distance and operation come from the
// generic tool-param dialog. The geometry is model/feature's AssemblyExtrudeFeature.

// assemblyExtrudeOps maps the tool's operation choice index to the kernel operation. Cut is first
// because an assembly extrude is most often a pocket cut across components. assemblyExtrudeOpNames
// are the matching chooser labels — kept beside the ops so the two slices stay index-aligned;
// shared by every sketch/profile machining tool (extrude, revolve).
var (
	assemblyExtrudeOps     = []ops.PartFeatureOperation{ops.Cut, ops.Join, ops.Intersect, ops.NewBody}
	assemblyExtrudeOpNames = []string{"Cut", "Join", "Intersect", "New body"}
)

// AssemblyExtrudeTool extrudes a picked assembly-sketch profile into a machining feature applied to
// every participant.
type AssemblyExtrudeTool struct {
	profile   *ProfileHandle
	distance  float64
	operation int // index into assemblyExtrudeOps
}

// NewAssemblyExtrudeTool returns an extrude tool with a 1-unit Cut default.
func NewAssemblyExtrudeTool() *AssemblyExtrudeTool { return &AssemblyExtrudeTool{distance: 1} }
func (t *AssemblyExtrudeTool) Name() string        { return "Extrude" }
func (t *AssemblyExtrudeTool) Prompt(*Session) string {
	return "Pick a sketch profile, set the distance and operation, then OK."
}
func (t *AssemblyExtrudeTool) Start(*Session) {}

// Pick collects the profile region clicked in the viewport.
func (t *AssemblyExtrudeTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(ProfileHandle); ok {
		t.profile = &h
	}
}
func (t *AssemblyExtrudeTool) Cancel(*Session) { t.profile = nil }
func (t *AssemblyExtrudeTool) CanCommit() bool { return t.profile != nil && t.distance > 0 }

func (t *AssemblyExtrudeTool) Commit(s *Session) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	if t.profile == nil {
		return errors.New("extrude: pick a sketch profile first")
	}
	d := t.distance
	op := assemblyExtrudeOps[t.operation]
	af := asm.AddFeature(feature.NewAssemblyExtrudeFeature(t.profile.Sketch, t.profile.ProfileIndex, op, func() float64 { return d }))
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	s.recordEdit(asm, "Extrude") // the feature program persists + undoes (#785)
	return nil
}

func (t *AssemblyExtrudeTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{
			{"Distance", func() float64 { return t.distance }, func(v float64) { t.distance = v }},
		},
		Choices: []ChoiceParam{
			{Label: "Operation", Options: assemblyExtrudeOpNames,
				Get: func() int { return t.operation }, Set: func(i int) { t.operation = i }},
		},
	}
}
