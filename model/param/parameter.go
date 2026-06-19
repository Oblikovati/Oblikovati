// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	"strconv"

	"oblikovati.org/api/types"
)

// ParameterKind is a parameter's category. The type, its stable ids, String, and
// Editable are defined once in the Apache-2.0 contract ([types.ParameterKind]);
// this alias keeps the historical param.ParameterKind / param.UserParam spelling
// working unchanged across the implementation (ADR-0018).
type ParameterKind = types.ParameterKind

const (
	ModelParam     = types.ModelParam
	UserParam      = types.UserParam
	ReferenceParam = types.ReferenceParam
	DerivedParam   = types.DerivedParam
	TableParam     = types.TableParam
)

// HealthStatus is a parameter's evaluation health (contract: HealthStatusEnum).
type HealthStatus uint8

const (
	// Healthy: value is current and valid.
	Healthy HealthStatus = 0
	// OutOfDate: the expression depends on parameters not yet recomputed.
	OutOfDate HealthStatus = 1
	// Failed: evaluation failed (dimensional mismatch, cycle, undefined ref).
	Failed HealthStatus = 2
)

// Health is a parameter's status plus a human-readable reason when not healthy.
// Modeling problems are health, never panics (parametric-cad skill §2).
type Health struct {
	Status HealthStatus
	Reason string
}

// OK reports whether the parameter is healthy.
func (h Health) OK() bool { return h.Status == Healthy }

// Parameter is one parametric variable: the Expression→Value→ModelValue triad
// plus identity, units, tolerance, and presentation. Value is the evaluated
// quantity in database units; ModelValue applies the tolerance. Construct
// parameters through a [Parameters] collection, never directly.
type Parameter struct {
	id     ID
	name   string
	expr   Expr
	value  Quantity
	text   string // the value when this is a Text parameter (Unit == Text)
	tol    Tolerance
	kind   ParameterKind
	health Health

	exprList    []string // multi-value choices (empty ⇒ single-valued); see expression_list.go
	allowCustom bool     // a value outside exprList is accepted (the one custom value)
	customOrder bool     // keep exprList in authored order instead of sorting it

	// modelValueType selects which value within the tolerance band the model
	// consumes; the zero value reads as Nominal (see ModelValueType).
	modelValueType ModelValueType

	Comment           string
	IsKey             bool
	Visible           bool
	Precision         int
	DisplayFormat     ParameterDisplayFormat
	ExposedAsProperty bool
	CustomProperty    CustomPropertyFormat
}

// ID returns the parameter's stable identity.
func (p *Parameter) ID() ID { return p.id }

// Name returns the display label.
func (p *Parameter) Name() string { return p.name }

// Kind returns the parameter category.
func (p *Parameter) Kind() ParameterKind { return p.kind }

// Unit returns the unit of the evaluated value.
func (p *Parameter) Unit() Unit { return p.value.Unit }

// IsText / IsBoolean / IsNumeric report the parameter's value flavor. Only numeric
// parameters support expressions and tolerances (Inventor: "Only numeric parameters
// support expressions"); text and true/false parameters carry a literal value.
func (p *Parameter) IsText() bool    { return p.value.Unit == Text }
func (p *Parameter) IsBoolean() bool { return p.value.Unit == Boolean }
func (p *Parameter) IsNumeric() bool { return !p.IsText() && !p.IsBoolean() }

// Text returns the literal of a text parameter ("" for non-text parameters).
func (p *Parameter) Text() string { return p.text }

// Bool returns the value of a true/false parameter (false for non-boolean parameters).
func (p *Parameter) Bool() bool { return p.IsBoolean() && p.value.Value != 0 }

// Value returns the evaluated quantity (database units).
func (p *Parameter) Value() Quantity { return p.value }

// ModelValue returns the value the model consumes after the tolerance band and
// the parameter's model-value selection are applied (database units).
func (p *Parameter) ModelValue() float64 {
	switch p.ModelValueType() {
	case Upper:
		return p.value.Value + p.tol.Upper
	case Lower:
		return p.value.Value + p.tol.Lower
	case Median:
		return p.value.Value + (p.tol.Upper+p.tol.Lower)/2
	default:
		return p.value.Value
	}
}

// ModelValueType returns which value within the tolerance band the model
// consumes; the zero value reads as Nominal.
func (p *Parameter) ModelValueType() ModelValueType {
	if p.modelValueType == 0 {
		return Nominal
	}
	return p.modelValueType
}

