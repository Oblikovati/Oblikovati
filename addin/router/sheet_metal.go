// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sheetmetal"
)

// The sheet-metal rule/style surface (M13-F01, #373/#369): read the active part's rule,
// edit it (recomputing so a thickness/K-factor change repropagates), and preview the
// developed flat length of one bend. A part is in the sheet-metal environment when it was
// created with the sheet-metal subtype, which seeds its rule.

// registerSheetMetalHandlers wires the sheetMetal.* methods.
func (r *Router) registerSheetMetalHandlers() {
	r.readOnly(wire.MethodSheetMetalGetStyle, ctxQuery(resolveSheetMetalPart, sheetMetalGetStyle))
	r.mutating(wire.MethodSheetMetalSetStyle, "Edit Sheet Metal Style", typedCtx(resolveSheetMetalPart, sheetMetalSetStyle))
	r.readOnly(wire.MethodSheetMetalBendAllowance, typedCtx(resolveSheetMetalPart, sheetMetalBendAllowance))
	r.readOnly(wire.MethodSheetMetalBends, ctxQuery(resolveSheetMetalPart, sheetMetalBends))
	r.mutating(wire.MethodSheetMetalUnfold, "Create Flat Pattern", ctxQuery(resolveSheetMetalPart, sheetMetalUnfold))
}

// activeSheetMetal returns the active part and its rule, or an error if the active document
// is not a sheet-metal part.
func activeSheetMetal(s *app.Session) (*compdef.PartComponentDefinition, *sheetmetal.Rule, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, nil, err
	}
	rule := part.SheetMetal()
	if rule == nil {
		return nil, nil, fmt.Errorf("sheetMetal: the active part is not a sheet-metal part (create it with subType %q)", types.SubTypeSheetMetalPart)
	}
	return part, rule, nil
}

// sheetMetalPart bundles the active sheet-metal part with its rule — the context the
// sheetMetal.* and flatPattern.* adapters resolve before decoding, wrapping activeSheetMetal
// as a typedCtx/ctxQuery resolver (#1649).
type sheetMetalPart struct {
	part *compdef.PartComponentDefinition
	rule *sheetmetal.Rule
}

// resolveSheetMetalPart adapts activeSheetMetal to the (Ctx, error) shape typedCtx/ctxQuery expect.
func resolveSheetMetalPart(s *app.Session) (sheetMetalPart, error) {
	part, rule, err := activeSheetMetal(s)
	if err != nil {
		return sheetMetalPart{}, err
	}
	return sheetMetalPart{part: part, rule: rule}, nil
}

func sheetMetalGetStyle(_ *app.Session, ctx sheetMetalPart) (wire.SheetMetalStyleResult, error) {
	return wire.SheetMetalStyleResult{Style: styleInfo(ctx.part, ctx.rule)}, nil
}

func sheetMetalSetStyle(_ *app.Session, ctx sheetMetalPart, in wire.SetSheetMetalStyleArgs) (wire.SheetMetalStyleResult, error) {
	if err := applyStyleEdits(ctx.part, ctx.rule, in); err != nil {
		return wire.SheetMetalStyleResult{}, err
	}
	// A rule edit (e.g. thickness) changes the inputs every wall/bend reads live, but those
	// reads are not tracked feature dependencies — invalidate the whole program so the sheet
	// rebuilds at the new gauge (the same full-rebuild a parameter edit triggers).
	ctx.part.Features().MarkAllDirty()
	ctx.part.Recompute()
	return wire.SheetMetalStyleResult{Style: styleInfo(ctx.part, ctx.rule)}, nil
}

// applyStyleEdits mutates the rule (and its backing parameters) per the non-empty fields of
// in. Lengths are re-authored on the Thickness/BendRadius parameters so the rule stays
// parameter-backed; relief/gap/unfold edits land on the rule directly.
func applyStyleEdits(part *compdef.PartComponentDefinition, rule *sheetmetal.Rule, in wire.SetSheetMetalStyleArgs) error {
	if in.Thickness != "" {
		if err := part.SetSheetMetalLengthParam(compdef.ThicknessParamName(), in.Thickness); err != nil {
			return err
		}
	}
	if in.BendRadius != "" {
		if err := part.SetSheetMetalLengthParam(compdef.BendRadiusParamName(), in.BendRadius); err != nil {
			return err
		}
	}
	if err := applyReliefEdits(part, rule, in); err != nil {
		return err
	}
	if err := applyCornerReliefEdits(part, rule, in); err != nil {
		return err
	}
	if in.MinimumGap != "" {
		gap, err := resolveQuantity(part, in.MinimumGap, param.Length)
		if err != nil {
			return fmt.Errorf("sheetMetal minimumGap %q: %w", in.MinimumGap, err)
		}
		rule.SetGap(sheetmetal.Constant(gap.Value))
	}
	return applyUnfoldEdits(rule, in)
}

