// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import "oblikovati.org/api/contract"

// The occurrence model satisfies the scalar read surfaces the public contract exposes for
// assembly occurrences (ADR-0018, #728); the mutators travel over api/wire.
var (
	_ contract.ComponentOccurrence  = (*Occurrence)(nil)
	_ contract.ComponentOccurrences = (*Occurrences)(nil)
)

// The pattern model satisfies the public contract's scalar pattern read surfaces (#729);
// pattern elements are created/read as occurrences over api/wire (assembly.patternCreate).
var (
	_ contract.OccurrencePattern        = (*OccurrencePattern)(nil)
	_ contract.OccurrencePatternElement = (*OccurrencePatternElement)(nil)
)
