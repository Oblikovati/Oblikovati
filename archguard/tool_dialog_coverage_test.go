// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Audit I4: head/ui once carried a hand-maintained roll-call of per-tool drawXDialog
// calls in chrome.go; a new feature tool minus its call was headless-invisible — the tool
// activated, collected nothing, and the user saw no UI (the #1521 shape). Dialogs now
// self-register (head/ui/tool_dialog_registry.go) and this source-level gate asserts every
// feature tool resolves to exactly one dialog PATH: a bespoke registered dialog, the
// generic parameter panel (ParameterizedTool.Params), or the explicit no-dialog allowlist
// below. A new feature tool without any of the three fails CI, not a live session.

// featureToolsWithoutDialog are the feature tools that legitimately need no property
// panel: they are driven entirely by viewport/browser picks and commit with no
// configurable parameter, so a dialog would be an empty window. Each entry names why —
// an undeclared gap (a tool that SHOULD have a panel) fails the coverage test instead.
var featureToolsWithoutDialog = map[string]string{
	"HullTool":                       "no inputs — the running solids are the input, just OK (feature_extra_tools.go)",
	"PatchTool":                      "AutoCommitOnPick — fills the region the instant it is clicked, nothing to configure",
	"FeatureSketchDrivenPatternTool": "driven by source-feature + driving-sketch picks; one occurrence per sketch point, no scalar params",
}

func TestEveryFeatureToolHasADialogPath(t *testing.T) {
	feature := featureToolConstructors(t)
	param := toolsImplementingParams(t)
	dialog := toolsWithRegisteredDialog(t)
	for tool := range feature {
		if _, ok := param[tool]; ok {
			continue
		}
		if _, ok := dialog[tool]; ok {
			continue
		}
		if _, ok := featureToolsWithoutDialog[tool]; ok {
			continue
		}
		t.Errorf("feature tool %s has no dialog path: it neither implements ParameterizedTool "+
			"(generic panel), registers a bespoke dialog (head/ui registerToolDialog), nor is "+
			"declared in featureToolsWithoutDialog — a new tool minus its UI must fail here, not a "+
			"live session (#1521, audit I4)", tool)
	}
	assertAllowlistFresh(t, feature, param, dialog)
}

// assertAllowlistFresh fails when a featureToolsWithoutDialog entry is stale — the tool is
// no longer a feature tool, or has since gained a dialog/params — so the allowlist cannot
// silently mask a real gap after the tool changes.
func assertAllowlistFresh(t *testing.T, feature, param, dialog map[string]struct{}) {
	t.Helper()
	for tool := range featureToolsWithoutDialog {
		if _, ok := feature[tool]; !ok {
			t.Errorf("featureToolsWithoutDialog entry %q is stale — no longer a feature tool; delete it", tool)
		}
		if _, ok := param[tool]; ok {
			t.Errorf("featureToolsWithoutDialog entry %q now implements ParameterizedTool; delete the allowlist entry", tool)
		}
		if _, ok := dialog[tool]; ok {
			t.Errorf("featureToolsWithoutDialog entry %q now has a bespoke dialog; delete the allowlist entry", tool)
		}
	}
}

var (
	startFeatureToolRe = regexp.MustCompile(`StartFeatureTool\(New(\w+)\(`)
	partFeatureThunkRe = regexp.MustCompile(`func\(\) PartFeatureTool \{ return New(\w+)\(\)`)
	paramsMethodRe     = regexp.MustCompile(`func \(\w+ \*(\w+Tool)\) Params\(\) ToolParams`)
	registerDialogRe   = regexp.MustCompile(`(?s)registerToolDialog\(([^)]*)\)`)
	toolLiteralRe      = regexp.MustCompile(`"(\w+Tool)"`)
)

// featureToolConstructors collects every part-feature tool activated through
// StartFeatureTool (directly or via a func() PartFeatureTool flyout thunk).
func featureToolConstructors(t *testing.T) map[string]struct{} {
	t.Helper()
	out := map[string]struct{}{}
	for _, src := range readGoSources(t, "../app/commands_*.go") {
		collectMatches(startFeatureToolRe, src, out)
		collectMatches(partFeatureThunkRe, src, out)
	}
	if len(out) == 0 {
		t.Fatal("no feature-tool constructors found — the StartFeatureTool scan is broken")
	}
	return out
}

// toolsImplementingParams collects the tools that expose the generic parameter panel.
func toolsImplementingParams(t *testing.T) map[string]struct{} {
	t.Helper()
	out := map[string]struct{}{}
	for _, src := range readGoSources(t, "../app/*.go") {
		collectMatches(paramsMethodRe, src, out)
	}
	return out
}

// toolsWithRegisteredDialog collects the tool keys claimed by head/ui registerToolDialog
// calls — the bespoke-dialog side of the contract.
func toolsWithRegisteredDialog(t *testing.T) map[string]struct{} {
	t.Helper()
	out := map[string]struct{}{}
	for _, src := range readGoSources(t, "../head/ui/*.go") {
		for _, call := range registerDialogRe.FindAllStringSubmatch(src, -1) {
			for _, lit := range toolLiteralRe.FindAllStringSubmatch(call[1], -1) {
				out[lit[1]] = struct{}{}
			}
		}
	}
	return out
}

// collectMatches records capture group 1 of every match of re in src.
func collectMatches(re *regexp.Regexp, src string, out map[string]struct{}) {
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		out[m[1]] = struct{}{}
	}
}

// readGoSources reads every non-test .go file matching the glob (relative to the archguard
// package dir) and returns their contents.
func readGoSources(t *testing.T, glob string) []string {
	t.Helper()
	files, err := filepath.Glob(glob)
	if err != nil || len(files) == 0 {
		t.Fatalf("globbing %q: %v (found %d)", glob, err, len(files))
	}
	out := make([]string, 0, len(files))
	for _, f := range files {
		if filepath.Base(f) == "" || len(f) > 8 && f[len(f)-8:] == "_test.go" {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		out = append(out, string(src))
	}
	return out
}
