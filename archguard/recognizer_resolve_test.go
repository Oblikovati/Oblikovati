// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/token"
)

// RESOLUTION for the recognizer derivation (Oblikovati/Oblikovati#3522): given the index of the
// classification's package tree (recognizer_index_test.go), what does one NAME in its source mean?
//
// Three questions, and none of them is answered by a spelling:
//
//   - Is this declaration a VERDICT? Its result-type set decides — a final bool, and payloads before
//     it — not a match against `bool` and `(<something>Trim, bool)`.
//   - What does this TYPE expression denote? A canonical key, through pointers and parentheses and
//     through a package of the tree; anything else resolves to nothing and can be neither payload nor
//     bool.
//   - What does this CALL reach? A bare name, a tree package's `pkg.F`, a generic instantiation's
//     underlying callee, or a method — and a method only when the AST states a receiver whose type the
//     tree declares, which is what keeps a foreign `when.IsZero()` from inflating the pin (review I1).
//
// Everything here is syntactic on purpose. The alternative is a full type-check of kernel/ops/tessellate
// inside a guard, and the limits of the syntactic reading are enumerated in the index's own header
// rather than left for a reader to discover.

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
//	f(…)       a function of this package
//	x.f(…)     a METHOD, claimed only when the AST states a receiver of a type the tree declares
//	pkg.F(…)   a package-qualified call, resolved ONLY into the classification's own tree
//	f[T](…)    the same function with its type argument written out
//
// A selector on an import from outside the tree returns no name at all: resolving it by its bare
// selector would let `stdmath.Signbit(…)` answer to a local `Signbit`, which the probe test plants.
func (idx *recognizerIndex) calleeName(call *ast.CallExpr, d declaredHere) (string, bool) {
	return idx.calleeExprName(call.Fun, d)
}

// calleeExprName resolves the expression a call invokes. An explicit generic instantiation wraps the
// callee it instantiates — `f[T](…)` is an *ast.IndexExpr and `f[T1, T2](…)` an *ast.IndexListExpr —
// so unwrapping it is what keeps a generic recognizer as visible as the same recognizer with its type
// argument inferred (review I2). The same unwrapping makes `table[i](…)` resolve to `table`, which
// names no declaration, so a func value taken out of a slice remains invisible.
func (idx *recognizerIndex) calleeExprName(fun ast.Expr, d declaredHere) (string, bool) {
	switch x := fun.(type) {
	case *ast.Ident:
		return x.Name, true
	case *ast.SelectorExpr:
		return idx.selectorCallee(x, d)
	case *ast.ParenExpr:
		return idx.calleeExprName(x.X, d)
	case *ast.IndexExpr:
		return idx.calleeExprName(x.X, d)
	case *ast.IndexListExpr:
		return idx.calleeExprName(x.X, d)
	}
	return "", false
}

// selectorCallee splits `x.f` into the two calls it can be: into a package of the tree, or a method on
// a value. A method is claimed ONLY when the AST states a receiver whose type the tree declares.
// Claiming it on the bare selector alone is a channel that INFLATES the pin: `when.IsZero()` on a
// time.Time would answer to an in-tree `IsZero` and be counted as a recognizer the classification never
// reads (review I1), and an inflated base makes a later FALL look real when nothing was deleted. A
// receiver the AST does not state is claimed by neither branch, and assertNoUnresolvableMethodReads
// refuses it separately when its name reaches a verdict.
func (idx *recognizerIndex) selectorCallee(sel *ast.SelectorExpr, d declaredHere) (string, bool) {
	if base, isIdent := sel.X.(*ast.Ident); isIdent {
		if path, imported := d.imports[base.Name]; imported {
			return idx.treePackageCallee(path, sel.Sel.Name)
		}
	}
	if idx.receiverOriginOf(sel.X, d) != receiverInTree {
		return "", false
	}
	return sel.Sel.Name, true
}

// treePackageCallee is `pkg.F(…)` resolved into the classification's tree, or nothing at all.
func (idx *recognizerIndex) treePackageCallee(path, name string) (string, bool) {
	scope, inTree := idx.scopes[path]
	if !inTree {
		return "", false
	}
	return scope.qualify(name), true
}

// receiverOrigin says what a method call's receiver is, as far as the AST states it.
type receiverOrigin int

const (
	// receiverUnstated is `x := f()`: this index runs no type checker, so nothing about x is known.
	receiverUnstated receiverOrigin = iota
	// receiverInTree is a value of a type the classification's package tree declares.
	receiverInTree
	// receiverOutsideTree is a time.Time, a geom.Cone, a topo.Face — a method that is not ours.
	receiverOutsideTree
)

// receiverOriginOf resolves the type of a method call's receiver from what the AST states about it.
func (idx *recognizerIndex) receiverOriginOf(base ast.Expr, d declaredHere) receiverOrigin {
	stated, in := idx.statedTypeOf(base, d)
	if stated == nil {
		return receiverUnstated
	}
	if idx.treeTypes[idx.typeKey(stated, in)] {
		return receiverInTree
	}
	return receiverOutsideTree
}

