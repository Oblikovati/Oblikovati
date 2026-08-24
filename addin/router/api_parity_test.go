// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"oblikovati.org/addin/opregistry"
)

// TestEveryWireMethodHasAHandler is the API↔router parity guard: every wire.Method*
// constant declared in the public contract MUST be registered in the router, or this
// fails. It is what stops a new wire method from shipping in the API while the router
// silently never handles it.
func TestEveryWireMethodHasAHandler(t *testing.T) {
	methods := wireConstants(t, "Method")
	if len(methods) == 0 {
		t.Fatal("parsed zero wire Method constants — the wire source lookup is broken")
	}
	r := New(opregistry.Default())
	var missing []string
	for name, value := range methods {
		if notYetHandled[name] {
			continue
		}
		if _, ok := r.handlers[value]; !ok {
			missing = append(missing, name+" = "+strconv.Quote(value))
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("%d wire method(s) declared in the API have NO router handler:\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
	// The allowlist may only shrink: a now-handled (or no-longer-declared) method must be removed.
	for name := range notYetHandled {
		value, declared := methods[name]
		if !declared {
			t.Errorf("notYetHandled lists %q, which is not a wire Method constant", name)
			continue
		}
		if _, ok := r.handlers[value]; ok {
			t.Errorf("method %q is now handled — delete it from notYetHandled", name)
		}
	}
}

// notYetHandled are wire methods the API declares ahead of the router handler that will serve
// them. Tracked debt: add an entry only when the contract genuinely lands before its handler, and
// DELETE it the moment the handler lands (the guard above fails on a stale entry, so this list may
// only shrink).
var notYetHandled = map[string]bool{
	// model.assignOpenPBRAppearance (M45-F02 #2126): the OpenPBRAppearance library/CRUD
	// contract lands in PBI-339, but assignment needs the document-embedding and
	// assignment-store plumbing that PBI-350 (M45-F05, "wire kRealisticRendering end to
	// end") builds once the renderer can actually consume an OpenPBR surface. Delete this
	// entry the moment that handler lands.
	"MethodModelAssignOpenPBRAppearance": true,
}

// notYetRelayed are wire events the API declares ahead of the host behavior that would
// emit them (the representations / model-state surface has no app-level event yet). They
// are tracked debt: when that feature fires an app event, relay it in addin/events and
// DELETE it from this list. Any OTHER unrelayed event fails the test below — so this list
// may only shrink. See https://github.com/Oblikovati/Oblikovati/issues/901.
var notYetRelayed = map[string]bool{}

// TestEveryWireEventIsRelayed guards the other direction: every wire.Event* constant
// must be referenced by the add-in's event relay (addin/events) or the router, so a
// declared host→add-in event cannot be left unrelayed (except the tracked debt above).
func TestEveryWireEventIsRelayed(t *testing.T) {
	events := wireConstants(t, "Event")
	if len(events) == 0 {
		t.Fatal("parsed zero wire Event constants — the wire source lookup is broken")
	}
	used := identifiersUsedIn(t, "oblikovati.org/addin/events", "oblikovati.org/addin/router")
	var missing []string
	for name := range events {
		if !used[name] && !notYetRelayed[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("%d wire event(s) declared in the API are never relayed:\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
	// The allowlist may only shrink: a tracked event that is now relayed (or is no longer
	// a declared event) must be removed from it.
	for name := range notYetRelayed {
		if used[name] {
			t.Errorf("event %q is now relayed — delete it from notYetRelayed", name)
		}
		if _, declared := events[name]; !declared {
			t.Errorf("notYetRelayed lists %q, which is not a wire Event constant", name)
		}
	}
}

// wireConstants parses the oblikovati.org/api/wire package source and returns the
// {name: value} of every string const whose name starts with prefix (e.g. "Method").
func wireConstants(t *testing.T, prefix string) map[string]string {
	t.Helper()
	out := map[string]string{}
	fset := token.NewFileSet()
	for _, dir := range packageDirs(t, "oblikovati.org/api/wire") {
		pkgs, err := parser.ParseDir(fset, dir, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				collectStringConsts(file, prefix, out)
			}
		}
	}
	return out
}

// collectStringConsts records every `Name = "value"` const whose name starts with prefix.
func collectStringConsts(file *ast.File, prefix string, out map[string]string) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
				continue
			}
			lit, ok := vs.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			if name := vs.Names[0].Name; strings.HasPrefix(name, prefix) {
				if v, err := strconv.Unquote(lit.Value); err == nil {
					out[name] = v
				}
			}
		}
	}
}

// identifiersUsedIn returns the set of identifier names referenced anywhere in the given
// packages' .go files (excluding tests) — used to confirm an event constant is consumed.
func identifiersUsedIn(t *testing.T, importPaths ...string) map[string]bool {
	t.Helper()
	used := map[string]bool{}
	fset := token.NewFileSet()
	for _, ip := range importPaths {
		for _, dir := range packageDirs(t, ip) {
			pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
				return !strings.HasSuffix(fi.Name(), "_test.go")
			}, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", dir, err)
			}
			for _, pkg := range pkgs {
				for _, file := range pkg.Files {
					ast.Inspect(file, func(n ast.Node) bool {
						if id, ok := n.(*ast.Ident); ok {
							used[id.Name] = true
						}
						return true
					})
				}
			}
		}
	}
	return used
}

// packageDirs resolves an import path to its on-disk directory via `go list` (which
// honors the go.work / CI replace that points oblikovati.org/api at the sibling checkout).
func packageDirs(t *testing.T, importPath string) []string {
	t.Helper()
	out, err := exec.Command("go", "list", "-f", "{{.Dir}}", importPath).Output()
	if err != nil {
		t.Fatalf("go list %s: %v", importPath, err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" || !filepath.IsAbs(dir) {
		t.Fatalf("go list %s returned %q", importPath, dir)
	}
	return []string{dir}
}
