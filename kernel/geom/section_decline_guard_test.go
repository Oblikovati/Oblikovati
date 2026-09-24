// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// The guard over the [SectionDecline] vocabulary (Oblikovati/Oblikovati#3525).
//
// Everything it checks is DERIVED from the source: the constant list from the const block's own AST,
// the return sites by walking every kernel package that mentions the type. Nothing here is a
// hand-written list, so a decline added tomorrow is covered the day it lands and a decline DELETED
// tomorrow (the torus × torus section, #3514) leaves the guard green without an edit. The test it
// replaces bounded its loop by the last constant as written that day, which made the next constant —
// the one most likely to arrive without its name — invisible to it.

// declineTypeNames are the types a refusal travels in: geom's own enum, and the brep struct that
// carries one plus the value the gate measured.
var declineTypeNames = map[string]bool{"SectionDecline": true, "sectionRefusal": true}

// TestEverySectionDeclineIsNamed: a reason with no name reaches a user as "SectionDecline(?)", which is
// worse than no diagnostic. The enumeration is the const block itself, so this fails when the next
// reason lands without its string — however many land after it.
func TestEverySectionDeclineIsNamed(t *testing.T) {
	t.Parallel()
	declared := declaredSectionDeclines(t)
	if len(declared) != len(sectionDeclineNames) {
		t.Fatalf("%d SectionDecline constants are declared but sectionDeclineNames holds %d entries: %v",
			len(declared), len(sectionDeclineNames), declared)
	}
	seen := map[string]string{}
	for i, constName := range declared {
		name := SectionDecline(i).String()
		if name == unnamedSectionDecline || name == "" {
			t.Errorf("SectionDecline %s (=%d) has no name in sectionDeclineNames (got %q)", constName, i, name)
		}
		if prev, dup := seen[name]; dup {
			t.Errorf("SectionDecline %s reuses %s's name %q", constName, prev, name)
		}
		seen[name] = constName
	}
	if got := SectionDecline(len(declared) + 100).String(); got != unnamedSectionDecline {
		t.Errorf("an out-of-range decline names itself %q, want %q", got, unnamedSectionDecline)
	}
}

// unnamedSectionDecline is what [SectionDecline.String] answers for a value with no entry.
const unnamedSectionDecline = "SectionDecline(?)"

// TestNoDeclineReturnSiteIsAnonymous: a refusal that returns DeclineNone tells the user only that
// something refused, which is the silence #3525 names — a torus pair, an ill-conditioned lane and a
// section that does not close all read the same. Every function that returns a decline is walked, and
// each of its refusal returns must name a reason.
func TestNoDeclineReturnSiteIsAnonymous(t *testing.T) {
	t.Parallel()
	sites := 0
	forEachDecliningFunc(t, func(file string, fn *ast.FuncDecl, why, ok int) {
		for _, ret := range returnStmtsOf(fn) {
			if len(ret.Results) <= max(why, ok) {
				continue // a naked return, or a forwarded call: nothing to read here
			}
			sites++
			if isFalseLiteral(ret.Results[ok]) && namesNoDecline(ret.Results[why]) {
				t.Errorf("%s: %s returns a refusal with no reason (DeclineNone with ok=false)", file, fn.Name.Name)
			}
		}
	})
	if sites == 0 {
		t.Fatal("the guard found no decline return site at all — it is passing vacuously")
	}
	t.Logf("walked %d decline return sites", sites)
}

// TestEveryDeclineNameIsReachable: a name nothing returns is a name the user can never be shown, and
// the delete-first rule says the constant goes with the site that raised it. Derived both ways, so
// removing a gate AND its reason keeps this green.
//
// It proves a SYNTACTIC mention, not a live producer: a constant returned only from a branch no input
// can reach passes here identically. The runtime half is brep's TestEveryImprintRefusalIsNamed, which
// logs the names its corpus actually raised — the gap between the two lists is the honest measure of
// which reasons a user can meet today (Oblikovati/Oblikovati#3525, review round 1, finding 11).
func TestEveryDeclineNameIsReachable(t *testing.T) {
	t.Parallel()
	raised := declineIdentsRaisedByAGate(t)
	for _, constName := range declaredSectionDeclines(t) {
		if constName == "DeclineNone" {
			continue // the "no refusal" member: read by comparison, never raised by a gate
		}
		if !raised[constName] {
			t.Errorf("%s is declared and named but no gate ever returns it: delete it with the gate that did",
				constName)
		}
	}
}

// declaredSectionDeclines returns the const block's member names in declaration (iota) order.
func declaredSectionDeclines(t *testing.T) []string {
	t.Helper()
	var out []string
	forEachKernelFile(t, func(_ string, file *ast.File) {
		for _, decl := range file.Decls {
			gen, isConst := decl.(*ast.GenDecl)
			if !isConst || gen.Tok != token.CONST || !constBlockIsSectionDecline(gen) {
				continue
			}
			for _, spec := range gen.Specs {
				out = append(out, spec.(*ast.ValueSpec).Names[0].Name)
			}
		}
	})
	if len(out) == 0 {
		t.Fatal("no SectionDecline const block found: the guard cannot enumerate what it does not see")
	}
	return out
}

