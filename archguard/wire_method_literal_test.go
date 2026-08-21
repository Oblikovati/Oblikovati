// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The house rule "never re-declare a DTO or method-name string — import it from api/wire"
// (CLAUDE.md) protects diagnostics and dispatch: a wire method renamed in one place must
// not leave a stale bare-string copy behind (a router error context reporting the old
// name, an add-in call 404-ing at runtime) that still compiles green (#1618, audit B7).
// The 585=585=585 wire/router/client parity guard covers REGISTRATION sites keyed on
// wire.Method*; this guard extends that discipline to EVERY occurrence site by diffing the
// value set of the wire.Method* constants against bare string literals in the GPL module.
//
// It parses the constant VALUES from api/wire via go/ast (not a regex over "x.y" shapes,
// which both false-positives on unrelated dotted strings and false-negatives on gofmt
// quirks) and, for each, requires call sites to reference the constant, not its value.

// wireMethodLiteralAllowlist justifies any bare occurrence of a wire method value that is
// NOT a re-declaration (e.g. a doc example). Shrink-only, and empty by design: a real hit
// is a bug to fix (import the constant), not to allowlist. Keyed "file:value".
var wireMethodLiteralAllowlist = map[string]string{}

func TestNoReDeclaredWireMethodLiterals(t *testing.T) {
	wireDir := apiSubdir(t, "wire") // resolved via go list -m, so this runs in CI (_api) too
	values := wireMethodConstValues(t, wireDir)
	if len(values) == 0 {
		t.Fatalf("found no wire.Method* string constants in %s — the guard would be a no-op", wireDir)
	}
	walkGoSourceFiles(t, "..", func(path string, file *ast.File, fset *token.FileSet) {
		reportWireLiteralHits(t, path, file, fset, values)
	})
}

// wireMethodConstValues returns the string value -> constant-name map of every exported
// `Method*` const in the wire package.
func wireMethodConstValues(t *testing.T, wireDir string) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, wireDir, notATestFile, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", wireDir, err)
	}
	values := map[string]string{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			collectMethodConsts(file, values)
		}
	}
	return values
}

// collectMethodConsts records every `Method<Name> = "value"` const spec in file.
func collectMethodConsts(file *ast.File, values map[string]string) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			addMethodConst(spec, values)
		}
	}
}

// addMethodConst records one const spec when it binds a `Method*` name to a string literal.
func addMethodConst(spec ast.Spec, values map[string]string) {
	vs, ok := spec.(*ast.ValueSpec)
	if !ok || len(vs.Names) != len(vs.Values) {
		return
	}
	for i, name := range vs.Names {
		if !strings.HasPrefix(name.Name, "Method") {
			continue
		}
		if v, ok := stringLit(vs.Values[i]); ok {
			values[v] = name.Name
		}
	}
}

// reportWireLiteralHits fails on every string literal in file whose value equals a wire
// method value — the caller must reference wire.<Const> instead.
func reportWireLiteralHits(t *testing.T, path string, file *ast.File, fset *token.FileSet, values map[string]string) {
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		v, ok := unquote(lit.Value)
		if !ok {
			return true
		}
		if constName, isMethod := values[v]; isMethod {
			flagWireLiteral(t, path, fset.Position(lit.Pos()).Line, v, constName)
		}
		return true
	})
}

// flagWireLiteral reports one re-declared method literal unless it is justified.
func flagWireLiteral(t *testing.T, path string, line int, value, constName string) {
	if wireMethodLiteralAllowlist[path+":"+value] != "" {
		return
	}
	t.Errorf("%s:%d re-declares the wire method name %q as a bare string literal — reference "+
		"wire.%s instead (a rename must not leave a stale copy that compiles green). CLAUDE.md, #1618.",
		path, line, value, constName)
}

// stringLit returns the unquoted value of an expression that is a string literal.
func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	return unquote(lit.Value)
}

// unquote strips Go quoting from a string-literal token, reporting failure rather than panicking.
func unquote(tok string) (string, bool) {
	s, err := strconv.Unquote(tok)
	if err != nil {
		return "", false
	}
	return s, true
}

// walkGoSourceFiles parses every non-test .go file under root (skipping non-source trees)
// and invokes visit for each, sharing the FileSet so positions are reportable.
func walkGoSourceFiles(t *testing.T, root string, visit func(path string, file *ast.File, fset *token.FileSet)) {
	t.Helper()
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return skipNonSourceDir(d.Name())
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, perr := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if perr != nil {
			return nil // unparseable non-test file is another guard's problem, not this one's
		}
		if isGeneratedFile(file) {
			return nil // generated artifacts are derived from wire, not hand-maintained re-declarations
		}
		visit(path, file, fset)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s for wire literals: %v", root, err)
	}
}

// isGeneratedFile reports whether file carries the standard `// Code generated ... DO NOT
// EDIT.` marker (https://pkg.go.dev/cmd/go#hdr-Generate_Go_files). Such files materialize the
// wire catalog from its single source; the generator, not the artifact, owns the strings.
func isGeneratedFile(file *ast.File) bool {
	for _, group := range file.Comments {
		for _, c := range group.List {
			if strings.HasPrefix(c.Text, "// Code generated ") && strings.HasSuffix(c.Text, " DO NOT EDIT.") {
				return true
			}
		}
	}
	return false
}
