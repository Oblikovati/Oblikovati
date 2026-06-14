// SPDX-License-Identifier: GPL-2.0-only

package assembly

import "oblikovati.org/api/contract"

// Compile-time assertions that the constraint engine satisfies the public Apache-2.0
// contract (ADR-0018): the read surfaces an in-proc consumer binds against.
var (
	_ contract.AssemblyConstraints           = (*ConstraintSet)(nil)
	_ contract.AssemblyConstraintsEnumerator = (*OccurrenceConstraints)(nil)
	_ contract.ConstraintLimits              = (*limits)(nil)
	_ contract.MateConstraint                = (*MateConstraint)(nil)
	_ contract.FlushConstraint               = (*FlushConstraint)(nil)
	_ contract.AngleConstraint               = (*AngleConstraint)(nil)
	_ contract.TangentConstraint             = (*TangentConstraint)(nil)
	_ contract.InsertConstraint              = (*InsertConstraint)(nil)
	_ contract.AssemblySymmetryConstraint    = (*AssemblySymmetryConstraint)(nil)
)
