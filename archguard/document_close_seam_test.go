// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Closing a document has to release the session state that points into it — the armed tool and
// the open sketch edit. Before #2040 every close path called Workspace().Close directly and
// released nothing, so the commit bar and the contextual Sketch tab survived the close and
// sketch commands answered ok while mutating the destroyed document's sketch.
//
// Session.CloseDocument is now the single seam that does the teardown. This guard keeps it that
// way: a new close path that reaches past it reintroduces the leak silently, which is exactly
// how the first one shipped.
const documentCloseSeam = "app/session_document_close.go"

// TestWorkspaceCloseOnlyReachedThroughTheSessionSeam fails when any non-test source outside
// model/doc (which owns Close) and the seam itself calls Workspace().Close.
// Red-verify by pointing head/ui/document_close.go back at s.Workspace().Close.
func TestWorkspaceCloseOnlyReachedThroughTheSessionSeam(t *testing.T) {
	for _, f := range moduleGoSources(t) {
		if strings.HasPrefix(f.rel, "model/doc/") || f.rel == documentCloseSeam {
			continue
		}
		if strings.Contains(f.src, "Workspace().Close(") {
			t.Errorf("%s calls Workspace().Close directly — use Session.CloseDocument so the armed "+
				"tool and the open sketch edit are released with the document (#2040). The teardown "+
				"lives in %s.", f.rel, documentCloseSeam)
		}
	}
}

// goSource is one non-test .go file of this module, with its module-relative path.
type goSource struct {
	rel string
	src string
}

// skippedSourceDirs are trees whose contents are not this module's own Go sources.
var skippedSourceDirs = map[string]bool{
	".git": true, "dist": true, "build": true, "experiments": true,
	"test-utilities": true, "docs": true, "architecture": true,
}

// moduleGoSources reads every non-test .go file in the repo, head/ included (it is a separate
// module but the same application). Paths are relative to the repo root so failures name a file
// the reader can open.
func moduleGoSources(t *testing.T) []goSource {
	t.Helper()
	root := ".."
	var out []goSource
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Dot-directories hold tooling state, not sources — .claude/worktrees in particular
			// carries whole checkouts of the repo that would be scanned as if they were this one.
			if skippedSourceDirs[d.Name()] || (path != root && strings.HasPrefix(d.Name(), ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, goSource{rel: filepath.ToSlash(rel), src: string(src)})
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s for Go sources: %v", root, err)
	}
	if len(out) == 0 {
		t.Fatalf("found no Go sources under %s — the walk is broken, not the code", root)
	}
	return out
}
