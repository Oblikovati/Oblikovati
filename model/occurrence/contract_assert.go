// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import "oblikovati.org/api/contract"

// The occurrence read surfaces satisfy the api/contract (ADR-0018, #1619). Most host types satisfy
// their contract interface directly; only the pattern SET needs a thin view, because its own Item
// returns the concrete *OccurrencePattern for host callers while the contract's Item returns the
// element interface. These assertions were missing since #1976 defined the interfaces — the archguard
// TestEveryContractInterfaceIsAsserted only runs with the api sibling checked out, so CI never
// flagged them.
var (
	_ contract.ComponentOccurrence      = (*Occurrence)(nil)
	_ contract.ComponentOccurrences     = (*Occurrences)(nil)
	_ contract.OccurrencePattern        = (*OccurrencePattern)(nil)
	_ contract.OccurrencePatternElement = (*OccurrencePatternElement)(nil)
	_ contract.OccurrencePatterns       = OccurrencePatternsView{}
)

// OccurrencePatternsView adapts an OccurrencePatternSet to the contract.OccurrencePatterns read
// surface — the in-proc surface an add-in reads an assembly's patterns through (ADR-0018) — by
// returning each pattern as the contract element interface.
type OccurrencePatternsView struct{ set *OccurrencePatternSet }

// AsContract exposes the set as the api/contract read surface (its element pointers satisfy
// contract.OccurrencePattern, so the view just narrows the Item return type).
func (s *OccurrencePatternSet) AsContract() OccurrencePatternsView { return OccurrencePatternsView{s} }

// Count returns how many patterns the assembly holds.
func (v OccurrencePatternsView) Count() int { return v.set.Count() }

// Item returns the i-th pattern as the contract element interface.
func (v OccurrencePatternsView) Item(i int) contract.OccurrencePattern { return v.set.Item(i) }
