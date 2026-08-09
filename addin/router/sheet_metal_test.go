// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// newSheetMetalPart creates a sheet-metal part document over the router (the create path
// that seeds the rule) and makes it active, returning the router + session.
func newSheetMetalPart(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := seededSession(t)
	var doc wire.DocumentInfo
	call(t, r, s, "documents.create", fmt.Sprintf(`{"type":"part","name":"panel.obk","subType":%q}`, string(types.SubTypeSheetMetalPart)), &doc)
	call(t, r, s, "documents.activate", fmt.Sprintf(`{"id":%d}`, doc.ID), nil)
	return r, s
}

// TestSheetMetalGetStyleDefaults a part created with the sheet-metal subtype reports the
// seeded default rule (1 mm thickness, straight relief per Inventor's Default style, K-factor
// unfold).
func TestSheetMetalGetStyleDefaults(t *testing.T) {
	r, s := newSheetMetalPart(t)
	var res wire.SheetMetalStyleResult
	call(t, r, s, wire.MethodSheetMetalGetStyle, "{}", &res)
	if res.Style.UnfoldMethod != types.KFactorUnfold.String() {
		t.Errorf("unfold method = %q, want kFactor", res.Style.UnfoldMethod)
	}
	if res.Style.ReliefShape != types.ReliefStraight.String() {
		t.Errorf("relief shape = %q, want straight", res.Style.ReliefShape)
	}
	if math.Abs(res.Style.KFactor-0.44) > 1e-9 {
		t.Errorf("K-factor = %v, want 0.44", res.Style.KFactor)
	}
}

// TestSheetMetalGetStyleRejectsPlainPart getStyle on a non-sheet-metal part errors.
func TestSheetMetalGetStyleRejectsPlainPart(t *testing.T) {
	r, s := seededSession(t) // the seeded part is an ordinary part
	if _, err := r.Handle(s, wire.MethodSheetMetalGetStyle, []byte("{}")); err == nil {
		t.Fatal("getStyle on a plain part must error")
	}
}

// TestSheetMetalSetStyleEditsRuleAndKFactor setStyle changes thickness and K-factor; the
// developed bend allowance changes accordingly (the F01 acceptance criterion over the wire).
func TestSheetMetalSetStyleEditsRuleAndKFactor(t *testing.T) {
	r, s := newSheetMetalPart(t)

	var before wire.BendAllowanceResult
	call(t, r, s, wire.MethodSheetMetalBendAllowance, `{"angle":"90 deg","radius":"2 mm"}`, &before)

	var styled wire.SheetMetalStyleResult
	call(t, r, s, wire.MethodSheetMetalSetStyle, `{"thickness":"2 mm","kFactor":0.5}`, &styled)
	if math.Abs(styled.Style.KFactor-0.5) > 1e-9 {
		t.Errorf("K-factor after setStyle = %v, want 0.5", styled.Style.KFactor)
	}

	var after wire.BendAllowanceResult
	call(t, r, s, wire.MethodSheetMetalBendAllowance, `{"angle":"90 deg","radius":"2 mm"}`, &after)
	// BA = angle·(r + K·t); a thicker sheet with a larger K develops a longer flat.
	if !(after.BendAllowance > before.BendAllowance) {
		t.Errorf("bend allowance did not grow after thickening: before=%.6f after=%.6f", before.BendAllowance, after.BendAllowance)
	}
	wantAfter := (math.Pi / 2) * (0.2 + 0.5*0.2) // r=2mm=0.2cm, t=2mm=0.2cm, K=0.5
	if math.Abs(after.BendAllowance-wantAfter) > 1e-6 {
		t.Errorf("bend allowance = %.6f, want %.6f", after.BendAllowance, wantAfter)
	}
}

// TestSheetMetalSetStyleRelief setStyle switches the relief shape.
func TestSheetMetalSetStyleRelief(t *testing.T) {
	r, s := newSheetMetalPart(t)
	var res wire.SheetMetalStyleResult
	// "square" is the older spelling of the rectangular bend relief; it still sets the style and
	// reads back under the canonical name (#1960).
	call(t, r, s, wire.MethodSheetMetalSetStyle, `{"reliefShape":"square"}`, &res)
	if res.Style.ReliefShape != types.ReliefStraight.String() {
		t.Errorf("relief shape after setStyle = %q, want straight", res.Style.ReliefShape)
	}
}

