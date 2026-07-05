//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"bytes"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// One generic property panel serves every tool that exposes parameters
// (app.ParameterizedTool) — the sketch Move/Copy/Rotate/Scale/Stretch, the rectangular/
// circular patterns, Slot/Chamfer/Text/Fillet/Offset, the feature pattern/modify tools —
// instead of a bespoke cgo dialog per tool (core/09 reflection-driven editing). It
// follows the property-panel schema: a breadcrumb naming the tool and a Behavior section
// of label/control rows, one per parameter, with OK/Cancel beneath. Modeless: the user
// still picks geometry in the viewport.

// toolText is the persistent edit buffer for a tool's text parameter (Dear ImGui's
// InputText needs a stable buffer across frames).
var toolText = struct {
	buf  []byte
	open bool
}{buf: make([]byte, 256)}

// drawToolParamsDialog shows the generic parameter panel while a parameterized tool runs.
func drawToolParamsDialog(s *app.Session) {
	title, params, ok := activeToolParams(s)
	if !ok {
		toolText.open = false
		return
	}
	dialogSizeOnce(320, 280)
	if native.Begin(title) {
		drawFeatureBreadcrumb(title, "")
		if propertySection("Behavior") {
			drawToolFloatParams(s, params)
			drawToolIntParams(params)
			drawToolBoolParams(params)
			drawToolChoiceParams(params)
			drawToolTextParams(params)
		}
		native.Separator()
		drawCommitCancelButtons(s, s.ActiveTool().Tool().CanCommit())
	}
	native.End()
}

// activeToolParams returns the active parameterized tool's title + params, or ok=false when
// there is no active tool, it is not parameterized, or it exposes no parameters.
func activeToolParams(s *app.Session) (string, app.ToolParams, bool) {
	ti := s.ActiveTool()
	if ti == nil {
		return "", app.ToolParams{}, false
	}
	pt, ok := ti.Tool().(app.ParameterizedTool)
	if !ok {
		return "", app.ToolParams{}, false
	}
	params := pt.Params()
	if params.Empty() {
		return "", app.ToolParams{}, false
	}
	return ti.Name(), params, true
}

func drawToolBoolParams(params app.ToolParams) {
	for _, b := range params.Bools {
		propertyRow("")
		v := b.Get()
		if native.Checkbox(b.Label, &v) {
			b.Set(v)
		}
	}
}

// drawToolChoiceParams renders each one-of-N parameter (e.g. text alignment, font) as a
// labelled dropdown row bound to the tool's selected index.
func drawToolChoiceParams(params app.ToolParams) {
	for _, c := range params.Choices {
		drawToolChoice(c)
	}
}

func drawToolChoice(c app.ChoiceParam) {
	propertyRow(c.Label)
	native.SetNextItemWidth(propertyFieldWidth)
	cur := c.Get()
	preview := ""
	if cur >= 0 && cur < len(c.Options) {
		preview = c.Options[cur]
	}
	if !native.BeginCombo(toolParamPrefix+c.Label, preview) {
		return
	}
	for i, opt := range c.Options {
		if native.Selectable(opt, i == cur) {
			c.Set(i)
		}
	}
	native.EndCombo()
}

func drawToolFloatParams(s *app.Session, params app.ToolParams) {
	for _, f := range params.Floats {
		v := float32(f.Get())
		// Add-in float params carry no document unit; ParameterInput still owns the row (no bare
		// InputFloat in a tool dialog — see TestParameterInputIsEnforced) and accepts formulas.
		if parameterFloatRow(s, f.Label, "tool-param-"+f.Label, paramUnitless, "", &v) {
			f.Set(float64(v))
		}
	}
}

func drawToolIntParams(params app.ToolParams) {
	for _, n := range params.Ints {
		propertyRow(n.Label)
		native.SetNextItemWidth(propertyFieldWidth)
		v := int32(n.Get())
		if native.InputInt(toolParamPrefix+n.Label, &v) {
			n.Set(int(v))
		}
	}
}

// drawToolTextParams renders the text params, seeding the shared buffer once when the
// dialog opens so the user's typing persists across frames.
func drawToolTextParams(params app.ToolParams) {
	if len(params.Texts) == 0 {
		toolText.open = false
		return
	}
	tp := params.Texts[0]
	if !toolText.open {
		seedToolText(tp.Get())
		toolText.open = true
	}
	propertyRow(tp.Label)
	native.SetNextItemWidth(propertyComboWidth)
	if native.InputText(toolParamPrefix+tp.Label, toolText.buf) {
		tp.Set(string(bytes.TrimRight(toolText.buf, "\x00")))
	}
}

func seedToolText(text string) {
	for i := range toolText.buf {
		toolText.buf[i] = 0
	}
	copy(toolText.buf, text)
}
