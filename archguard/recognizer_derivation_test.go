// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Deriving the recognizer count from the classification's own source (final fix wave, finding 6).
//
// curvedTrimRecognizers was a hand-written registry checked only against the switch's case names and
// "each name is still declared": a fourth sphere rim form, a second cone topology inside
// coneApexTrimOf, or a new recogniser gate inside an existing arm moved the number by zero — the exact
// events a "recognizers" ratchet exists to catch. The count is now DERIVED from the source and the
// registry must equal the derivation, name for name.
//
// A recogniser is a package function the classification READS for a shape verdict. Reads are found by
// their syntactic shape, starting at classifyCurvedTrim and following each read into its callee:
//
//	R1  if v, ok := f(…); ok { … }          a payload gate
//	R2  if f(…) { … }                        a boolean gate
//	R3  v, ok := f(…)  (ok never `!ok`-guarded)   an inventory read — every form evaluated, none gating
//	R4  return … a(…) || (!g(…) && b(…))     positive operands of a returned boolean; a negated call is a guard
//	R5  case …: return f(…)                  a form dispatched by name
//
// Only a VERDICT function is a read: one declared to return `bool`, or `(<payload>, bool)` where the
// payload is one of the classification's own `…Trim` types (coneApexTrim, sphereCapTrim, curvedTrim…).
// That is the contract every arm already keeps, and it is what stops the walk at a geometric helper —
// capAxis returns a vector, chooseSphereChart a chart, splitWrappingHoles two slices — instead of
// following it into the mesher.
//
// A function that reads nothing is a leaf and counts once. A function that reads others counts itself
// too when it also OWNS a verdict — a returned bool built from a call that is no read (coneApexTrimOf's
// `len(rim) != len(outer3D)` beside its faceIsConeApexCap gate) — and not when every verdict it returns
// is one of its reads' (classifySphereTrim, sphereCapTrimOf, sphereCapRimOfForm, ruledTwoRimBandHolds
// are pure dispatchers; a call under `!` is a guard there, not a verdict). The sphere family's rim FORMS
// are also asserted as an inventory: the sphereCapRimForm constants and sphereCapRimOfForm's cases must
// agree, so a form declared without a case, or a case without a form, fails on its own.

// recognizerRoot is the classification the derivation starts from.
const recognizerRoot = "classifyCurvedTrim"

// derivedRecognizers walks the classification and returns every recogniser it reads, sorted.
func derivedRecognizers(t *testing.T) []string {
	t.Helper()
	decls := tessellateFuncDecls(t)
	seen := map[string]bool{}
	var walk func(name string)
	walk = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		for _, r := range shapeReads(decls[name], decls) {
			walk(r)
		}
	}
	walk(recognizerRoot)
	var out []string
	for name := range seen {
		if name != recognizerRoot && countsAsRecognizer(decls[name], decls) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// countsAsRecognizer: a leaf always; a reader only when it owns a verdict of its own.
func countsAsRecognizer(fn *ast.FuncDecl, decls map[string]*ast.FuncDecl) bool {
	reads := shapeReads(fn, decls)
	return len(reads) == 0 || ownsVerdict(fn, reads)
}

// tessellateFuncDecls parses every non-test file of kernel/ops/tessellate into name → declaration.
func tessellateFuncDecls(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	dir := filepath.Join("..", "kernel", "ops", "tessellate")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, parseErr := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, e.Name()), nil, 0)
		if parseErr != nil {
			t.Fatalf("parsing %s: %v", e.Name(), parseErr)
		}
		for _, d := range f.Decls {
			if fd, isFunc := d.(*ast.FuncDecl); isFunc && fd.Recv == nil {
				decls[fd.Name.Name] = fd
			}
		}
	}
	return decls
}

// shapeReads is every package function fn reads for a verdict (R1–R5), in source order, once each.
func shapeReads(fn *ast.FuncDecl, decls map[string]*ast.FuncDecl) []string {
	if fn == nil || fn.Body == nil {
		return nil
	}
	guarded := negatedGuards(fn.Body)
	var out []string
	add := func(name string) {
		if d, declared := decls[name]; declared && isVerdictFunc(d) && !contains(out, name) {
			out = append(out, name)
		}
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		for _, name := range readsOfStatement(n, guarded) {
			add(name)
		}
		return true
	})
	return out
}

