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

// The ground rules allow ONE predicate package: "Every topological decision (side,
// inside/outside, coincidence, orientation, convexity) uses an exact or filtered predicate. One
// predicate package (kernel/predicates). Delete duplicates."
//
// #2193 reported a second live package, math/predicate, imported by four files in kernel/ops.
// It no longer exists — it was deleted before this guard was written. That is precisely why the
// guard is here: a deletion that nothing enforces comes back. Two independent orientation
// predicates that can disagree on a near-degenerate sign is the worst kind of duplicate,
// because both are "right" in isolation and the bodies they build are inconsistent.

// predicateAuthority is the one package allowed to DECIDE a geometric sign from scratch.
const predicateAuthority = "kernel/predicates"

// exactRationalTier is the one package allowed to compute an orientation sign without calling
// predicateAuthority, and only because it cannot: kernel/meshbool works on CONSTRUCTED
// intersection points that are exact rationals, not binary64, so the float-filtered predicates
// cannot represent its operands at all. Its fast path still delegates (predicates.Orient3D on
// the binary64 quad), and TestOrient3DFastPathMatchesExact pins the two tiers to the same
// answer — so this is one authority with a wider number type, not a second opinion.
const exactRationalTier = "kernel/meshbool"

// signDecider matches a function that names itself an orientation or in-circle predicate. Those
// names are the ones whose sign a topological decision is allowed to trust.
var signDecider = regexp.MustCompile(`^func (?:\([^)]*\) )?(?i:orient2d|orient3d|incircle)\w*\(`)

// TestOnePredicatePackage fails if a second predicate package appears, if the deleted
// math/predicate comes back, or if a sign-deciding function outside the authority computes its
// own answer instead of delegating.
func TestOnePredicatePackage(t *testing.T) {
	t.Parallel()
	var rival, revived, ownSign []string
	err := filepath.WalkDir("..", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			// Skip dot-directories (.git, .claude/worktrees — agent scratch copies of this very
			// tree, which would otherwise report every rule as violated) and vendored trees.
			if p != ".." && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "third_party") {
				return filepath.SkipDir
			}
			if (name == "predicate" || name == "predicates") && !strings.HasSuffix(filepath.ToSlash(p), predicateAuthority) {
				rival = append(rival, filepath.ToSlash(p))
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(strings.TrimPrefix(filepath.ToSlash(p), "../"))
		if strings.HasPrefix(rel, predicateAuthority+"/") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		text := string(src)
		if strings.Contains(text, `"oblikovati.org/math/predicate"`) {
			revived = append(revived, rel)
		}
		if strings.HasPrefix(rel, exactRationalTier+"/") {
			return nil
		}
		for _, fn := range signDeciders(text) {
			if !strings.Contains(fn, "predicates.") {
				ownSign = append(ownSign, rel)
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the repo: %v", err)
	}
	sortStrings(rival)
	sortStrings(revived)
	sortStrings(ownSign)
	if len(rival) > 0 {
		t.Errorf("a SECOND predicate package appeared — there is one (%s), and duplicates are "+
			"deleted, not added:\n  %s", predicateAuthority, strings.Join(rival, "\n  "))
	}
	if len(revived) > 0 {
		t.Errorf("math/predicate is back — it was deleted as the duplicate of %s (#2193):\n  %s",
			predicateAuthority, strings.Join(revived, "\n  "))
	}
	if len(ownSign) > 0 {
		t.Errorf("these files decide an orientation/in-circle sign WITHOUT %s — a naive float "+
			"determinant returns the wrong sign on near-degenerate input, and two deciders that "+
			"can disagree build inconsistent bodies. Delegate, or state why the operands cannot "+
			"be represented (see %s):\n  %s",
			predicateAuthority, exactRationalTier, strings.Join(ownSign, "\n  "))
	}
}

// signDeciders returns the body of each function in src whose name declares it an orientation or
// in-circle predicate. A body is read to its closing brace at column 0, which is what gofmt
// guarantees for a top-level declaration.
func signDeciders(src string) []string {
	var out []string
	lines := strings.Split(src, "\n")
	for i, l := range lines {
		if !signDecider.MatchString(l) {
			continue
		}
		var body []string
		for j := i; j < len(lines); j++ {
			body = append(body, lines[j])
			if j > i && lines[j] == "}" {
				break
			}
		}
		out = append(out, strings.Join(body, "\n"))
	}
	return out
}
