// SPDX-License-Identifier: GPL-2.0-only

package cmdline

import (
	"os"
	"strings"
	"testing"
)

// manualDocPath is the generated command manual, relative to this package directory.
const manualDocPath = "../../architecture/mapping/autocad-command-map.md"

// TestManualComplete asserts the manual is well-formed: every command has at least one
// multi-letter word with the canonical first, plus a non-empty summary and example — the
// fields a generated manual needs. (DefaultVocabulary also panics on a violation; this gives
// a readable failure.)
func TestManualComplete(t *testing.T) {
	for _, c := range DefaultVocabulary().Manual() {
		if len(c.Words) == 0 || c.Canonical != c.Words[0] {
			t.Errorf("%s: canonical %q is not the first word in %v", c.Action, c.Canonical, c.Words)
		}
		for _, w := range c.Words {
			if len([]rune(w)) < 2 {
				t.Errorf("%s: single-letter word %q in the static manual", c.Action, w)
			}
		}
		if strings.TrimSpace(c.Summary) == "" || strings.TrimSpace(c.Example) == "" {
			t.Errorf("%s (%s): missing summary or example", c.Action, c.Canonical)
		}
	}
}

// TestCommandManualInSync keeps the committed Markdown manual byte-identical to RenderManual,
// so the doc can never drift from the vocabulary. Regenerate with:
//
//	UPDATE_MANUAL=1 go test ./app/cmdline -run TestCommandManualInSync
func TestCommandManualInSync(t *testing.T) {
	got := DefaultVocabulary().RenderManual()
	if os.Getenv("UPDATE_MANUAL") != "" {
		if err := os.WriteFile(manualDocPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write manual: %v", err)
		}
		return
	}
	want, err := os.ReadFile(manualDocPath)
	if err != nil {
		t.Fatalf("read manual %s: %v (regenerate with UPDATE_MANUAL=1)", manualDocPath, err)
	}
	if string(want) != got {
		t.Errorf("command manual %s is out of date; regenerate with UPDATE_MANUAL=1 go test ./app/cmdline -run TestCommandManualInSync", manualDocPath)
	}
}
