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

// The SECOND shape a first-fit ladder takes (ADR-0061 stage 5, #3409).
//
// TestNoFirstFitDispatchLadders matches a range loop over a table of funcs. When that table is
// dissolved, the ladder does not have to die with it: writing the same rungs out as consecutive
// statements
//
//	if a, ok := recogniseX(...); ok { return a }
//	if b, ok := recogniseY(...); ok { return b }
//
// is the identical mechanism — each call both recognises and produces the payload, so the ORDER
// decides which recogniser gets the face, and two of them accepting is not a detected contradiction
// but the mechanism working. The first pass at retiring the tessellator's ladder did exactly this to
// the sphere family, and nothing saw it.
//
// The shape ALONE cannot say which is which: a classification whose recognisers are proved disjoint
// reads the same way, and `classifyCurvedTrim` is one. So this is a REGISTRY, like dispatchLadders:
// every chain that exists is listed with what makes it acceptable, a NEW one fails the test, and a
// chain that disappears must lose its entry. It is scoped to kernel/ops/tessellate — the package
// ADR-0061 stage 5 owns and where the disjointness proof lives; kernel-wide the shape has 16
// instances across packages other work owns, and registering those from here would make this guard
// a merge conflict rather than a ratchet (see ADR-0061).

// payloadGatedChains are the consecutive payload-gated recogniser chains in kernel/ops/tessellate,
// each with what makes it not a first-fit ladder. It may only shrink.
var payloadGatedChains = map[string]string{
	// PROVED DISJOINT. TestCurvedTrimKindsAreMutuallyExclusive evaluates every recogniser on every
	// curved face of the corpus and fails if two answer, and TestTheClassificationCorpusReachesEveryArm
	// keeps that proof from covering nothing. Reordering this function changes no answer.
	"kernel/ops/tessellate/curved_trim_classify.go:classifyCurvedTrim": "proved disjoint (#3409)",
	// DEBT. The B-spline face router: three shapes tried in order, each gated on a loop shape no other
	// claims, but nothing proves it. #3410 gives it a classification and a disjointness proof.
	"kernel/ops/tessellate/tessellate_trim.go:splineFaceMesh": "#3410",
	// DEBT. The seam-crossing router. Every face still reaching it has the chart mesher decline
	// (measured 182 of 182), so it is the chartless-face path; its rungs go when the chart mesher can
	// take them. #3411.
	"kernel/ops/tessellate/tessellate_trim.go:meshSeamCrossingFace": "#3411",
}

func TestNoUnprovenPayloadGatedChains(t *testing.T) {
	t.Parallel()
	found := scanPayloadGatedChains(t, filepath.Join("..", "kernel", "ops", "tessellate"))
	var added, gone []string
	for _, f := range found {
		if _, known := payloadGatedChains[f]; !known {
			added = append(added, f)
		}
	}
	seen := map[string]bool{}
	for _, f := range found {
		seen[f] = true
	}
	for f, why := range payloadGatedChains {
		if !seen[f] {
			gone = append(gone, f+" (was "+why+")")
		}
	}
	sort.Strings(added)
	sort.Strings(gone)
	if len(added) > 0 {
		t.Errorf("a payload-gated recogniser chain appeared — each call both recognises and produces "+
			"the result, so its ORDER decides the answer, exactly like the table ladder it looks "+
			"nothing like. Prove the recognisers disjoint and register it, or classify instead:\n  %s",
			strings.Join(added, "\n  "))
	}
	if len(gone) > 0 {
		t.Errorf("these chains are gone — good; DELETE their payloadGatedChains entries so the guard "+
			"holds the new floor:\n  %s", strings.Join(gone, "\n  "))
	}
}

// scanPayloadGatedChains returns "file:function" for every function in root holding a run of two or
// more consecutive `if x, ok := f(...); ok { … return }` statements over DISTINCT functions.
func scanPayloadGatedChains(t *testing.T, root string) []string {
	t.Helper()
	fset := token.NewFileSet()
	hits := map[string]bool{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		f, parseErr := parser.ParseFile(fset, p, nil, 0)
		if parseErr != nil {
			t.Fatalf("parsing %s: %v", p, parseErr)
		}
		collectChainedFuncs(f, filepath.ToSlash(strings.TrimPrefix(p, "../")), hits)
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	out := make([]string, 0, len(hits))
	for h := range hits {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

// collectChainedFuncs records every function of f whose body holds such a run.
func collectChainedFuncs(f *ast.File, path string, hits map[string]bool) {
	ast.Inspect(f, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			return true
		}
		if hasDistinctGatedRun(fd.Body.List) {
			hits[path+":"+fd.Name.Name] = true
		}
		return true
	})
}

// hasDistinctGatedRun reports whether the statements hold a maximal run of >= 2 consecutive gated
// recogniser calls naming DIFFERENT functions. Two gated calls to the SAME function in a row are a
// retry, not a ladder, and are not this guard's business.
func hasDistinctGatedRun(stmts []ast.Stmt) bool {
	var run []string
	for _, st := range append(append([]ast.Stmt{}, stmts...), nil) {
		name, isGated := "", false
		if st != nil {
			name, isGated = payloadGatedCallName(st)
		}
		if isGated {
			run = append(run, name)
			continue
		}
		if len(run) >= 2 && allDistinct(run) {
			return true
		}
		run = nil
	}
	return false
}

// allDistinct reports whether every name in the run is different.
func allDistinct(run []string) bool {
	seen := map[string]bool{}
	for _, n := range run {
		if seen[n] {
			return false
		}
		seen[n] = true
	}
	return true
}

// payloadGatedCallName returns the callee of `if <payload…>, ok := call(…); ok { … return }`.
func payloadGatedCallName(st ast.Stmt) (string, bool) {
	ifs, ok := st.(*ast.IfStmt)
	if !ok || ifs.Init == nil || !endsInReturn(ifs.Body) {
		return "", false
	}
	as, isAssign := ifs.Init.(*ast.AssignStmt)
	if !isAssign || len(as.Rhs) != 1 || len(as.Lhs) < 2 {
		return "", false
	}
	call, isCall := as.Rhs[0].(*ast.CallExpr)
	last, isIdent := as.Lhs[len(as.Lhs)-1].(*ast.Ident)
	cond, isCond := ifs.Cond.(*ast.Ident)
	if !isCall || !isIdent || !isCond || cond.Name != last.Name {
		return "", false
	}
	return calleeName(call.Fun)
}

// endsInReturn reports whether the block's last statement is a return — what makes the gate an EXIT
// rather than an ordinary early-out inside one algorithm.
func endsInReturn(b *ast.BlockStmt) bool {
	if b == nil || len(b.List) == 0 {
		return false
	}
	_, isRet := b.List[len(b.List)-1].(*ast.ReturnStmt)
	return isRet
}

// calleeName is the plain name of a called function or method.
func calleeName(fun ast.Expr) (string, bool) {
	switch fn := fun.(type) {
	case *ast.Ident:
		return fn.Name, true
	case *ast.SelectorExpr:
		return fn.Sel.Name, true
	}
	return "", false
}
