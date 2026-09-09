// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/token"
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
// A recogniser is a function the classification READS for a shape verdict. Reads are found by their
// syntactic shape, starting at classifyCurvedTrim and following each read into its callee:
//
//	R1  if v, ok := f(…); ok { … }          a payload gate
//	R2  if f(…) { … }                        a boolean gate
//	R3  v, ok := f(…)  (ok never `!ok`-guarded)   an inventory read — every form evaluated, none gating
//	R4  return … a(…) || (!g(…) && b(…))     positive operands of a returned boolean; a negated call is a guard
//	R5  case …: return f(…)                  a form dispatched by name
//
// WHICH declaration a read reaches, and WHETHER that declaration returns a verdict, are both resolved
// by recognizerIndex (recognizer_index_test.go). Neither answer is a spelling: a callee may be a bare
// call, a method, or a call into a package of the classification's own tree, and a verdict is any
// result set ending in a bool whose earlier results are the payloads the verdict struct carries, or
// bools (#3522). The index's own doc states what it still cannot see, and the probe plants it.
//
// R1–R5 are the read SHAPES, and #3522 did not widen them — it widened only WHICH declaration a read
// reaches and WHETHER that declaration is a verdict. Nesting is not a limit: a gate inside a `for` body
// is read, because shapeReads inspects the whole body. A SINGLE-LHS init gate is: `if ok := f(); ok { … }`
// is invisible, because payloadGatedCall requires at least two left-hand names and ifReads gives up on
// an `if` that carries an Init. A recognizer written that way moves the count by zero.
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
	return derivedRecognizersIn(t, tessellateIndex(t))
}

// derivedRecognizersIn is the walk itself, over any index — the kernel's, or the probe's planted one.
func derivedRecognizersIn(t *testing.T, idx *recognizerIndex) []string {
	t.Helper()
	assertUnambiguousVerdictNames(t, idx)
	reached := reachableFromClassification(t, idx)
	assertNoUnresolvableMethodReads(t, idx, reached)
	return recognizersAmong(reached, idx)
}

// reachableFromClassification is every declaration the walk reaches from the classification root.
func reachableFromClassification(t *testing.T, idx *recognizerIndex) map[string]bool {
	t.Helper()
	if idx.lookup(recognizerRoot) == nil {
		t.Fatalf("no %s is declared in the indexed tree; the derivation has nothing to walk", recognizerRoot)
	}
	seen := map[string]bool{}
	var walk func(name string)
	walk = func(name string) {
		d := idx.lookup(name)
		if seen[name] || d == nil {
			return
		}
		seen[name] = true
		for _, r := range (&reader{*d, idx}).shapeReads() {
			walk(r)
		}
	}
	walk(recognizerRoot)
	return seen
}

