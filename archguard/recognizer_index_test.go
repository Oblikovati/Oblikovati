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

// The INDEX the recognizer derivation resolves names against (Oblikovati/Oblikovati#3522).
//
// The derivation (recognizer_derivation_test.go) counts the bespoke shape recognizers standing behind
// classifyCurvedTrim, and that count is the "recognizers" ratchet a later slice must drive toward zero
// by DELETING arms. A ratchet is only worth its number if the walk can see what it claims to count,
// and the first cut could not see two whole shapes:
//
//   - a recognizer reached through a call that is not a bare identifier — a METHOD on a value, or a
//     function moved into a helper package of the classification's own tree and called `trim.F(…)`;
//   - a recognizer returning any verdict but the two spellings `bool` and `(<something>Trim, bool)` —
//     a pointer payload, a third result, a payload renamed off the "Trim" suffix, a named bool.
//
// Either one moves the count by ZERO, so a recognizer could be added inside an arm, or an existing one
// relocated, with the ratchet reporting no change at all.
//
// This file is the resolution those two blind spots needed, and it makes both answers structural:
//
//   - NAMES. Every function AND method of the classification's package, plus every package under it,
//     is indexed under the key a caller writes: a bare name in the package itself, `pkg.Name` for a
//     package of the tree. A selector on a package OUTSIDE the tree resolves to nothing, so
//     `stdmath.Signbit(…)` can never answer to a local `Signbit` — planted, and measured, by
//     walkBoundaryShapes.
//   - VERDICTS. A verdict is read off the RESULT-TYPE SET: the last result is a bool, and every
//     earlier result is one of the classification's own payload types. The payload set is DERIVED
//     from the verdict struct — curvedTrim and the types of its fields — because that is what makes a
//     payload a payload: an arm's recognition has to land in the verdict the classification returns.
//     Renaming coneApexTrim keeps it counted; a chart a recognizer merely USES stays outside the walk.
//     An earlier result may also be a bool, so `(found, ok bool)` is a verdict too.
//
// What this index still cannot see, stated rather than implied (the discipline #3536 records for the
// FMA scan's own generic blind spot):
//
//   - A recognizer moved OUT of the classification's tree entirely — into kernel/geom, say — is not
//     followed, and the count falls by one without a shape being deleted. That is deliberate: a
//     predicate general enough to live in geom is what the ground rules ASK for ("behaviour many
//     operations need is a method on geom.Surface"), and following bare selectors into every imported
//     package would count library predicates as bespoke recognizers. A relocation out of the tree
//     must therefore be justified as a deletion in the ADR, which the index cannot do for you. The
//     limit is not merely stated: walkBoundaryShapes plants a call into a package outside the tree and
//     asserts the derivation reads nothing from it.
//   - Type resolution is syntactic. `type verdict bool` is a bool; `type verdict2 verdict` is not
//     resolved through the second hop, and neither is a payload reached through an interface or a
//     type parameter.
//   - A callee is resolved by NAME, not by the receiver's type. Two declarations answering to one key
//     are refused rather than guessed (assertUnambiguousVerdictNames).

const (
	// classificationDir is the package classifyCurvedTrim lives in; the index covers it and every
	// package beneath it.
	classificationDir = "../kernel/ops/tessellate"
	// classificationImportPath is what that directory is imported as, so a `pkg.F(…)` call inside the
	// tree can be told apart from one into geom or the standard library.
	classificationImportPath = "oblikovati.org/kernel/ops/tessellate"
	// verdictStructName is the struct every arm's recognition lands in; its fields ARE the payload set.
	verdictStructName = "curvedTrim"
)

// packageScope is one package of the classification's tree: the prefix its declarations are indexed
// under ("" for the classification itself, "trim." for a package beneath it) and the types it declares.
type packageScope struct {
	prefix string
	types  map[string]bool
}

// qualify is the index key a name declared in this scope is reachable under.
func (s *packageScope) qualify(name string) string { return s.prefix + name }

// typeKey canonicalises a type name written inside this scope. A name the scope does not DECLARE is
// predeclared (bool, int, error) or comes from elsewhere, and keeps its bare spelling — which is what
// keeps `bool` spelled `bool` inside a subpackage.
func (s *packageScope) typeKey(name string) string {
	if !s.types[name] {
		return name
	}
	return s.qualify(name)
}

