// SPDX-License-Identifier: GPL-2.0-only

package testguard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// scanDir classifies one directory, or returns nil when it holds no _test.go files.
func scanDir(dir string) (*Package, error) {
	files, err := testFiles(dir)
	if err != nil || len(files) == 0 {
		return nil, err
	}
	bodies, err := parseFuncs(files)
	if err != nil {
		return nil, err
	}
	aware := shortAware(bodies)
	pkg := &Package{WholePackage: aware["TestMain"], guarded: map[string]bool{}}
	for name := range bodies {
		if strings.HasPrefix(name, "Test") && aware[name] {
			pkg.guarded[name] = true
		}
	}
	return pkg, nil
}

// testFiles lists the _test.go files directly inside dir.
func testFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "_test.go") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}

// funcBody is what the guard closure needs from one function declaration.
type funcBody struct {
	callsShort bool     // the body calls testing.Short()
	calls      []string // same-package functions it calls
}

// parseFuncs reads every top-level function in the files, keyed by name. Methods are
// ignored: a guard is written in a test or in a package-level helper it calls.
func parseFuncs(files []string) (map[string]funcBody, error) {
	out := map[string]funcBody{}
	fset := token.NewFileSet()
	for _, f := range files {
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", f, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Body != nil {
				out[fn.Name.Name] = readBody(fn.Body)
			}
		}
	}
	return out, nil
}

// readBody records whether the body calls testing.Short() and which same-package
// functions it calls.
func readBody(body *ast.BlockStmt) funcBody {
	var fb funcBody
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			if id, ok := fun.X.(*ast.Ident); ok && id.Name == "testing" && fun.Sel.Name == "Short" {
				fb.callsShort = true
			}
		case *ast.Ident:
			fb.calls = append(fb.calls, fun.Name)
		}
		return true
	})
	return fb
}

// shortAware returns the functions that reach testing.Short(), directly or through
// same-package helpers. A corpus harness such as model/feature's runCorpusGrids
// guards several tests from one place, so a direct-call check alone would report
// those tests as unguarded.
func shortAware(bodies map[string]funcBody) map[string]bool {
	aware := map[string]bool{}
	for name, fb := range bodies {
		if fb.callsShort {
			aware[name] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for name, fb := range bodies {
			if aware[name] {
				continue
			}
			for _, callee := range fb.calls {
				if aware[callee] {
					aware[name] = true
					changed = true
					break
				}
			}
		}
	}
	return aware
}