// recognizersAmong keeps, of everything the walk reached, the names that COUNT as recognizers.
func recognizersAmong(seen map[string]bool, idx *recognizerIndex) []string {
	var out []string
	for name := range seen {
		if name == recognizerRoot {
			continue
		}
		if (&reader{*idx.lookup(name), idx}).countsAsRecognizer() {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// assertUnambiguousVerdictNames refuses a tree in which one key names two declarations and either could
// be a verdict. "Ambiguous key resolution is an error; unguarded first-match lookups are forbidden."
func assertUnambiguousVerdictNames(t *testing.T, idx *recognizerIndex) {
	t.Helper()
	if names := idx.ambiguousVerdictNames(); len(names) > 0 {
		t.Errorf("these names reach more than one declaration and at least one of each pair is "+
			"verdict-shaped, so a read would resolve against whichever was indexed first — rename one "+
			"side:\n  %s", strings.Join(names, "\n  "))
	}
}

// assertNoUnresolvableMethodReads refuses a method call the index cannot attribute: the AST states no
// type for its receiver, and its bare name reaches a verdict declaration of the tree. Such a call is
// EITHER a recognizer read or a call on a foreign value that happens to share the name. Counting it
// INFLATES the pin and dropping it deflates it, and both corrupt a later fall — so neither is guessed.
func assertNoUnresolvableMethodReads(t *testing.T, idx *recognizerIndex, reached map[string]bool) {
	t.Helper()
	if sites := unresolvableReads(idx, reached); len(sites) > 0 {
		t.Errorf("these method calls reach a verdict-shaped name of the classification's tree, but the "+
			"AST states no type for their receiver, so the derivation cannot tell a recognizer read from "+
			"a call on a foreign value of the same method name — give the receiver a stated type (a "+
			"parameter, a `var`, a composite literal) or rename one side:\n  %s",
			strings.Join(sites, "\n  "))
	}
}

// unresolvableReads is every unattributable method call inside the declarations the walk reached,
// sorted, so the failure is byte-identical across runs.
func unresolvableReads(idx *recognizerIndex, reached map[string]bool) []string {
	var out []string
	for name := range reached {
		out = append(out, (&reader{*idx.lookup(name), idx}).unattributableMethodCalls()...)
	}
	sort.Strings(out)
	return out
}

// reader is one declaration being walked, with the index its names resolve against.
type reader struct {
	d   declaredHere
	idx *recognizerIndex
}

// countsAsRecognizer: a leaf always; a reader only when it owns a verdict of its own.
func (r *reader) countsAsRecognizer() bool {
	reads := r.shapeReads()
	return len(reads) == 0 || r.ownsVerdict(reads)
}

// shapeReads is every function r reads for a verdict (R1–R5), in source order, once each.
func (r *reader) shapeReads() []string {
	if r.d.fn == nil || r.d.fn.Body == nil {
		return nil
	}
	guarded := negatedGuards(r.d.fn.Body)
	var out []string
	ast.Inspect(r.d.fn.Body, func(n ast.Node) bool {
		for _, name := range r.readsOfStatement(n, guarded) {
			if d := r.idx.lookup(name); d != nil && r.idx.isVerdict(*d) && !contains(out, name) {
				out = append(out, name)
			}
		}
		return true
	})
	return out
}

// readsOfStatement returns the callee names one node reads for a verdict.
func (r *reader) readsOfStatement(n ast.Node, guarded map[string]bool) []string {
	switch st := n.(type) {
	case *ast.IfStmt:
		return r.ifReads(st)
	case *ast.AssignStmt:
		return r.inventoryRead(st, guarded)
	case *ast.ReturnStmt:
		return r.returnedVerdictReads(st)
	case *ast.CaseClause:
		return r.caseReturnRead(st)
	}
	return nil
}

// ifReads: R1 (payload gate) and R2 (boolean gate).
func (r *reader) ifReads(st *ast.IfStmt) []string {
	if name, ok := r.payloadGateName(st); ok {
		return []string{name}
	}
	if st.Init == nil {
		return r.positiveCallOperands(st.Cond)
	}
	return nil
}

// payloadGateName is the callee of `if v, ok := f(…); ok { … }`, resolved through the index so a
// method or a tree-package call is as visible as a bare one.
func (r *reader) payloadGateName(st *ast.IfStmt) (string, bool) {
	call, ok := payloadGatedCall(st)
	if !ok {
		return "", false
	}
	return r.idx.calleeName(call, r.d)
}

// inventoryRead: R3 — `v, ok := f(…)` whose ok is never a negated guard.
func (r *reader) inventoryRead(st *ast.AssignStmt, guarded map[string]bool) []string {
	if len(st.Lhs) != 2 || len(st.Rhs) != 1 {
		return nil
	}
	call, isCall := st.Rhs[0].(*ast.CallExpr)
	ok, isIdent := st.Lhs[1].(*ast.Ident)
	if !isCall || !isIdent || guarded[ok.Name] {
		return nil
	}
	if name, resolved := r.idx.calleeName(call, r.d); resolved {
		return []string{name}
	}
	return nil
}

// returnedVerdictReads: R4 — the positive call operands of a returned boolean expression.
func (r *reader) returnedVerdictReads(st *ast.ReturnStmt) []string {
	if len(st.Results) == 0 {
		return nil
	}
	return r.positiveCallOperands(st.Results[len(st.Results)-1])
}

// caseReturnRead: R5 — a case clause whose body is `return f(…)`.
func (r *reader) caseReturnRead(cc *ast.CaseClause) []string {
	if len(cc.List) == 0 || len(cc.Body) != 1 {
		return nil
	}
	ret, isRet := cc.Body[0].(*ast.ReturnStmt)
	if !isRet || len(ret.Results) != 1 {
		return nil
	}
	if call, isCall := ret.Results[0].(*ast.CallExpr); isCall {
		if name, resolved := r.idx.calleeName(call, r.d); resolved {
			return []string{name}
		}
	}
	return nil
}

// positiveCallOperands walks a boolean expression and returns the calls that are NOT under a
// negation — `a() || (!g() && b())` gives a and b; g is a guard.
func (r *reader) positiveCallOperands(e ast.Expr) []string {
	switch x := e.(type) {
	case *ast.CallExpr:
		if name, resolved := r.idx.calleeName(x, r.d); resolved {
			return []string{name}
		}
	case *ast.ParenExpr:
		return r.positiveCallOperands(x.X)
	case *ast.BinaryExpr:
		if x.Op == token.LOR || x.Op == token.LAND {
			return append(r.positiveCallOperands(x.X), r.positiveCallOperands(x.Y)...)
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
func (r *reader) ownsVerdict(reads []string) bool {
	owns := false
	ast.Inspect(r.d.fn.Body, func(n ast.Node) bool {
		if ret, isRet := n.(*ast.ReturnStmt); isRet && len(ret.Results) > 0 {
			owns = owns || r.callsOutsideReads(ret.Results[len(ret.Results)-1], reads)
		}
		return true
	})
	return owns
}

// callsOutsideReads walks a boolean expression, skipping anything negated, and reports a call whose
// callee is not one of the reads — a builtin, a call the index cannot resolve, or a function that is
// no verdict.
func (r *reader) callsOutsideReads(e ast.Expr, reads []string) bool {
	switch x := e.(type) {
	case *ast.UnaryExpr:
		return x.Op != token.NOT && r.callsOutsideReads(x.X, reads)
	case *ast.ParenExpr:
		return r.callsOutsideReads(x.X, reads)
	case *ast.BinaryExpr:
		return r.callsOutsideReads(x.X, reads) || r.callsOutsideReads(x.Y, reads)
	case *ast.CallExpr:
		name, resolved := r.idx.calleeName(x, r.d)
		return !resolved || !contains(reads, name)
	}
	return false
}

// unattributableMethodCalls are the method calls in this declaration whose receiver the AST does not
// state and whose name reaches a verdict of the tree.
func (r *reader) unattributableMethodCalls() []string {
	if r.d.fn == nil || r.d.fn.Body == nil {
		return nil
	}
	var out []string
	ast.Inspect(r.d.fn.Body, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall {
			return true
		}
		if name, unattributable := r.idx.unattributableMethod(call, r.d); unattributable {
			out = append(out, name+" at "+r.idx.fset.Position(call.Pos()).String())
		}
		return true
	})
	return out
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
	idx := tessellateIndex(t)
	form := idx.lookup("sphereCapRimOfForm")
	if form == nil {
		t.Fatal("sphereCapRimOfForm is not declared in the classification's tree")
	}
	cases := 0
	r := &reader{*form, idx}
	ast.Inspect(form.fn.Body, func(n ast.Node) bool {
		if cc, isCase := n.(*ast.CaseClause); isCase && len(r.caseReturnRead(cc)) == 1 {
			cases++
		}
		return true
	})
	assertRimFormInventoriesAgree(t, cases)
}

// assertRimFormInventoriesAgree compares the dispatched cases with the constants and the registry.
func assertRimFormInventoriesAgree(t *testing.T, cases int) {
	t.Helper()
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
