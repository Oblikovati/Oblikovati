// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// "One naming mechanism, one namespace per feature INSTANCE." ADR-0043 §3 spells out what that
// means: model/feature passes each feature's unique name (Fillet1, Fillet2, …) as the lineage
// tag, "replacing the hardcoded fillet/chamfer constants, so two features of the same kind never
// share a namespace".
//
// A literal tag is a shared namespace. Two Fillet features that both mint fillet:v#3 have given
// two different vertices the same name, and the lookup that resolves it picks one — which is the
// wrong-rebind this rule and #2979 exist together to prevent. Neither half was guarded (#2185).
//
// The kernel is about half converted: 105 call sites pass a feat variable, 112 pass a literal.
// So this is a ratchet on the literal form, per package. kernel/ops/blend's 16 include
// filletAssemblyTag, the constant ADR-0043 §3 named explicitly and #2997 removes.

// literalLineageTags is the per-package count of topo.Tok calls whose namespace is a string
// literal rather than the owning feature's unique name. Baseline 2026-09-01. It may only shrink.
var literalLineageTags = map[string]int{
	"kernel/brep":                61,
	"kernel/ops":                 12,
	"kernel/ops/blend":           17,
	"kernel/ops/boolean":         1,
	"kernel/ops/internal/retopo": 4,
	"kernel/ops/surface":         11,
	"model/bodyapi":              1,
	"model/compdef":              1,
	"model/feature":              4,
}

func TestLineageTagsArePerInstance(t *testing.T) {
	t.Parallel()
	got := scanLiteralLineageTags(t)
	var rose, fell, stale []string
	for pkg, n := range got {
		switch owed := literalLineageTags[pkg]; {
		case n > owed:
			rose = append(rose, pkg+": "+strconv.Itoa(n)+" literal lineage tag(s), budget "+strconv.Itoa(owed))
		case n < owed:
			fell = append(fell, `"`+pkg+`": `+strconv.Itoa(n)+",")
		}
	}
	for pkg := range literalLineageTags {
		if _, ok := got[pkg]; !ok {
			stale = append(stale, pkg)
		}
	}
	sort.Strings(rose)
	sort.Strings(fell)
	sort.Strings(stale)
	if len(rose) > 0 {
		t.Errorf("a lineage tag is a string LITERAL — that is one namespace shared by every "+
			"instance of the feature, so two of them mint the same key for different entities. "+
			"Pass the feature's unique name (ADR-0043 §3, #2185):\n%s", strings.Join(rose, "\n"))
	}
	if len(fell) > 0 {
		t.Errorf("literal-tag debt FELL — good; lower these literalLineageTags entries so the "+
			"ratchet holds the new floor:\n%s", strings.Join(fell, "\n"))
	}
	if len(stale) > 0 {
		t.Errorf("these packages no longer mint a literal lineage tag — DELETE their entries:\n  %s",
			strings.Join(stale, "\n  "))
	}
}

// scanLiteralLineageTags counts, per package, the Tok calls whose first argument is a string
// literal. kernel/topo is skipped: it DEFINES Tok, and its own doc examples are not a namespace.
func scanLiteralLineageTags(t *testing.T) map[string]int {
	t.Helper()
	fset := token.NewFileSet()
	got := map[string]int{}
	seenAny := false
	for _, root := range []string{"kernel", "model"} {
		err := filepath.WalkDir(filepath.Join("..", root), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return err
			}
			rel := filepath.ToSlash(strings.TrimPrefix(filepath.ToSlash(p), "../"))
			if strings.HasPrefix(rel, "kernel/topo/") {
				return nil
			}
			f, err := parser.ParseFile(fset, p, nil, 0)
			if err != nil {
				t.Fatalf("parsing %s: %v", p, err)
			}
			pkg := filepath.ToSlash(filepath.Dir(rel))
			ast.Inspect(f, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if !ok || len(c.Args) == 0 {
					return true
				}
				sel, ok := c.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Tok" {
					return true
				}
				if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "topo" {
					return true
				}
				seenAny = true
				if lit, ok := c.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					got[pkg]++
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
	if !seenAny {
		t.Fatal("found no topo.Tok calls — the guard would pass vacuously; check whether Tok was renamed")
	}
	return got
}
