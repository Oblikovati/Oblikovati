// SPDX-License-Identifier: GPL-2.0-only

package param

import "fmt"

// ID is a parameter's stable identity. Expression references and the
// dependency graph bind to it, never to display text, so renaming a parameter
// is a relabel rather than a global string rewrite (architecture core/04).
type ID uint64

// Scope resolves a bound reference's current value during evaluation.
type Scope interface {
	// ValueOf returns the current value of the parameter with the given id.
	ValueOf(id ID) (Quantity, bool)
}

// node is one AST node of a parsed expression.
type node interface {
	// eval computes the node's value; scope may be nil for reference-free nodes.
	eval(scope Scope) (Quantity, error)
	// walkRefs invokes fn on every reference node in the subtree.
	walkRefs(fn func(*refNode))
}

// numberNode is a literal, optionally carrying a unit ("5 mm" → {Value:0.5, Unit:Length}).
type numberNode struct{ q Quantity }

// refNode is a reference to another parameter by name; the parser records the
// name and the graph binds it to a stable id (see [Expr.Bind]).
type refNode struct {
	name  string
	id    ID
	bound bool
}

// binaryNode is an arithmetic operation (op is one of + - * /).
type binaryNode struct {
	op       byte
	lhs, rhs node
}

// unaryNode is a unary negation.
type unaryNode struct{ operand node }

// callNode is a built-in function call.
type callNode struct {
	fn   string
	args []node
}

func (n numberNode) walkRefs(func(*refNode))  {}
func (n *refNode) walkRefs(fn func(*refNode)) { fn(n) }
func (n binaryNode) walkRefs(fn func(*refNode)) {
	n.lhs.walkRefs(fn)
	n.rhs.walkRefs(fn)
}
func (n unaryNode) walkRefs(fn func(*refNode)) { n.operand.walkRefs(fn) }
func (n callNode) walkRefs(fn func(*refNode)) {
	for _, a := range n.args {
		a.walkRefs(fn)
	}
}

// Expr is a parsed, immutable parameter expression: its authored source, the
// AST root, and the distinct names it references (in source order). Bind it to
// stable ids before evaluating references.
type Expr struct {
	src  string
	root node
	refs []string
}

// Parse compiles source text into an [Expr], folding reference-free subtrees to
// constants. It returns a positioned error for malformed input.
func Parse(src string) (Expr, error) {
	root, err := parse(src)
	if err != nil {
		return Expr{}, err
	}
	root = foldConstants(root)
	return Expr{src: src, root: root, refs: distinctRefs(root)}, nil
}

// Source returns the authored expression text.
func (e Expr) Source() string { return e.src }

// References returns the distinct parameter names the expression reads, in the
// order first seen.
func (e Expr) References() []string { return e.refs }

// Bind resolves each reference name to a stable id via resolve, marking it
// bound. It returns the names that did not resolve (the caller marks those
// parameters sick — an undefined reference is a health state, not an error).
func (e Expr) Bind(resolve func(name string) (ID, bool)) (unresolved []string) {
	if e.root == nil {
		return nil
	}
	e.root.walkRefs(func(r *refNode) {
		if id, ok := resolve(r.name); ok {
			r.id, r.bound = id, true
		} else {
			r.bound = false
			unresolved = append(unresolved, r.name)
		}
	})
	return unresolved
}

// Eval evaluates the expression against scope. A nil scope is allowed only for
// reference-free expressions.
func (e Expr) Eval(scope Scope) (Quantity, error) {
	if e.root == nil {
		return Quantity{}, fmt.Errorf("param: cannot evaluate empty expression")
	}
	return e.root.eval(scope)
}

// boundRefs returns the distinct stable ids this expression references, after
// [Expr.Bind] has resolved them. Unbound references are skipped.
func (e Expr) boundRefs() []ID {
	if e.root == nil {
		return nil
	}
	var ids []ID
	seen := map[ID]bool{}
	e.root.walkRefs(func(r *refNode) {
		if r.bound && !seen[r.id] {
			seen[r.id] = true
			ids = append(ids, r.id)
		}
	})
	return ids
}

// renameRef updates the display name of every reference bound to id, so a
// dependent expression tracks a driver's rename (references stay bound by id).
func (e Expr) renameRef(id ID, newName string) {
	if e.root == nil {
		return
	}
	e.root.walkRefs(func(r *refNode) {
		if r.bound && r.id == id {
			r.name = newName
		}
	})
}

// distinctRefs returns the reference names in the subtree, de-duplicated, in
// first-seen order.
func distinctRefs(root node) []string {
	var names []string
	seen := map[string]bool{}
	root.walkRefs(func(r *refNode) {
		if !seen[r.name] {
			seen[r.name] = true
			names = append(names, r.name)
		}
	})
	return names
}
