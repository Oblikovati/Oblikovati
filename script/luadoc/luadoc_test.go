// SPDX-License-Identifier: GPL-2.0-only

package luadoc

import (
	"os/exec"
	"strings"
	"testing"
)

// apiDir resolves the on-disk oblikovati.org/api module root via `go list` (honoring the
// go.work / CI replace), the same way the router's api-parity test locates the contract.
func apiDir(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "oblikovati.org/api").Output()
	if err != nil {
		t.Fatalf("go list api: %v", err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		t.Fatal("go list returned empty api dir")
	}
	return dir
}

// TestMethodsCoverWireSurface asserts the generator sees the full scriptable surface: a few
// hundred methods across many groups, with the expected dotted names present.
func TestMethodsCoverWireSurface(t *testing.T) {
	ms, err := Methods(apiDir(t))
	if err != nil {
		t.Fatalf("Methods: %v", err)
	}
	if len(ms) < 450 {
		t.Fatalf("parsed only %d methods; expected the full wire surface (~500)", len(ms))
	}
	want := map[string]bool{"documents.create": false, "sketch.create": false, "features.add": false, "parameters.set": false}
	groups := map[string]bool{}
	for _, m := range ms {
		groups[m.Group] = true
		if _, ok := want[m.Wire]; ok {
			want[m.Wire] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("expected method %q in the generated reference", name)
		}
	}
	if len(groups) < 40 {
		t.Errorf("expected many groups, got %d", len(groups))
	}
}

// TestMostMethodsHaveSummaries guards that the mcp:summary reuse actually lands: the large
// majority of methods carry a description (a parser regression would drop them to ~0).
func TestMostMethodsHaveSummaries(t *testing.T) {
	ms, err := Methods(apiDir(t))
	if err != nil {
		t.Fatalf("Methods: %v", err)
	}
	withSummary := 0
	for _, m := range ms {
		if strings.TrimSpace(m.Summary) != "" {
			withSummary++
		}
	}
	if withSummary*100 < len(ms)*90 {
		t.Errorf("only %d/%d methods have summaries; the mcp:summary join looks broken", withSummary, len(ms))
	}
}

// TestGenerateProducesManual checks the full render has the guide, the example, and the
// reference, and is internally consistent.
func TestGenerateProducesManual(t *testing.T) {
	exs := []Example{{Name: "demo.lua", Description: "A demo.", Source: "print('hi')\n"}}
	md, err := Generate(apiDir(t), exs)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, want := range []string{"# Lua Scripting", "## Examples", "### demo.lua", "## API reference", "### documents", "oblikovati.documents.create"} {
		if !strings.Contains(md, want) {
			t.Errorf("generated manual missing %q", want)
		}
	}
}