// applyReliefEdits updates the relief shape and notch sizes from the non-empty fields.
func applyReliefEdits(part *compdef.PartComponentDefinition, rule *sheetmetal.Rule, in wire.SetSheetMetalStyleArgs) error {
	relief := rule.Relief()
	if in.ReliefShape != "" {
		shape, ok := types.ParseReliefShape(in.ReliefShape)
		if !ok {
			return fmt.Errorf("sheetMetal reliefShape %q: want round|straight|tear", in.ReliefShape)
		}
		relief.Shape = shape
	}
	for _, e := range []struct {
		expr  string
		field *func() float64
	}{{in.ReliefWidth, &relief.Width}, {in.ReliefDepth, &relief.Depth}} {
		if e.expr == "" {
			continue
		}
		v, err := resolveQuantity(part, e.expr, param.Length)
		if err != nil {
			return fmt.Errorf("sheetMetal relief size %q: %w", e.expr, err)
		}
		*e.field = sheetmetal.Constant(v.Value)
	}
	rule.SetRelief(relief)
	return nil
}

// applyCornerReliefEdits updates the CORNER relief block from the non-empty fields (#1960). It is
// a separate property from the bend relief: the cut where two flanges meet, not the one at a
// bend's ends.
func applyCornerReliefEdits(part *compdef.PartComponentDefinition, rule *sheetmetal.Rule, in wire.SetSheetMetalStyleArgs) error {
	corner := rule.CornerRelief()
	for _, e := range []struct {
		name  string
		field *types.CornerReliefShape
	}{{in.CornerReliefShape, &corner.Shape}, {in.ThreeBendReliefShape, &corner.ThreeBendShape}} {
		if e.name == "" {
			continue
		}
		shape, ok := types.ParseCornerReliefShape(e.name)
		if !ok {
			return fmt.Errorf("sheetMetal corner relief shape %q: want trimToBend|round|square|tear|"+
				"fullRound|roundWithRadius|intersection", e.name)
		}
		*e.field = shape
	}
	if in.CornerReliefPlacement != "" {
		placement, ok := types.ParseCornerReliefPlacement(in.CornerReliefPlacement)
		if !ok {
			return fmt.Errorf("sheetMetal cornerReliefPlacement %q: want bendTangent|bendIntersection|alongBend",
				in.CornerReliefPlacement)
		}
		corner.Placement = placement
	}
	if err := applyCornerReliefSizes(part, &corner, in); err != nil {
		return err
	}
	rule.SetCornerRelief(corner)
	return nil
}

// applyCornerReliefSizes resolves the two corner-relief size expressions.
func applyCornerReliefSizes(part *compdef.PartComponentDefinition, corner *sheetmetal.CornerRelief, in wire.SetSheetMetalStyleArgs) error {
	for _, e := range []struct {
		expr  string
		field *func() float64
	}{{in.CornerReliefSize, &corner.Size}, {in.ThreeBendReliefSize, &corner.ThreeBendSize}} {
		if e.expr == "" {
			continue
		}
		v, err := resolveQuantity(part, e.expr, param.Length)
		if err != nil {
			return fmt.Errorf("sheetMetal corner relief size %q: %w", e.expr, err)
		}
		*e.field = sheetmetal.Constant(v.Value)
	}
	return nil
}

// applyUnfoldEdits sets the unfold method. F01 wires the K-factor method (the default and
// most common); a bend-table or equation method carries data the simple style DTO does not,
// so those are configured by their own surface (M13-F04) and rejected here with a clear note.
func applyUnfoldEdits(rule *sheetmetal.Rule, in wire.SetSheetMetalStyleArgs) error {
	method := in.UnfoldMethod
	if method == "" && in.KFactor == 0 {
		return nil
	}
	if method != "" && method != types.KFactorUnfold.String() {
		return fmt.Errorf("sheetMetal unfoldMethod %q: only %q is settable via setStyle (bend-table/equation are configured separately)", method, types.KFactorUnfold)
	}
	k := in.KFactor
	if k == 0 {
		k = rule.Unfold().KFactor // keep the current K-factor when only the method was named
	}
	rule.SetUnfold(sheetmetal.KFactorMethod(k))
	return nil
}

func sheetMetalBendAllowance(_ *app.Session, ctx sheetMetalPart, in wire.BendAllowanceArgs) (wire.BendAllowanceResult, error) {
	angle, err := resolveQuantity(ctx.part, in.Angle, param.Angle)
	if err != nil {
		return wire.BendAllowanceResult{}, fmt.Errorf("sheetMetal bendAllowance angle %q: %w", in.Angle, err)
	}
	radius := 0.0
	if in.Radius != "" {
		r, err := resolveQuantity(ctx.part, in.Radius, param.Length)
		if err != nil {
			return wire.BendAllowanceResult{}, fmt.Errorf("sheetMetal bendAllowance radius %q: %w", in.Radius, err)
		}
		radius = r.Value
	}
	return wire.BendAllowanceResult{
		BendAllowance: ctx.rule.BendAllowance(angle.Value, radius),
		BendDeduction: ctx.rule.BendDeduction(angle.Value, radius),
	}, nil
}