// constBlockIsSectionDecline reports whether an iota block declares SectionDecline members — the
// first spec carries the type, the rest inherit it.
func constBlockIsSectionDecline(gen *ast.GenDecl) bool {
	for _, spec := range gen.Specs {
		vs, isValue := spec.(*ast.ValueSpec)
		if !isValue {
			return false
		}
		if id, isIdent := vs.Type.(*ast.Ident); isIdent && id.Name == "SectionDecline" {
			return true
		}
	}
	return false
}

// forEachDecliningFunc calls fn for every function in the kernel tree whose results carry a refusal,
// with the result indices of the refusal and of the bool that says whether the pair was handled.
func forEachDecliningFunc(t *testing.T, visit func(file string, fn *ast.FuncDecl, why, ok int)) {
	t.Helper()
	forEachKernelFile(t, func(path string, file *ast.File) {
		for _, decl := range file.Decls {
			fn, isFunc := decl.(*ast.FuncDecl)
			if !isFunc || fn.Body == nil || fn.Type.Results == nil {
				continue
			}
			if why, ok, found := refusalResultIndices(fn.Type.Results); found {
				visit(path, fn, why, ok)
			}
		}
	})
}

// refusalResultIndices locates the refusal and the handled-bool in a result list, or found=false when
// the signature carries no refusal (or carries one without a bool beside it, which is a total answer).
func refusalResultIndices(results *ast.FieldList) (why, ok int, found bool) {
	why, ok = -1, -1
	for i, name := range flatResultTypeNames(results) {
		switch {
		case declineTypeNames[name] && why < 0:
			why = i
		case name == "bool" && ok < 0:
			ok = i
		}
	}
	return why, ok, why >= 0 && ok >= 0
}

// flatResultTypeNames names each result position's type, expanding a grouped field ("a, b bool") and
// dropping a package qualifier so geom.SectionDecline and SectionDecline read alike.
func flatResultTypeNames(results *ast.FieldList) []string {
	var out []string
	for _, field := range results.List {
		name := typeLeafName(field.Type)
		for range max(len(field.Names), 1) {
			out = append(out, name)
		}
	}
	return out
}

// typeLeafName is the bare type name of an expression: "SectionDecline" for both SectionDecline and
// geom.SectionDecline, and "" for anything with no single name (a slice, a func type).
func typeLeafName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return ""
	}
}

// returnStmtsOf collects the function's return statements, nested ones included.
func returnStmtsOf(fn *ast.FuncDecl) []*ast.ReturnStmt {
	var out []*ast.ReturnStmt
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if ret, isReturn := n.(*ast.ReturnStmt); isReturn {
			out = append(out, ret)
		}
		return true
	})
	return out
}

// isFalseLiteral reports the untyped `false` written straight into a return.
func isFalseLiteral(expr ast.Expr) bool {
	id, isIdent := expr.(*ast.Ident)
	return isIdent && id.Name == "false"
}

// namesNoDecline reports a refusal expression that carries no reason: the DeclineNone constant, or the
// constructor that stands for it. A variable or any other call is a FORWARD — the reason was named
// further down and travels through — so it passes.
func namesNoDecline(expr ast.Expr) bool {
	if call, isCall := expr.(*ast.CallExpr); isCall {
		return typeLeafName(call.Fun) == "solved"
	}
	return typeLeafName(expr) == "DeclineNone"
}

// declineIdentsRaisedByAGate collects every Decline* identifier the kernel tree mentions inside a
// FUNCTION BODY. Declarations are deliberately out of scope: the const block names the member and the
// names array gives it its string, so counting either would let a reason nothing raises look reachable
// — which is the orphan this checks for.
func declineIdentsRaisedByAGate(t *testing.T) map[string]bool {
	t.Helper()
	used := map[string]bool{}
	forEachKernelFile(t, func(_ string, file *ast.File) {
		for _, decl := range file.Decls {
			fn, isFunc := decl.(*ast.FuncDecl)
			if !isFunc || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if id, isIdent := n.(*ast.Ident); isIdent && strings.HasPrefix(id.Name, "Decline") {
					used[id.Name] = true
				}
				return true
			})
		}
	})
	return used
}

// forEachKernelFile parses every non-test .go file under kernel/ and hands it to visit. The whole
// kernel is walked rather than a named package list, so a refusal minted in a package that does not
// exist yet is covered the day it lands.
func forEachKernelFile(t *testing.T, visit func(path string, file *ast.File)) {
	t.Helper()
	fset := token.NewFileSet()
	const root = ".." // the test runs in kernel/geom, so ".." is the kernel tree
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		parsed, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			return perr
		}
		visit(path, parsed)
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s for decline sites: %v", root, err)
	}
}
