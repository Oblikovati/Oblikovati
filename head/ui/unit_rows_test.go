// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hardcodedUnitLiterals are unit labels that must NOT be passed as a literal to
// propertyFloatRow — a length/angle field's unit has to come from the document
// (s.LengthUnitName()/s.AngleUnitName(), via lengthCmRow/angleDegRow) so it
// follows the document's units (Oblikovati/Oblikovati#146).
var hardcodedUnitLiterals = map[string]bool{
	"mm": true, "cm": true, "m": true, "km": true,
	"in": true, "ft": true, "deg": true, "rad": true, "°": true,
}

// TestNoHardcodedUnitLabelsInRows guards every dialog: a propertyFloatRow whose
// suffix argument is a hardcoded unit string is a units bug waiting to happen.
// Non-unit suffixes ("× (1 = off)", "") and dynamic s.*UnitName() calls are fine.
func TestNoHardcodedUnitLabelsInRows(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	fset := token.NewFileSet()
	var offences []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		tree, err := parser.ParseFile(fset, f, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		ast.Inspect(tree, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isIdentCall(call, "propertyFloatRow") || len(call.Args) < 3 {
				return true
			}
			if lit, ok := call.Args[2].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				// Flag a bare unit ("deg") or a composite that LEADS with one
				// ("deg (+out / −in)"); a non-unit hint ("× (1 = off)", "") is fine.
				val := strings.Trim(lit.Value, "`\"")
				if first, _, _ := strings.Cut(val, " "); hardcodedUnitLiterals[first] {
					offences = append(offences, f+": propertyFloatRow suffix "+lit.Value)
				}
			}
			return true
		})
	}
	if len(offences) > 0 {
		t.Errorf("hardcoded unit labels (use lengthCmRow/angleDegRow instead):\n  %s",
			strings.Join(offences, "\n  "))
	}
}

// isIdentCall reports whether call is a call to the named bare function.
func isIdentCall(call *ast.CallExpr, name string) bool {
	id, ok := call.Fun.(*ast.Ident)
	return ok && id.Name == name
}
