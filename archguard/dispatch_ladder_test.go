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

// "Dispatch is a classification that selects exactly one path. Never add to an ordered try-list,
// a first-fit ladder, or a 'load-bearing' order." (kernel ground rules.)
//
// A first-fit ladder is the shape that lets a kernel grow by accretion: each new failing input
// gets a rung, the rungs acquire an order nobody can justify, and the order becomes load-bearing
// because two rungs both accept some input and the earlier one wins by position. Nothing detected
// it (#2186).
//
// This guard matches the SHAPE, not the type. `[]func` is not the offence — running the entries
// until one succeeds is. A table whose entries ALL run and must all succeed is a composition, and
// the rules allow it: buildSetbackLoops (kernel/ops/blend/fillet_setback_extract.go) runs every
// band builder and honest-rejects the whole edge if any fails, so it is not flagged even though
// #2186 listed it as a third ladder. Detecting the pattern rather than the declaration is what
// keeps the guard from pressuring correct code into a worse shape.

// dispatchLadders are the first-fit ladders that exist today, each with the issue that replaces
// it with a classification. It may only SHRINK: a new ladder fails this test, and removing one
// demands its entry go with it.
var dispatchLadders = map[string]string{
	// 26 analytic recognizers tried in order; the comment on the table says the order is
	// load-bearing. ADR-0056 "What this deletes" names the recognizers a completed
	// reconstruction retires.
	"kernel/ops/boolean/boolean_curved.go": "#3397",
	// Surface-specific meshers tried in priority order before the generic (u,v) trim path.
	// #3235-#3251 delete them one at a time as the general mesher covers each.
	"kernel/ops/tessellate/tessellate_trim_special.go": "#3409",
}

func TestNoFirstFitDispatchLadders(t *testing.T) {
	t.Parallel()
	found := scanDispatchLadders(t)
	var added, gone []string
	for _, f := range found {
		if _, known := dispatchLadders[f]; !known {
			added = append(added, f)
		}
	}
	seen := map[string]bool{}
	for _, f := range found {
		seen[f] = true
	}
	for f, issue := range dispatchLadders {
		if !seen[f] {
			gone = append(gone, f+" (was "+issue+")")
		}
	}
	sort.Strings(added)
	sort.Strings(gone)
	if len(added) > 0 {
		t.Errorf("a first-fit dispatch ladder appeared — the loop runs candidates until one "+
			"succeeds, so its ORDER decides the answer. Classify the input and select exactly one "+
			"path instead (#2186):\n  %s", strings.Join(added, "\n  "))
	}
	if len(gone) > 0 {
		t.Errorf("these ladders are gone — good; DELETE their dispatchLadders entries so the "+
			"guard holds the new floor:\n  %s", strings.Join(gone, "\n  "))
	}
}

// scanDispatchLadders returns each kernel file containing a range loop that calls its own range
// value and escapes on that call's SUCCESS.
func scanDispatchLadders(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	hits := map[string]bool{}
	err := filepath.WalkDir(filepath.Join("..", "kernel"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", p, err)
		}
		rel := filepath.ToSlash(strings.TrimPrefix(filepath.ToSlash(p), "../"))
		ast.Inspect(f, func(n ast.Node) bool {
			rs, ok := n.(*ast.RangeStmt)
			if !ok || rs.Value == nil {
				return true
			}
			id, ok := rs.Value.(*ast.Ident)
			if ok && firstFitBody(rs.Body, id.Name) {
				hits[rel] = true
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking kernel/: %v", err)
	}
	out := make([]string, 0, len(hits))
	for f := range hits {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// firstFitBody reports whether the loop body CALLS the range value and escapes on its SUCCESS.
// Escaping on FAILURE is the composition shape (every entry must succeed), which is allowed.
func firstFitBody(body *ast.BlockStmt, name string) bool {
	called, ladder := false, false
	ast.Inspect(body, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if id, ok := c.Fun.(*ast.Ident); ok && id.Name == name {
				called = true
			}
		}
		if is, ok := n.(*ast.IfStmt); ok && escapesBlock(is.Body) && successGuard(is.Cond) {
			ladder = true
		}
		return true
	})
	return called && ladder
}

// escapesBlock reports whether a block returns or breaks out of the enclosing loop.
func escapesBlock(b *ast.BlockStmt) bool {
	found := false
	ast.Inspect(b, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.ReturnStmt, *ast.BranchStmt:
			found = true
		}
		return true
	})
	return found
}

// successGuard reports whether cond fires when the candidate SUCCEEDED. `ok` is a success guard;
// `!ok`, and any ||-chain leading with a negation, guards failure.
func successGuard(cond ast.Expr) bool {
	switch c := cond.(type) {
	case *ast.UnaryExpr:
		return c.Op != token.NOT
	case *ast.BinaryExpr:
		if c.Op == token.LOR {
			return successGuard(c.X)
		}
		return c.Op != token.NEQ
	}
	return true
}