// TestSheetMetalSetStyleRejectsUnsupportedUnfold setStyle rejects a bend-table/equation
// method (configured by its own surface), naming the value.
func TestSheetMetalSetStyleRejectsUnsupportedUnfold(t *testing.T) {
	r, s := newSheetMetalPart(t)
	if _, err := r.Handle(s, wire.MethodSheetMetalSetStyle, []byte(`{"unfoldMethod":"bendTable"}`)); err == nil {
		t.Fatal("setStyle must reject bendTable (no table payload in the style DTO)")
	}
}

// TestSheetMetalSetStyleAllProperties setStyle edits every simple property in one call:
// bend radius, relief width/depth, minimum gap, and the named K-factor method.
func TestSheetMetalSetStyleAllProperties(t *testing.T) {
	r, s := newSheetMetalPart(t)
	var res wire.SheetMetalStyleResult
	call(t, r, s, wire.MethodSheetMetalSetStyle,
		`{"bendRadius":"3 mm","reliefShape":"square","reliefWidth":"0.5 mm","reliefDepth":"0.5 mm","minimumGap":"1 mm","unfoldMethod":"kFactor"}`, &res)
	if res.Style.BendRadius != "3 mm" {
		t.Errorf("bend radius = %q, want 3 mm", res.Style.BendRadius)
	}
	if res.Style.ReliefWidth != "0.5 mm" || res.Style.ReliefDepth != "0.5 mm" {
		t.Errorf("relief size = %q/%q, want 0.5 mm/0.5 mm", res.Style.ReliefWidth, res.Style.ReliefDepth)
	}
	if res.Style.MindGap != "1 mm" {
		t.Errorf("minimum gap = %q, want 1 mm", res.Style.MindGap)
	}
	if res.Style.UnfoldMethod != types.KFactorUnfold.String() {
		t.Errorf("unfold = %q, want kFactor", res.Style.UnfoldMethod)
	}
}

// TestSheetMetalSetStyleRejectsBadExpressions setStyle reports a clear error for an
// unparseable length and a bad relief shape.
func TestSheetMetalSetStyleRejectsBadExpressions(t *testing.T) {
	r, s := newSheetMetalPart(t)
	// reliefShape/minimumGap/reliefWidth go through a hard unit parse, so a bad value errors.
	// (thickness/bendRadius re-author a parameter, where an unparseable token becomes a sick
	// parameter rather than a setStyle error — that path is the parameter engine's contract.)
	for _, bad := range []string{
		`{"reliefShape":"hexagon"}`,
		`{"minimumGap":"zzz"}`,
		`{"reliefWidth":"zzz"}`,
		`{"reliefDepth":"zzz"}`,
	} {
		if _, err := r.Handle(s, wire.MethodSheetMetalSetStyle, []byte(bad)); err == nil {
			t.Errorf("setStyle(%s) should error", bad)
		}
	}
}

// TestSheetMetalBendAllowanceRejectsBadInput bendAllowance reports a clear error for a bad
// angle or radius expression.
func TestSheetMetalBendAllowanceRejectsBadInput(t *testing.T) {
	r, s := newSheetMetalPart(t)
	for _, bad := range []string{`{"angle":"oops"}`, `{"angle":"90 deg","radius":"oops"}`} {
		if _, err := r.Handle(s, wire.MethodSheetMetalBendAllowance, []byte(bad)); err == nil {
			t.Errorf("bendAllowance(%s) should error", bad)
		}
	}
}

