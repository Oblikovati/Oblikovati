// SPDX-License-Identifier: GPL-2.0-only

package param

import "fmt"

// eval returns the literal value.
func (n numberNode) eval(Scope) (Quantity, error) { return n.q, nil }

// eval resolves the reference through the scope. The reference must be bound to
// a stable id first (see [Expr.Bind]); an unbound or unknown reference is an
// error the parameter layer turns into a health state.
func (n *refNode) eval(scope Scope) (Quantity, error) {
	if !n.bound {
		return Quantity{}, fmt.Errorf("param: unbound reference %q", n.name)
	}
	if scope == nil {
		return Quantity{}, fmt.Errorf("param: reference %q evaluated without a scope", n.name)
	}
	v, ok := scope.ValueOf(n.id)
	if !ok {
		return Quantity{}, fmt.Errorf("param: reference %q (id %d) has no value", n.name, n.id)
	}
	return v, nil
}

// eval applies the arithmetic operator with dimensional checking.
func (n binaryNode) eval(scope Scope) (Quantity, error) {
	lhs, err := n.lhs.eval(scope)
	if err != nil {
		return Quantity{}, err
	}
	rhs, err := n.rhs.eval(scope)
	if err != nil {
		return Quantity{}, err
	}
	switch n.op {
	case '+':
		return lhs.Add(rhs)
	case '-':
		return lhs.Sub(rhs)
	case '*':
		return lhs.Mul(rhs)
	case '/':
		return lhs.Div(rhs)
	default:
		return Quantity{}, fmt.Errorf("param: unknown operator %q", string(n.op))
	}
}

// eval negates the operand.
func (n unaryNode) eval(scope Scope) (Quantity, error) {
	v, err := n.operand.eval(scope)
	if err != nil {
		return Quantity{}, err
	}
	return v.Negate(), nil
}

// eval evaluates the arguments then applies the named built-in function.
func (n callNode) eval(scope Scope) (Quantity, error) {
	fn, ok := functions[n.fn]
	if !ok {
		return Quantity{}, fmt.Errorf("param: unknown function %q", n.fn)
	}
	args := make([]Quantity, len(n.args))
	for i, a := range n.args {
		v, err := a.eval(scope)
		if err != nil {
			return Quantity{}, err
		}
		args[i] = v
	}
	return fn(args)
}

// hasRefs reports whether the subtree contains any reference node.
func hasRefs(n node) bool {
	found := false
	n.walkRefs(func(*refNode) { found = true })
	return found
}

// foldConstants replaces reference-free subtrees with their computed value, so
// "2 * (3 + 4)" becomes a single constant. A subtree whose evaluation errors
// (e.g. a dimensional mismatch) is left intact, so the error surfaces at Eval
// as a health state rather than a parse failure.
func foldConstants(n node) node {
	switch t := n.(type) {
	case binaryNode:
		t.lhs, t.rhs = foldConstants(t.lhs), foldConstants(t.rhs)
		return tryFold(t)
	case unaryNode:
		t.operand = foldConstants(t.operand)
		return tryFold(t)
	case callNode:
		for i := range t.args {
			t.args[i] = foldConstants(t.args[i])
		}
		return tryFold(t)
	default:
		return n
	}
}

// tryFold collapses n to a constant when it has no references and evaluates
// without error.
func tryFold(n node) node {
	if hasRefs(n) {
		return n
	}
	if q, err := n.eval(nil); err == nil {
		return numberNode{q}
	}
	return n
}
