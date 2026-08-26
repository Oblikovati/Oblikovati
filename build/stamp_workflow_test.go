// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestReleaseWorkflowStampsCorrectSymbolPath is the regression guard for issue #1228: the release
// build (.github/workflows/build.yml) addresses build.Version/Commit/Date with
// `-ldflags -X ${CORE}/build.Version=…`, and CORE MUST equal the module path in go.mod. When it did
// not ("oblikovati" vs "oblikovati.org"), the linker silently ignored every -X and shipped builds
// stamped Version="dev", so the channel-aware update check treated nightlies as dev and never
// offered updates. A linker -X to a non-existent symbol is a no-op, so only this static check
// catches the drift.
func TestReleaseWorkflowStampsCorrectSymbolPath(t *testing.T) {
	module := readModulePath(t)
	core := readWorkflowCore(t)
	if core != module {
		t.Fatalf("build.yml CORE = %q but go.mod module = %q; -ldflags -X %s/build.Version would be\n"+
			"ignored, shipping Version=%q (issue #1228). Set CORE to the module path.",
			core, module, core, Version)
	}
}

// readModulePath returns the module path from the repo-root go.mod (the build package lives at the
// module root, so go.mod is one directory up).
func readModulePath(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	t.Fatal("no module directive in go.mod")
	return ""
}

// readWorkflowCore returns the CORE env value from the reusable build workflow.
func readWorkflowCore(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../.github/workflows/build.yml")
	if err != nil {
		t.Fatalf("read build.yml: %v", err)
	}
	m := regexp.MustCompile(`(?m)^\s*CORE:\s*(\S+)\s*$`).FindStringSubmatch(string(b))
	if m == nil {
		t.Fatal("no CORE: entry in build.yml")
	}
	return m[1]
}