func sheetMetalBends(_ *app.Session, ctx sheetMetalPart) (wire.BendsResult, error) {
	bends := ctx.part.Bends()
	out := wire.BendsResult{Bends: make([]wire.BendInfo, 0, len(bends))}
	for _, b := range bends {
		out.Bends = append(out.Bends, wire.BendInfo{
			Feature:   b.Feature,
			Angle:     b.Angle * degPerRad,
			Radius:    b.Radius,
			Thickness: b.Thickness,
			Allowance: b.Allowance,
			Deduction: b.Deduction,
		})
		out.TotalAllowance += b.Allowance
	}
	return out, nil
}

// degPerRad converts a bend's stored angle (radians) to the degrees the wire reports.
const degPerRad = 180.0 / 3.141592653589793

func sheetMetalUnfold(_ *app.Session, ctx sheetMetalPart) (wire.UnfoldResult, error) {
	flat, err := ctx.part.Unfold()
	if err != nil {
		return wire.UnfoldResult{}, err
	}
	return wire.UnfoldResult{Flat: flatInfo(flat)}, nil
}

// flatInfo renders a developed flat pattern as wire: its extents, gauge, developed footprint
// area (the constant-thickness plate's volume divided by the gauge), and the fold lines.
func flatInfo(flat *feature.FlatPattern) wire.FlatPatternInfo {
	area := 0.0
	if flat.Thickness > 0 {
		area = ops.BodyGeometryProperties(flat.Body, ops.Quality{ChordTolerance: 1e-3}).Volume / flat.Thickness
	}
	info := wire.FlatPatternInfo{
		Extents:   types.Box2d{Min: point2d(flat.Extents.Min), Max: point2d(flat.Extents.Max)},
		Thickness: flat.Thickness,
		Area:      area,
		Bends:     make([]wire.FlatBendLineInfo, 0, len(flat.Bends)),
	}
	for _, b := range flat.Bends {
		info.Bends = append(info.Bends, wire.FlatBendLineInfo{
			Start: point2d(b.A), End: point2d(b.B), Angle: b.Angle * degPerRad,
		})
	}
	info.Punches = punchInfos(flat)
	return info
}

// punchInfos projects the flat's developed punches onto the wire (#1963). The geometry was always
// computed for the flat; it simply had no way out of the host.
func punchInfos(flat *feature.FlatPattern) []wire.FlatPunchInfo {
	out := make([]wire.FlatPunchInfo, 0, len(flat.Punches))
	for _, p := range flat.Punches {
		info := wire.FlatPunchInfo{
			ID: p.Token, Position: point2d(p.Position), Angle: p.Angle * degPerRad,
			DirectionUp: p.DirectionUp, HasDepth: p.HasDepth, Depth: p.Depth,
			Outline: make([]types.Point2d, 0, len(p.Outline)),
		}
		for _, v := range p.Outline {
			info.Outline = append(info.Outline, point2d(v))
		}
		out = append(out, info)
	}
	return out
}

// point2d converts a model 2D point to the wire value type.
func point2d(p gmath.Point2) types.Point2d {
	return types.Point2d{X: float64(p.X), Y: float64(p.Y)}
}

// styleInfo renders the active rule as wire, formatting lengths in the document's units and
// including the bend allowance of a 90° bend as a convenience preview. The two
// parameter-backed lengths report their authored expression (e.g. "3 mm" or "t*2"), which
// is both more faithful and free of the floating-point noise a unit reconversion introduces.
func styleInfo(part *compdef.PartComponentDefinition, rule *sheetmetal.Rule) wire.SheetMetalStyleInfo {
	fmtLen := func(v float64) string { return part.Units().Format(param.Q(v, param.Length)) }
	paramExpr := func(name string, value float64) string {
		if p, ok := part.Parameters().ByName(name); ok {
			return p.Expression()
		}
		return fmtLen(value)
	}
	unfold := rule.Unfold()
	return wire.SheetMetalStyleInfo{
		Name:          rule.Name(),
		Thickness:     paramExpr(compdef.ThicknessParamName(), rule.Thickness()),
		BendRadius:    paramExpr(compdef.BendRadiusParamName(), rule.BendRadius()),
		ReliefShape:   rule.Relief().Shape.String(),
		ReliefWidth:   fmtLen(rule.ReliefWidth()),
		ReliefDepth:   fmtLen(rule.ReliefDepth()),
		MindGap:       fmtLen(rule.Gap()),
		UnfoldMethod:  unfold.Type.String(),
		KFactor:       unfold.KFactor,
		BendAllowance: rule.BendAllowance(halfPi, 0),

		CornerReliefShape:     rule.CornerRelief().Shape.String(),
		CornerReliefSize:      fmtLen(rule.CornerReliefSize()),
		CornerReliefPlacement: rule.CornerRelief().Placement.String(),
		ThreeBendReliefShape:  rule.CornerRelief().ThreeBendShape.String(),
		ThreeBendReliefSize:   fmtLen(rule.ThreeBendReliefSize()),
	}
}

// halfPi is a 90° bend in radians — the angle styleInfo previews the bend allowance at.
const halfPi = 1.5707963267948966
