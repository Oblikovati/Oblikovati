//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"sort"

	"oblikovati.org/app"
)

// A tool's property dialog is a modeless ImGui window drawn every frame while its tool is
// active; each dialog self-gates by querying the session (e.g. s.ActiveChamfer() returns
// nil when its tool is not running) and returns early otherwise. Rather than a
// hand-maintained roll-call in chrome.go — where a new tool minus its call is
// headless-invisible, the #1521 shape (audit I4) — the dialogs are assembled into one
// explicit set by defaultToolDialogSet; chrome draws the set, and a source-level
// completeness check (archguard TestEveryFeatureToolHasADialogPath) asserts every feature
// tool resolves to a registered dialog, the generic parameter panel (ParameterizedTool),
// or an explicit no-dialog allowlist.
//
// The set is built by ONE explicit constructor at package scope, never by scattered init()
// side effects (audit B6, #1617): init()-time registration makes correctness depend on a
// binary's import list (a forgotten blank import silently drops a dialog) and blocks
// building a minimal set in tests. Enumerating every dialog in one constructor keeps the
// contract in a single readable place.

// toolDialogDraw draws one tool's property panel for the frame. It MUST self-gate on the
// session — a no-op when its tool is not active — because the set draws every dialog each
// frame and relies on at most one painting a window.
type toolDialogDraw func(s *app.Session)

// toolDialogSet is an explicitly constructed collection of tool property dialogs plus the
// set of tool type names each covers (for the completeness check).
type toolDialogSet struct {
	draws        []toolDialogDraw
	coveredTools map[string]struct{}
}

// newToolDialogSet returns an empty set; tests build a minimal one, the head uses the
// default assembled by defaultToolDialogSet.
func newToolDialogSet() *toolDialogSet {
	return &toolDialogSet{coveredTools: map[string]struct{}{}}
}

// registerToolDialog adds a bespoke dialog draw and the tool type name(s) it serves (e.g.
// "ChamferTool"). It panics on a duplicate tool key — two dialogs claiming one tool is a
// construction bug, caught when the default set is built, not in a live session.
func (s *toolDialogSet) registerToolDialog(draw toolDialogDraw, tools ...string) {
	if draw == nil || len(tools) == 0 {
		panic(fmt.Sprintf("registerToolDialog: need a draw func and ≥1 tool key, got draw!=nil=%v tools=%v", draw != nil, tools))
	}
	for _, name := range tools {
		if _, dup := s.coveredTools[name]; dup {
			panic(fmt.Sprintf("registerToolDialog: duplicate dialog registration for tool %q", name))
		}
		s.coveredTools[name] = struct{}{}
	}
	s.draws = append(s.draws, draw)
}

// draw renders every dialog in the set for the frame; each draw self-gates, so at most one
// paints. This replaces chrome.go's per-tool roll-call.
func (s *toolDialogSet) draw(sess *app.Session) {
	for _, d := range s.draws {
		d(sess)
	}
}

// toolNames returns the sorted tool type names the set covers.
func (s *toolDialogSet) toolNames() []string {
	names := make([]string, 0, len(s.coveredTools))
	for n := range s.coveredTools {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// toolDialogs is the head's default dialog set, assembled once by explicit construction —
// the composition point for head/ui's own tool panels (every DrawChrome caller shares it).
var toolDialogs = defaultToolDialogSet()

// defaultToolDialogSet assembles the full set of bespoke tool dialogs. Split into
// panel-scoped helpers so each stays within the statement budget and the grouping mirrors
// the ribbon.
func defaultToolDialogSet() *toolDialogSet {
	s := newToolDialogSet()
	registerSolidToolDialogs(s)
	registerSurfaceToolDialogs(s)
	registerSheetMetalToolDialogs(s)
	return s
}

// registerSolidToolDialogs adds the solid-feature and interaction dialogs.
func registerSolidToolDialogs(s *toolDialogSet) {
	s.registerToolDialog(drawExtrudeDialog, "ExtrudeTool")
	s.registerToolDialog(drawRevolveDialog, "RevolveTool")
	s.registerToolDialog(drawCoilDialog, "CoilTool")
	s.registerToolDialog(drawLoftDialog, "LoftTool")
	s.registerToolDialog(drawSweepDialog, "SweepTool")
	s.registerToolDialog(drawHoleDialog, "HoleTool")
	s.registerToolDialog(drawChamferDialog, "ChamferTool")
	s.registerToolDialog(drawThreadDialog, "ThreadTool")
	s.registerToolDialog(drawFilletDialog, "FilletTool")
	s.registerToolDialog(drawFaceFilletDialog, "FaceFilletTool")
	s.registerToolDialog(drawFullRoundFilletDialog, "FullRoundFilletTool")
	s.registerToolDialog(drawShellDialog, "ShellTool")
	s.registerToolDialog(drawSplitDialog, "SplitTool")
	s.registerToolDialog(drawGripSnapDialog, "GripSnapTool")
	s.registerToolDialog(drawMeasureDialog, "MeasureTool")
}

// registerSurfaceToolDialogs adds the surface-edit dialogs.
func registerSurfaceToolDialogs(s *toolDialogSet) {
	s.registerToolDialog(drawFaceOffsetDialog, "FaceOffsetTool")
	s.registerToolDialog(drawDraftDialog, "DraftTool")
	s.registerToolDialog(drawDeleteFaceDialog, "DeleteFaceTool")
	s.registerToolDialog(drawReplaceFaceDialog, "ReplaceFaceTool")
	s.registerToolDialog(drawThickenDialog, "ThickenTool")
	s.registerToolDialog(drawUnwrapDialog, "UnwrapTool")
	s.registerToolDialog(drawSimplifyDialog, "SimplifyTool")
}

// registerSheetMetalToolDialogs adds the one Sheet Metal router, which serves all seventeen
// wall/modify/flat tools by their session accessor. SheetMetalStyleTool is a settings
// dialog (drawSheetMetalStyle), covered inside the router, not a feature commit.
func registerSheetMetalToolDialogs(s *toolDialogSet) {
	s.registerToolDialog(drawSheetMetalDialogs,
		"SheetMetalFaceTool", "SheetMetalFlangeTool", "SheetMetalContourFlangeTool",
		"SheetMetalLoftedFlangeTool", "SheetMetalHemTool", "SheetMetalContourRollTool",
		"SheetMetalBendTool", "SheetMetalFoldTool", "SheetMetalCornerTool",
		"SheetMetalCornerSeamTool", "SheetMetalCutTool", "SheetMetalPunchTool",
		"SheetMetalRipTool", "SheetMetalUnfoldTool", "SheetMetalRefoldTool",
		"SheetMetalCosmeticBendTool", "SheetMetalLipTool")
}

// drawRegisteredToolDialogs draws the head's default dialog set for the frame.
func drawRegisteredToolDialogs(s *app.Session) { toolDialogs.draw(s) }

// registeredDialogTools returns the sorted tool type names that have a bespoke dialog in
// the default set. The head-side completeness test reads it; the archguard CI gate reads
// the source instead.
func registeredDialogTools() []string { return toolDialogs.toolNames() }