// declaredHere is one function or method declaration with everything needed to resolve the names its
// body writes: the scope it lives in and what its own FILE can call each import.
type declaredHere struct {
	fn      *ast.FuncDecl
	scope   *packageScope
	imports map[string]string // file-local package name → import path
	where   string
}

// recognizerIndex answers the two questions the derivation asks of a name: which declaration does this
// callee reach, and is that declaration a verdict.
type recognizerIndex struct {
	scopes    map[string]*packageScope // import path → scope
	decls     map[string][]declaredHere
	boolTypes map[string]bool // canonical keys of named types whose underlying is bool
	payloads  map[string]bool // canonical keys of the classification's own verdict payload types
	verdict   *ast.StructType
	verdictIn declaredHere // the scope and imports the verdict struct's field types are written in
	fset      *token.FileSet
}

// tessellateIndex indexes the classification's own package tree.
func tessellateIndex(t *testing.T) *recognizerIndex {
	t.Helper()
	return newRecognizerIndex(t, classificationDir, classificationImportPath)
}

// newRecognizerIndex indexes root and every package beneath it, under the keys a caller inside the
// tree writes. root is a parameter so the probe test can plant blind shapes in a package of its own
// rather than in the kernel.
func newRecognizerIndex(t *testing.T, root, importPath string) *recognizerIndex {
	t.Helper()
	idx := &recognizerIndex{
		scopes: map[string]*packageScope{}, decls: map[string][]declaredHere{},
		boolTypes: map[string]bool{}, fset: token.NewFileSet(),
	}
	for _, dir := range packageDirsUnder(t, root) {
		idx.addPackage(t, dir, treeImportPath(importPath, root, dir), dir == root)
	}
	idx.payloads = idx.verdictPayloads(t)
	return idx
}

// packageDirsUnder is every directory at or below root that holds non-test Go source, sorted, so the
// index is built in one deterministic order on every platform.
func packageDirsUnder(t *testing.T, root string) []string {
	t.Helper()
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && len(goSourceNames(t, path)) > 0 {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	sort.Strings(dirs)
	return dirs
}

// goSourceNames is the non-test .go files of one directory, sorted.
func goSourceNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// treeImportPath is the path a directory of the tree is imported under.
func treeImportPath(importPath, root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return importPath
	}
	return importPath + "/" + filepath.ToSlash(rel)
}

// addPackage indexes one package: its type names first (a result type resolves against them), then the
// named bools among them, the verdict struct if this is the root, and finally every declaration.
func (idx *recognizerIndex) addPackage(t *testing.T, dir, importPath string, isRoot bool) {
	t.Helper()
	files := idx.parsePackage(t, dir)
	scope := &packageScope{types: map[string]bool{}}
	if !isRoot {
		scope.prefix = files[0].Name.Name + "."
	}
	idx.scopes[importPath] = scope
	for _, f := range files {
		collectTypeNames(f, scope)
	}
	for _, f := range files {
		idx.collectTypeShapes(f, scope, isRoot)
		idx.collectDeclarations(f, scope)
	}
}

// parsePackage parses the non-test files of one directory, in name order.
func (idx *recognizerIndex) parsePackage(t *testing.T, dir string) []*ast.File {
	t.Helper()
	var files []*ast.File
	for _, name := range goSourceNames(t, dir) {
		f, err := parser.ParseFile(idx.fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", filepath.Join(dir, name), err)
		}
		files = append(files, f)
	}
	return files
}

// collectTypeNames records which names this package declares as types.
func collectTypeNames(f *ast.File, scope *packageScope) {
	forEachTypeSpec(f, func(spec *ast.TypeSpec) {
		scope.types[spec.Name.Name] = true
	})
}

