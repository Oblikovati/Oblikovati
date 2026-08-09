// SPDX-License-Identifier: GPL-2.0-only

package sheetmetal

import "oblikovati.org/api/types"

// Relief geometry default sizes, as multiples of thickness — the standard starting point a
// rule uses until the user overrides them.
const (
	defaultReliefWidthFactor = 0.5 // relief notch width = 0.5·thickness
	defaultReliefDepthFactor = 0.5 // relief notch depth = 0.5·thickness
	defaultGapFactor         = 0.0 // corner gap (0 = closed) until set
	// defaultCornerReliefFactor is Inventor's Default-style corner relief size, Thickness*4.
	defaultCornerReliefFactor = 4.0
)

// Relief describes the cut placed at the ends of a bend so the adjacent web folds without
// tearing: its shape plus the notch width and depth (parameter-backed lengths in database
// units, cm).
type Relief struct {
	Shape ReliefShape
	Width func() float64
	Depth func() float64
}

// CornerRelief is the cut made where two flanges MEET — a separate property from the bend
// relief, with its own shape, size and placement, and a distinct shape/size for the three-bend
// corner (Inventor's SheetMetalStyle corner-relief block, #1960).
type CornerRelief struct {
	Shape          types.CornerReliefShape
	Size           func() float64
	Placement      types.CornerReliefPlacement
	ThreeBendShape types.CornerReliefShape
	ThreeBendSize  func() float64
}

// Rule is the active sheet-metal style: the constant material thickness, the default bend
// radius, the relief geometry, the corner gap, and the unfold method that develops bends
// into the flat pattern. Every length is a closure so the rule stays parameter-backed —
// editing the part's Thickness parameter repropagates to every wall and bend through the
// normal feature engine.
type Rule struct {
	name       string
	thickness  func() float64
	bendRadius func() float64
	gap        func() float64
	relief     Relief
	corner     CornerRelief
	unfold     UnfoldMethod
}

// NewRule builds a rule from parameter-backed closures. The caller (the compdef layer)
// supplies closures that read the part's named parameters; tests pass constant closures.
func NewRule(name string, thickness, bendRadius, gap func() float64, relief Relief, unfold UnfoldMethod) *Rule {
	return &Rule{name: name, thickness: thickness, bendRadius: bendRadius, gap: gap, relief: relief, unfold: unfold}
}

// Constant returns a closure that always reports v — for tests and constant defaults.
func Constant(v float64) func() float64 { return func() float64 { return v } }

// DefaultRule returns a rule with constant defaults, matching what Inventor's own Default style
// ships with (#1960): a STRAIGHT bend relief at half thickness, the corner trimmed to the bend at
// four times thickness, a three-bend corner rounded at the bend radius, and K-factor unfold. Used
// as the seed when a part enters the sheet-metal environment before the user picks a style.
func DefaultRule(thickness, bendRadius float64) *Rule {
	relief := Relief{
		Shape: types.ReliefStraight,
		Width: Constant(defaultReliefWidthFactor * thickness),
		Depth: Constant(defaultReliefDepthFactor * thickness),
	}
	corner := CornerRelief{
		Shape:          types.CornerTrimToBend,
		Size:           Constant(defaultCornerReliefFactor * thickness),
		Placement:      types.CornerReliefAtBendTangent,
		ThreeBendShape: types.CornerRoundWithRadius,
		ThreeBendSize:  Constant(bendRadius),
	}
	r := NewRule("Default", Constant(thickness), Constant(bendRadius), Constant(defaultGapFactor), relief, KFactorMethod(defaultKFactor))
	r.SetCornerRelief(corner)
	return r
}

// Name returns the rule's style name.
func (r *Rule) Name() string { return r.name }

// SetName renames the rule.
func (r *Rule) SetName(name string) { r.name = name }

// Thickness returns the material thickness (database units, cm).
func (r *Rule) Thickness() float64 { return call(r.thickness) }

// BendRadius returns the default inside bend radius (cm).
func (r *Rule) BendRadius() float64 { return call(r.bendRadius) }

// Gap returns the corner gap (cm).
func (r *Rule) Gap() float64 { return call(r.gap) }

// Relief returns the rule's relief geometry.
func (r *Rule) Relief() Relief { return r.relief }

// ReliefWidth and ReliefDepth report the relief notch size (cm).
func (r *Rule) ReliefWidth() float64 { return call(r.relief.Width) }
func (r *Rule) ReliefDepth() float64 { return call(r.relief.Depth) }

// Unfold returns the rule's unfold method.
func (r *Rule) Unfold() UnfoldMethod { return r.unfold }

// SetUnfold replaces the unfold method (e.g. switching from K-factor to a bend table).
func (r *Rule) SetUnfold(m UnfoldMethod) { r.unfold = m }

// SetThickness/SetBendRadius/SetGap replace a length closure (the compdef layer rebinds
// these to the edited parameter; tests pass constants).
func (r *Rule) SetThickness(f func() float64)  { r.thickness = f }
func (r *Rule) SetBendRadius(f func() float64) { r.bendRadius = f }
func (r *Rule) SetGap(f func() float64)        { r.gap = f }

// SetRelief replaces the bend-relief geometry.
func (r *Rule) SetRelief(relief Relief) { r.relief = relief }

// CornerRelief returns the rule's corner-relief block, and SetCornerRelief replaces it (#1960).
func (r *Rule) CornerRelief() CornerRelief     { return r.corner }
func (r *Rule) SetCornerRelief(c CornerRelief) { r.corner = c }

// CornerReliefSize and ThreeBendReliefSize report the corner cut sizes (cm).
func (r *Rule) CornerReliefSize() float64    { return call(r.corner.Size) }
func (r *Rule) ThreeBendReliefSize() float64 { return call(r.corner.ThreeBendSize) }

// BendAllowance and BendDeduction develop a single bend under this rule's unfold method,
// defaulting the radius to the rule's bend radius when a non-positive radius is passed.
func (r *Rule) BendAllowance(angle, radius float64) float64 {
	return r.unfold.BendAllowance(angle, r.radiusOr(radius), r.Thickness())
}

func (r *Rule) BendDeduction(angle, radius float64) float64 {
	return r.unfold.BendDeduction(angle, r.radiusOr(radius), r.Thickness())
}

func (r *Rule) radiusOr(radius float64) float64 {
	if radius > 0 {
		return radius
	}
	return r.BendRadius()
}

// call evaluates a length closure, treating nil as zero so a half-built rule never panics.
func call(f func() float64) float64 {
	if f == nil {
		return 0
	}
	return f()
}
