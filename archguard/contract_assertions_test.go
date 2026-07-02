// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-0018 step 2 mandates a compile-time assertion (var _ contract.X = ...) for every
// api/contract interface the host implements. Implicit satisfaction at a usage site is
// fragile — a refactor that removes the one conversion also removes the compile check,
// and the next contract drift breaks add-ins at runtime instead of this build (#1619).
// This guard diffs the declared contract interfaces against the assertion sites found
// anywhere in this repo (all modules, head included) and fails on any interface that is
// neither asserted nor justified in pendingContractAssertions.

// apiContractDir is the sibling Apache-2.0 module's contract package.
const apiContractDir = "../../Oblikovati.API/contract"

// pendingContractAssertions lists contract interfaces knowingly without a host-side
// assertion, each with the issue that owns closing the gap. Shrink-only: entries that
// gain an assertion (or leave the contract) fail the guard until removed here.
var pendingContractAssertions = map[string]string{
	// The client-object graphics model is declared but has no host implementation yet —
	// its "compile-time asserted" doc claim is the subject of #1613 (audit B2). Implement
	// or retire there; every entry below must leave this list as #1613 lands.
	"GraphicsCoordinateSet":        "#1613 (B2): graphics object model unimplemented",
	"GraphicsColorSet":             "#1613 (B2): graphics object model unimplemented",
	"GraphicsIndexSet":             "#1613 (B2): graphics object model unimplemented",
	"GraphicsNormalSet":            "#1613 (B2): graphics object model unimplemented",
	"GraphicsTextureCoordinateSet": "#1613 (B2): graphics object model unimplemented",
	"GraphicsImageSet":             "#1613 (B2): graphics object model unimplemented",
	"GraphicsColorMapper":          "#1613 (B2): graphics object model unimplemented",
	"GraphicsDataSets":             "#1613 (B2): graphics object model unimplemented",
	"GraphicsPrimitive":            "#1613 (B2): graphics object model unimplemented",
	"LineGraphics":                 "#1613 (B2): graphics object model unimplemented",
	"LineStripGraphics":            "#1613 (B2): graphics object model unimplemented",
	"PointGraphics":                "#1613 (B2): graphics object model unimplemented",
	"TextGraphics":                 "#1613 (B2): graphics object model unimplemented",
	"TriangleGraphics":             "#1613 (B2): graphics object model unimplemented",
	"TriangleStripGraphics":        "#1613 (B2): graphics object model unimplemented",
	"TriangleFanGraphics":          "#1613 (B2): graphics object model unimplemented",
	"SurfaceGraphics":              "#1613 (B2): graphics object model unimplemented",
	"CurveGraphics":                "#1613 (B2): graphics object model unimplemented",
	"ImageGraphics":                "#1613 (B2): graphics object model unimplemented",
	"SweepGraphics":                "#1613 (B2): graphics object model unimplemented",
	"GraphicsObjectNode":           "#1613 (B2): graphics object model unimplemented",
	"ComponentGraphics":            "#1613 (B2): graphics object model unimplemented",

	// The utility-collection contracts are implemented and asserted CLIENT-side
	// (api/client/transient_objects.go) — the host deliberately has no variant bag
	// (typed structs replace the COM NameValueMap habit; see event/doc.go).
	"NameValueMap":               "client-implemented: api/client/transient_objects.go",
	"ObjectsEnumerator":          "client-implemented: api/client/transient_objects.go",
	"ObjectCollection":           "client-implemented: api/client/transient_objects.go",
	"ObjectsEnumeratorByVariant": "client-implemented: api/client/transient_objects.go",
	"ObjectCollectionByVariant":  "client-implemented: api/client/transient_objects.go",
	"TransientObjects":           "client-implemented: api/client/transient_objects.go",
}

