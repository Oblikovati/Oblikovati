// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	"sort"
)

// Multi-value parameters (the reference API's ExpressionList): a parameter can offer a fixed list
// of allowed expressions chosen from a dropdown, optionally accepting one custom value
// outside the list. The current selection is just the parameter's expression — the
// dropdown the UI shows is the list plus the current value when it is a custom one, so
// there is at most one custom value at a time ("only one custom value", per the journey).

// SetExpressionList makes the parameter multi-value with the given choices. allowCustom
// permits selecting a value outside the list. An empty list clears the multi-value state.
// It does not change the current value. The authored order is kept (CustomOrder becomes
// true, per the reference ExpressionList); SetCustomOrder(false) re-enables sorting.
func (p *Parameter) SetExpressionList(list []string, allowCustom bool) error {
	if !p.kind.Editable() {
		return fmt.Errorf(errReadOnly, p.kind, p.name)
	}
	p.exprList = append([]string(nil), list...)
	p.allowCustom = allowCustom && len(list) > 0
	p.customOrder = len(list) > 0
	return nil
}

// ClearExpressionList drops the multi-value choices, returning the parameter to a plain
// single value (its current value is kept).
func (p *Parameter) ClearExpressionList() {
	p.exprList, p.allowCustom, p.customOrder = nil, false, false
}

// CustomOrder reports whether the multi-value choices stay in authored order
// (true) instead of being kept sorted (false).
func (p *Parameter) CustomOrder() bool { return p.customOrder }

// SetCustomOrder toggles authored-order presentation; turning it off sorts the
// current choices alphabetically (the reference behavior of CustomOrder=False).
func (p *Parameter) SetCustomOrder(custom bool) {
	p.customOrder = custom && p.IsMultiValue()
	if !custom {
		sort.Strings(p.exprList)
	}
}

// ExpressionList returns the multi-value choices (empty when single-valued).
func (p *Parameter) ExpressionList() []string {
	return append([]string(nil), p.exprList...)
}

// IsMultiValue reports whether the parameter offers a list of choices.
func (p *Parameter) IsMultiValue() bool { return len(p.exprList) > 0 }

// AllowsCustomValue reports whether a value outside the list is accepted.
func (p *Parameter) AllowsCustomValue() bool { return p.allowCustom }

// SelectValue sets the parameter to one of its list choices (or to a custom value when
// allowed), routing to the numeric or text setter. It errors when the value is not in the
// list and custom values are not allowed.
func (p *Parameter) SelectValue(expr string) error {
	if p.IsMultiValue() && !p.allowCustom && !contains(p.exprList, expr) {
		return fmt.Errorf("param: %q is not an allowed value for %q", expr, p.name)
	}
	if p.IsText() {
		return p.SetText(expr)
	}
	return p.SetExpression(expr)
}

// contains reports whether s is in list.
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
