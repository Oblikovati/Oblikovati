// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestNoUnjustifiedAbsoluteEpsilons is the mechanical guard for ADR-0042 / #1399: a kernel
// decision must not be gated on a cm-anchored absolute length epsilon. A model-relative
// tolerance carries no literal — it reads res.Weld()/res.Plane() from a geom.Resolution — so a
// bare `1e-N` that survives is either a length tolerance still needing relativising (a defect)
// or a deliberately dimensionless one that must SAY SO with a `// tol:<kind>` annotation.
//
// Allowed annotations (the literal stays absolute, justified):
//
//	tol:angular     — a cosine/sine/dot threshold on unit vectors (dimensionless)
//	tol:parametric  — a normalised curve/surface parameter u/v/t in [0,1] or knot space
//	tol:numeric     — a convergence / determinant / rank / slope guard, not a model length
//	tol:area        — scales with size² (a near-zero-area test)
//	tol:volume      — scales with size³
//	tol:calibrated  — a length tolerance deliberately kept absolute because it is validated
//	                  against an external oracle in a normalised frame (e.g. the ruled (u,v)
//	                  arrangement split); converting it shifts borderline classifications
//
// SCOPE (#2189): every non-test .go file under kernel/, which is what CLAUDE.md always claimed.
// It previously scanned a hand-maintained list of 91 paths — 10% of the 861 files — so the
// other 90%, including every fillet tolerance and the identity-deciding literals in the weld
// paths, were unguarded. The list is now INVERTED: clean is the default and toleranceDebt is
// the exception, so a new file is covered the moment it is written rather than when someone
// remembers to add it.
//
// It moved out of kernel/ops for the same reason: a rule over the whole kernel does not belong
// to one package inside it.
func TestNoUnjustifiedAbsoluteEpsilons(t *testing.T) {
	t.Parallel()
	got := scanToleranceDebt(t)
	var rose, fell, stale []string
	for path, n := range got {
		if owed := toleranceDebt[path]; n > owed {
			rose = append(rose, path+": "+strconv.Itoa(n)+" unjustified literal(s), budget "+strconv.Itoa(owed))
		} else if n < owed {
			fell = append(fell, "\""+path+"\": "+strconv.Itoa(n)+",")
		}
	}
	for path := range toleranceDebt {
		if _, ok := got[path]; !ok {
			stale = append(stale, path)
		}
	}
	sort3(rose, fell, stale)
	if len(rose) > 0 {
		t.Errorf("unjustified absolute length epsilon(s) added — relativise them "+
			"(geom.ResolutionForBox/ForSize(...).Weld()/.Plane()) or annotate `// tol:<kind>` on the "+
			"line:\n%s", strings.Join(rose, "\n"))
	}
	if len(fell) > 0 {
		t.Errorf("tolerance debt FELL — good; lower these toleranceDebt entries so the ratchet "+
			"holds the new floor:\n%s", strings.Join(fell, "\n"))
	}
	if len(stale) > 0 {
		t.Errorf("these toleranceDebt files are now clean or gone — DELETE their entries:\n  %s",
			strings.Join(stale, "\n  "))
	}
}

// scanToleranceDebt counts, per kernel file, the lines carrying an unjustified absolute
// tolerance literal. The key is the path relative to kernel/, so moving a file between
// packages inside the kernel shows up as one deletion and one addition rather than as a
// mystery.
func scanToleranceDebt(t *testing.T) map[string]int {
	t.Helper()
	got := map[string]int{}
	err := filepath.WalkDir("../kernel", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		n := 0
		for _, line := range strings.Split(string(src), "\n") {
			code, comment, _ := strings.Cut(line, "//")
			if toleranceLiteral(code) && !strings.Contains(comment, "tol:") {
				n++
			}
		}
		if n > 0 {
			rel, _ := filepath.Rel("../kernel", p)
			got[filepath.ToSlash(rel)] = n
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking kernel/: %v", err)
	}
	return got
}

// epsilonLiteral matches a bare small scientific literal (1e-N).
var epsilonLiteral = regexp.MustCompile(`[^0-9a-zA-Z_.]1(?:\.0)?e-[0-9]+`)

// reciprocalGrid matches the evasion forms of an absolute tolerance written as its
// reciprocal (#1610): a quantization multiplier (`* 1e5`, `* 100000.0`) or a division of
// one by a named tolerance/grid (`1.0/tol`). These carry the same cm-anchored scale
// assumption as a bare 1e-N and must be relativised or annotated identically.
var reciprocalGrid = regexp.MustCompile(
	`\*\s*1(?:\.0)?e\+?[0-9]+|\*\s*10{4,}(?:\.0)?\b|[^0-9a-zA-Z_.]1(?:\.0)?\s*/\s*[A-Za-z_]*(?:eps|tol|grid|Eps|Tol|Grid)`)

// toleranceLiteral reports whether a code line (comment already stripped) carries an
// absolute-tolerance literal in either direct or reciprocal form.
func toleranceLiteral(code string) bool {
	return epsilonLiteral.MatchString(" "+code) || reciprocalGrid.MatchString(" "+code)
}

// sort3 sorts the three report slices so the failure text is stable run to run.
func sort3(a, b, c []string) {
	for _, s := range [][]string{a, b, c} {
		sortStrings(s)
	}
}

// sortStrings is an insertion sort — the slices are short and this keeps the guard
// dependency-free.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// TestToleranceGuardCatchesEvasionForms is the guard-the-guard self-test (#1610): the
// matcher must catch each known evasion spelling, and must not fire on ordinary
// arithmetic — otherwise the whitelist gives false confidence.
func TestToleranceGuardCatchesEvasionForms(t *testing.T) {
	t.Parallel()
	caught := []string{
		"k := int64(x * 1e6)",     // the old quantCoord reciprocal grid
		"const q = x * 1e5",       // the old weldKey reciprocal grid
		"grid := v * 100000.0",    // spelled-out reciprocal
		"cells := 1.0 / tol",      // one-over-a-named-tolerance
		"n := 1 / weldEps",        // one-over-eps identifier
		"if d < 1e-6 {",           // the classic direct form
		"tol := span*1e-9 + 1e-9", // absolute floor hiding behind a relative term
	}
	for _, line := range caught {
		if !toleranceLiteral(line) {
			t.Errorf("evasion form not caught: %q", line)
		}
	}
	clean := []string{
		"area := w * h",
		"x := 2 * stdmath.Pi",
		"s := v.Scale(1 / norm)",
		"r := 1 / (radius * radius)",
		"i := n * 100",
	}
	for _, line := range clean {
		if toleranceLiteral(line) {
			t.Errorf("ordinary arithmetic falsely flagged: %q", line)
		}
	}
}