// collectTypeShapes records the named bools of a package and, in the root, the verdict struct whose
// fields are the payload set.
func (idx *recognizerIndex) collectTypeShapes(f *ast.File, scope *packageScope, isRoot bool) {
	imports := fileImports(f)
	forEachTypeSpec(f, func(spec *ast.TypeSpec) {
		if id, isIdent := spec.Type.(*ast.Ident); isIdent && id.Name == "bool" {
			idx.boolTypes[scope.qualify(spec.Name.Name)] = true
		}
		st, isStruct := spec.Type.(*ast.StructType)
		if isRoot && isStruct && spec.Name.Name == verdictStructName {
			idx.verdict, idx.verdictIn = st, declaredHere{scope: scope, imports: imports}
		}
	})
}

// forEachTypeSpec visits every type declaration of a file.
func forEachTypeSpec(f *ast.File, visit func(*ast.TypeSpec)) {
	for _, d := range f.Decls {
		gd, isGen := d.(*ast.GenDecl)
		if !isGen || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			if ts, isType := spec.(*ast.TypeSpec); isType {
				visit(ts)
			}
		}
	}
}

// collectDeclarations indexes every function AND method of a file. Methods are indexed under their
// bare name because that is the only name their call site writes: `x.f(…)` says nothing about x's type.
func (idx *recognizerIndex) collectDeclarations(f *ast.File, scope *packageScope) {
	imports := fileImports(f)
	for _, d := range f.Decls {
		fd, isFunc := d.(*ast.FuncDecl)
		if !isFunc {
			continue
		}
		key := scope.qualify(fd.Name.Name)
		idx.decls[key] = append(idx.decls[key], declaredHere{
			fn: fd, scope: scope, imports: imports, where: idx.fset.Position(fd.Pos()).String(),
		})
	}
}

// fileImports maps the name a file calls each import by to that import's path.
func fileImports(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, spec := range f.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		name := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		out[name] = path
	}
	return out
}

// verdictPayloads is the classification's own payload set, derived from the verdict struct: the struct
// itself plus the type of every field it carries. Deriving it is what replaces the "…Trim" spelling —
// an arm's recognition has to land in the verdict, so the fields ARE the inventory of payloads.
func (idx *recognizerIndex) verdictPayloads(t *testing.T) map[string]bool {
	t.Helper()
	if idx.verdict == nil {
		t.Fatalf("no %s struct is declared in %s: the derivation reads the payload set off that "+
			"struct's fields, so it cannot resolve a verdict without it", verdictStructName, classificationDir)
	}
	out := map[string]bool{idx.verdictIn.scope.qualify(verdictStructName): true}
	for _, field := range idx.verdict.Fields.List {
		if key := idx.typeKey(field.Type, idx.verdictIn); key != "" {
			out[key] = true
		}
	}
	return out
}

// lookup is the declaration a callee name reaches, or nil. A verdict declaration wins over a
// non-verdict one of the same name; that the two can coexist at all is refused separately, by
// assertUnambiguousVerdictNames, rather than resolved here.
func (idx *recognizerIndex) lookup(name string) *declaredHere {
	found := idx.decls[name]
	for i := range found {
		if idx.isVerdict(found[i]) {
			return &found[i]
		}
	}
	if len(found) == 0 {
		return nil
	}
	return &found[0]
}

