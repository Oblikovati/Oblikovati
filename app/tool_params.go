// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/math"

// Tool parameter descriptors: a Tool may expose a set of editable parameters (a label +
// a bound get/set per value) so the head can render ONE generic property dialog for any
// tool instead of a bespoke cgo dialog per tool (core/09 reflection-driven editing). The
// descriptors are pure data with closures over the tool's fields, so the wiring is tested
// headlessly here and the head only renders the fields.

// FloatParam is an editable floating-point tool parameter (e.g. a distance or angle).
type FloatParam struct {
	Label string
	Get   func() float64
	Set   func(float64)
}

// IntParam is an editable integer tool parameter (e.g. a pattern count).
type IntParam struct {
	Label string
	Get   func() int
	Set   func(int)
}

// TextParam is an editable string tool parameter (e.g. the text-box string).
type TextParam struct {
	Label string
	Get   func() string
	Set   func(string)
}

// BoolParam is an editable boolean tool parameter (e.g. a helix's clockwise flag),
// rendered as a checkbox.
type BoolParam struct {
	Label string
	Get   func() bool
	Set   func(bool)
}

// ChoiceParam is a one-of-N tool parameter (e.g. text alignment), rendered as a dropdown.
// Get/Set carry the selected index into Options.
type ChoiceParam struct {
	Label   string
	Options []string
	Get     func() int
	Set     func(int)
}

// ToolParams is the set of editable parameters a tool exposes; the head renders each
// non-empty group as a labelled input row.
type ToolParams struct {
	Floats  []FloatParam
	Ints    []IntParam
	Texts   []TextParam
	Bools   []BoolParam
	Choices []ChoiceParam
}

// Empty reports whether the tool exposes no parameters (so the head shows no dialog).
func (p ToolParams) Empty() bool {
	return len(p.Floats) == 0 && len(p.Ints) == 0 && len(p.Texts) == 0 &&
		len(p.Bools) == 0 && len(p.Choices) == 0
}

// ParameterizedTool is a Tool that exposes editable parameters for the property dialog.
// The head queries the active tool for this interface each frame and, when present,
// renders the generic dialog bound to the tool's parameters.
type ParameterizedTool interface {
	Tool
	Params() ToolParams
}

// degrees/radians conversion for angle parameters surfaced to the user in degrees.
const radPerDeg = 0.017453292519943295

func degFromRad(rad float64) float64 { return rad / radPerDeg }
func radFromDeg(deg float64) float64 { return deg * radPerDeg }

// xyParams binds a vector/point's X and Y as two float params with the given label prefix.
func xyParams(prefix string, x, y *math.Scalar) []FloatParam {
	return []FloatParam{
		{prefix + " X", func() float64 { return float64(*x) }, func(v float64) { *x = math.Scalar(v) }},
		{prefix + " Y", func() float64 { return float64(*y) }, func(v float64) { *y = math.Scalar(v) }},
	}
}

// scalarParam binds a single math.Scalar field as a float param.
func scalarParam(label string, p *math.Scalar) FloatParam {
	return FloatParam{label, func() float64 { return float64(*p) }, func(v float64) { *p = math.Scalar(v) }}
}
