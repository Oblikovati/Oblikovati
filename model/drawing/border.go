// SPDX-License-Identifier: GPL-2.0-only

package drawing

// BorderDefinition is a reusable border template: the printable-area margins inset
// (millimetres) from the sheet edge on each side. A drawing standard owns a set of
// these; V1 ships one default (custom border definitions are a follow-up).
type BorderDefinition struct {
	name                     string
	left, right, top, bottom float64
}

// DefaultBorderDefinition is the standard border: 10 mm on three sides and a wider
// 20 mm left (binding) edge — a common drafting default.
func DefaultBorderDefinition() *BorderDefinition {
	return &BorderDefinition{name: "Default", left: 20, right: 10, top: 10, bottom: 10}
}

// Name returns the border definition's name.
func (d *BorderDefinition) Name() string { return d.name }

// Border is a sheet's border — an instance of a [BorderDefinition].
type Border struct {
	def *BorderDefinition
}

func newBorder(def *BorderDefinition) *Border { return &Border{def: def} }

// DefinitionName returns the name of the border definition this border instantiates.
func (b *Border) DefinitionName() string { return b.def.name }

// Margins returns the left, right, top and bottom inset in millimetres
// (contract.DrawingBorder).
func (b *Border) Margins() (left, right, top, bottom float64) {
	return b.def.left, b.def.right, b.def.top, b.def.bottom
}
