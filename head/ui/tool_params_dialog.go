//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"bytes"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
)

// One generic property dialog serves every tool that exposes parameters
// (app.ParameterizedTool) — the sketch Move/Copy/Rotate/Scale/Stretch, the rectangular/
// circular patterns, Slot/Chamfer/Text/Fillet/Offset — instead of a bespoke cgo dialog per
// tool (core/09 reflection-driven editing). It renders each parameter as a labelled input
// bound to the tool's field; OK commits, Cancel aborts. Modeless: the user still picks
// geometry in the viewport.

// toolText is the persistent edit buffer for a tool's text parameter (Dear ImGui's
// InputText needs a stable buffer across frames).
var toolText = struct {
	buf  []byte
	open bool
}{buf: make([]byte, 256)}

// drawToolParamsDialog shows the generic parameter dialog while a parameterized tool runs.
func drawToolParamsDialog(s *app.Session) {
	ti := s.ActiveTool()
	if ti == nil {
		toolText.open = false
		return
	}
	pt, ok := ti.Tool().(app.ParameterizedTool)
	if !ok {
		toolText.open = false
		return
	}
	params := pt.Params()
	if params.Empty() {
		toolText.open = false
		return
	}
	native.SetNextWindowSize(280, 240)
	if native.Begin(ti.Name()) {
		drawToolFloatParams(params)
		drawToolIntParams(params)
		drawToolBoolParams(params)
		drawToolTextParams(params)
		drawToolParamButtons(s)
	}
	native.End()
}

func drawToolBoolParams(params app.ToolParams) {
	for _, b := range params.Bools {
		v := b.Get()
		if native.Checkbox(b.Label, &v) {
			b.Set(v)
		}
	}
}

func drawToolFloatParams(params app.ToolParams) {
	for _, f := range params.Floats {
		v := float32(f.Get())
		if native.InputFloat(f.Label, &v) {
			f.Set(float64(v))
		}
	}
}

func drawToolIntParams(params app.ToolParams) {
	for _, n := range params.Ints {
		v := int32(n.Get())
		if native.InputInt(n.Label, &v) {
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
	if native.InputText(tp.Label, toolText.buf) {
		tp.Set(string(bytes.TrimRight(toolText.buf, "\x00")))
	}
}

// drawToolParamButtons draws OK/Cancel; OK is disabled until the tool is ready.
func drawToolParamButtons(s *app.Session) {
	native.BeginDisabled(!s.ActiveTool().Tool().CanCommit())
	if native.Button("OK") {
		_ = s.OK()
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
	}
}

func seedToolText(text string) {
	for i := range toolText.buf {
		toolText.buf[i] = 0
	}
	copy(toolText.buf, text)
}
