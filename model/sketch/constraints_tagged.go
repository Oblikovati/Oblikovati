// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// Tag constraints (M06-F11, Oblikovati/Oblikovati#626): constraints that
// carry meaning rather than residuals. The text-box anchor records the
// system-owned tie between a TextBox and its anchor position and is not
// deletable on its own (it dies with its text box); the custom constraint is
// an add-in-owned named marker on sketch entities (the reference API's
// CustomConstraint scoped as a record, not a solver callback).

// NonDeletable marks constraints the user (and the wire's deleteConstraint)
// may not remove directly.
type NonDeletable interface {
	// Deletable reports whether explicit deletion is allowed.
	Deletable() bool
}

// TextBoxAnchorConstraint is the auto-created anchor of a TextBox. It has no
// residuals (the anchor is a stored position, not a solver variable) but
// appears in the constraint enumeration so callers see why the text is tied
// down, with Deletable() false.
type TextBoxAnchorConstraint struct {
	constraintBase
	Text *TextBox
}

// Residuals and Variables implement [Constraint]: an anchoring record
// constrains nothing numerically.
func (c *TextBoxAnchorConstraint) Residuals() []float64      { return nil }
func (c *TextBoxAnchorConstraint) Variables() []*math.Scalar { return nil }

// Deletable implements [NonDeletable]: the anchor lives and dies with its
// text box.
func (c *TextBoxAnchorConstraint) Deletable() bool { return false }

// CustomConstraint is an add-in-owned tag on sketch entities: a named,
// attribute-carrying record (payload travels in attribute sets, not here).
// Solver-callback semantics are explicitly out of scope (#626).
type CustomConstraint struct {
	constraintBase
	ClientID string
	Name     string
	Entities []Entity
}

// Residuals and Variables implement [Constraint]: a tag constrains nothing.
func (c *CustomConstraint) Residuals() []float64      { return nil }
func (c *CustomConstraint) Variables() []*math.Scalar { return nil }

// AddCustom tags the given entities for the owning client. ClientID is
// required — an anonymous tag could never be cleaned up by its owner.
func (g *GeometricConstraints) AddCustom(clientID, name string, entities []Entity) (*CustomConstraint, error) {
	if clientID == "" {
		return nil, fmt.Errorf("a custom constraint needs the owning add-in's clientId (name %q)", name)
	}
	c := &CustomConstraint{
		constraintBase: newConstraint(),
		ClientID:       clientID, Name: name,
		Entities: append([]Entity(nil), entities...),
	}
	g.add(c)
	return c, nil
}

// DeleteAllowed removes a constraint like Delete but refuses system-owned
// (non-deletable) ones, naming the offender.
func (g *GeometricConstraints) DeleteAllowed(c Constraint) error {
	if nd, ok := c.(NonDeletable); ok && !nd.Deletable() {
		return fmt.Errorf("constraint %d is system-owned and cannot be deleted directly", c.EntityID())
	}
	if !g.Delete(c) {
		return fmt.Errorf("constraint %d is not in this sketch", c.EntityID())
	}
	return nil
}

// anchorTextBox auto-creates the non-deletable anchor record for a new text
// box; deleteTextBoxAnchor drops it when the text box goes away.
func (s *Sketch) anchorTextBox(t *TextBox) *TextBoxAnchorConstraint {
	c := &TextBoxAnchorConstraint{constraintBase: newConstraint(), Text: t}
	s.geomCons.add(c)
	return c
}

func (s *Sketch) deleteTextBoxAnchor(t *TextBox) {
	for _, c := range s.geomCons.All() {
		if anchor, ok := c.(*TextBoxAnchorConstraint); ok && anchor.Text == t {
			s.geomCons.Delete(c)
			return
		}
	}
}