// TestCornerReliefStyleRoundTrips (#1960): the CORNER relief is a separate style property from the
// bend relief — the cut where two flanges meet — with its own shape, size, placement and a
// distinct three-bend pair. setStyle must carry all five and getStyle report them back.
func TestCornerReliefStyleRoundTrips(t *testing.T) {
	r, s := newSheetMetalPart(t)
	var res wire.SheetMetalStyleResult
	call(t, r, s, wire.MethodSheetMetalSetStyle, `{"cornerReliefShape":"square","cornerReliefSize":"3 mm",`+
		`"cornerReliefPlacement":"bendIntersection","threeBendReliefShape":"fullRound","threeBendReliefSize":"1 mm"}`, &res)
	if res.Style.CornerReliefShape != "square" || res.Style.CornerReliefPlacement != "bendIntersection" {
		t.Errorf("corner relief = %q at %q, want square at bendIntersection",
			res.Style.CornerReliefShape, res.Style.CornerReliefPlacement)
	}
	if res.Style.ThreeBendReliefShape != "fullRound" {
		t.Errorf("three-bend relief shape = %q, want fullRound", res.Style.ThreeBendReliefShape)
	}

	var got wire.SheetMetalStyleResult
	call(t, r, s, wire.MethodSheetMetalGetStyle, `{}`, &got)
	if got.Style.CornerReliefShape != "square" || got.Style.ThreeBendReliefShape != "fullRound" {
		t.Errorf("getStyle reported %+v, want the corner relief just set", got.Style)
	}
}

// TestDefaultCornerReliefMatchesInventor (#1960): a fresh sheet-metal part reports Inventor's
// Default-style corner relief, so a round-trip of an unedited part does not silently restyle it.
func TestDefaultCornerReliefMatchesInventor(t *testing.T) {
	r, s := newSheetMetalPart(t)
	var res wire.SheetMetalStyleResult
	call(t, r, s, wire.MethodSheetMetalGetStyle, `{}`, &res)
	if res.Style.ReliefShape != "straight" {
		t.Errorf("default bend relief = %q, want straight (Inventor's Default style)", res.Style.ReliefShape)
	}
	if res.Style.CornerReliefShape != "trimToBend" || res.Style.ThreeBendReliefShape != "roundWithRadius" {
		t.Errorf("default corner relief = %q / three-bend %q, want trimToBend / roundWithRadius",
			res.Style.CornerReliefShape, res.Style.ThreeBendReliefShape)
	}
	if res.Style.CornerReliefPlacement != "bendTangent" {
		t.Errorf("default corner relief placement = %q, want bendTangent", res.Style.CornerReliefPlacement)
	}
}

// TestUnknownCornerReliefIsRefused: an unrecognised shape or placement must not fall through to
// the default and quietly cut a different corner.
func TestUnknownCornerReliefIsRefused(t *testing.T) {
	r, s := newSheetMetalPart(t)
	for _, args := range []string{
		`{"cornerReliefShape":"laserWeld"}`,
		`{"cornerReliefPlacement":"middle"}`,
		`{"threeBendReliefShape":"tearDrop"}`,
	} {
		var res wire.SheetMetalStyleResult
		if _, err := r.Handle(s, wire.MethodSheetMetalSetStyle, []byte(args)); err == nil {
			t.Errorf("setStyle %s should be refused", args)
		}
		_ = res
	}
}

// TestBendTransitionStyleRoundTrips (#1959): the transition and its arc radius are style
// properties, so setStyle must carry them and getStyle report them back.
func TestBendTransitionStyleRoundTrips(t *testing.T) {
	r, s := newSheetMetalPart(t)
	var res wire.SheetMetalStyleResult
	call(t, r, s, wire.MethodSheetMetalGetStyle, "{}", &res)
	if res.Style.BendTransition != "none" {
		t.Errorf("default bend transition = %q, want none (Inventor's Default style)", res.Style.BendTransition)
	}
	call(t, r, s, wire.MethodSheetMetalSetStyle,
		`{"bendTransition":"arc","bendTransitionArcRadius":"3 mm"}`, &res)
	if res.Style.BendTransition != "arc" {
		t.Errorf("bend transition after setStyle = %q, want arc", res.Style.BendTransition)
	}
	var got wire.SheetMetalStyleResult
	call(t, r, s, wire.MethodSheetMetalGetStyle, "{}", &got)
	if got.Style.BendTransition != "arc" {
		t.Errorf("getStyle reported %q, want the arc transition just set", got.Style.BendTransition)
	}
	if _, err := r.Handle(s, wire.MethodSheetMetalSetStyle, []byte(`{"bendTransition":"spline"}`)); err == nil {
		t.Error("an unknown bend transition should be refused")
	}
}