// SetModelValueType selects which value within the tolerance band the model
// consumes. It errors for non-numeric parameters and unknown selections.
func (p *Parameter) SetModelValueType(m ModelValueType) error {
	if err := p.requireNumericTolerance(); err != nil {
		return err
	}
	switch m {
	case Nominal, Upper, Lower, Median:
		p.modelValueType = m
		return nil
	default:
		return fmt.Errorf("param: unknown model value type %d for %q; want nominal/lower/upper/median", int32(m), p.name)
	}
}

// Expression returns the authored expression source.
func (p *Parameter) Expression() string { return p.expr.Source() }

// Health returns the current evaluation health.
func (p *Parameter) Health() Health { return p.health }

// Tolerance returns the engineering tolerance.
func (p *Parameter) Tolerance() Tolerance { return p.tol }

// SetTolerance sets the engineering tolerance; this changes ModelValue but not
// the nominal Value.
func (p *Parameter) SetTolerance(t Tolerance) { p.tol = t }

// SetExpression replaces the expression and re-evaluates if it is constant. A
// reference-bearing expression is left OutOfDate for the graph to recompute. It
// errors for read-only kinds and for malformed source.
func (p *Parameter) SetExpression(src string) error {
	if !p.kind.Editable() {
		return fmt.Errorf(errReadOnly, p.kind, p.name)
	}
	if !p.IsNumeric() {
		return fmt.Errorf("param: %q is a %s parameter; only numeric parameters take expressions", p.name, p.value.Unit)
	}
	e, err := Parse(src)
	if err != nil {
		return err
	}
	p.expr = e
	p.reevaluateConstant()
	return nil
}

// SetValue sets a constant value, equivalent to assigning a constant
// expression. It errors for read-only kinds.
func (p *Parameter) SetValue(q Quantity) error {
	if !p.kind.Editable() {
		return fmt.Errorf(errReadOnly, p.kind, p.name)
	}
	if !p.IsNumeric() {
		return fmt.Errorf("param: %q is a %s parameter; use SetText/SetBool", p.name, p.value.Unit)
	}
	p.expr = constantExpr(q)
	p.value = q
	p.health = Health{Status: Healthy}
	return nil
}

// SetText sets the literal of a text parameter. It errors for read-only kinds and for
// non-text parameters. The stored expression is the value quoted (Inventor surfaces a
// text parameter's equation as a quoted string).
func (p *Parameter) SetText(s string) error {
	if !p.kind.Editable() {
		return fmt.Errorf(errReadOnly, p.kind, p.name)
	}
	if !p.IsText() {
		return fmt.Errorf("param: %q is not a text parameter (unit %s)", p.name, p.value.Unit)
	}
	p.text = s
	p.expr = literalExpr(strconv.Quote(s))
	p.health = Health{Status: Healthy}
	return nil
}

// SetBool sets the value of a true/false parameter. It errors for read-only kinds and for
// non-boolean parameters. Booleans are stored as 0/1 in the value quantity.
func (p *Parameter) SetBool(b bool) error {
	if !p.kind.Editable() {
		return fmt.Errorf(errReadOnly, p.kind, p.name)
	}
	if !p.IsBoolean() {
		return fmt.Errorf("param: %q is not a true/false parameter (unit %s)", p.name, p.value.Unit)
	}
	p.value = Q(boolValue(b), Boolean)
	p.expr = literalExpr(strconv.FormatBool(b))
	p.health = Health{Status: Healthy}
	return nil
}

// boolValue maps a bool to its 0/1 stored representation.
func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// literalExpr wraps a fixed source string as an [Expr] for non-numeric parameters: its
// Source is the text the UI shows, and it is never evaluated through the graph.
func literalExpr(src string) Expr { return Expr{src: src} }

// reevaluateConstant evaluates a reference-free expression immediately; an
// expression with references is marked OutOfDate for the dependency graph.
func (p *Parameter) reevaluateConstant() {
	if p.expr.root == nil {
		return
	}
	if hasRefs(p.expr.root) {
		p.health = Health{Status: OutOfDate, Reason: "awaiting dependency recompute"}
		return
	}
	q, err := p.expr.Eval(nil)
	if err != nil {
		p.health = Health{Status: Failed, Reason: err.Error()}
		return
	}
	p.value, p.health = q, Health{Status: Healthy}
}

// constantExpr builds an [Expr] that evaluates to q, without going through the
// parser (so area/volume units, whose names contain '^', are representable).
func constantExpr(q Quantity) Expr {
	src := strconv.FormatFloat(q.Value, 'g', -1, 64) + " " + q.Unit.String()
	return Expr{src: src, root: numberNode{q: q}}
}