func TestEveryContractInterfaceIsAsserted(t *testing.T) {
	if _, err := os.Stat(apiContractDir); err != nil {
		t.Skipf("api module not checked out at %s: %v", apiContractDir, err)
	}
	declared := contractInterfaceNames(t)
	asserted := assertedContractNames(t)
	for _, name := range declared {
		if _, ok := asserted[name]; !ok && pendingContractAssertions[name] == "" {
			t.Errorf("contract.%s has no compile-time assertion (var _ contract.%s = ...) in the "+
				"host — add one next to the implementing type, or justify it in "+
				"pendingContractAssertions (ADR-0018, #1619).", name, name)
		}
	}
	reportStaleAllowlist(t, declared, asserted)
}

// reportStaleAllowlist keeps pendingContractAssertions shrink-only: an entry that is now
// asserted, or that no longer names a declared interface, must be deleted from the list.
func reportStaleAllowlist(t *testing.T, declared []string, asserted map[string]bool) {
	names := map[string]bool{}
	for _, n := range declared {
		names[n] = true
	}
	for name, why := range pendingContractAssertions {
		if asserted[name] {
			t.Errorf("pendingContractAssertions[%q] (%s) is stale — the assertion exists now; delete the entry.", name, why)
		}
		if !names[name] {
			t.Errorf("pendingContractAssertions[%q] (%s) names no declared contract interface; delete the entry.", name, why)
		}
	}
}

// contractInterfaceNames parses the api/contract package and returns its exported
// interface names (AST-level, no type checking — the contract package is plain Go).
func contractInterfaceNames(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, apiContractDir, notATestFile, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", apiContractDir, err)
	}
	var names []string
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			names = append(names, exportedInterfaces(file)...)
		}
	}
	return names
}

func notATestFile(fi fs.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }

// exportedInterfaces returns the exported interface type names declared in file.
func exportedInterfaces(file *ast.File) []string {
	var names []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts := spec.(*ast.TypeSpec)
			if _, isIface := ts.Type.(*ast.InterfaceType); isIface && ts.Name.IsExported() {
				names = append(names, ts.Name.Name)
			}
		}
	}
	return names
}

// assertedContractNames walks the whole repo (every module, head included — the
// AddInAutomation assertion lives there) and collects the contract interface names
// bound by `var _ contract.X = ...` declarations in non-test files.
func assertedContractNames(t *testing.T) map[string]bool {
	t.Helper()
	asserted := map[string]bool{}
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return skipNonSourceDir(d.Name())
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			collectFileAssertions(t, path, asserted)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo for contract assertions: %v", err)
	}
	return asserted
}

// skipNonSourceDir prunes trees that never hold shipped host code.
func skipNonSourceDir(name string) error {
	if strings.HasPrefix(name, ".") && name != "." && name != ".." { // .git AND .claude agent worktrees
		return filepath.SkipDir
	}
	switch name {
	case "experiments", "test-utilities", "node_modules":
		return filepath.SkipDir
	}
	return nil
}

// collectFileAssertions parses one file (pre-filtered on a cheap byte scan) and records
// the contract names asserted in it.
func collectFileAssertions(t *testing.T, path string, asserted map[string]bool) {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !bytes.Contains(src, []byte("oblikovati.org/api/contract")) {
		return
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	for _, name := range assertionNames(file, contractImportName(file)) {
		asserted[name] = true
	}
}

// contractImportName resolves the local identifier the file uses for api/contract
// (usually "contract", but an aliased import must still count).
func contractImportName(file *ast.File) string {
	for _, imp := range file.Imports {
		if strings.Trim(imp.Path.Value, `"`) != "oblikovati.org/api/contract" {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "contract"
	}
	return ""
}

// assertionNames returns the X of every top-level `var _ pkg.X = ...` in file, where
// pkg is the local name of the api/contract import.
func assertionNames(file *ast.File, pkg string) []string {
	if pkg == "" {
		return nil
	}
	var names []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		names = append(names, blankVarContractTypes(gen, pkg)...)
	}
	return names
}

// blankVarContractTypes extracts the contract type names from one var declaration's
// blank-identifier specs (`_ contract.X = ...`), including grouped var blocks.
func blankVarContractTypes(gen *ast.GenDecl, pkg string) []string {
	var names []string
	for _, spec := range gen.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "_" {
			continue
		}
		if sel, ok := vs.Type.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == pkg {
				names = append(names, sel.Sel.Name)
			}
		}
	}
	return names
}
