// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// "One naming mechanism, one namespace per feature instance. Ambiguous key resolution is an
// error; unguarded first-match lookups are forbidden." (kernel ground rules.)
//
// #2191 framed this as a rule about the 79 CALLERS of Find*ByKey — no such lookup outside a
// guarded resolver. That is the wrong place to enforce it. The invariant belongs to the lookup
// itself: if Find*ByKey refuses an ambiguous key, every caller is safe and the rule needs no
// per-caller policing; if it does not, no amount of caller discipline helps, because the
// ambiguity is already resolved wrongly by the time a caller sees the result.
//
// So this guard is on the IMPLEMENTATION. A function named Find<Thing>ByKey must not take the
// first element of a multi-match slice without deciding what more than one means. Today all
// three do — EdgesByKey/FacesByKey return every match and their doc comment names the hazard
// exactly, but the Find* wrappers then return m[0] anyway. #2979 fixes that; until it lands the
// three are pinned here, so a FOURTH cannot be written the same way.

// firstMatchKeyLookups are the Find*ByKey functions that still return the first of an ambiguous
// match. Pinned to #2979. This map may only SHRINK — it is not a licence, it is a countdown.
var firstMatchKeyLookups = map[string]bool{
	"kernel/topo/body.go::FindFaceByKey":   true,
	"kernel/topo/body.go::FindEdgeByKey":   true,
	"kernel/topo/body.go::FindVertexByKey": true,
}

func TestKeyLookupsRefuseAmbiguity(t *testing.T) {
	t.Parallel()
	found := scanFirstMatchLookups(t)
	var added, fixed []string
	for _, f := range found {
		if !firstMatchKeyLookups[f] {
			added = append(added, f)
		}
	}
	seen := map[string]bool{}
	for _, f := range found {
		seen[f] = true
	}
	for f := range firstMatchKeyLookups {
		if !seen[f] {
			fixed = append(fixed, f)
		}
	}
	sort.Strings(added)
	sort.Strings(fixed)
	if len(added) > 0 {
		t.Errorf("a key lookup returns the FIRST of a possibly-ambiguous match — two entities "+
			"sharing a key is a naming collision, and picking one silently is the wrong-rebind "+
			"ADR-0043 §4 exists to prevent. Decide what more than one match means (#2191, "+
			"#2979):\n  %s", strings.Join(added, "\n  "))
	}
	if len(fixed) > 0 {
		t.Errorf("these lookups now refuse an ambiguous key — good; DELETE their "+
			"firstMatchKeyLookups entries so a new one cannot reuse the exemption:\n  %s",
			strings.Join(fixed, "\n  "))
	}
}

// scanFirstMatchLookups returns each Find*ByKey function that indexes [0] without ever comparing
// a match count against one.
func scanFirstMatchLookups(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	var hits []string
	seenAny := false
	err := filepath.WalkDir(filepath.Join("..", "kernel"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", p, err)
		}
		rel := filepath.ToSlash(strings.TrimPrefix(filepath.ToSlash(p), "../"))
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil || !keyLookupName(fd.Name.Name) {
				continue
			}
			seenAny = true
			if takesFirstMatch(fd.Body) {
				hits = append(hits, rel+"::"+fd.Name.Name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking kernel/: %v", err)
	}
	if !seenAny {
		t.Fatal("found no Find*ByKey functions under kernel/ — the guard would pass vacuously; " +
			"check keyLookupName if they were renamed")
	}
	sort.Strings(hits)
	return hits
}

// keyLookupName reports whether a function name declares a by-key lookup: Find<Thing>ByKey.
func keyLookupName(n string) bool {
	return strings.HasPrefix(n, "Find") && strings.HasSuffix(n, "ByKey")
}

// takesFirstMatch reports whether the body indexes element 0 of a match slice without ever
// comparing a count against 1. `len(m) > 0` is precisely the unguarded form: it proves there is
// at least one match, which is not the same as proving there is only one.
func takesFirstMatch(b *ast.BlockStmt) bool {
	indexesZero, countsOne := false, false
	ast.Inspect(b, func(n ast.Node) bool {
		if ix, ok := n.(*ast.IndexExpr); ok {
			if lit, ok := ix.Index.(*ast.BasicLit); ok && lit.Kind == token.INT && lit.Value == "0" {
				indexesZero = true
			}
		}
		if be, ok := n.(*ast.BinaryExpr); ok && comparesLenToOne(be) {
			countsOne = true
		}
		return true
	})
	return indexesZero && !countsOne
}

// comparesLenToOne reports whether a binary expression compares a len() call against the literal
// 1 — the shape of an ambiguity check, in either operand order.
func comparesLenToOne(be *ast.BinaryExpr) bool {
	return (isLenCall(be.X) && isOne(be.Y)) || (isLenCall(be.Y) && isOne(be.X))
}

func isLenCall(e ast.Expr) bool {
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	id, ok := c.Fun.(*ast.Ident)
	return ok && id.Name == "len"
}

func isOne(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "1"
}