// ambiguousVerdictNames are the keys more than one declaration answers to where at least one is
// verdict-shaped. `foo()` and `x.foo()` are the same key to this index, so such a pair would let a
// read resolve against whichever declaration happened to be indexed first.
func (idx *recognizerIndex) ambiguousVerdictNames() []string {
	var out []string
	for name, found := range idx.decls {
		if len(found) < 2 {
			continue
		}
		for i := range found {
			if idx.isVerdict(found[i]) {
				out = append(out, name+" ("+placesOf(found)+")")
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// placesOf lists where a name's declarations are, so an ambiguity failure names both.
func placesOf(found []declaredHere) string {
	var at []string
	for _, d := range found {
		at = append(at, d.where)
	}
	return strings.Join(at, ", ")
}

// isVerdict reports whether a declaration returns a classification verdict, read off its RESULT-TYPE
// SET rather than off a list of spellings (#3522): the last result is a bool (or a named bool), and
// every earlier result is one of the classification's own payload types, or another bool.
//
// That is what stops the walk at a geometric helper without naming one: capAxis returns a vector,
// chooseSphereChart a chart, splitWrappingHoles two slices — none of those is a field of curvedTrim,
// so none of them is a verdict.
func (idx *recognizerIndex) isVerdict(d declaredHere) bool {
	results := idx.resultTypeKeys(d)
	if len(results) == 0 || !idx.isBool(results[len(results)-1]) {
		return false
	}
	for _, key := range results[:len(results)-1] {
		if !idx.payloads[key] && !idx.isBool(key) {
			return false
		}
	}
	return true
}

// isBool reports whether a canonical type key denotes a boolean.
func (idx *recognizerIndex) isBool(key string) bool {
	return key == "bool" || idx.boolTypes[key]
}

// resultTypeKeys is the declaration's results ONE PER RESULT — `(a, b bool)` is one field and two
// results — each canonicalised to the key its type is indexed under.
func (idx *recognizerIndex) resultTypeKeys(d declaredHere) []string {
	if d.fn == nil || d.fn.Type.Results == nil {
		return nil
	}
	var out []string
	for _, field := range d.fn.Type.Results.List {
		key := idx.typeKey(field.Type, d)
		for range max(1, len(field.Names)) {
			out = append(out, key)
		}
	}
	return out
}

// typeKey canonicalises a type expression written inside d. A pointer is the type it points at — a
// recognizer handing back `*coneApexTrim` returns the same payload — and anything else (a slice, a map,
// a func) is deliberately unresolvable, so it can be neither a payload nor a bool.
func (idx *recognizerIndex) typeKey(e ast.Expr, d declaredHere) string {
	switch x := e.(type) {
	case *ast.ParenExpr:
		return idx.typeKey(x.X, d)
	case *ast.StarExpr:
		return idx.typeKey(x.X, d)
	case *ast.Ident:
		return d.scope.typeKey(x.Name)
	case *ast.SelectorExpr:
		return idx.qualifiedTypeKey(x, d)
	}
	return ""
}

// qualifiedTypeKey resolves `pkg.T` when pkg is a package of the classification's own tree — a payload
// that moved into a helper package is still that payload. A type from anywhere else (geom.Cone,
// topo.Face) resolves to nothing, which is what keeps it out of the payload set.
func (idx *recognizerIndex) qualifiedTypeKey(sel *ast.SelectorExpr, d declaredHere) string {
	scope, inTree := idx.treeScopeOf(sel.X, d.imports)
	if !inTree {
		return ""
	}
	return scope.typeKey(sel.Sel.Name)
}

// treeScopeOf is the scope a selector's base names, when that base is an import of the classification's
// own tree.
func (idx *recognizerIndex) treeScopeOf(base ast.Expr, imports map[string]string) (*packageScope, bool) {
	id, isIdent := base.(*ast.Ident)
	if !isIdent {
		return nil, false
	}
	path, imported := imports[id.Name]
	if !imported {
		return nil, false
	}
	scope, inTree := idx.scopes[path]
	return scope, inTree
}

// calleeName is the index key a call reaches, in each shape Go writes a call in:
//
//	f(…)      a function of this package
//	x.f(…)    a METHOD, or a call through a field; resolved by its bare name, which only names
//	          something when the tree declares an f of its own
//	pkg.F(…)  a package-qualified call, resolved ONLY into the classification's own tree
//
// A selector on an import from outside the tree returns no name at all: resolving it by its bare
// selector would let `stdmath.Min(…)` answer to a local `Min`, which the probe test plants.
func (idx *recognizerIndex) calleeName(call *ast.CallExpr, d declaredHere) (string, bool) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name, true
	case *ast.SelectorExpr:
		return idx.selectorCallee(fun, d)
	}
	return "", false
}

// selectorCallee splits `x.f` into the two calls it can be: into a package of the tree, or a method.
func (idx *recognizerIndex) selectorCallee(sel *ast.SelectorExpr, d declaredHere) (string, bool) {
	if base, isIdent := sel.X.(*ast.Ident); isIdent {
		if path, imported := d.imports[base.Name]; imported {
			scope, inTree := idx.scopes[path]
			if !inTree {
				return "", false
			}
			return scope.qualify(sel.Sel.Name), true
		}
	}
	return sel.Sel.Name, true
}