// readsOfStatement returns the callee names one node reads for a verdict.
func readsOfStatement(n ast.Node, guarded map[string]bool) []string {
	switch st := n.(type) {
	case *ast.IfStmt:
		return ifReads(st)
	case *ast.AssignStmt:
		return inventoryRead(st, guarded)
	case *ast.ReturnStmt:
		return returnedVerdictReads(st)
	case *ast.CaseClause:
		return caseReturnRead(st)
	}
	return nil
}

// ifReads: R1 (payload gate) and R2 (boolean gate).
func ifReads(st *ast.IfStmt) []string {
	if name, ok := payloadGatedCallName(st); ok {
		return []string{name}
	}
	if st.Init == nil {
		return positiveCallOperands(st.Cond)
	}
	return nil
}

// inventoryRead: R3 — `v, ok := f(…)` whose ok is never a negated guard.
func inventoryRead(st *ast.AssignStmt, guarded map[string]bool) []string {
	if len(st.Lhs) != 2 || len(st.Rhs) != 1 {
		return nil
	}
	call, isCall := st.Rhs[0].(*ast.CallExpr)
	ok, isIdent := st.Lhs[1].(*ast.Ident)
	if !isCall || !isIdent || guarded[ok.Name] {
		return nil
	}
	if name, plain := plainCallee(call); plain {
		return []string{name}
	}
	return nil
}

// returnedVerdictReads: R4 — the positive call operands of a returned boolean expression.
func returnedVerdictReads(st *ast.ReturnStmt) []string {
	if len(st.Results) == 0 {
		return nil
	}
	return positiveCallOperands(st.Results[len(st.Results)-1])
}

// caseReturnRead: R5 — a case clause whose body is `return f(…)`.
func caseReturnRead(cc *ast.CaseClause) []string {
	if len(cc.List) == 0 || len(cc.Body) != 1 {
		return nil
	}
	ret, isRet := cc.Body[0].(*ast.ReturnStmt)
	if !isRet || len(ret.Results) != 1 {
		return nil
	}
	if call, isCall := ret.Results[0].(*ast.CallExpr); isCall {
		if name, plain := plainCallee(call); plain {
			return []string{name}
		}
	}
	return nil
}

// positiveCallOperands walks a boolean expression and returns the plain calls that are NOT under a
// negation — `a() || (!g() && b())` gives a and b; g is a guard.
func positiveCallOperands(e ast.Expr) []string {
	switch x := e.(type) {
	case *ast.CallExpr:
		if name, plain := plainCallee(x); plain {
			return []string{name}
		}
	case *ast.ParenExpr:
		return positiveCallOperands(x.X)
	case *ast.BinaryExpr:
		if x.Op == token.LOR || x.Op == token.LAND {
			return append(positiveCallOperands(x.X), positiveCallOperands(x.Y)...)
		}
	}
	return nil
}

// negatedGuards is every identifier the body tests as `if !ident` — a precondition, not a verdict.
func negatedGuards(body *ast.BlockStmt) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		ifs, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		if u, isUnary := ifs.Cond.(*ast.UnaryExpr); isUnary && u.Op == token.NOT {
			if id, isIdent := u.X.(*ast.Ident); isIdent {
				out[id.Name] = true
			}
		}
		return true
	})
	return out
}

// ownsVerdict reports whether a reading function also returns a verdict of its own: a returned bool
// that calls something which is no read of its — a measurement, not a delegation. A call under `!` is a
// guard and never a verdict.
func ownsVerdict(fn *ast.FuncDecl, reads []string) bool {
	owns := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if ret, isRet := n.(*ast.ReturnStmt); isRet && len(ret.Results) > 0 {
			owns = owns || callsOutsideReads(ret.Results[len(ret.Results)-1], reads)
		}
		return true
	})
	return owns
}

