// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sheetmetal"
)

// The sheet-metal rule/style surface (M13-F01, #373/#369): read the active part's rule,
// edit it (recomputing so a thickness/K-factor change repropagates), and preview the
// developed flat length of one bend. A part is in the sheet-metal environment when it was
// created with the sheet-metal subtype, which seeds its rule.

// registerSheetMetalHandlers wires the sheetMetal.* methods.
func (r *Router) registerSheetMetalHandlers() {
	r.handlers[wire.MethodSheetMetalGetStyle] = sheetMetalGetStyle
	r.handlers[wire.MethodSheetMetalSetStyle] = sheetMetalSetStyle
	r.handlers[wire.MethodSheetMetalBendAllowance] = sheetMetalBendAllowance
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

func sheetMetalGetStyle(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, rule, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.SheetMetalStyleResult{Style: styleInfo(part, rule)})
}

func sheetMetalSetStyle(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, rule, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetSheetMetalStyleArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := applyStyleEdits(part, rule, in); err != nil {
		return nil, err
	}
	part.Recompute()
	return json.Marshal(wire.SheetMetalStyleResult{Style: styleInfo(part, rule)})
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
	if in.MinimumGap != "" {
		gap, err := part.Units().Parse(in.MinimumGap, param.Length)
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
			return fmt.Errorf("sheetMetal reliefShape %q: want round|square|tear", in.ReliefShape)
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
		v, err := part.Units().Parse(e.expr, param.Length)
		if err != nil {
			return fmt.Errorf("sheetMetal relief size %q: %w", e.expr, err)
		}
		*e.field = sheetmetal.Constant(v.Value)
	}
	rule.SetRelief(relief)
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

func sheetMetalBendAllowance(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, rule, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	var in wire.BendAllowanceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	angle, err := part.Units().Parse(in.Angle, param.Angle)
	if err != nil {
		return nil, fmt.Errorf("sheetMetal bendAllowance angle %q: %w", in.Angle, err)
	}
	radius := 0.0
	if in.Radius != "" {
		r, err := part.Units().Parse(in.Radius, param.Length)
		if err != nil {
			return nil, fmt.Errorf("sheetMetal bendAllowance radius %q: %w", in.Radius, err)
		}
		radius = r.Value
	}
	return json.Marshal(wire.BendAllowanceResult{
		BendAllowance: rule.BendAllowance(angle.Value, radius),
		BendDeduction: rule.BendDeduction(angle.Value, radius),
	})
}

// styleInfo renders the active rule as wire, formatting lengths in the document's units and
// including the bend allowance of a 90° bend as a convenience preview.
func styleInfo(part *compdef.PartComponentDefinition, rule *sheetmetal.Rule) wire.SheetMetalStyleInfo {
	fmtLen := func(v float64) string { return part.Units().Format(param.Q(v, param.Length)) }
	unfold := rule.Unfold()
	return wire.SheetMetalStyleInfo{
		Name:          rule.Name(),
		Thickness:     fmtLen(rule.Thickness()),
		BendRadius:    fmtLen(rule.BendRadius()),
		ReliefShape:   rule.Relief().Shape.String(),
		ReliefWidth:   fmtLen(rule.ReliefWidth()),
		ReliefDepth:   fmtLen(rule.ReliefDepth()),
		MindGap:       fmtLen(rule.Gap()),
		UnfoldMethod:  unfold.Type.String(),
		KFactor:       unfold.KFactor,
		BendAllowance: rule.BendAllowance(halfPi, 0),
	}
}

// halfPi is a 90° bend in radians — the angle styleInfo previews the bend allowance at.
const halfPi = 1.5707963267948966
