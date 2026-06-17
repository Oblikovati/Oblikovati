// SPDX-License-Identifier: GPL-2.0-only

package param

import "fmt"

// The unit conversion / expression service that backs the public units API
// (Oblikovati/Oblikovati#146). It converts between registered unit names,
// resolves a unit name's category, and evaluates unit-bearing expressions
// against a document's parameter set.

// ParseUnitCategory resolves a category name (the inverse of [Unit.String], e.g.
// "length") back to its [Unit].
func ParseUnitCategory(name string) (Unit, bool) {
	for u, n := range unitNames {
		if n == name {
			return u, true
		}
	}
	return 0, false
}

// UnitCategoryOf returns the category a registered unit NAME ("mm", "deg")
// belongs to.
func UnitCategoryOf(unitName string) (Unit, bool) {
	d, ok := lookupUnit(unitName)
	if !ok {
		return 0, false
	}
	return d.category, true
}

// ConvertValue converts a value expressed in unit From to unit To. Both must be
// registered units of the SAME category; a mismatch is an error naming both.
func ConvertValue(value float64, from, to string) (float64, error) {
	fd, ok := lookupUnit(from)
	if !ok {
		return 0, fmt.Errorf("param: unknown unit %q", from)
	}
	td, ok := lookupUnit(to)
	if !ok {
		return 0, fmt.Errorf("param: unknown unit %q", to)
	}
	if fd.category != td.category {
		return 0, fmt.Errorf("param: cannot convert %q (%s) to %q (%s): different categories",
			from, fd.category, to, td.category)
	}
	return value * fd.factor / td.factor, nil
}

// EvaluateExpression parses and evaluates a unit-bearing expression against the
// parameter set, returning the database-unit [Quantity]. A parse error, an
// undefined reference, or a dimensional error is returned (a units-service
// caller wants a definite value or a clear reason it has none).
//
// Example: with width = 4 cm, EvaluateExpression("width * 2") → {8 cm, Length}.
func (ps *Parameters) EvaluateExpression(src string) (Quantity, error) {
	expr, err := Parse(src)
	if err != nil {
		return Quantity{}, err
	}
	if unresolved := expr.Bind(ps.resolver()); len(unresolved) > 0 {
		return Quantity{}, fmt.Errorf("param: expression %q references undefined parameter(s) %v", src, unresolved)
	}
	return expr.Eval(ps)
}

// ExpressionReferences returns the distinct parameter names an expression reads,
// in first-seen order (the driving parameters of the expression). It does not
// need a parameter set — it is a pure parse.
func ExpressionReferences(src string) ([]string, error) {
	expr, err := Parse(src)
	if err != nil {
		return nil, err
	}
	return expr.References(), nil
}