// callsOutsideReads walks a boolean expression, skipping anything negated, and reports a call whose
// callee is not one of the reads — a builtin, a method, or a package function that is no verdict.
func callsOutsideReads(e ast.Expr, reads []string) bool {
	switch x := e.(type) {
	case *ast.UnaryExpr:
		return x.Op != token.NOT && callsOutsideReads(x.X, reads)
	case *ast.ParenExpr:
		return callsOutsideReads(x.X, reads)
	case *ast.BinaryExpr:
		return callsOutsideReads(x.X, reads) || callsOutsideReads(x.Y, reads)
	case *ast.CallExpr:
		name, plain := plainCallee(x)
		return !plain || !contains(reads, name)
	}
	return false
}

// isVerdictFunc reports whether a declaration returns a classification verdict: `bool` alone, or a
// `…Trim` payload with a `bool`.
func isVerdictFunc(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}
	var types []string
	for _, f := range fn.Type.Results.List {
		id, isIdent := f.Type.(*ast.Ident)
		if !isIdent {
			return false
		}
		types = append(types, id.Name)
	}
	if len(types) == 1 {
		return types[0] == "bool"
	}
	return len(types) == 2 && strings.HasSuffix(types[0], "Trim") && types[1] == "bool"
}

// plainCallee is the bare identifier a call invokes; a method or a package-qualified call is not one.
func plainCallee(call *ast.CallExpr) (string, bool) {
	id, isIdent := call.Fun.(*ast.Ident)
	if !isIdent {
		return "", false
	}
	return id.Name, true
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// TestTheRecognizerRegistryEqualsItsDerivation is the assertion itself: the hand-readable registry
// (curvedTrimRecognizers, which says WHICH arm each recognizer stands behind) must name exactly the
// recognizers the classification's source reads. A recogniser that appears in the source and not in the
// registry fails, as does one the registry keeps after the source dropped it.
func TestTheRecognizerRegistryEqualsItsDerivation(t *testing.T) {
	t.Parallel()
	derived := derivedRecognizers(t)
	var registered []string
	for _, names := range curvedTrimRecognizers {
		registered = append(registered, names...)
	}
	sort.Strings(registered)
	if strings.Join(derived, ",") != strings.Join(registered, ",") {
		t.Errorf("the recognizer registry does not match what classifyCurvedTrim reads:\n  derived:    %v\n"+
			"  registered: %v\nupdate curvedTrimRecognizers and kernelNetDeltaPin[\"recognizers\"] in this "+
			"commit, and say why in the ADR if the number ROSE", derived, registered)
	}
}

// TestTheSphereRimFormsAreOneInventory: the rim-form constants, the cases sphereCapRimOfForm dispatches
// on, and the registry's kindSphereCapFan entries are three readings of ONE inventory and must agree.
func TestTheSphereRimFormsAreOneInventory(t *testing.T) {
	t.Parallel()
	decls := tessellateFuncDecls(t)
	cases := 0
	ast.Inspect(decls["sphereCapRimOfForm"].Body, func(n ast.Node) bool {
		if cc, isCase := n.(*ast.CaseClause); isCase && len(caseReturnRead(cc)) == 1 {
			cases++
		}
		return true
	})
	forms := sphereCapRimFormConstants(t)
	registered := len(curvedTrimRecognizers["kindSphereCapFan"])
	if cases != forms || forms != registered {
		t.Errorf("sphere cap rim forms disagree: %d constants, %d dispatched cases, %d registered — a form "+
			"declared without a case (or a case without a form) is a shape the inventory cannot read",
			forms, cases, registered)
	}
}

// sphereCapRimFormConstants counts the constants of the sphereCapRimForm iota block, less its range
// sentinel (the block's last entry).
func sphereCapRimFormConstants(t *testing.T) int {
	t.Helper()
	f := parseKernelFile(t, "kernel/ops/tessellate/sphere_trim_form.go")
	for _, d := range f.Decls {
		gd, isGen := d.(*ast.GenDecl)
		if !isGen || gd.Tok != token.CONST || len(gd.Specs) == 0 {
			continue
		}
		if vs, isValue := gd.Specs[0].(*ast.ValueSpec); isValue && vs.Type != nil {
			if id, isIdent := vs.Type.(*ast.Ident); isIdent && id.Name == "sphereCapRimForm" {
				return len(gd.Specs) - 1
			}
		}
	}
	t.Fatal("no sphereCapRimForm const block found in sphere_trim_form.go")
	return 0
}
