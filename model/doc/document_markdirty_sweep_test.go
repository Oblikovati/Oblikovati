// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"testing"
)

// M40 S6 (#1641) completeness guard: every *Document method that marks the document dirty on a
// metadata mutation must be undoable — it has to ride the app-layer metadata snapshot (documentMetaStore
// folds body names, color styles and sketch/display settings into the recipe snapshot). A new dirtying
// metadata setter that is NOT classified here fails this test, forcing a visible decision — the same
// anti-drift discipline the feature-editor and serialization registries use.
//
// The value documents WHY the setter is covered (or, for a future exemption, why it is not undoable).
var documentDirtyMetadataSetters = map[string]string{
	"SetDisplaySettings": "rides the app metadata undo snapshot (documentMetaStore, S6 #1641)",
	"SetSketchSettings":  "rides the app metadata undo snapshot (documentMetaStore, S6 #1641)",
	"SetBodyName":        "rides the app metadata undo snapshot (documentMetaStore, S6 #1641)",
	"SetBodyColorStyle":  "rides the app metadata undo snapshot (documentMetaStore, S5 #1640 / S6 #1641)",
}

// TestDocumentMarkDirtySettersAreClassified fails when a *Document method calls MarkDirty without an
// entry in documentDirtyMetadataSetters — every persisted-metadata mutation must be a deliberate,
// undoable (or explicitly justified) part of the undo model.
func TestDocumentMarkDirtySettersAreClassified(t *testing.T) {
	fset := token.NewFileSet()
	pkg, err := parser.ParseDir(fset, ".", noTestFiles, 0)
	if err != nil {
		t.Fatalf("parse model/doc: %v", err)
	}
	for _, f := range pkg["doc"].Files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !isDocumentMethod(fn) || fn.Name.Name == "MarkDirty" {
				continue
			}
			if !bodyCallsMarkDirty(fn) {
				continue
			}
			if _, classified := documentDirtyMetadataSetters[fn.Name.Name]; !classified {
				t.Errorf("*Document.%s calls MarkDirty but is not classified in documentDirtyMetadataSetters: "+
					"a new persisted-metadata setter must ride the undo snapshot or be justified (#1641)", fn.Name.Name)
			}
		}
	}
}

// noTestFiles excludes _test.go from the parse so the sweep scans only production code.
func noTestFiles(info fs.FileInfo) bool {
	return !strings.HasSuffix(info.Name(), "_test.go")
}

// isDocumentMethod reports whether fn is a method with a *Document receiver.
func isDocumentMethod(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return false
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	id, ok := star.X.(*ast.Ident)
	return ok && id.Name == "Document"
}

// bodyCallsMarkDirty reports whether fn's body contains a MarkDirty call.
func bodyCallsMarkDirty(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "MarkDirty" {
			found = true
		}
		return !found
	})
	return found
}