// statedTypeOf is the type expression the AST gives for a method call's receiver, together with the
// declaration that type expression is WRITTEN IN — a field's type is spelled in its struct's scope, not
// in the caller's, and canonicalising it against the wrong scope would resolve the wrong type.
func (idx *recognizerIndex) statedTypeOf(base ast.Expr, d declaredHere) (ast.Expr, declaredHere) {
	switch x := base.(type) {
	case *ast.ParenExpr:
		return idx.statedTypeOf(x.X, d)
	case *ast.UnaryExpr:
		return idx.addressOfType(x, d)
	case *ast.CompositeLit:
		return x.Type, d
	case *ast.Ident:
		return declaredTypeOfName(x.Name, d.fn), d
	case *ast.SelectorExpr:
		return idx.fieldTypeOf(x, d)
	}
	return nil, d
}

// addressOfType unwraps `&T{…}`; any other unary expression states no type.
func (idx *recognizerIndex) addressOfType(u *ast.UnaryExpr, d declaredHere) (ast.Expr, declaredHere) {
	if u.Op != token.AND {
		return nil, d
	}
	return idx.statedTypeOf(u.X, d)
}

// fieldTypeOf resolves `x.field` when x's own type is a struct the tree declares. The chart mesher
// writes `b.r.covers(u, v)`, where r is a field of an in-tree struct the index has ALREADY parsed;
// giving up on it made seven ordinary call sites in real kernel source unattributable, and the remedy
// the refusal prescribes would have been to rewrite clean code to appease a guard (review N1).
func (idx *recognizerIndex) fieldTypeOf(sel *ast.SelectorExpr, d declaredHere) (ast.Expr, declaredHere) {
	owner, in := idx.statedTypeOf(sel.X, d)
	if owner == nil {
		return nil, d
	}
	declared, isStruct := idx.structs[idx.typeKey(owner, in)]
	if !isStruct {
		return nil, d
	}
	return fieldTypeNamed(sel.Sel.Name, declared.fields.Fields), declared.in
}

// declaredTypeOfName is the type fn states for a name: its own receiver, a parameter, a named result, a
// `var` declaration, or a short declaration from a composite literal. `x := someCall()` states none.
func declaredTypeOfName(name string, fn *ast.FuncDecl) ast.Expr {
	if fn == nil {
		return nil
	}
	for _, list := range []*ast.FieldList{fn.Recv, fn.Type.Params, fn.Type.Results} {
		if stated := fieldTypeNamed(name, list); stated != nil {
			return stated
		}
	}
	return bodyDeclaredType(name, fn.Body)
}

// fieldTypeNamed is the type of the field declaring name, or nil.
func fieldTypeNamed(name string, list *ast.FieldList) ast.Expr {
	if list == nil {
		return nil
	}
	for _, field := range list.List {
		for _, id := range field.Names {
			if id.Name == name {
				return field.Type
			}
		}
	}
	return nil
}

// bodyDeclaredType is the type a body states for name: `var x T`, or `x := T{}` / `x := &T{}`.
func bodyDeclaredType(name string, body *ast.BlockStmt) ast.Expr {
	if body == nil {
		return nil
	}
	var found ast.Expr
	ast.Inspect(body, func(n ast.Node) bool {
		switch st := n.(type) {
		case *ast.ValueSpec:
			if st.Type != nil && identsInclude(st.Names, name) {
				found = st.Type
			}
		case *ast.AssignStmt:
			if stated := shortDeclaredType(name, st); stated != nil {
				found = stated
			}
		}
		return found == nil
	})
	return found
}

// shortDeclaredType is the type `name := T{}` or `name := &T{}` states, or nil.
func shortDeclaredType(name string, st *ast.AssignStmt) ast.Expr {
	if st.Tok != token.DEFINE || len(st.Lhs) != len(st.Rhs) {
		return nil
	}
	for i, lhs := range st.Lhs {
		if id, isIdent := lhs.(*ast.Ident); isIdent && id.Name == name {
			return compositeLitType(st.Rhs[i])
		}
	}
	return nil
}

// compositeLitType is the type of `T{…}` or `&T{…}`, and nothing for any other expression.
func compositeLitType(e ast.Expr) ast.Expr {
	if u, isUnary := e.(*ast.UnaryExpr); isUnary && u.Op == token.AND {
		e = u.X
	}
	if lit, isLit := e.(*ast.CompositeLit); isLit {
		return lit.Type
	}
	return nil
}

// identsInclude reports whether the name list holds name.
func identsInclude(names []*ast.Ident, name string) bool {
	for _, id := range names {
		if id.Name == name {
			return true
		}
	}
	return false
}

// unattributableMethod reports a method call whose receiver type the AST does not state and whose bare
// name reaches a verdict declaration of the tree. Such a call is EITHER a recognizer read or a call on
// a foreign value that happens to share the name; guessing either way moves the pin, so it is refused.
func (idx *recognizerIndex) unattributableMethod(call *ast.CallExpr, d declaredHere) (string, bool) {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel {
		return "", false
	}
	if base, isIdent := sel.X.(*ast.Ident); isIdent {
		if _, imported := d.imports[base.Name]; imported {
			return "", false
		}
	}
	if idx.receiverOriginOf(sel.X, d) != receiverUnstated {
		return "", false
	}
	target := idx.lookup(sel.Sel.Name)
	return sel.Sel.Name, target != nil && idx.isVerdict(*target)
}
