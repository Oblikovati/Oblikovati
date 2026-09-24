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
// This file holds the INDEX — what the tree contains. What a name in it MEANS (verdict, type key,
// callee) is resolved in recognizer_resolve_test.go. Together they make both answers structural:
//
//   - NAMES. Every function AND method of the classification's package, plus every package under it,
//     is indexed under the key a caller writes: a bare name in the package itself, `pkg.Name` for a
//     package of the tree, and the same name again when the call spells its type argument out
//     (`f[T](…)`). A selector on a package OUTSIDE the tree resolves to nothing, so
//     `stdmath.Signbit(…)` can never answer to a local `Signbit`.
//   - RECEIVERS. A method is claimed only when the AST states a receiver whose type the tree DECLARES.
//     This is the half that keeps the pin from being INFLATED: `when.IsZero()` on a time.Time must not
//     answer to an in-tree `IsZero`, and an inflated base is worse than a missed recognizer, because a
//     later fall measured from it looks real when nothing was deleted. Both directions are planted by
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
//     type parameter. A generic recognizer's CALL is resolved (both spellings); a generic recognizer
//     whose PAYLOAD is a type parameter is not.
//   - A payload returned inside a slice, array or map — `([]coneApexTrim, bool)` — is not resolved, so
//     such a recognizer is invisible. Unwrapping the element would also make `func f() []bool` a
//     verdict, and that is measured, not guessed: on today's tree the unwrap turns 4 existing
//     declarations into verdicts (ConsistentOutwardFlips, floodInside, frustratedFaces, and seamEndMask,
//     which returns `([]bool, bool)`) and catches 0 recognizers, since no `([]payload, bool)`
//     declaration exists. Inflation 4, catch 0 — so the limit stands, planted by walkBoundaryShapes.
//   - Two declarations answering to one key are refused rather than guessed
//     (assertUnambiguousVerdictNames), and so is a method call whose receiver type the AST does not
//     state when its name reaches a verdict (assertNoUnresolvableMethodReads). Neither refusal can
//     fire on today's tree; both are planted.
//   - A receiver's type is read off what the AST states — a parameter, a named result, the enclosing
//     receiver, a `var`, or a composite literal. `x := f()` states none, and that call is refused
//     rather than resolved.

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
	prefix  string
	pkgName string
	types   map[string]bool
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
	treeTypes map[string]bool // canonical keys of EVERY type the tree declares, for receiver origin
	payloads  map[string]bool // canonical keys of the classification's own verdict payload types
	structs   map[string]treeStruct
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
		boolTypes: map[string]bool{}, treeTypes: map[string]bool{},
		structs: map[string]treeStruct{}, fset: token.NewFileSet(),
	}
	packages := idx.parseTree(t, root, importPath)
	for _, pkg := range packages {
		idx.declareTypeNames(pkg)
	}
	for _, pkg := range packages {
		idx.indexPackage(pkg)
	}
	idx.payloads = idx.verdictPayloads(t)
	return idx
}

// treePackage is one parsed package, held between the indexing phases. The phases exist because every
// scope must be registered before any FILE's imports are read: an unaliased import is named by the
// imported package's PACKAGE name, which only the parsed package knows, and guessing it from the
// directory would silently drop every recognizer behind a package whose two names differ (review M7).
type treePackage struct {
	importPath string
	scope      *packageScope
	files      []*ast.File
}

// parseTree parses every package at or below root and registers its scope.
func (idx *recognizerIndex) parseTree(t *testing.T, root, importPath string) []treePackage {
	t.Helper()
	var out []treePackage
	for _, dir := range packageDirsUnder(t, root) {
		files := idx.parsePackage(t, dir)
		path := treeImportPath(importPath, root, dir)
		scope := &packageScope{pkgName: files[0].Name.Name, types: map[string]bool{}}
		if path != importPath {
			scope.prefix = scope.pkgName + "."
		}
		idx.scopes[path] = scope
		out = append(out, treePackage{importPath: path, scope: scope, files: files})
	}
	return out
}

// declareTypeNames records the types a package declares, in its own scope and in the tree-wide set a
// method call's receiver is resolved against.
func (idx *recognizerIndex) declareTypeNames(pkg treePackage) {
	for _, f := range pkg.files {
		forEachTypeSpec(f, func(spec *ast.TypeSpec) {
			pkg.scope.types[spec.Name.Name] = true
			idx.treeTypes[pkg.scope.qualify(spec.Name.Name)] = true
		})
	}
}

// indexPackage records a package's named bools and structs, and every function and method it declares.
func (idx *recognizerIndex) indexPackage(pkg treePackage) {
	for _, f := range pkg.files {
		idx.collectTypeShapes(f, pkg.scope)
		idx.collectDeclarations(f, pkg.scope)
	}
}

// treeStruct is a struct the tree declares, with the declaration its FIELD TYPES are written in. The
// verdict struct is one of these; so is every struct a method call's receiver may be a field of.
type treeStruct struct {
	fields *ast.StructType
	in     declaredHere
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

// collectTypeShapes records the named bools and the structs of a package. The verdict struct, whose
// fields are the payload set, is just the one whose key is verdictStructName — only the classification
// itself has an empty prefix, so a struct of that name in a package beneath it cannot be mistaken for it.
func (idx *recognizerIndex) collectTypeShapes(f *ast.File, scope *packageScope) {
	in := declaredHere{scope: scope, imports: idx.fileImports(f)}
	forEachTypeSpec(f, func(spec *ast.TypeSpec) {
		if id, isIdent := spec.Type.(*ast.Ident); isIdent && id.Name == "bool" {
			idx.boolTypes[scope.qualify(spec.Name.Name)] = true
		}
		if st, isStruct := spec.Type.(*ast.StructType); isStruct {
			idx.structs[scope.qualify(spec.Name.Name)] = treeStruct{fields: st, in: in}
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
	imports := idx.fileImports(f)
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
func (idx *recognizerIndex) fileImports(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, spec := range f.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		out[idx.importLocalName(spec, path)] = path
	}
	return out
}

// importLocalName is what one file calls one import: its alias, else the imported package's own name.
// For a package of the tree the index has parsed that name; for anything else the last path element is
// the only available guess, and it is only ever used to REJECT a call, never to claim one.
func (idx *recognizerIndex) importLocalName(spec *ast.ImportSpec, path string) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	if scope, inTree := idx.scopes[path]; inTree {
		return scope.pkgName
	}
	return path[strings.LastIndex(path, "/")+1:]
}

// verdictPayloads is the classification's own payload set, derived from the verdict struct: the struct
// itself plus the type of every field it carries. Deriving it is what replaces the "…Trim" spelling —
// an arm's recognition has to land in the verdict, so the fields ARE the inventory of payloads.
func (idx *recognizerIndex) verdictPayloads(t *testing.T) map[string]bool {
	t.Helper()
	verdict, declared := idx.structs[verdictStructName]
	if !declared {
		t.Fatalf("no %s struct is declared in %s: the derivation reads the payload set off that "+
			"struct's fields, so it cannot resolve a verdict without it", verdictStructName, classificationDir)
	}
	out := map[string]bool{verdictStructName: true}
	for _, field := range verdict.fields.Fields.List {
		if key := idx.typeKey(field.Type, verdict.in); key != "" {
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
