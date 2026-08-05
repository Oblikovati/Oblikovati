// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"regexp"
	"testing"
)

// TestEveryFeatureToolHasADialogPath covers the part-feature tools. The ~110 tools activated
// through the plain StartTool — every sketch, 3D-sketch, drawing, assembly, work-feature,
// measure and settings tool — had no equivalent guard at all, so one of them could ship with no
// way for the user to see what it wanted or what it had gathered (#2051).
//
// A non-feature tool need not have a property panel: many are pure pick-and-go. But it should
// offer SOMETHING of its own — a bespoke dialog, the generic parameter panel, or at least a
// status-bar Prompt telling the user what to click. A tool with none of the three falls back to
// BuildStatus's generic commit-readiness line, which names no step and no target.
//
// The allowlist below is a RATCHET, not a blessing: it pins the tools that shipped that way so
// no new one joins them. Each is worth a specific prompt the next time it is touched.

// promptMethodRe matches a tool's Prompt method — the minimum interaction surface.
var promptMethodRe = regexp.MustCompile(`func \(\w+ \*(\w+Tool)\) Prompt\(`)

// plainToolsWithoutInteraction are the plain-StartTool tools with no surface of their own,
// pinned at the count they shipped with (#2051). Delete an entry when its tool gains one.
var plainToolsWithoutInteraction = map[string]string{
	"CloudMoveTool":         "drag-driven cloud move; the drag itself is the affordance (#645)",
	"ContinuityCheckTool":   "analysis probe: picks two faces and reports in the status bar",
	"CropBoxTool":           "drag-driven crop box; the box handles are the affordance (#645)",
	"IncludeGeometry3DTool": "3D-sketch include: picks model geometry and converts it",
	"Line3DTool":            "3D-sketch geometry: click points in model space",
	"PartsListTool":         "drawing parts list: places the table where clicked",
	"PlaceComponentTool":    "assembly place: the file dialog IS its interaction",
	"Point3DTool":           "3D-sketch geometry: click a point in model space",
	"ProjectGeometryTool":   "sketch project: picks model geometry and projects it",
	"SheetMetalStyleTool":   "settings dialog drawn by the sheet-metal router, not a tool panel",
	"SketchCircleTool":      "sketch geometry: click centre then radius",
	"SketchRectangleTool":   "sketch geometry: click two corners",
	"SurfaceCurve3DTool":    "3D-sketch surface curve: picks the surface and the curve",
}

// TestPlainStartToolsHaveAnInteractionSurface fails when a plain-StartTool tool offers no
// dialog, no parameters and no prompt. Red-verify by deleting a sketch tool's Prompt method.
func TestPlainStartToolsHaveAnInteractionSurface(t *testing.T) {
	started := plainStartedToolConstructors(t)
	param := toolsImplementingParams(t)
	dialog := toolsWithRegisteredDialog(t)
	prompt := toolsWithPrompt(t)
	returns := toolConstructorReturns(t)

	for name := range started {
		tool := name
		if typ, ok := returns["New"+name]; ok {
			tool = typ
		}
		if hasAnyInteraction(tool, param, dialog, prompt) {
			continue
		}
		if _, allowed := plainToolsWithoutInteraction[tool]; allowed {
			continue
		}
		t.Errorf("%s reaches the plain StartTool with no interaction surface: no registered dialog, "+
			"no ParameterizedTool.Params, and no Prompt — the user gets an armed tool that says "+
			"nothing about what to click (#2051). Give it one, or declare it in "+
			"plainToolsWithoutInteraction with the reason.", tool)
	}
	for tool := range plainToolsWithoutInteraction {
		if hasAnyInteraction(tool, param, dialog, prompt) {
			t.Errorf("plainToolsWithoutInteraction entry %q now has an interaction surface; delete the entry", tool)
		}
	}
}

// hasAnyInteraction reports whether the tool offers any of the three surfaces.
func hasAnyInteraction(tool string, param, dialog, prompt map[string]struct{}) bool {
	for _, set := range []map[string]struct{}{param, dialog, prompt} {
		if _, ok := set[tool]; ok {
			return true
		}
	}
	return false
}

// toolsWithPrompt collects the tools that declare a Prompt method.
func toolsWithPrompt(t *testing.T) map[string]struct{} {
	t.Helper()
	out := map[string]struct{}{}
	for _, src := range readGoSources(t, "../app/*.go") {
		collectMatches(promptMethodRe, src, out)
	}
	return out
}
