// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// model/feature's Add*/Set* methods on the *Features collections are how a feature gets authored.
// An exported one with no non-test caller is either dead surface — a convenience wrapper a richer
// sibling superseded — or a capability nothing in the product can reach. Both are worth failing on:
// the first is maintenance cost that looks like the way to author the feature, and the second is a
// feature gap that otherwise only surfaces in an audit (#2052).
//
// A method reached only from another method inside model/feature counts as called; that is a
// genuine internal helper and should simply not be exported.

// authoringMethodDecl matches an exported authoring method on a feature collection.
var authoringMethodDecl = regexp.MustCompile(`^func \(\w+ \*(\w*Features)\) ((?:Add|Set)[A-Za-z0-9_]*)\(`)

// authoringMethodsWithoutCaller is the allowlist: an exported authoring method that nothing calls
// yet, with the issue that will give it one. Entries go stale loudly — a method that gains a
// caller fails this test until its entry is deleted.
var authoringMethodsWithoutCaller = map[string]string{
	"AddFromCage":         "#2048 — freeform cage editing has no UI; the builder lands with that work",
	"AddSplitFacesByPath": "#2068 — split-face-along-a-path is buildable via the model API; the wire sketch reference + UI tool that call it are the follow-up",
}

// TestExportedAuthoringMethodsHaveACaller fails on an exported model/feature authoring method
// that no non-test source calls. Red-verify by adding an unused exported Add* to a collection.
func TestExportedAuthoringMethodsHaveACaller(t *testing.T) {
	sources := moduleGoSources(t)
	decls := authoringMethodDeclarations(t)
	if len(decls) < 50 {
		t.Fatalf("found only %d authoring methods — the declaration scan is broken, not the code", len(decls))
	}
	for name, where := range decls {
		called := authoringMethodIsCalled(name, sources)
		reason, allowed := authoringMethodsWithoutCaller[name]
		switch {
		case called && allowed:
			t.Errorf("authoringMethodsWithoutCaller entry %q is stale — it now has a caller; delete the entry", name)
		case !called && !allowed:
			t.Errorf("%s declares exported %s with no non-test caller. Either delete it (a richer "+
				"sibling supersedes it) or give it one — an authoring method nothing calls is dead "+
				"surface or an unreachable capability (#2052). To defer, add it to "+
				"authoringMethodsWithoutCaller with the issue.", where, name)
		case !called && allowed:
			t.Logf("%s: %s (allowlisted: %s)", where, name, reason)
		}
	}
}

// authoringMethodIsCalled reports whether any non-test source calls the method by name.
//
// It matches on ".Name(", so an identically named method on an unrelated type counts as a call.
// That direction is safe for this guard: it can only make a dead method look live, never the
// reverse, so a failure is always a real finding.
func authoringMethodIsCalled(name string, sources []goSource) bool {
	call := "." + name + "("
	for _, f := range sources {
		if strings.Contains(f.src, call) {
			return true
		}
	}
	return false
}

// authoringMethodDeclarations maps each exported authoring method to the file:line declaring it.
// occtparity is skipped: it is a test-support package, not authoring surface.
func authoringMethodDeclarations(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	root := "../model/feature"
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "occtparity" {
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
		for i, ln := range strings.Split(string(src), "\n") {
			if m := authoringMethodDecl.FindStringSubmatch(ln); m != nil {
				out[m[2]] = filepath.ToSlash(strings.TrimPrefix(path, "../")) + ":" + itoa(i+1)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return out
}

// itoa avoids pulling strconv in for one call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
