// SPDX-License-Identifier: GPL-2.0-only

package assembly

import "oblikovati.org/solve"

// The motion constraints couple two components' DRIVEN RATES — a gear ratio, a rack-and-
// pinion pitch — rather than their static positions. Per ADR-0011 a motion coupling has
// no static-positioning residual: it constrains how the driven variables move relative to
// one another during a drive (M12-F03), not where the parts rest. So each contributes no
// residual to the static solve; it stores its ratio/distance, lists, and is driven later.

// RotateRotateConstraint couples two rotations by a gear ratio (revolutions of B per
// revolution of A).
type RotateRotateConstraint struct {
	*constraintBase
	ratio float64
}

// Value returns the gear ratio.
func (c *RotateRotateConstraint) Value() float64 { return c.ratio }

// SetValue overrides the rotate-rotate ratio (a positional representation, M12-F04).
func (c *RotateRotateConstraint) SetValue(v float64) { c.ratio = v }

// Ratio returns the gear ratio (revolutions of B per revolution of A).
func (c *RotateRotateConstraint) Ratio() float64 { return c.ratio }

// bind contributes no static residual (the coupling is applied by the drive, F03).
func (c *RotateRotateConstraint) bind(binder) []solve.Residual { return nil }

// RotateTranslateConstraint couples a rotation to a translation by the distance moved per
// revolution (rack and pinion).
type RotateTranslateConstraint struct {
	*constraintBase
	distance float64
}

// Value returns the distance moved per revolution.
func (c *RotateTranslateConstraint) Value() float64 { return c.distance }

// SetValue overrides the rotate-translate distance (a positional representation, M12-F04).
func (c *RotateTranslateConstraint) SetValue(v float64) { c.distance = v }

// Distance returns the translation moved per revolution (cm).
func (c *RotateTranslateConstraint) Distance() float64 { return c.distance }

// bind contributes no static residual (the coupling is applied by the drive, F03).
func (c *RotateTranslateConstraint) bind(binder) []solve.Residual { return nil }

// TranslateTranslateConstraint couples two translations by a ratio.
type TranslateTranslateConstraint struct {
	*constraintBase
	ratio float64
}

// Value returns the translation ratio.
func (c *TranslateTranslateConstraint) Value() float64 { return c.ratio }

// SetValue overrides the translate-translate ratio (a positional representation, M12-F04).
func (c *TranslateTranslateConstraint) SetValue(v float64) { c.ratio = v }

// Ratio returns the translation ratio (distance of B per unit distance of A).
func (c *TranslateTranslateConstraint) Ratio() float64 { return c.ratio }

// bind contributes no static residual (the coupling is applied by the drive, F03).
func (c *TranslateTranslateConstraint) bind(binder) []solve.Residual { return nil }
